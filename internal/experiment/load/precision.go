package load

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// BuildPrecisionPilotPlan extends the accepted four-second 20 tx/s benign
// plan to twelve steady seconds, still within the unchanged S06 60s/25 tx/s
// and 64 KiB/s caps. Every event uses a distinct funded genesis account.
// It is for bounded File 02 precision QA, not a confirmatory study workload.
func BuildPrecisionPilotPlan(f *PilotFixture, p researchv2.Profile, runID string,
	seed int64, c Config, configHash string) (Plan, error) {
	return buildExtendedBenignPlan(f, p, runID, seed, c, configHash, 20)
}

// BuildKneePilotPlan builds a bounded benign plan for the separately
// authorized T06 knee search; it cannot exceed the unchanged S06 caps.
func BuildKneePilotPlan(f *PilotFixture, p researchv2.Profile, runID string,
	seed int64, c Config, configHash string, rate int) (Plan, error) {
	if rate != 20 && rate != 23 && rate != 25 {
		return Plan{}, ErrPlan
	}
	return buildExtendedBenignPlan(f, p, runID, seed, c, configHash, rate)
}

func buildExtendedBenignPlan(f *PilotFixture, p researchv2.Profile, runID string,
	seed int64, c Config, configHash string, rate int) (Plan, error) {
	const steadySeconds = 12
	if f == nil || len(f.Senders) < 1+rate*steadySeconds {
		return Plan{}, ErrPlan
	}
	plan, err := BuildPilotPlan(f, p, EqualTxRate, "constant", runID, seed, c, configHash, rate)
	if err != nil {
		return Plan{}, err
	}
	plan.SteadyNS = int64(steadySeconds * time.Second)
	initial, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
	if err != nil {
		return Plan{}, err
	}
	slot := len(plan.Events)
	for second := 4; second < steadySeconds; second++ {
		for j := 0; j < rate; j++ {
			raw, err := f.signed(p, slot, 1)
			if err != nil {
				return Plan{}, err
			}
			_, outcome, err := researchv2.EvaluateTrusted(initial, p, raw)
			if err != nil || outcome.Code != researchv2.CodeOK || outcome.LastStage != researchv2.StageAccepted {
				return Plan{}, fmt.Errorf("%w: direct precision oracle slot %d", ErrPlan, slot)
			}
			h := sha256.Sum256(raw)
			plan.Events = append(plan.Events, PlannedTx{
				ID: fmt.Sprintf("event-%03d", slot),
				OffsetNS: int64(time.Second) + int64(second)*int64(time.Second) +
					int64(j+1)*int64(time.Second)/int64(rate+1),
				Traffic: "legitimate", Scenario: "valid_high_rate",
				Capability: "C_registered_key_holder", Source: "rpc",
				AccountSlot: slot, Nonce: 0, BytesHex: hex.EncodeToString(raw),
				SHA256: hex.EncodeToString(h[:]), ExpectedCode: researchv2.CodeOK,
				ExpectedStage: researchv2.StageAccepted,
			})
			slot++
		}
	}
	if err := plan.Validate(c, configHash); err != nil {
		return Plan{}, err
	}
	return plan, nil
}
