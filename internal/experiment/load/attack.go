package load

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	researchv2 "example.com/pluto/internal/experiment/profiles"
	researchv2test "example.com/pluto/internal/experiment/profiles/testfixture"
)

// BuildAttackPairPlans uses the same immutable genesis and fixed legitimate
// byte stream in both arms. A fresh network is required for each arm.
func BuildAttackPairPlans(f *PilotFixture, p researchv2.Profile, caseID string, pair int,
	config Config, configHash string) (Plan, Plan, error) {
	if f == nil || pair < 0 || pair > 99 || len(f.Senders) < 241 ||
		(caseID != "A01" && caseID != "A02" && caseID != "A03" && caseID != "A04" && caseID != "A05") ||
		(caseID == "A04" && p != researchv2.ProfileH) || (caseID != "A04" && p != researchv2.ProfileH && p != researchv2.ProfileC) {
		return Plan{}, Plan{}, ErrPlan
	}
	genesisHash := sha256.Sum256(f.Genesis)
	base := Plan{Version: 1, ConfigSHA256: configHash, GenesisSHA256: hex.EncodeToString(genesisHash[:]),
		Profile: p, Waveform: "constant", Seed: int64(2026111300 + pair),
		WarmupNS: int64(time.Second), SteadyNS: int64(12 * time.Second),
		DrainNS: int64(5 * time.Second), WindowNS: int64(time.Second)}
	benign, attack := base, base
	benign.RunID = fmt.Sprintf("%s-p%02d-benign", caseID, pair)
	attack.RunID = fmt.Sprintf("%s-p%02d-attack", caseID, pair)
	benign.Regime, benign.TargetTxPS = EqualTxRate, 20
	attack.Regime, attack.TargetLegitTPS, attack.TargetInvalidTPS = FixedLegitimatePlusAdversary, 6, 14
	appendTx := func(plan *Plan, offset int64, traffic, scenario, capability string,
		code researchv2.Code, stage researchv2.Stage, raw []byte, slot int) {
		sum := sha256.Sum256(raw)
		plan.Events = append(plan.Events, PlannedTx{ID: fmt.Sprintf("event-%03d", len(plan.Events)),
			OffsetNS: offset, Traffic: traffic, Scenario: scenario, Capability: capability, Source: "rpc",
			AccountSlot: slot, Nonce: 0, BytesHex: hex.EncodeToString(raw),
			SHA256: hex.EncodeToString(sum[:]), ExpectedCode: code, ExpectedStage: stage})
	}
	for slot := 0; slot < 241; slot++ {
		offset := int64(200 * time.Millisecond)
		fixed := true
		if slot > 0 {
			second, position := (slot-1)/20, (slot-1)%20
			offset = int64(time.Second) + int64(second)*int64(time.Second) + int64(position+1)*int64(time.Second)/21
			fixed = position == 2 || position == 5 || position == 8 || position == 11 || position == 14 || position == 17
		}
		valid, err := f.signed(p, slot, 1)
		if err != nil {
			return Plan{}, Plan{}, err
		}
		benignScenario := "benign_background"
		if fixed {
			benignScenario = "fixed_legitimate"
		}
		appendTx(&benign, offset, "legitimate", benignScenario, "C_registered_key_holder",
			researchv2.CodeOK, researchv2.StageAccepted, valid, slot)
		if fixed {
			appendTx(&attack, offset, "legitimate", "fixed_legitimate", "C_registered_key_holder",
				researchv2.CodeOK, researchv2.StageAccepted, valid, slot)
			continue
		}
		invalid, code, stage, capability, err := makeAttackWire(f, p, caseID, slot, valid)
		if err != nil {
			return Plan{}, Plan{}, err
		}
		scenario := caseID
		if caseID == "A01" {
			scenario = []string{"A01_wrong_magic", "A01_wrong_declared_length", "A01_bad_chain_encoding"}[slot%3]
		}
		appendTx(&attack, offset, "adversarial", scenario, capability, code, stage, invalid, slot)
		if caseID == "A05" {
			attack.Events[len(attack.Events)-1].Nonce = 1
		}
	}
	if err := benign.Validate(config, configHash); err != nil {
		return Plan{}, Plan{}, err
	}
	if err := attack.Validate(config, configHash); err != nil {
		return Plan{}, Plan{}, err
	}
	state, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
	if err != nil {
		return Plan{}, Plan{}, err
	}
	for _, e := range attack.Events {
		raw, _ := hex.DecodeString(e.BytesHex)
		_, result, err := researchv2.EvaluateTrusted(state, p, raw)
		if err != nil || result.Code != e.ExpectedCode || result.LastStage != e.ExpectedStage {
			return Plan{}, Plan{}, fmt.Errorf("%w: attack oracle %s", ErrPlan, e.ID)
		}
	}
	return benign, attack, nil
}

func makeAttackWire(f *PilotFixture, p researchv2.Profile, caseID string, slot int,
	valid []byte) ([]byte, researchv2.Code, researchv2.Stage, string, error) {
	modified := bytes.Clone(valid)
	switch caseID {
	case "A01":
		switch slot % 3 {
		case 0:
			modified[0] ^= 1
		case 1:
			modified[11]++
		case 2:
			modified[12] = '/'
		}
		return modified, researchv2.CodeMalformed, researchv2.StageStructural, "A_arbitrary_sender", nil
	case "A02":
		// Change the registered sender address. The existing proofs are never reached.
		modified[12+len(f.ChainID)] ^= 1
		return modified, researchv2.CodeBadBinding, researchv2.StageBinding, "A_arbitrary_sender", nil
	case "A03":
		tx, err := researchv2.Parse(modified)
		if err != nil {
			return nil, 0, "", "", err
		}
		tx.SecpSignature[0] ^= 1
		modified, err = researchv2.Encode(tx)
		return modified, researchv2.CodeBadSecpSignature, researchv2.StageSecp, "B_signed_transaction_modifier", err
	case "A04":
		tx, err := researchv2.Parse(modified)
		if err != nil {
			return nil, 0, "", "", err
		}
		tx.MLSignature[len(tx.MLSignature)-1] ^= 1
		modified, err = researchv2.Encode(tx)
		return modified, researchv2.CodeBadMLSignature, researchv2.StageML, "B_signed_transaction_modifier", err
	case "A05":
		owner := &researchv2test.Fixture{ChainID: f.ChainID}
		identity := f.Senders[slot]
		modified, err := owner.SignedBy(p, identity, identity.Binding.Account,
			f.Receiver.Binding.Account, 1, 1)
		return modified, researchv2.CodeInvalidNonce, researchv2.StageState, "C_registered_key_holder", err
	default:
		return nil, 0, "", "", ErrPlan
	}
}
