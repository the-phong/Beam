package load

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var mempoolFullPattern = regexp.MustCompile(`mempool is full: number of txs ([0-9]+) \(max: ([0-9]+)\), total txs bytes ([0-9]+) \(max: ([0-9]+)\)`)
var txTooLargePattern = regexp.MustCompile(`Tx too large\. Max size is ([0-9]+), but got ([0-9]+)`)

func classifyRPCAdmission(message string, nextTxBytes int) string {
	if parts := txTooLargePattern.FindStringSubmatch(message); len(parts) == 3 {
		max, _ := strconv.Atoi(parts[1])
		actual, _ := strconv.Atoi(parts[2])
		if actual == nextTxBytes && actual > max {
			return "mempool_tx_too_large"
		}
	}
	parts := mempoolFullPattern.FindStringSubmatch(message)
	if len(parts) != 5 {
		return "rpc_reject"
	}
	count, _ := strconv.ParseInt(parts[1], 10, 64)
	maxCount, _ := strconv.ParseInt(parts[2], 10, 64)
	bytes, _ := strconv.ParseInt(parts[3], 10, 64)
	maxBytes, _ := strconv.ParseInt(parts[4], 10, 64)
	bytesWouldOverflow := nextTxBytes > 0 && bytes > maxBytes-int64(nextTxBytes)
	if count >= maxCount && bytesWouldOverflow {
		return "mempool_count_and_bytes_full"
	}
	if count >= maxCount {
		return "mempool_count_full"
	}
	if bytesWouldOverflow {
		return "mempool_bytes_full"
	}
	return "mempool_full_unresolved"
}

const (
	StatusOK             = "OK"
	StatusRateMiss       = "RATE_MISS"
	StatusSafetyAbort    = "SAFETY_ABORT"
	StatusOracleMismatch = "ORACLE_MISMATCH"
)

type SendResult struct {
	WrittenAt time.Time
	Completed time.Time
	Code      *uint32
	Reason    string
	Err       error
}

type Sender interface {
	Send(context.Context, []byte) SendResult
}

type Guard interface {
	Check(context.Context) error
}

type GuardFunc func(context.Context) error

func (f GuardFunc) Check(ctx context.Context) error { return f(ctx) }

type EventResult struct {
	ID              string  `json:"id"`
	Traffic         string  `json:"traffic"`
	Source          string  `json:"source"`
	PlannedUTC      string  `json:"planned_utc"`
	ActualSendUTC   *string `json:"actual_send_utc"`
	CompletedUTC    *string `json:"completed_utc"`
	PlannedOffsetNS int64   `json:"planned_offset_ns"`
	ActualOffsetNS  *int64  `json:"actual_offset_ns"`
	WireBytes       int     `json:"wire_bytes"`
	ExpectedCode    uint32  `json:"expected_code"`
	ActualCode      *uint32 `json:"actual_code"`
	Reason          string  `json:"reason"`
	SubmissionError string  `json:"submission_error,omitempty"`
}

type Result struct {
	Version         int           `json:"version"`
	RunID           string        `json:"run_id"`
	ConfigSHA256    string        `json:"config_sha256"`
	PlanSHA256      string        `json:"plan_sha256"`
	GenesisSHA256   string        `json:"genesis_sha256"`
	Profile         byte          `json:"profile"`
	Regime          string        `json:"regime"`
	Status          string        `json:"status"`
	Reason          string        `json:"reason,omitempty"`
	StartedUTC      string        `json:"started_utc"`
	CompletedUTC    string        `json:"completed_utc"`
	PlannedCount    int           `json:"planned_count"`
	ActualSendCount int           `json:"actual_send_count"`
	PlannedBytes    int           `json:"planned_bytes"`
	ActualSendBytes int           `json:"actual_send_bytes"`
	Events          []EventResult `json:"events"`
}

type HTTPSender struct {
	origin string
	client *http.Client
}

