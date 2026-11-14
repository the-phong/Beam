package load

import (
	"encoding/hex"
	"testing"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func TestBuildAttackPairPlans(t *testing.T) {
	config, hash, err := ReadConfig("../../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := NewPilotFixture("pluto-file02-a01", 241)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		id      string
		profile researchv2.Profile
		code    researchv2.Code
		stage   researchv2.Stage
	}{
		{"A01", researchv2.ProfileH, researchv2.CodeMalformed, researchv2.StageStructural},
		{"A02", researchv2.ProfileH, researchv2.CodeBadBinding, researchv2.StageBinding},
		{"A03", researchv2.ProfileH, researchv2.CodeBadSecpSignature, researchv2.StageSecp},
		{"A04", researchv2.ProfileH, researchv2.CodeBadMLSignature, researchv2.StageML},
		{"A05", researchv2.ProfileH, researchv2.CodeInvalidNonce, researchv2.StageState},
	} {
		t.Run(item.id, func(t *testing.T) {
			benign, attack, err := BuildAttackPairPlans(fixture, item.profile, item.id, 0, config, hash)
			if err != nil {
				t.Fatal(err)
			}
			if len(benign.Events) != 241 || len(attack.Events) != 241 {
				t.Fatal("pair event count")
			}
			var fixed, invalid int
			variants := map[string]int{}
			for i, event := range attack.Events {
				control := benign.Events[i]
				if event.OffsetNS != control.OffsetNS || event.AccountSlot != control.AccountSlot ||
					len(event.BytesHex) != len(control.BytesHex) {
					t.Fatalf("pair mismatch at %d", i)
				}
				if event.Scenario == "fixed_legitimate" {
					fixed++
					if event.BytesHex != control.BytesHex || event.SHA256 != control.SHA256 {
						t.Fatal("fixed legitimate stream changed")
					}
				} else {
					invalid++
					variants[event.Scenario]++
					if event.ExpectedCode != item.code || event.ExpectedStage != item.stage {
						t.Fatal("attack oracle mismatch")
					}
					if item.id == "A05" {
						wire, _ := hex.DecodeString(event.BytesHex)
						tx, err := researchv2.Parse(wire)
						if err != nil || tx.Nonce != event.Nonce || tx.Nonce != 1 {
							t.Fatal("state-invalid nonce metadata mismatch")
						}
					}
				}
			}
			if fixed != 73 || invalid != 168 {
				t.Fatalf("fixed/invalid count %d/%d", fixed, invalid)
			}
			if item.id == "A01" && (variants["A01_wrong_magic"] == 0 ||
				variants["A01_wrong_declared_length"] == 0 || variants["A01_bad_chain_encoding"] == 0) {
				t.Fatalf("missing structural variants: %#v", variants)
			}
		})
	}
	if _, _, err := BuildAttackPairPlans(fixture, researchv2.ProfileC, "A04", 0, config, hash); err == nil {
		t.Fatal("A04 should require hybrid profile")
	}
}

func TestA08MixedAttackRatioAndStage(t *testing.T) {
	config, hash, err := ReadConfig("../../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := NewPilotFixture("pluto-file02-a08", 241)
	if err != nil {
		t.Fatal(err)
	}
	benign, attack, err := BuildAttackPairPlans(fixture, researchv2.ProfileH, "A08", 0, config, hash)
	if err != nil {
		t.Fatal(err)
	}
	counts := [12]map[string]int{}
	for i := range counts {
		counts[i] = map[string]int{}
	}
	for i, event := range attack.Events {
		if event.Scenario == "fixed_legitimate" {
			if event.SHA256 != benign.Events[i].SHA256 {
				t.Fatal("A08 fixed stream drift")
			}
			continue
		}
		second := (i - 1) / 20
		counts[second][event.Scenario]++
		want := map[string]struct {
			code  researchv2.Code
			stage researchv2.Stage
		}{"A08_structural": {researchv2.CodeMalformed, researchv2.StageStructural},
			"A08_bad_ml":        {researchv2.CodeBadMLSignature, researchv2.StageML},
			"A08_state_invalid": {researchv2.CodeInvalidNonce, researchv2.StageState}}[event.Scenario]
		if event.ExpectedCode != want.code || event.ExpectedStage != want.stage ||
			event.Scenario == "A08_state_invalid" && event.Nonce != 1 {
			t.Fatalf("A08 oracle mismatch %s", event.Scenario)
		}
	}
	for second, got := range counts {
		if got["A08_structural"] != 4 || got["A08_bad_ml"] != 5 || got["A08_state_invalid"] != 5 {
			t.Fatalf("A08 second %d ratio %v", second, got)
		}
	}
}
