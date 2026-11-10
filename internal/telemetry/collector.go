package telemetry

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProcessSample struct {
	PID        int    `json:"pid"`
	StartTicks uint64 `json:"start_ticks"`
	CPU_NS     uint64 `json:"cpu_ns"`
	RSSBytes   uint64 `json:"rss_bytes"`
}

// NodeSample contains only metrics whose source is independently observable.
// A nil pointer means NOT MEASURED, never zero.
type NodeSample struct {
	Schema                    string         `json:"schema"`
	Kind                      string         `json:"kind"`
	SampleStartedUTC          string         `json:"sample_started_utc"`
	ObservedUTC               string         `json:"observed_utc"`
	SampleDurationNS          int64          `json:"sample_duration_ns"`
	AcceptedMempoolCount      *int64         `json:"accepted_mempool_count"`
	AcceptedMempoolBytes      *int64         `json:"accepted_mempool_bytes"`
	FailedTxTotal             *float64       `json:"failed_tx_total"`
	ResourceRejectedTxTotal   *float64       `json:"resource_rejected_tx_total"`
	RecheckTotal              *float64       `json:"recheck_total"`
	RelayReceiveEnvelopeBytes *float64       `json:"relay_receive_envelope_bytes"`
	RelaySendEnvelopeBytes    *float64       `json:"relay_send_envelope_bytes"`
	P2PIngressTxTotal         *float64       `json:"p2p_ingress_tx_total"`
	P2PIngressTxBytesTotal    *float64       `json:"p2p_ingress_tx_bytes_total"`
	P2PIngressTxPerSecond     *float64       `json:"p2p_ingress_tx_per_second"`
	P2PIngressBytesPerSecond  *float64       `json:"p2p_ingress_bytes_per_second"`
	CheckTxSocketQueueCount   *float64       `json:"checktx_socket_queue_count"`
	CheckTxSocketQueueSeconds *float64       `json:"checktx_socket_queue_seconds"`
	Comet                     *ProcessSample `json:"comet_process"`
	Application               *ProcessSample `json:"application_process"`
	Unavailable               []string       `json:"unavailable,omitempty"`
	DroppedSamples            uint64         `json:"dropped_samples"`
}

type Collector struct {
	client              *http.Client
	rpcURL              string
	metricsURL          string
	cometPID            int
	appPID              int
	clockTicks          uint64
	dropped             uint64
	lastP2PAt           time.Time
	lastP2PTx           float64
	lastP2PBytes        float64
	lastCometStartTicks uint64
}

func NewCollector(rpcURL, metricsURL string, cometPID, appPID int) (*Collector, error) {
	for _, raw := range []string{rpcURL, metricsURL} {
		if _, err := localOrigin(raw); err != nil {
			return nil, err
		}
	}
	if cometPID <= 0 || appPID <= 0 {
		return nil, errors.New("both process PIDs are required")
	}
	rawTicks, err := exec.Command("getconf", "CLK_TCK").Output()
	if err != nil {
		return nil, fmt.Errorf("read process clock tick rate: %w", err)
	}
	ticks, err := strconv.ParseUint(strings.TrimSpace(string(rawTicks)), 10, 64)
	if err != nil || ticks == 0 {
		return nil, errors.New("invalid process clock tick rate")
	}
	return &Collector{client: &http.Client{Timeout: 2 * time.Second}, rpcURL: rpcURL, metricsURL: metricsURL, cometPID: cometPID, appPID: appPID, clockTicks: ticks}, nil
}

func localOrigin(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return nil, errors.New("collector endpoints must be local HTTP origins")
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, errors.New("collector endpoints must bind to loopback")
	}
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return nil, errors.New("collector endpoints must bind to loopback")
		}
	}
	return u, nil
}

func (c *Collector) Sample(ctx context.Context) NodeSample {
	started := time.Now()
	s := NodeSample{Schema: Schema, Kind: "node_sample", SampleStartedUTC: started.UTC().Format(time.RFC3339Nano)}
	if err := c.mempool(ctx, &s); err != nil {
		s.Unavailable = append(s.Unavailable, "accepted_mempool:"+err.Error())
	}
	if err := c.prometheus(ctx, &s); err != nil {
		s.Unavailable = append(s.Unavailable, "comet_metrics:"+err.Error())
	}
	if p, err := readProcess(c.cometPID, c.clockTicks); err == nil {
		s.Comet = &p
	} else {
		s.Unavailable = append(s.Unavailable, "comet_process:"+err.Error())
	}
	if p, err := readProcess(c.appPID, c.clockTicks); err == nil {
		s.Application = &p
	} else {
		s.Unavailable = append(s.Unavailable, "application_process:"+err.Error())
	}
	if len(s.Unavailable) != 0 {
		c.dropped++
	}
	s.DroppedSamples = c.dropped
	completed := time.Now()
	if s.Comet != nil && s.P2PIngressTxTotal != nil && s.P2PIngressTxBytesTotal != nil {
		if !c.lastP2PAt.IsZero() && c.lastCometStartTicks == s.Comet.StartTicks &&
			*s.P2PIngressTxTotal >= c.lastP2PTx && *s.P2PIngressTxBytesTotal >= c.lastP2PBytes {
			elapsed := completed.Sub(c.lastP2PAt).Seconds()
			if elapsed > 0 {
				txRate := (*s.P2PIngressTxTotal - c.lastP2PTx) / elapsed
				byteRate := (*s.P2PIngressTxBytesTotal - c.lastP2PBytes) / elapsed
				s.P2PIngressTxPerSecond = &txRate
				s.P2PIngressBytesPerSecond = &byteRate
			}
		}
		c.lastP2PAt = completed
		c.lastP2PTx = *s.P2PIngressTxTotal
		c.lastP2PBytes = *s.P2PIngressTxBytesTotal
		c.lastCometStartTicks = s.Comet.StartTicks
	} else {
		c.lastP2PAt = time.Time{}
	}
	s.SampleDurationNS = completed.Sub(started).Nanoseconds()
	s.ObservedUTC = completed.UTC().Format(time.RFC3339Nano)
	return s
}

