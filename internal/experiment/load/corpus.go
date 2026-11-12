package load

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"example.com/pluto/internal/experiment/profiles"
	"example.com/pluto/internal/experiment/profiles/testfixture"
)

var ErrCorpus = errors.New("invalid S06 workload corpus")

type ScenarioCase struct {
	ID             string             `json:"id"`
	Profile        researchv2.Profile `json:"profile"`
	Scenario       string             `json:"scenario"`
	Traffic        string             `json:"traffic"`
	Capability     string             `json:"capability"`
	PriorTransfers int                `json:"prior_transfers"`
	BytesHex       string             `json:"bytes_hex"`
	SHA256         string             `json:"sha256"`
	ExpectedCode   researchv2.Code    `json:"expected_code"`
	ExpectedStage  researchv2.Stage   `json:"expected_stage"`
}

type ScenarioCorpus struct {
	Version    int            `json:"version"`
	ChainID    string         `json:"chain_id"`
	GenesisHex string         `json:"genesis_hex"`
	Cases      []ScenarioCase `json:"cases"`
}

// BuildScenarioCorpus creates one publication-safe signed corpus. It never
// serializes the ephemeral signing keys used to produce its public vectors.
func BuildScenarioCorpus() (ScenarioCorpus, error) {
	f, err := researchv2test.New("pluto-s06-public")
	if err != nil {
		return ScenarioCorpus{}, err
	}
	c := ScenarioCorpus{Version: 1, ChainID: f.ChainID, GenesisHex: hex.EncodeToString(f.Genesis)}
	add := func(p researchv2.Profile, name, traffic, capability string, prior int, raw []byte, code researchv2.Code, stage researchv2.Stage) {
		h := sha256.Sum256(raw)
		c.Cases = append(c.Cases, ScenarioCase{ID: fmt.Sprintf("p%d-%s", p, name), Profile: p, Scenario: name, Traffic: traffic, Capability: capability,
			PriorTransfers: prior, BytesHex: hex.EncodeToString(raw), SHA256: hex.EncodeToString(h[:]), ExpectedCode: code, ExpectedStage: stage})
	}
	for _, p := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		valid := make([][]byte, 4)
		for nonce := range valid {
			valid[nonce], err = f.Signed(p, 1, uint64(nonce))
			if err != nil {
				return ScenarioCorpus{}, err
			}
		}
		add(p, "benign_low", "legitimate", "C_registered_key_holder", 0, valid[0], researchv2.CodeOK, researchv2.StageAccepted)
		add(p, "benign_high", "legitimate", "C_registered_key_holder", 1, valid[1], researchv2.CodeOK, researchv2.StageAccepted)
		add(p, "benign_burst", "legitimate", "C_registered_key_holder", 2, valid[2], researchv2.CodeOK, researchv2.StageAccepted)
		add(p, "valid_high_rate", "legitimate", "C_registered_key_holder", 3, valid[3], researchv2.CodeOK, researchv2.StageAccepted)
		early := []byte(fmt.Sprintf("s06-early-p%d", p))
		add(p, "invalid_early", "adversarial", "A_arbitrary_sender", 0, early, researchv2.CodeMalformed, researchv2.StageStructural)
		late, err := f.Signed(p, 101, 0)
		if err != nil {
			return ScenarioCorpus{}, err
		}
		add(p, "late_state_invalid", "adversarial", "C_registered_key_holder", 0, late, researchv2.CodeInsufficientBalance, researchv2.StageState)
		if p.UsesSecp() {
			secp := bytes.Clone(valid[0])
			secp[12+len(f.ChainID)+56] ^= 1
			add(p, "invalid_ecdsa", "adversarial", "B_signed_transaction_modifier", 0, secp, researchv2.CodeBadSecpSignature, researchv2.StageSecp)
		}
		if p.UsesML() {
			ml := bytes.Clone(valid[0])
			ml[len(ml)-1] ^= 1
			name := "invalid_ml_p"
			if p == researchv2.ProfileH {
				name = "invalid_ml_after_ecdsa"
			}
			add(p, name, "adversarial", "B_signed_transaction_modifier", 0, ml, researchv2.CodeBadMLSignature, researchv2.StageML)
		}
	}
	if err := VerifyScenarioCorpus(c); err != nil {
		return ScenarioCorpus{}, err
	}
	return c, nil
}

func ReadScenarioCorpus(path string) (ScenarioCorpus, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ScenarioCorpus{}, err
	}
	var c ScenarioCorpus
	if err := json.Unmarshal(raw, &c); err != nil {
		return ScenarioCorpus{}, err
	}
	return c, VerifyScenarioCorpus(c)
}

func WriteScenarioCorpus(path string, c ScenarioCorpus) error {
	if err := VerifyScenarioCorpus(c); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}

