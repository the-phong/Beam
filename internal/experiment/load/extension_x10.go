package load

import (
	"fmt"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// BuildX10TripletPlans preserves the accepted A04 valid/late-bad-ML wire
// fixture while adding a six-valid-only arm. All three arms share the exact
// fixed legitimate bytes, offsets, genesis and signer identities per block.
func BuildX10TripletPlans(f *PilotFixture, block int, c Config, configHash string) (Plan, Plan, Plan, error) {
	if block < 0 || block >= 24 {
		return Plan{}, Plan{}, Plan{}, ErrPlan
	}
	benign, invalid, err := BuildAttackPairPlans(f, researchv2.ProfileH, "A04", block, c, configHash)
	if err != nil {
		return Plan{}, Plan{}, Plan{}, err
	}
	l := invalid
	l.RunID = fmt.Sprintf("X10-b%02d-L", block)
	l.Regime, l.TargetTxPS, l.TargetLegitTPS, l.TargetInvalidTPS = EqualTxRate, 6, 0, 0
	l.Events = make([]PlannedTx, 0, 73)
	for _, event := range invalid.Events {
		if event.Scenario == "fixed_legitimate" {
			l.Events = append(l.Events, event)
		}
	}
	benign.RunID = fmt.Sprintf("X10-b%02d-B", block)
	invalid.RunID = fmt.Sprintf("X10-b%02d-I", block)
	for _, plan := range []Plan{l, benign, invalid} {
		if err := plan.Validate(c, configHash); err != nil {
			return Plan{}, Plan{}, Plan{}, err
		}
	}
	return l, benign, invalid, nil
}
