package researchv2_test

import (
	"reflect"
	"runtime"
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

func TestStageCPUClockPreservesHybridValidation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("stage CPU clock is Linux only")
	}
	f, err := researchv2test.New("pluto-file02-x16")
	if err != nil {
		t.Fatal(err)
	}
	s, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.Signed(researchv2.ProfileH, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range [][]byte{raw, raw[:2]} {
		plainState, plainResult, err := researchv2.EvaluateTrusted(s, researchv2.ProfileH, candidate)
		if err != nil {
			t.Fatal(err)
		}
		cpu := map[researchv2.Stage]int64{}
		measuredState, measuredResult, err := researchv2.EvaluateTrustedObservedCPU(
			s, researchv2.ProfileH, candidate, nil, nil,
			func(stage researchv2.Stage, delta *int64) {
				if delta == nil || *delta <= 0 {
					t.Fatalf("stage %s missing CPU", stage)
				}
				cpu[stage] = *delta
			})
		if err != nil || !reflect.DeepEqual(measuredResult, plainResult) ||
			measuredState.AppHash() != plainState.AppHash() || cpu[researchv2.StageStructural] <= 0 {
			t.Fatalf("stage CPU path changed validation: plain=%+v measured=%+v err=%v", plainResult, measuredResult, err)
		}
		if len(candidate) == len(raw) && (cpu[researchv2.StageSecp] <= 0 || cpu[researchv2.StageML] <= 0 || cpu[researchv2.StageState] <= 0) {
			t.Fatalf("hybrid CPU stages incomplete: %+v", cpu)
		}
	}
}