// NewHTTPSender refuses a DNS name, proxy, redirect or non-loopback target.
// Thus a plan cannot move this generator to a public network by changing a URL.
func NewHTTPSender(raw string, timeout time.Duration) (*HTTPSender, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("RPC must be an HTTP origin on 127.0.0.1")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil || host != "127.0.0.1" || port == "" {
		return nil, errors.New("RPC must be an HTTP origin on 127.0.0.1")
	}
	if parsed, err := strconv.Atoi(port); err != nil || parsed < 1 || parsed > 65535 || timeout <= 0 {
		return nil, errors.New("invalid private RPC port or timeout")
	}
	tr := &http.Transport{Proxy: nil, MaxIdleConnsPerHost: 16}
	return &HTTPSender{origin: raw, client: &http.Client{Timeout: timeout, Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("redirect prohibited for isolated load")
	}}}, nil
}

func (s *HTTPSender) Send(ctx context.Context, raw []byte) SendResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.origin+"/broadcast_tx_sync?tx=0x"+hex.EncodeToString(raw), nil)
	if err != nil {
		return SendResult{Err: err}
	}
	var written time.Time
	var writtenMu sync.Mutex
	request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				writtenMu.Lock()
				written = time.Now()
				writtenMu.Unlock()
			}
		},
	}))
	response, err := s.client.Do(request)
	completed := time.Now()
	writtenMu.Lock()
	wroteAt := written
	writtenMu.Unlock()
	if err != nil {
		return SendResult{WrittenAt: wroteAt, Completed: completed, Err: err}
	}
	defer response.Body.Close()
	// This bounded decoder exports only an application code and a fixed reason.
	var wire struct {
		Result struct {
			Code *uint32 `json:"code"`
		} `json:"result"`
		Error struct {
			Message string `json:"message"`
			Data    string `json:"data"`
		} `json:"error"`
	}
	if err := decodeBoundedJSON(response.Body, &wire); err != nil {
		return SendResult{WrittenAt: wroteAt, Completed: completed, Reason: "rpc_response_invalid", Err: err}
	}
	if response.StatusCode != http.StatusOK {
		return SendResult{WrittenAt: wroteAt, Completed: completed, Reason: classifyRPCAdmission(wire.Error.Message+" "+wire.Error.Data, len(raw)), Err: fmt.Errorf("RPC HTTP %d", response.StatusCode)}
	}
	if wire.Result.Code == nil {
		reason := classifyRPCAdmission(wire.Error.Message+" "+wire.Error.Data, len(raw))
		if reason == "rpc_reject" {
			reason = "rpc_code_missing"
		}
		return SendResult{WrittenAt: wroteAt, Completed: completed, Reason: reason, Err: errors.New("missing CheckTx code")}
	}
	return SendResult{WrittenAt: wroteAt, Completed: completed, Code: wire.Result.Code, Reason: "app_code"}
}

