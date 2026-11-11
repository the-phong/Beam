// Package telemetry records bounded, publication-safe observations from the
// controlled-profile application. It never stores transaction or key bytes.
package telemetry

import (
	"encoding/json"
	"io"
	"sync/atomic"
	"time"
)

const Schema = "pluto.telemetry.v1"

type Span struct {
	Stage      string `json:"stage"`
	DurationNS int64  `json:"duration_ns"`
}

type CheckTxEvent struct {
	Schema        string `json:"schema"`
	Kind          string `json:"kind"`
	Sequence      uint64 `json:"sequence"`
	ObservedUTC   string `json:"observed_utc"`
	Profile       uint8  `json:"profile"`
	CheckType     string `json:"check_type"`
	Source        string `json:"source"`
	WireBytes     int    `json:"wire_bytes"`
	Code          uint32 `json:"code"`
	LastStage     string `json:"last_stage"`
	ServiceNS     int64  `json:"service_ns"`
	AppLockWaitNS int64  `json:"app_lock_wait_ns"`
	Spans         []Span `json:"spans,omitempty"`
}

// Recorder drops excess samples instead of delaying consensus or admission.
// Capacity is fixed at construction and Drain is the only export operation.
type Recorder struct {
	sequence atomic.Uint64
	dropped  atomic.Uint64
	events   chan CheckTxEvent
}

func NewRecorder(capacity int) *Recorder {
	if capacity < 1 {
		capacity = 1
	}
	return &Recorder{events: make(chan CheckTxEvent, capacity)}
}

func (r *Recorder) Record(e CheckTxEvent) {
	if r == nil {
		return
	}
	e.Schema = Schema
	e.Kind = "check_tx"
	e.Sequence = r.sequence.Add(1)
	e.ObservedUTC = time.Now().UTC().Format(time.RFC3339Nano)
	e.Source = "unknown"
	select {
	case r.events <- e:
	default:
		r.dropped.Add(1)
	}
}

func (r *Recorder) Drain() ([]CheckTxEvent, uint64) {
	if r == nil {
		return nil, 0
	}
	var events []CheckTxEvent
	for {
		select {
		case e := <-r.events:
			events = append(events, e)
		default:
			return events, r.dropped.Swap(0)
		}
	}
}

// WriteJSONL writes an already drained batch. Callers keep this off the
// consensus and CheckTx path; a failed write never changes validation.
func WriteJSONL(w io.Writer, events []CheckTxEvent) error {
	enc := json.NewEncoder(w)
	for _, e := range events {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return nil
}
