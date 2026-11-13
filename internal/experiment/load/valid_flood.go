package load

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// BuildValidFloodPlans returns a matched H-only 20/25 tx/s pair. Fixed
// legitimate bytes and offsets are identical. All traffic is valid and uses
// distinct funded accounts; each arm requires a fresh network.
func BuildValidFloodPlans(f *PilotFixture, block int, c Config, configHash string) (Plan, Plan, error) {
	if f == nil || block < 0 || block > 99 || len(f.Senders) < 101 {
		return Plan{}, Plan{}, ErrPlan
	}
	genesisHash := sha256.Sum256(f.Genesis)
	base := Plan{Version: 1, ConfigSHA256: configHash,
		GenesisSHA256: hex.EncodeToString(genesisHash[:]), Profile: researchv2.ProfileH,
		Regime: EqualTxRate, Waveform: "constant", Seed: int64(2026111300 + block),
		WarmupNS: int64(time.Second), SteadyNS: int64(4 * time.Second),
		DrainNS: int64(3 * time.Second), WindowNS: int64(time.Second)}
	low, high := base, base
	low.RunID, high.RunID = fmt.Sprintf("A07-b%02d-rate20", block), fmt.Sprintf("A07-b%02d-rate25", block)
	low.TargetTxPS, high.TargetTxPS = 20, 25
	appendTx := func(plan *Plan, offset int64, slot int, scenario string, wire []byte) {
		h := sha256.Sum256(wire)
		plan.Events = append(plan.Events, PlannedTx{ID: fmt.Sprintf("event-%03d", len(plan.Events)),
			OffsetNS: offset, Traffic: "legitimate", Scenario: scenario,
			Capability: "C_registered_key_holder", Source: "rpc", AccountSlot: slot,
			Nonce: 0, BytesHex: hex.EncodeToString(wire), SHA256: hex.EncodeToString(h[:]),
			ExpectedCode: researchv2.CodeOK, ExpectedStage: researchv2.StageAccepted})
	}
	for slot := 0; slot < 101; slot++ {
		wire, err := f.signed(researchv2.ProfileH, slot, 1)
		if err != nil {
			return Plan{}, Plan{}, err
		}
		if slot == 0 {
			appendTx(&low, int64(200*time.Millisecond), slot, "warmup", wire)
			appendTx(&high, int64(200*time.Millisecond), slot, "warmup", wire)
			continue
		}
		second, position := (slot-1)/25, (slot-1)%25
		offset := int64(time.Second) + int64(second)*int64(time.Second) +
			int64(position+1)*int64(time.Second)/26
		fixed := position == 2 || position == 6 || position == 10 || position == 14 || position == 18 || position == 22
		scenario := "valid_background"
		if fixed {
			scenario = "fixed_legitimate"
		}
		appendTx(&high, offset, slot, scenario, wire)
		if fixed || (position != 24 && position != 23 && position != 21 && position != 20 && position != 19) {
			appendTx(&low, offset, slot, scenario, wire)
		}
	}
	if err := low.Validate(c, configHash); err != nil {
		return Plan{}, Plan{}, err
	}
	if err := high.Validate(c, configHash); err != nil {
		return Plan{}, Plan{}, err
	}
	return low, high, nil
}
