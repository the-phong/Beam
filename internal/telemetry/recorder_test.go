package telemetry

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRecorderBoundedAndSafeSchema(t *testing.T) {
	r := NewRecorder(1)
	r.Record(CheckTxEvent{Profile: 4, CheckType: "new", WireBytes: 2565, Code: 13, LastStage: "ml", Spans: []Span{{Stage: "ml", DurationNS: 12}}})
	r.Record(CheckTxEvent{Profile: 4, CheckType: "new"})
	events, dropped := r.Drain()
	if len(events) != 1 || dropped != 1 || events[0].Sequence != 1 || events[0].Schema != Schema || events[0].ObservedUTC == "" || events[0].Source != "unknown" {
		t.Fatalf("invalid bounded recording: %+v, drops=%d", events, dropped)
	}
	var out bytes.Buffer
	if err := WriteJSONL(&out, events); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &decoded); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"tx", "raw", "sender", "receiver", "key", "signature", "case_id"} {
		if _, exists := decoded[forbidden]; exists {
			t.Fatalf("private or label field %q in telemetry", forbidden)
		}
	}
	if more, n := r.Drain(); len(more) != 0 || n != 0 {
		t.Fatalf("drain did not reset: %+v %d", more, n)
	}
}
