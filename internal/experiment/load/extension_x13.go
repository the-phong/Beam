package load

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// BuildX13InvalidPlan keeps the six A04 fixed legitimate transactions and
// adds only new late-bad-ML background at 23/25 tx/s. The same fixture is
// used for all three rates in one matched block; each rate gets a fresh net.
func BuildX13InvalidPlan(f *PilotFixture, block, rate int, c Config, configHash string) (Plan, error) {
	if f == nil || block < 0 || block >= 12 || (rate != 20 && rate != 23 && rate != 25) || len(f.Senders) < 301 {
		return Plan{}, ErrPlan
	}
	_, plan, err := BuildAttackPairPlans(f, researchv2.ProfileH, "A04", block, c, configHash)
	if err != nil {
		return Plan{}, err
	}
	plan.RunID = fmt.Sprintf("X13-b%02d-r%d", block, rate)
	plan.TargetInvalidTPS = float64(rate - 6)
	for second := 0; second < 12; second++ {
		for j := 0; j < rate-20; j++ {
			slot := 241 + second*5 + j
			valid, err := f.signed(researchv2.ProfileH, slot, 1)
			if err != nil {
				return Plan{}, err
			}
			invalid, code, stage, capability, err := makeAttackWire(f, researchv2.ProfileH, "A04", slot, valid)
			if err != nil {
				return Plan{}, err
			}
			sum := sha256.Sum256(invalid)
			plan.Events = append(plan.Events, PlannedTx{
				OffsetNS: int64(time.Second) + int64(second)*int64(time.Second) + int64(j+1)*int64(time.Second)/int64(rate-19),
				Traffic:  "adversarial", Scenario: "A04", Capability: capability, Source: "rpc",
				AccountSlot: slot, Nonce: 0, BytesHex: hex.EncodeToString(invalid),
				SHA256: hex.EncodeToString(sum[:]), ExpectedCode: code, ExpectedStage: stage,
			})
		}
	}
	sort.SliceStable(plan.Events, func(i, j int) bool { return plan.Events[i].OffsetNS < plan.Events[j].OffsetNS })
	for i := range plan.Events {
		plan.Events[i].ID = fmt.Sprintf("event-%03d", i)
	}
	if err := plan.Validate(c, configHash); err != nil {
		return Plan{}, err
	}
	return plan, nil
}
