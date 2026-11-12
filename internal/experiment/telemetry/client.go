package telemetry

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"time"
)

// Submitter times the client's complete RPC round trip. It is deliberately
// separate from proxy ingress and ABCI CheckTx service time.
type Submitter struct {
	origin string
	client *http.Client
}

type Submission struct {
	Schema      string  `json:"schema"`
	Kind        string  `json:"kind"`
	SentUTC     string  `json:"sent_utc"`
	ObservedUTC string  `json:"observed_utc"`
	WireBytes   int     `json:"wire_bytes"`
	Status      int     `json:"http_status"`
	Code        *uint32 `json:"app_code"`
	Reason      string  `json:"reason"`
	ClientRPCNS int64   `json:"client_rpc_ns"`
}

func NewSubmitter(origin string) (*Submitter, error) {
	if _, err := localOrigin(origin); err != nil {
		return nil, err
	}
	return &Submitter{origin: origin, client: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (s *Submitter) BroadcastSync(ctx context.Context, raw []byte) (Submission, error) {
	if len(raw) == 0 || len(raw) > 8192 {
		return Submission{}, errors.New("transaction length is outside controlled RPC bounds")
	}
	u := s.origin + "/broadcast_tx_sync?tx=0x" + hex.EncodeToString(raw)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Submission{}, err
	}
	started := time.Now()
	sentUTC := started.UTC().Format(time.RFC3339Nano)
	resp, err := s.client.Do(req)
	if err != nil {
		return Submission{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return Submission{}, errors.New("invalid bounded RPC response")
	}
	latency := time.Since(started)
	code, reason := classifyRPC(resp.StatusCode, body)
	return Submission{Schema: Schema, Kind: "client_rpc", SentUTC: sentUTC, ObservedUTC: time.Now().UTC().Format(time.RFC3339Nano), WireBytes: len(raw), Status: resp.StatusCode, Code: code, Reason: reason, ClientRPCNS: latency.Nanoseconds()}, nil
}