func VerifyScenarioCorpus(c ScenarioCorpus) error {
	if c.Version != 1 || !researchv2.ValidChainID(c.ChainID) || len(c.Cases) < 25 {
		return ErrCorpus
	}
	genesis, err := hex.DecodeString(c.GenesisHex)
	if err != nil {
		return ErrCorpus
	}
	initial, err := researchv2.ParseGenesis(genesis, c.ChainID)
	if err != nil {
		return ErrCorpus
	}
	seen := map[string]bool{}
	counts := map[researchv2.Profile]map[string]int{}
	for _, item := range c.Cases {
		if seen[item.ID] || !item.Profile.Valid() || item.ID != fmt.Sprintf("p%d-%s", item.Profile, item.Scenario) || item.PriorTransfers < 0 || item.PriorTransfers > 3 {
			return ErrCorpus
		}
		seen[item.ID] = true
		if counts[item.Profile] == nil {
			counts[item.Profile] = map[string]int{}
		}
		counts[item.Profile][item.Scenario]++
		raw, err := hex.DecodeString(item.BytesHex)
		if err != nil || len(raw) == 0 {
			return ErrCorpus
		}
		h := sha256.Sum256(raw)
		if hex.EncodeToString(h[:]) != item.SHA256 {
			return ErrCorpus
		}
		state := initial
		for nonce := 0; nonce < item.PriorTransfers; nonce++ {
			priorID := fmt.Sprintf("p%d-%s", item.Profile, []string{"benign_low", "benign_high", "benign_burst"}[nonce])
			var priorRaw []byte
			for _, prior := range c.Cases {
				if prior.ID == priorID {
					priorRaw, _ = hex.DecodeString(prior.BytesHex)
					break
				}
			}
			if priorRaw == nil {
				return ErrCorpus
			}
			var result researchv2.Result
			state, result, err = researchv2.EvaluateTrusted(state, item.Profile, priorRaw)
			if err != nil || result.Code != researchv2.CodeOK {
				return ErrCorpus
			}
			state, err = state.AdvanceBlock()
			if err != nil {
				return ErrCorpus
			}
		}
		before := state.AppHash()
		next, result, err := researchv2.EvaluateTrusted(state, item.Profile, raw)
		if err != nil || result.Code != item.ExpectedCode || result.LastStage != item.ExpectedStage || (result.Code != researchv2.CodeOK && next.AppHash() != before) {
			return fmt.Errorf("%w: oracle mismatch in %s", ErrCorpus, item.ID)
		}
		if err := verifyScenarioMetadata(item); err != nil {
			return err
		}
	}
	for _, p := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		for _, name := range []string{"benign_low", "benign_high", "benign_burst", "valid_high_rate", "invalid_early", "late_state_invalid"} {
			if counts[p][name] != 1 {
				return ErrCorpus
			}
		}
		if p.UsesSecp() && counts[p]["invalid_ecdsa"] != 1 || p == researchv2.ProfileH && counts[p]["invalid_ml_after_ecdsa"] != 1 || p == researchv2.ProfileP && counts[p]["invalid_ml_p"] != 1 {
			return ErrCorpus
		}
	}
	return nil
}

func verifyScenarioMetadata(item ScenarioCase) error {
	switch item.Scenario {
	case "benign_low", "benign_high", "benign_burst", "valid_high_rate":
		if item.Traffic != "legitimate" || item.Capability != "C_registered_key_holder" || item.ExpectedCode != researchv2.CodeOK || item.ExpectedStage != researchv2.StageAccepted {
			return ErrCorpus
		}
		prior := map[string]int{"benign_low": 0, "benign_high": 1, "benign_burst": 2, "valid_high_rate": 3}
		if item.PriorTransfers != prior[item.Scenario] {
			return ErrCorpus
		}
	case "invalid_early":
		if item.Traffic != "adversarial" || item.Capability != "A_arbitrary_sender" || item.PriorTransfers != 0 || item.ExpectedCode != researchv2.CodeMalformed || item.ExpectedStage != researchv2.StageStructural {
			return ErrCorpus
		}
	case "invalid_ecdsa":
		if item.Traffic != "adversarial" || item.Capability != "B_signed_transaction_modifier" || item.PriorTransfers != 0 || !item.Profile.UsesSecp() || item.ExpectedCode != researchv2.CodeBadSecpSignature || item.ExpectedStage != researchv2.StageSecp {
			return ErrCorpus
		}
	case "invalid_ml_p", "invalid_ml_after_ecdsa":
		if item.Traffic != "adversarial" || item.Capability != "B_signed_transaction_modifier" || item.PriorTransfers != 0 || item.ExpectedCode != researchv2.CodeBadMLSignature || item.ExpectedStage != researchv2.StageML ||
			(item.Scenario == "invalid_ml_p" && item.Profile != researchv2.ProfileP) || (item.Scenario == "invalid_ml_after_ecdsa" && item.Profile != researchv2.ProfileH) {
			return ErrCorpus
		}
	case "late_state_invalid":
		if item.Traffic != "adversarial" || item.Capability != "C_registered_key_holder" || item.ExpectedCode != researchv2.CodeInsufficientBalance || item.ExpectedStage != researchv2.StageState {
			return ErrCorpus
		}
	default:
		return ErrCorpus
	}
	return nil
}
