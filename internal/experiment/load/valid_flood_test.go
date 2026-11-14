package load

import (
	"testing"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func TestA07ValidFloodMatchedFixedStreamAndCaps(t *testing.T) {
	c, hash := pilotConfig(t)
	f, err := NewPilotFixture("pluto-file02-a07", 101)
	if err != nil {
		t.Fatal(err)
	}
	low, high, err := BuildValidFloodPlans(f, 0, c, hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(low.Events) != 81 || len(high.Events) != 101 ||
		low.TargetTxPS != 20 || high.TargetTxPS != 25 || low.Profile != researchv2.ProfileH {
		t.Fatal("A07 plan shape")
	}
	for _, p := range []Plan{low, high} {
		if err := p.Validate(c, hash); err != nil {
			t.Fatal(err)
		}
		for _, e := range p.Events {
			if e.Scenario != "fixed_legitimate" {
				continue
			}
			found := false
			for _, other := range high.Events {
				if other.SHA256 == e.SHA256 && other.OffsetNS == e.OffsetNS {
					found = true
				}
			}
			if !found {
				t.Fatal("A07 fixed stream drift")
			}
		}
	}
}