func (c *Collector) get(ctx context.Context, raw string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (c *Collector) mempool(ctx context.Context, s *NodeSample) error {
	body, err := c.get(ctx, c.rpcURL+"/num_unconfirmed_txs")
	if err != nil {
		return err
	}
	defer body.Close()
	var wire struct {
		Result struct {
			Total      string `json:"total"`
			TotalBytes string `json:"total_bytes"`
		} `json:"result"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(body, 1<<20)).Decode(&wire); err != nil {
		return err
	}
	if len(wire.Error) != 0 && string(wire.Error) != "null" {
		return errors.New("RPC error")
	}
	count, err := strconv.ParseInt(wire.Result.Total, 10, 64)
	if err != nil || count < 0 {
		return errors.New("invalid mempool count")
	}
	bytes, err := strconv.ParseInt(wire.Result.TotalBytes, 10, 64)
	if err != nil || bytes < 0 {
		return errors.New("invalid mempool byte count")
	}
	s.AcceptedMempoolCount, s.AcceptedMempoolBytes = &count, &bytes
	return nil
}

func (c *Collector) prometheus(ctx context.Context, s *NodeSample) error {
	body, err := c.get(ctx, c.metricsURL+"/metrics")
	if err != nil {
		return err
	}
	defer body.Close()
	raw, err := io.ReadAll(io.LimitReader(body, (4<<20)+1))
	if err != nil || len(raw) > 4<<20 {
		return errors.New("invalid bounded metrics response")
	}
	var failed, rejected, recheck, p2pTxs, p2pBytes, queueCount, queueSeconds *float64
	values := map[string]**float64{
		"cometbft_mempool_failed_txs":                    &failed,
		"cometbft_mempool_rejected_txs":                  &rejected,
		"cometbft_mempool_recheck_times":                 &recheck,
		"pluto_comet_p2p_ingress_txs_total":              &p2pTxs,
		"pluto_comet_p2p_ingress_tx_bytes_total":         &p2pBytes,
		"pluto_comet_checktx_socket_queue_seconds_count": &queueCount,
		"pluto_comet_checktx_socket_queue_seconds_sum":   &queueSeconds,
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var send, receive float64
	var sendSeen, receiveSeen bool
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.SplitN(fields[0], "{", 2)[0]
		value, err := strconv.ParseFloat(fields[1], 64)
		if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		if target, ok := values[name]; ok {
			*target = &value
		}
		if !strings.Contains(fields[0], `chID="0x30"`) {
			continue
		}
		switch name {
		case "cometbft_p2p_peer_send_bytes_total":
			send += value
			sendSeen = true
		case "cometbft_p2p_peer_receive_bytes_total":
			receive += value
			receiveSeen = true
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	s.FailedTxTotal, s.ResourceRejectedTxTotal, s.RecheckTotal = failed, rejected, recheck
	s.P2PIngressTxTotal, s.P2PIngressTxBytesTotal = p2pTxs, p2pBytes
	s.CheckTxSocketQueueCount, s.CheckTxSocketQueueSeconds = queueCount, queueSeconds
	if sendSeen {
		s.RelaySendEnvelopeBytes = &send
	}
	if receiveSeen {
		s.RelayReceiveEnvelopeBytes = &receive
	}
	return nil
}

func readProcess(pid int, clockTicks uint64) (ProcessSample, error) {
	root := filepath.Join("/proc", strconv.Itoa(pid))
	rawCPU, err := os.ReadFile(filepath.Join(root, "stat"))
	if err != nil {
		return ProcessSample{}, err
	}
	closeParen := strings.LastIndexByte(string(rawCPU), ')')
	if closeParen < 0 {
		return ProcessSample{}, errors.New("invalid process stat")
	}
	fields := strings.Fields(string(rawCPU[closeParen+1:]))
	if len(fields) < 20 {
		return ProcessSample{}, errors.New("incomplete process stat")
	}
	user, err := strconv.ParseUint(fields[11], 10, 64) // /proc stat field 14
	if err != nil {
		return ProcessSample{}, err
	}
	system, err := strconv.ParseUint(fields[12], 10, 64) // field 15
	if err != nil {
		return ProcessSample{}, err
	}
	startTicks, err := strconv.ParseUint(fields[19], 10, 64) // field 22
	if err != nil {
		return ProcessSample{}, err
	}
	rawMem, err := os.ReadFile(filepath.Join(root, "statm"))
	if err != nil {
		return ProcessSample{}, err
	}
	fields = strings.Fields(string(rawMem))
	if len(fields) < 2 {
		return ProcessSample{}, errors.New("invalid statm")
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return ProcessSample{}, err
	}
	return ProcessSample{PID: pid, StartTicks: startTicks, CPU_NS: (user + system) * uint64(time.Second) / clockTicks, RSSBytes: pages * uint64(os.Getpagesize())}, nil
}
