package load

import (
	"path/filepath"
	"testing"
)

func TestBuildX10TripletPlans(t *testing.T) {
	c, hash, err := ReadConfig(filepath.Join("..", "..", "..", "configs", "experiments", "s06_pilot.json"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewPilotFixture("pluto-file02-x10", 241)
	if err != nil {
		t.Fatal(err)
	}
	l, b, i, err := BuildX10TripletPlans(f, 0, c, hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Events) != 73 || len(b.Events) != 241 || len(i.Events) != 241 ||
		l.TargetTxPS != 6 || b.TargetTxPS != 20 || i.TargetLegitTPS != 6 || i.TargetInvalidTPS != 14 {
		t.Fatal("wrong three-arm rate/count contract")
	}
	for _, event := range l.Events {
		if event.Scenario != "fixed_legitimate" {
			t.Fatal("L contains a background transaction")
		}
		if b.Events[event.AccountSlot].SHA256 != event.SHA256 ||
			i.Events[event.AccountSlot].SHA256 != event.SHA256 ||
			b.Events[event.AccountSlot].OffsetNS != event.OffsetNS ||
			i.Events[event.AccountSlot].OffsetNS != event.OffsetNS {
			t.Fatal("fixed legitimate bytes or offsets differ across arms")
		}
	}
	for slot := 0; slot < 241; slot++ {
		if b.Events[slot].OffsetNS != i.Events[slot].OffsetNS ||
			len(b.Events[slot].BytesHex) != len(i.Events[slot].BytesHex) {
			t.Fatal("B/I offered timing or bytes mismatch")
		}
	}
	if _, _, _, err := BuildX10TripletPlans(f, 24, c, hash); err == nil {
		t.Fatal("invalid block accepted")
	}
}
