package load

import (
	"testing"
	"time"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func TestPrecisionPilotPlanStaysBoundedAndDistinct(t *testing.T) {
	c, hash := pilotConfig(t)
	f, err := NewPilotFixture("pluto-t06-precision", 241)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPrecisionPilotPlan(f, researchv2.ProfileH, "precision-unit", 4406, c, hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Events) != 241 || plan.SteadyNS != int64(12*time.Second) ||
		plan.TargetTxPS != 20 || plan.WarmupNS+plan.SteadyNS+plan.DrainNS > int64(time.Duration(c.Safety.MaxDurationSeconds)*time.Second) {
		t.Fatal("precision plan exceeds locked bounds")
	}
	seen := make(map[string]bool)
	for index, event := range plan.Events {
		if seen[event.SHA256] || event.AccountSlot != index || event.Nonce != 0 || event.Traffic != "legitimate" {
			t.Fatalf("non-distinct precision event %d", index)
		}
		seen[event.SHA256] = true
	}
	f.Senders = f.Senders[:240]
	if _, err := BuildPrecisionPilotPlan(f, researchv2.ProfileH, "precision-short", 4406, c, hash); err == nil {
		t.Fatal("insufficient funded accounts accepted")
	}
}
