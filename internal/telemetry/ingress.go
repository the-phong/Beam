package telemetry

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// RPCEvent is observed by the controlled localhost proxy before Comet's RPC
// handler. It intentionally has no raw transaction, hash, account or label.
type RPCEvent struct {
	Schema      string  `json:"schema"`
	Kind        string  `json:"kind"`
	IngressUTC  string  `json:"ingress_utc"`
	ObservedUTC string  `json:"observed_utc"`
	Source      string  `json:"source"`
	WireBytes   int     `json:"wire_bytes"`
	Status      int     `json:"http_status"`
	Code        *uint32 `json:"app_code"`
	Reason      string  `json:"reason"`
	RoundTripNS int64   `json:"proxy_roundtrip_ns"`
}

type IngressProxy struct {
	backend *url.URL
	client  *http.Client
	events  chan RPCEvent
	dropped atomic.Uint64
}

func NewIngressProxy(backend string, capacity int) (*IngressProxy, error) {
	u, err := localOrigin(backend)
	if err != nil {
		return nil, err
	}
	if capacity < 1 {
		return nil, errors.New("proxy event capacity must be positive")
	}
	return &IngressProxy{backend: u, client: &http.Client{Timeout: 30 * time.Second}, events: make(chan RPCEvent, capacity)}, nil
}

func (p *IngressProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || (r.URL.Path != "/broadcast_tx_sync" && r.URL.Path != "/broadcast_tx_commit") {
		http.Error(w, "unsupported RPC method", http.StatusNotFound)
		return
	}
	received := time.Now()
	ingressUTC := received.UTC().Format(time.RFC3339Nano)
	rawHex := r.URL.Query().Get("tx")
	if !strings.HasPrefix(rawHex, "0x") || len(rawHex) < 4 || len(rawHex) > 16386 || (len(rawHex)-2)%2 != 0 {
		http.Error(w, "invalid transaction encoding", http.StatusBadRequest)
		p.record(RPCEvent{IngressUTC: ingressUTC, Status: http.StatusBadRequest, Reason: "proxy_encoding", RoundTripNS: time.Since(received).Nanoseconds()})
		return
	}
	raw, err := hex.DecodeString(rawHex[2:])
	if err != nil {
		http.Error(w, "invalid transaction encoding", http.StatusBadRequest)
		p.record(RPCEvent{IngressUTC: ingressUTC, Status: http.StatusBadRequest, Reason: "proxy_encoding", RoundTripNS: time.Since(received).Nanoseconds()})
		return
	}
	target := *p.backend
	target.Path = r.URL.Path
	target.RawQuery = r.URL.RawQuery
	forward, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		http.Error(w, "proxy request failed", http.StatusBadGateway)
		p.record(RPCEvent{IngressUTC: ingressUTC, WireBytes: len(raw), Status: http.StatusBadGateway, Reason: "proxy_request_invalid", RoundTripNS: time.Since(received).Nanoseconds()})
		return
	}
	resp, err := p.client.Do(forward)
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		p.record(RPCEvent{IngressUTC: ingressUTC, WireBytes: len(raw), Status: http.StatusBadGateway, Reason: "transport_unavailable", RoundTripNS: time.Since(received).Nanoseconds()})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		http.Error(w, "upstream response invalid", http.StatusBadGateway)
		p.record(RPCEvent{IngressUTC: ingressUTC, WireBytes: len(raw), Status: http.StatusBadGateway, Reason: "upstream_invalid", RoundTripNS: time.Since(received).Nanoseconds()})
		return
	}
	latency := time.Since(received)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
	code, reason := classifyRPC(resp.StatusCode, body)
	p.record(RPCEvent{IngressUTC: ingressUTC, WireBytes: len(raw), Status: resp.StatusCode, Code: code, Reason: reason, RoundTripNS: latency.Nanoseconds()})
}

func classifyRPC(status int, body []byte) (*uint32, string) {
	var result struct {
		Result struct {
			Code    *uint32 `json:"code"`
			CheckTx struct {
				Code *uint32 `json:"code"`
			} `json:"check_tx"`
		} `json:"result"`
		Error struct {
			Message string `json:"message"`
			Data    string `json:"data"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "unknown_response"
	}
	code := result.Result.Code
	if code == nil {
		code = result.Result.CheckTx.Code
	}
	if code != nil {
		if *code == 0 {
			return code, "accepted_by_checktx"
		}
		return code, "app_reject"
	}
	message := strings.ToLower(result.Error.Message + " " + result.Error.Data)
	if strings.Contains(message, "tx already exists in cache") {
		return nil, "duplicate_cache"
	}
	if strings.Contains(message, "mempool is full") || strings.Contains(message, "mempool full") {
		return nil, "mempool_full"
	}
	if status >= 500 {
		return nil, "precheck_or_rpc_unknown"
	}
	return nil, "unknown_response"
}

func (p *IngressProxy) record(e RPCEvent) {
	e.Schema = Schema
	e.Kind = "rpc_proxy"
	e.Source = "rpc"
	e.ObservedUTC = time.Now().UTC().Format(time.RFC3339Nano)
	select {
	case p.events <- e:
	default:
		p.dropped.Add(1)
	}
}

func (p *IngressProxy) Drain() ([]RPCEvent, uint64) {
	var result []RPCEvent
	for {
		select {
		case e := <-p.events:
			result = append(result, e)
		default:
			return result, p.dropped.Swap(0)
		}
	}
}
