package load

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"time"

	"example.com/pluto/internal/experiment/profiles"
	"example.com/pluto/internal/experiment/profiles/testfixture"
)

// PilotFixture holds ephemeral test-only signers. It is used in the testbed
// controller process; only a pre-signed Plan reaches cmd/research/workload.
type PilotFixture struct {
	ChainID  string
	Genesis  []byte
	Senders  []researchv2test.Identity
	Receiver researchv2test.Identity
}

func NewPilotFixture(chainID string, senderCount int) (*PilotFixture, error) {
	if !researchv2.ValidChainID(chainID) || senderCount < 1 || senderCount+1 > researchv2.MaxGenesisAccounts {
		return nil, ErrPlan
	}
	f := &PilotFixture{ChainID: chainID, Senders: make([]researchv2test.Identity, senderCount)}
	var err error
	f.Receiver, err = researchv2test.NewIdentity()
	if err != nil {
		return nil, err
	}
	allocations := make([]researchv2.Allocation, 0, senderCount+1)
	allocations = append(allocations, researchv2.Allocation{Binding: f.Receiver.Binding, Balance: 0})
	for i := range f.Senders {
		f.Senders[i], err = researchv2test.NewIdentity()
		if err != nil {
			return nil, err
		}
		allocations = append(allocations, researchv2.Allocation{Binding: f.Senders[i].Binding, Balance: 2})
	}
	sort.Slice(allocations, func(i, j int) bool {
		return bytes.Compare(allocations[i].Binding.Account[:], allocations[j].Binding.Account[:]) < 0
	})
	f.Genesis, err = researchv2.EncodeGenesis(chainID, allocations)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (f *PilotFixture) signed(p researchv2.Profile, slot int, amount uint64) ([]byte, error) {
	if slot >= len(f.Senders) || slot < 0 {
		return nil, ErrPlan
	}
	owner := &researchv2test.Fixture{ChainID: f.ChainID}
	identity := f.Senders[slot]
	return owner.SignedBy(p, identity, identity.Binding.Account, f.Receiver.Binding.Account, amount, 0)
}

// BuildPilotPlan creates a small, pre-signed same-host pilot. Each legitimate
// transaction uses a fresh genesis-funded account at nonce zero; thus the
// offered stream cannot depend on which prior transaction already committed.
// A fresh network is still required for every run.
func BuildPilotPlan(f *PilotFixture, p researchv2.Profile, regime, waveform, runID string, seed int64, c Config, configHash string, txRate int) (Plan, error) {
	if f == nil || !p.Valid() || txRate < 0 || (waveform != "constant" && waveform != "burst" && waveform != "ramp") ||
		(waveform == "ramp" && txRate%5 != 0) {
		return Plan{}, ErrPlan
	}
	genesisHash := sha256.Sum256(f.Genesis)
	plan := Plan{Version: 1, RunID: runID, ConfigSHA256: configHash, GenesisSHA256: hex.EncodeToString(genesisHash[:]), Profile: p,
		Regime: regime, Waveform: waveform, Seed: seed, WarmupNS: int64(time.Second), SteadyNS: int64(4 * time.Second), DrainNS: int64(3 * time.Second), WindowNS: int64(time.Second)}
	var steadyPerSecond int
	switch regime {
	case EqualTxRate:
		if txRate < 1 || txRate > c.Safety.MaxTxPerSecond {
			return Plan{}, ErrPlan
		}
		plan.TargetTxPS = float64(txRate)
		steadyPerSecond = txRate
	case EqualByteRate:
		if waveform != "constant" {
			return Plan{}, ErrPlan
		}
		hSize, err := researchv2.ProfileH.WireSize(f.ChainID)
		if err != nil {
			return Plan{}, err
		}
		size, err := p.WireSize(f.ChainID)
		if err != nil {
			return Plan{}, err
		}
		plan.TargetBytesPS = float64(hSize)
		steadyPerSecond = int(math.Round(float64(hSize) / float64(size)))
	case FixedLegitimatePlusAdversary:
		if waveform != "constant" {
			return Plan{}, ErrPlan
		}
		plan.TargetLegitTPS, plan.TargetInvalidTPS = 1, 1
		steadyPerSecond = 2
	default:
		return Plan{}, ErrPlan
	}
	add := func(offset int64, traffic, scenario, capability string, code researchv2.Code, stage researchv2.Stage, raw []byte, slot int) {
		h := sha256.Sum256(raw)
		plan.Events = append(plan.Events, PlannedTx{ID: fmt.Sprintf("event-%03d", len(plan.Events)), OffsetNS: offset, Traffic: traffic, Scenario: scenario,
			Capability: capability, Source: "rpc", AccountSlot: slot, Nonce: 0, BytesHex: hex.EncodeToString(raw), SHA256: hex.EncodeToString(h[:]), ExpectedCode: code, ExpectedStage: stage})
	}
	slot := 0
	raw, err := f.signed(p, slot, 1)
	if err != nil {
		return Plan{}, err
	}
	add(int64(200*time.Millisecond), "legitimate", "benign_low", "C_registered_key_holder", researchv2.CodeOK, researchv2.StageAccepted, raw, slot)
	slot++
	for second := 0; second < 4; second++ {
		count := steadyPerSecond
		if waveform == "burst" {
			if second%2 == 0 {
				count = 0
			} else {
				count = steadyPerSecond * 2
			}
		} else if waveform == "ramp" {
			count = steadyPerSecond * (second + 1) * 2 / 5
		}
		for j := 0; j < count; j++ {
			offset := int64(time.Second) + int64(second)*int64(time.Second) + int64(j+1)*int64(time.Second)/int64(count+1)
			if regime == FixedLegitimatePlusAdversary && j == 1 {
				valid, err := f.signed(p, slot, 1)
				if err != nil {
					return Plan{}, err
				}
				modified := bytes.Clone(valid)
				scenario, stage, code := "invalid_ecdsa", researchv2.StageSecp, researchv2.CodeBadSecpSignature
				if p.UsesML() {
					scenario, stage, code = "invalid_ml_p", researchv2.StageML, researchv2.CodeBadMLSignature
					if p == researchv2.ProfileH {
						scenario = "invalid_ml_after_ecdsa"
					}
					modified[len(modified)-1] ^= 1
				} else {
					modified[12+len(f.ChainID)+56] ^= 1
				}
				add(offset, "adversarial", scenario, "B_signed_transaction_modifier", code, stage, modified, slot)
			} else {
				valid, err := f.signed(p, slot, 1)
				if err != nil {
					return Plan{}, err
				}
				scenario := "benign_low"
				if waveform == "burst" {
					scenario = "benign_burst"
				} else if waveform == "ramp" {
					scenario = "benign_ramp"
				} else if steadyPerSecond >= 5 {
					scenario = "valid_high_rate"
				}
				add(offset, "legitimate", scenario, "C_registered_key_holder", researchv2.CodeOK, researchv2.StageAccepted, valid, slot)
			}
			slot++
		}
	}
	if err := plan.Validate(c, configHash); err != nil {
		return Plan{}, err
	}
	initial, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
	if err != nil {
		return Plan{}, err
	}
	for _, e := range plan.Events {
		raw, _ := hex.DecodeString(e.BytesHex)
		_, result, err := researchv2.EvaluateTrusted(initial, p, raw)
		if err != nil || result.Code != e.ExpectedCode || result.LastStage != e.ExpectedStage {
			return Plan{}, fmt.Errorf("%w: direct oracle mismatch for %s", ErrPlan, e.ID)
		}
	}
	return plan, nil
}
