package load

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// BuildQ02StructuralPairPlans holds a profile-specific valid stream fixed
// while replacing valid background with structural-invalid bytes. It uses
// the same fixture across the four profiles of one Q02 block.
func BuildQ02StructuralPairPlans(f *PilotFixture, p researchv2.Profile, block int,
	c Config, configHash string) (Plan, Plan, error) {
	if f == nil || !p.Valid() || block < 0 || block >= 12 || len(f.Senders) < 241 ||
		f.ChainID != "pluto-file02-q02" {
		return Plan{}, Plan{}, ErrPlan
	}
	genesisHash := sha256.Sum256(f.Genesis)
	base := Plan{Version: 1, ConfigSHA256: configHash,
		GenesisSHA256: hex.EncodeToString(genesisHash[:]), Profile: p,
		Waveform: "constant", Seed: int64(2026111400 + block),
		WarmupNS: int64(time.Second), SteadyNS: int64(12 * time.Second),
		DrainNS: int64(5 * time.Second), WindowNS: int64(time.Second)}
	benign, attack := base, base
	label := map[researchv2.Profile]string{researchv2.ProfileC: "C",
		researchv2.ProfileCP: "CP", researchv2.ProfileP: "P", researchv2.ProfileH: "H"}[p]
	benign.RunID = fmt.Sprintf("Q02-b%02d-%s-benign", block, label)
	attack.RunID = fmt.Sprintf("Q02-b%02d-%s-attack", block, label)
	benign.Regime, benign.TargetTxPS = EqualTxRate, 20
	attack.Regime, attack.TargetLegitTPS, attack.TargetInvalidTPS =
		FixedLegitimatePlusAdversary, 6, 14
	add := func(plan *Plan, offset int64, slot int, scenario, traffic string,
		code researchv2.Code, stage researchv2.Stage, wire []byte) {
		hash := sha256.Sum256(wire)
		capability := "C_registered_key_holder"
		if traffic == "adversarial" {
			capability = "A_arbitrary_sender"
		}
		plan.Events = append(plan.Events, PlannedTx{
			ID: fmt.Sprintf("event-%03d", len(plan.Events)), OffsetNS: offset,
			Traffic: traffic, Scenario: scenario, Capability: capability,
			Source: "rpc", AccountSlot: slot, Nonce: 0,
			BytesHex: hex.EncodeToString(wire), SHA256: hex.EncodeToString(hash[:]),
			ExpectedCode: code, ExpectedStage: stage})
	}
	for slot := 0; slot < 241; slot++ {
		offset := int64(200 * time.Millisecond)
		fixed := true
		if slot > 0 {
			second, position := (slot-1)/20, (slot-1)%20
			offset = int64(time.Second) + int64(second)*int64(time.Second) +
				int64(position+1)*int64(time.Second)/21
			fixed = position == 2 || position == 5 || position == 8 ||
				position == 11 || position == 14 || position == 17
		}
		valid, err := f.signed(p, slot, 1)
		if err != nil {
			return Plan{}, Plan{}, err
		}
		scenario := "benign_background"
		if fixed {
			scenario = "fixed_legitimate"
		}
		add(&benign, offset, slot, scenario, "legitimate",
			researchv2.CodeOK, researchv2.StageAccepted, valid)
		if fixed {
			add(&attack, offset, slot, scenario, "legitimate",
				researchv2.CodeOK, researchv2.StageAccepted, valid)
			continue
		}
		invalid, code, stage, _, err := makeAttackWire(f, p, "A01", slot, valid)
		if err != nil {
			return Plan{}, Plan{}, err
		}
		add(&attack, offset, slot, "Q02_structural", "adversarial", code, stage, invalid)
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
		_, result, err := researchv2.EvaluateTrusted(state, p, wire)
		if err != nil || result.Code != researchv2.CodeMalformed ||
			result.LastStage != researchv2.StageStructural {
			return Plan{}, Plan{}, fmt.Errorf("%w: Q02 oracle %s", ErrPlan, event.ID)
		}
	}
	return benign, attack, nil
}
