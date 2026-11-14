package load

import (
	"errors"
	"testing"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func TestA06ReplayPlansAreScopedAndMatched(t *testing.T) {
	c, hash := pilotConfig(t)
	f, err := NewPilotFixture("pluto-file02-a06", 81)
	if err != nil {
		t.Fatal(err)
	}
	plans, err := BuildReplayPlans(f, 0, c, hash)
	if err != nil {
		t.Fatal(err)
	}
	for _, condition := range ReplayConditions {
		plan := plans[condition]
		if len(plan.Events) != 81 || plan.Profile != researchv2.ProfileH || plan.TargetTxPS != 20 {
			t.Fatalf("%s plan shape", condition)
		}
		if err := plan.Validate(c, hash); err != nil {
			t.Fatalf("%s: %v", condition, err)
		}
		for i, event := range plan.Events {
			if event.Scenario == "fixed_legitimate" && event.SHA256 != plans["valid_distinct"].Events[i].SHA256 {
				t.Fatalf("%s fixed stream drift at %d", condition, i)
			}
		}
	}
	for _, condition := range []string{"valid_repeated", "invalid_repeated"} {
		plan := plans[condition]
		plan.RunID = "unsafe-replay"
		if err := plan.Validate(c, hash); !errors.Is(err, ErrPlan) {
			t.Fatalf("replay accepted outside A06: %v", err)
		}
	}
	plan := plans["invalid_repeated"]
	plan.Events[2].Scenario = "A06_bad_secp_distinct"
	if err := plan.Validate(c, hash); !errors.Is(err, ErrPlan) {
		t.Fatalf("unmatched duplicate accepted: %v", err)
	}
}