func decodeBoundedJSON(input io.Reader, target any) error {
	raw, err := io.ReadAll(io.LimitReader(input, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return errors.New("RPC response exceeds size bound")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("invalid RPC response length or trailing document")
	}
	return nil
}

type workItem struct {
	index int
	raw   []byte
}

type workResult struct {
	index int
	sent  SendResult
}

// Run schedules all sends from absolute monotonic deadlines. A full work
// queue or missed offered rate is a failed run; it never lowers the plan.
func Run(parent context.Context, p Plan, c Config, configHash, planHash string, sender Sender, guard Guard) (Result, error) {
	if err := p.Validate(c, configHash); err != nil {
		return Result{}, err
	}
	if !validHash(planHash) || sender == nil || guard == nil {
		return Result{}, ErrPlan
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(c.Safety.MaxDurationSeconds)*time.Second)
	defer cancel()
	if err := guard.Check(ctx); err != nil {
		return Result{}, fmt.Errorf("preflight safety: %w", err)
	}
	start := time.Now().Add(100 * time.Millisecond)
	result := Result{Version: 1, RunID: p.RunID, ConfigSHA256: configHash, PlanSHA256: planHash, GenesisSHA256: p.GenesisSHA256, Profile: byte(p.Profile), Regime: p.Regime, Status: StatusOK, StartedUTC: start.UTC().Format(time.RFC3339Nano), PlannedCount: len(p.Events), Events: make([]EventResult, len(p.Events))}
	for i, e := range p.Events {
		raw, _ := hex.DecodeString(e.BytesHex)
		result.PlannedBytes += len(raw)
		result.Events[i] = EventResult{ID: e.ID, Traffic: e.Traffic, Source: e.Source, PlannedUTC: start.Add(time.Duration(e.OffsetNS)).UTC().Format(time.RFC3339Nano), PlannedOffsetNS: e.OffsetNS, WireBytes: len(raw), ExpectedCode: uint32(e.ExpectedCode)}
	}
	workers := min(4, c.Safety.MaxOutstanding)
	tasks := make(chan workItem, c.Safety.MaxOutstanding)
	completed := make(chan workResult, len(p.Events))
	semaphore := make(chan struct{}, c.Safety.MaxOutstanding)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				completed <- workResult{index: task.index, sent: sender.Send(ctx, task.raw)}
				<-semaphore
			}
		}()
	}
	guardFailure := make(chan error, 1)
	stopGuard := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Duration(c.Network.CollectorIntervalMS) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopGuard:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := guard.Check(ctx); err != nil {
					select {
					case guardFailure <- err:
					default:
					}
					cancel()
					return
				}
			}
		}
	}()
	setFailure := func(status, reason string) {
		if result.Status == StatusOK {
			result.Status, result.Reason = status, reason
		}
		cancel()
	}
	for i, e := range p.Events {
		deadline := start.Add(time.Duration(e.OffsetNS))
		if wait := time.Until(deadline); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-ctx.Done():
			}
			timer.Stop()
		}
		select {
		case err := <-guardFailure:
			setFailure(StatusSafetyAbort, err.Error())
		default:
		}
		if ctx.Err() != nil {
			if result.Status == StatusOK {
				setFailure(StatusSafetyAbort, "run context canceled or duration cap reached")
			}
			break
		}
		select {
		case semaphore <- struct{}{}:
			raw, _ := hex.DecodeString(e.BytesHex)
			tasks <- workItem{index: i, raw: raw}
		default:
			setFailure(StatusRateMiss, "bounded submission queue full")
		}
		if result.Status != StatusOK {
			break
		}
	}
	close(tasks)
	workerDone := make(chan struct{})
	go func() { wg.Wait(); close(workerDone) }()
	select {
	case <-workerDone:
	case <-ctx.Done():
		<-workerDone // HTTP requests share the canceled context and bounded timeout.
	}
	close(completed)
	for item := range completed {
		e := &result.Events[item.index]
		sent := item.sent
		if !sent.WrittenAt.IsZero() {
			actualUTC := sent.WrittenAt.UTC().Format(time.RFC3339Nano)
			actualOffset := sent.WrittenAt.Sub(start).Nanoseconds()
			e.ActualSendUTC, e.ActualOffsetNS = &actualUTC, &actualOffset
			e.ActualCode = sent.Code
			e.Reason = sent.Reason
			result.ActualSendCount++
			result.ActualSendBytes += e.WireBytes
		}
		if !sent.Completed.IsZero() {
			completedUTC := sent.Completed.UTC().Format(time.RFC3339Nano)
			e.CompletedUTC = &completedUTC
		}
		if sent.Err != nil {
			e.SubmissionError = classifySendError(sent.Err)
		}
	}
	if result.Status == StatusOK {
		if err := rateQuality(p, result.Events); err != nil {
			result.Status, result.Reason = StatusRateMiss, err.Error()
		}
	}
	if result.Status == StatusOK {
		for _, e := range result.Events {
			if e.SubmissionError != "" || e.ActualCode == nil || *e.ActualCode != e.ExpectedCode {
				result.Status, result.Reason = StatusOracleMismatch, "RPC CheckTx code or submission outcome differs from plan"
				break
			}
		}
	}
	if result.Status == StatusOK {
		deadline := start.Add(time.Duration(p.WarmupNS + p.SteadyNS + p.DrainNS))
		if wait := time.Until(deadline); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-ctx.Done():
			}
			timer.Stop()
		}
		if err := ctx.Err(); err != nil {
			result.Status, result.Reason = StatusSafetyAbort, "drain interrupted"
		}
	}
	select {
	case err := <-guardFailure:
		result.Status, result.Reason = StatusSafetyAbort, err.Error()
	default:
	}
	close(stopGuard)
	result.CompletedUTC = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func classifySendError(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "cancelled_or_deadline"
	}
	return "transport_or_rpc_error"
}

