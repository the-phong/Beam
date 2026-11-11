// Package researchv2corpus defines the public S04 correctness workload.
package researchv2corpus

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

type Case struct {
	CaseID          string             `json:"case_id"`
	Profile         researchv2.Profile `json:"profile"`
	BytesHex        string             `json:"bytes_hex"`
	SHA256          string             `json:"sha256"`
	ExpectedCode    researchv2.Code    `json:"expected_code"`
	Stage           researchv2.Stage   `json:"stage"`
	ActorCapability string             `json:"actor_capability"`
	Precondition    string             `json:"precondition"`
	SourcePath      string             `json:"source_path"`
	ScenarioLabel   string             `json:"scenario_label"`
	OracleResult    string             `json:"oracle_result"`
}
type Corpus struct {
	Version    int    `json:"version"`
	ChainID    string `json:"chain_id"`
	GenesisHex string `json:"genesis_hex"`
	Cases      []Case `json:"cases"`
}

var ErrCorpus = errors.New("invalid S04 corpus")

func Build() (Corpus, error) {
	f, err := researchv2test.New("pluto-s04-public")
	if err != nil {
		return Corpus{}, err
	}
	c := Corpus{Version: 1, ChainID: f.ChainID, GenesisHex: hex.EncodeToString(f.Genesis)}
	add := func(id string, p researchv2.Profile, raw []byte, code researchv2.Code, stage researchv2.Stage, actor, pre string) {
		h := sha256.Sum256(raw)
		c.Cases = append(c.Cases, Case{CaseID: id, Profile: p, BytesHex: hex.EncodeToString(raw), SHA256: hex.EncodeToString(h[:]), ExpectedCode: code, Stage: stage, ActorCapability: actor, Precondition: pre, SourcePath: "direct_abci", ScenarioLabel: "S04_CORRECTNESS_ONLY", OracleResult: "verified"})
	}
	for _, p := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		name := fmt.Sprintf("p%d", p)
		valid, err := f.Signed(p, 7, 0)
		if err != nil {
			return Corpus{}, err
		}
		add(name+"-valid", p, valid, researchv2.CodeOK, researchv2.StageAccepted, "C_registered_key_holder", "genesis_h0")
		add(name+"-duplicate", p, valid, researchv2.CodeInvalidNonce, researchv2.StageState, "B_signed_transaction_modifier", "valid_nonce0_committed")
		add(name+"-truncated", p, valid[:len(valid)-1], researchv2.CodeMalformed, researchv2.StageStructural, "B_signed_transaction_modifier", "genesis_h0")
		wrongProfile := researchv2.ProfileC
		if p == wrongProfile {
			wrongProfile = researchv2.ProfileH
		}
		wrong, err := f.Signed(wrongProfile, 7, 0)
		if err != nil {
			return Corpus{}, err
		}
		add(name+"-wrong-profile", p, wrong, researchv2.CodeWrongProfile, researchv2.StageProfile, "C_registered_key_holder", "genesis_h0")
		chain := bytes.Clone(valid)
		chain[12] ^= 1
		add(name+"-wrong-chain", p, chain, researchv2.CodeWrongChain, researchv2.StageChain, "B_signed_transaction_modifier", "genesis_h0")
		zero := bytes.Clone(valid)
		amountOffset := 12 + len(f.ChainID) + 40
		for i := 0; i < 8; i++ {
			zero[amountOffset+i] = 0
		}
		add(name+"-zero-amount", p, zero, researchv2.CodeZeroAmount, researchv2.StageAmount, "B_signed_transaction_modifier", "genesis_h0")
		binding := bytes.Clone(valid)
		binding[12+len(f.ChainID)] ^= 1
		add(name+"-unbound-sender", p, binding, researchv2.CodeBadBinding, researchv2.StageBinding, "B_signed_transaction_modifier", "genesis_h0")
		proof := bytes.Clone(valid)
		code, stage := researchv2.CodeBadSecpSignature, researchv2.StageSecp
		if p == researchv2.ProfileP {
			code, stage = researchv2.CodeBadMLSignature, researchv2.StageML
		}
		proof[12+len(f.ChainID)+56] ^= 1
		add(name+"-bad-proof", p, proof, code, stage, "B_signed_transaction_modifier", "genesis_h0")
		if p == researchv2.ProfileH {
			ml := bytes.Clone(valid)
			ml[len(ml)-1] ^= 1
			add(name+"-bad-ml", p, ml, researchv2.CodeBadMLSignature, researchv2.StageML, "B_signed_transaction_modifier", "genesis_h0")
		}
		if p == researchv2.ProfileCP {
			pad := bytes.Clone(valid)
			pad[len(pad)-1] = 1
			add(name+"-nonzero-pad", p, pad, researchv2.CodeMalformed, researchv2.StageStructural, "B_signed_transaction_modifier", "genesis_h0")
		}
	}
	return c, nil
}

