package load

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

var ReplayConditions = []string{"valid_distinct", "valid_repeated", "invalid_distinct", "invalid_repeated"}

// BuildReplayPlans keeps the 6/s fixed legitimate stream identical across
// arms. The replay-only duplicate exception is constrained by Plan.Validate.
func BuildReplayPlans(f *PilotFixture, block int, c Config, configHash string) (map[string]Plan, error) {
	if f == nil || block < 0 || block > 99 || len(f.Senders) < 81 {
		return nil, ErrPlan
	}
	base, err := BuildPilotPlan(f, researchv2.ProfileH, EqualTxRate, "constant",
		fmt.Sprintf("A06-b%02d-valid_distinct", block), int64(2026111300+block), c, configHash, 20)
	if err != nil || len(base.Events) != 81 {
		return nil, ErrPlan
	}
	firstRaw, _ := hex.DecodeString(base.Events[1].BytesHex)
	firstInvalid, err := badReplaySecp(firstRaw)
	if err != nil {
		return nil, err
	}
	firstInvalidHash := sha256.Sum256(firstInvalid)
	firstInvalidHex := hex.EncodeToString(firstInvalid)
	plans := make(map[string]Plan, len(ReplayConditions))
	for _, condition := range ReplayConditions {
		plan := base
		plan.RunID = fmt.Sprintf("A06-b%02d-%s", block, condition)
		plan.Events = append([]PlannedTx(nil), base.Events...)
		plan.ReplayMode = condition == "valid_repeated" || condition == "invalid_repeated"
		for i := range plan.Events {
			e := &plan.Events[i]
			fixed := i == 0
			if i > 0 {
				j := (i - 1) % 20
				fixed = j == 2 || j == 5 || j == 8 || j == 11 || j == 14 || j == 17
			}
			if fixed {
				e.Scenario = "fixed_legitimate"
				continue
			}
			switch condition {
			case "valid_distinct":
				e.Scenario = "benign_background"
			case "valid_repeated":
				e.Traffic, e.Scenario, e.Capability = "adversarial", "A06_valid_replay", "B_signed_transaction_replayer"
				e.AccountSlot, e.BytesHex, e.SHA256 = base.Events[1].AccountSlot, base.Events[1].BytesHex, base.Events[1].SHA256
			case "invalid_distinct", "invalid_repeated":
				e.Traffic, e.Scenario, e.Capability = "adversarial", "A06_bad_secp_distinct", "B_signed_transaction_modifier"
				e.ExpectedCode, e.ExpectedStage = researchv2.CodeBadSecpSignature, researchv2.StageSecp
				if condition == "invalid_repeated" {
					e.Scenario = "A06_bad_secp_repeated"
					e.AccountSlot, e.BytesHex, e.SHA256 = base.Events[1].AccountSlot, firstInvalidHex, hex.EncodeToString(firstInvalidHash[:])
				} else {
					valid, _ := hex.DecodeString(base.Events[i].BytesHex)
					invalid, err := badReplaySecp(valid)
					if err != nil {
						return nil, err
					}
					h := sha256.Sum256(invalid)
					e.BytesHex, e.SHA256 = hex.EncodeToString(invalid), hex.EncodeToString(h[:])
				}
			}
		}
		if err := plan.Validate(c, configHash); err != nil {
			return nil, fmt.Errorf("%s: %w", condition, err)
		}
		plans[condition] = plan
	}
	return plans, nil
}

func badReplaySecp(raw []byte) ([]byte, error) {
	tx, err := researchv2.Parse(raw)
	if err != nil {
		return nil, err
	}
	tx.SecpSignature[0] ^= 1
	return researchv2.Encode(tx)
}
