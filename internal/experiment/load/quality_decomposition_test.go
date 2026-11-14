package load

import (
	"testing"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func TestBuildQ02StructuralPairPlans(t *testing.T) {
	c, hash, err := ReadConfig("../../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewPilotFixture("pluto-file02-q02", 241)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		profile researchv2.Profile
		length  int
	}{
		{researchv2.ProfileC, 148}, {researchv2.ProfileCP, 2568},
		{researchv2.ProfileP, 2504}, {researchv2.ProfileH, 2568},
	} {
		benign, attack, err := BuildQ02StructuralPairPlans(f, item.profile, 0, c, hash)
		if err != nil {
			t.Fatal(err)
		}
		if len(benign.Events) != 241 || len(attack.Events) != 241 ||
			benign.GenesisSHA256 != attack.GenesisSHA256 {
			t.Fatal("Q02 pair count/genesis")
		}
		var fixed, invalid int
		for i, event := range attack.Events {
			control := benign.Events[i]
			if event.OffsetNS != control.OffsetNS || event.AccountSlot != control.AccountSlot ||
				len(event.BytesHex)/2 != item.length || len(control.BytesHex)/2 != item.length {
				t.Fatal("Q02 schedule or byte mismatch")
			}
			if event.Traffic == "legitimate" {
				fixed++
				if event.SHA256 != control.SHA256 || event.BytesHex != control.BytesHex {
					t.Fatal("Q02 fixed valid stream changed")
				}
			} else {
				invalid++
				if event.ExpectedCode != researchv2.CodeMalformed ||
					event.ExpectedStage != researchv2.StageStructural {
					t.Fatal("Q02 structural oracle")
				}
			}
		}
		if fixed != 73 || invalid != 168 {
			t.Fatal("Q02 fixed/invalid count")
		}
	}
	if _, _, err := BuildQ02StructuralPairPlans(f, researchv2.ProfileH, 12, c, hash); err == nil {
		t.Fatal("unregistered block accepted")
	}
}
