package load

import (
	"testing"
)

func TestA13DynamicsFixedStreamAndPhaseCaps(t *testing.T) {
	c, hash := pilotConfig(t)
	f, err := NewPilotFixture("pluto-file02-a13", 241)
	if err != nil {
		t.Fatal(err)
	}
	benign, attack, err := BuildAttackDynamicsPairPlans(f, 0, c, hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(benign.Events) != 157 || len(attack.Events) != 157 || benign.TargetTxPS != 13 {
		t.Fatal("A13 plan shape")
	}
	for _, plan := range []Plan{benign, attack} {
		var windows [12]int
		var classes [3]int
		for _, event := range plan.Events[1:] {
			second := (event.OffsetNS - 1_000_000_000) / 1_000_000_000
			windows[second]++
			if event.Traffic == "adversarial" {
				switch event.Scenario {
				case "A13_structural":
					classes[0]++
				case "A13_bad_ml":
					classes[1]++
				case "A13_state_invalid":
					classes[2]++
				default:
					t.Fatal("unregistered A13 class")
				}
			}
		}
		for second, count := range windows {
			want := 6
			if second >= 3 && second < 9 {
				want = 20
			}
			if count != want {
				t.Fatalf("A13 second %d: %d", second, count)
			}
		}
		if plan.RunID == attack.RunID && classes != [3]int{24, 30, 30} {
			t.Fatalf("A13 classes %v", classes)
		}
	}
	for i := range benign.Events {
		if benign.Events[i].Scenario == "fixed_legitimate" &&
			benign.Events[i].SHA256 != attack.Events[i].SHA256 {
			t.Fatal("A13 fixed stream drift")
		}
	}
}