func Write(path string, c Corpus) error {
	if err := Verify(c); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func Read(path string) (Corpus, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Corpus{}, err
	}
	var c Corpus
	if err := json.Unmarshal(raw, &c); err != nil {
		return Corpus{}, err
	}
	return c, Verify(c)
}
func Verify(c Corpus) error {
	if c.Version != 1 || len(c.Cases) < 32 {
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
	profileCounts := map[researchv2.Profile]int{}
	validCases := map[researchv2.Profile][]byte{}
	for _, item := range c.Cases {
		if item.CaseID == fmt.Sprintf("p%d-valid", item.Profile) && item.ExpectedCode == researchv2.CodeOK {
			valid, err := hex.DecodeString(item.BytesHex)
			if err != nil {
				return ErrCorpus
			}
			validCases[item.Profile] = valid
		}
	}
	for _, item := range c.Cases {
		if seen[item.CaseID] || !item.Profile.Valid() {
			return ErrCorpus
		}
		seen[item.CaseID] = true
		profileCounts[item.Profile]++
		raw, err := hex.DecodeString(item.BytesHex)
		if err != nil {
			return ErrCorpus
		}
		h := sha256.Sum256(raw)
		if hex.EncodeToString(h[:]) != item.SHA256 {
			return ErrCorpus
		}
		state := initial
		if item.Precondition == "valid_nonce0_committed" {
			valid, found := validCases[item.Profile]
			if !found {
				return ErrCorpus
			}
			var result researchv2.Result
			state, result, err = researchv2.EvaluateTrusted(state, item.Profile, valid)
			if err != nil || result.Code != researchv2.CodeOK {
				return ErrCorpus
			}
			state, err = state.AdvanceBlock()
			if err != nil {
				return ErrCorpus
			}
		} else if item.Precondition != "genesis_h0" {
			return ErrCorpus
		}
		before := state.AppHash()
		next, result, err := researchv2.EvaluateTrusted(state, item.Profile, raw)
		if err != nil || result.Code != item.ExpectedCode || result.LastStage != item.Stage {
			return fmt.Errorf("%w: %s code/stage", ErrCorpus, item.CaseID)
		}
		if result.Code != researchv2.CodeOK && next.AppHash() != before {
			return fmt.Errorf("%w: %s mutation", ErrCorpus, item.CaseID)
		}
		if item.ActorCapability != "C_registered_key_holder" && item.ActorCapability != "B_signed_transaction_modifier" {
			return ErrCorpus
		}
		if item.SourcePath != "direct_abci" || item.ScenarioLabel != "S04_CORRECTNESS_ONLY" || item.OracleResult != "verified" {
			return ErrCorpus
		}
	}
	for _, profile := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		want := 8
		if profile == researchv2.ProfileCP || profile == researchv2.ProfileH {
			want = 9
		}
		if profileCounts[profile] != want || len(validCases[profile]) == 0 {
			return ErrCorpus
		}
	}
	return nil
}
