package load

import (
	"testing"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func TestBuildX13InvalidPlan(t *testing.T) {
	c, hash, err := ReadConfig("../../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewPilotFixture("pluto-file02-x13", 301)
	if err != nil {
		t.Fatal(err)
	}
	var fixed []string
	for _, rate := range []int{20, 23, 25} {
		plan, err := BuildX13InvalidPlan(f, 0, rate, c, hash)
		if err != nil {
			t.Fatal(rate, err)
		}
		if len(plan.Events) != 1+12*rate || plan.TargetLegitTPS != 6 || plan.TargetInvalidTPS != float64(rate-6) {
			t.Fatalf("rate %d plan shape", rate)
		}
		var thisFixed []string
		for _, event := range plan.Events {
			if event.Scenario == "fixed_legitimate" {
				thisFixed = append(thisFixed, event.SHA256)
			} else if event.Traffic == "adversarial" && (event.ExpectedCode != researchv2.CodeBadMLSignature || event.ExpectedStage != researchv2.StageML) {
				t.Fatalf("rate %d invalid stage", rate)
			}
		}
		if len(thisFixed) != 73 {
			t.Fatalf("rate %d fixed count", rate)
		}
		if fixed == nil {
			fixed = thisFixed
		} else {
			for i := range fixed {
				if fixed[i] != thisFixed[i] {
					t.Fatalf("rate %d fixed stream drift at %d", rate, i)
				}
			}
		}
	}
	if _, err := BuildX13InvalidPlan(f, 0, 26, c, hash); err == nil {
		t.Fatal("26 tx/s must be rejected")
	}
}
