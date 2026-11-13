package load

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// BuildAttackDynamicsPairPlans keeps 6 H tx/s fixed while background is off
// for 3 seconds, on for 6, then off for 3. The on phase is exactly the A08
// 4/5/5 class ratio. Each arm must run on a fresh four-validator network.
func BuildAttackDynamicsPairPlans(f *PilotFixture, block int, c Config, configHash string) (Plan, Plan, error) {
	if f == nil || block < 0 || block > 99 || len(f.Senders) < 241 {
		return Plan{}, Plan{}, ErrPlan
	}
	genesisHash := sha256.Sum256(f.Genesis)
	base := Plan{Version: 1, ConfigSHA256: configHash,
		GenesisSHA256: hex.EncodeToString(genesisHash[:]), Profile: researchv2.ProfileH,
		Regime: EqualTxRate, Waveform: "step", Seed: int64(2026111300 + block),
		WarmupNS: int64(time.Second), SteadyNS: int64(12 * time.Second),
		DrainNS: int64(5 * time.Second), WindowNS: int64(time.Second), TargetTxPS: 13}
	benign, attack := base, base
	benign.RunID = fmt.Sprintf("A13-b%02d-benign", block)
	attack.RunID = fmt.Sprintf("A13-b%02d-attack", block)
	appendTx := func(plan *Plan, offset int64, slot int, scenario, traffic, capability string,
		code researchv2.Code, stage researchv2.Stage, wire []byte, nonce uint64) {
		h := sha256.Sum256(wire)
		plan.Events = append(plan.Events, PlannedTx{ID: fmt.Sprintf("event-%03d", len(plan.Events)),
			OffsetNS: offset, Traffic: traffic, Scenario: scenario, Capability: capability,
			Source: "rpc", AccountSlot: slot, Nonce: nonce,
			BytesHex: hex.EncodeToString(wire), SHA256: hex.EncodeToString(h[:]),
			ExpectedCode: code, ExpectedStage: stage})
	}
	for slot := 0; slot < 241; slot++ {
		valid, err := f.signed(researchv2.ProfileH, slot, 1)
		if err != nil {
			return Plan{}, Plan{}, err
		}
		if slot == 0 {
			for _, plan := range []*Plan{&benign, &attack} {
				appendTx(plan, int64(200*time.Millisecond), slot, "warmup", "legitimate",
					"C_registered_key_holder", researchv2.CodeOK, researchv2.StageAccepted, valid, 0)
			}
			continue
		}
		second, position := (slot-1)/20, (slot-1)%20
		on := second >= 3 && second < 9
		fixed := position == 2 || position == 5 || position == 8 || position == 11 || position == 14 || position == 17
		if !on && !fixed {
			continue
		}
		offset := int64(time.Second) + int64(second)*int64(time.Second) +
			int64(position+1)*int64(time.Second)/21
		if fixed {
			for _, plan := range []*Plan{&benign, &attack} {
				appendTx(plan, offset, slot, "fixed_legitimate", "legitimate",
					"C_registered_key_holder", researchv2.CodeOK, researchv2.StageAccepted, valid, 0)
			}
			continue
		}
		appendTx(&benign, offset, slot, "benign_background", "legitimate",
			"C_registered_key_holder", researchv2.CodeOK, researchv2.StageAccepted, valid, 0)
		invalid, code, stage, capability, err := makeAttackWire(f, researchv2.ProfileH, "A08", slot, valid)
		if err != nil {
			return Plan{}, Plan{}, err
		}
		class := mixedAttackClass(slot)
		scenario := map[string]string{"A01": "A13_structural", "A04": "A13_bad_ml", "A05": "A13_state_invalid"}[class]
		nonce := uint64(0)
		if class == "A05" {
			nonce = 1
		}
		appendTx(&attack, offset, slot, scenario, "adversarial", capability, code, stage, invalid, nonce)
	}
	if err := benign.Validate(c, configHash); err != nil {
		return Plan{}, Plan{}, err
	}
	if err := attack.Validate(c, configHash); err != nil {
		return Plan{}, Plan{}, err
	}
	state, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
	if err != nil {
		return Plan{}, Plan{}, err
	}
	for _, event := range attack.Events {
		if event.Traffic != "adversarial" {
			continue
		}
		wire, _ := hex.DecodeString(event.BytesHex)
		_, result, err := researchv2.EvaluateTrusted(state, researchv2.ProfileH, wire)
		if err != nil || result.Code != event.ExpectedCode || result.LastStage != event.ExpectedStage {
			return Plan{}, Plan{}, fmt.Errorf("%w: A13 oracle %s", ErrPlan, event.ID)
		}
	}
	return benign, attack, nil
}