func rateQuality(p Plan, actual []EventResult) error {
	// A measured offered send is the HTTP WroteRequest callback, not the
	// scheduler's queue insertion or the later RPC response.
	var plannedSteady, actualSteady, plannedBytes, actualBytes int
	var plannedLegit, actualLegit, plannedInvalid, actualInvalid int
	plannedByWindow := map[int64]int{}
	actualByWindow := map[int64]int{}
	plannedSplit := map[int64]map[string]int{}
	actualSplit := map[int64]map[string]int{}
	for i, event := range actual {
		planned := p.Events[i]
		if planned.OffsetNS < p.WarmupNS {
			continue
		}
		plannedSteady++
		plannedBytes += event.WireBytes
		plannedWindow := (planned.OffsetNS - p.WarmupNS) / p.WindowNS
		plannedByWindow[plannedWindow]++
		if plannedSplit[plannedWindow] == nil {
			plannedSplit[plannedWindow] = map[string]int{}
		}
		plannedSplit[plannedWindow][planned.Traffic]++
		if planned.Traffic == "legitimate" {
			plannedLegit++
		} else {
			plannedInvalid++
		}
		if event.ActualOffsetNS != nil && *event.ActualOffsetNS >= p.WarmupNS && *event.ActualOffsetNS < p.WarmupNS+p.SteadyNS {
			actualSteady++
			actualBytes += event.WireBytes
			actualWindow := (*event.ActualOffsetNS - p.WarmupNS) / p.WindowNS
			actualByWindow[actualWindow]++
			if actualSplit[actualWindow] == nil {
				actualSplit[actualWindow] = map[string]int{}
			}
			actualSplit[actualWindow][planned.Traffic]++
			if planned.Traffic == "legitimate" {
				actualLegit++
			} else {
				actualInvalid++
			}
		}
	}
	if plannedSteady == 0 || float64(actualSteady)/float64(plannedSteady) < .95 || float64(actualBytes)/float64(plannedBytes) < .95 {
		return errors.New("steady offered total below 95% of plan")
	}
	for window, expected := range plannedByWindow {
		if actualByWindow[window] < int(.9*float64(expected)+.999999) {
			return fmt.Errorf("steady window %d below 90%% of planned sends", window)
		}
		if p.Waveform == "burst" && actualByWindow[window] > int(1.1*float64(expected)) {
			return fmt.Errorf("burst window %d exceeds planned send count", window)
		}
		if p.Regime == FixedLegitimatePlusAdversary {
			for _, traffic := range []string{"legitimate", "adversarial"} {
				if n := plannedSplit[window][traffic]; n > 0 && actualSplit[window][traffic] < int(.9*float64(n)+.999999) {
					return fmt.Errorf("steady %s window %d below 90%% of plan", traffic, window)
				}
			}
		}
	}
	if p.Regime == FixedLegitimatePlusAdversary && (float64(actualLegit)/float64(plannedLegit) < .95 || float64(actualInvalid)/float64(plannedInvalid) < .95) {
		return errors.New("fixed legitimate or invalid offered stream below 95% of plan")
	}
	for window, count := range actualByWindow {
		if plannedByWindow[window] == 0 && count > 0 {
			return fmt.Errorf("unexpected send in planned-zero window %d", window)
		}
	}
	return nil
}
