package researchv2_test

import (
	"reflect"
	"testing"
	"time"

	researchv2 "example.com/pluto/internal/experiment/profiles"
	researchv2test "example.com/pluto/internal/experiment/profiles/testfixture"
)

func TestMeasurementSpansPreserveValidationAndSeparateStages(t *testing.T) {
	f, err := researchv2test.New("pluto-file02-r02")
	if err != nil {
		t.Fatal(err)
	}
	s, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		raw, err := f.Signed(profile, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		plainState, plainResult, err := researchv2.EvaluateTrusted(s, profile, raw)
		if err != nil {
			t.Fatal(err)
		}
		spans := map[researchv2.Stage]time.Duration{}
		observedState, observedResult, err := researchv2.EvaluateTrustedObserved(s, profile, raw, nil,
			func(stage researchv2.Stage, elapsed time.Duration) { spans[stage] = elapsed })
		if err != nil || !reflect.DeepEqual(observedResult, plainResult) ||
			observedState.AppHash() != plainState.AppHash() {
			t.Fatalf("profile %d changed validation: plain=%+v observed=%+v err=%v", profile, plainResult, observedResult, err)
		}
		for _, stage := range []researchv2.Stage{researchv2.StageStructural, researchv2.StageBinding,
			researchv2.StageDigest, researchv2.StageState, researchv2.StageStateCopy} {
			if spans[stage] <= 0 {
				t.Fatalf("profile %d missing %s duration", profile, stage)
			}
		}
		if profile.UsesSecp() && spans[researchv2.StageSecp] <= 0 ||
			profile.UsesML() && spans[researchv2.StageML] <= 0 {
			t.Fatalf("profile %d missing crypto duration", profile)
		}
	}
}
