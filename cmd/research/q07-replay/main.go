// q07-replay reconstructs a run's committed application state from frozen
// genesis and included valid transactions. It emits only keyless QC facts.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	researchv2 "example.com/pluto/internal/experiment/profiles"
)

type plannedEvent struct {
	ID       string `json:"id"`
	BytesHex string `json:"bytes_hex"`
	SHA256   string `json:"sha256"`
	Traffic  string `json:"traffic"`
}
type planRecord struct {
	RunID   string             `json:"run_id"`
	Profile researchv2.Profile `json:"profile"`
	Events  []plannedEvent     `json:"events"`
}
type commitEvent struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
	Height int64  `json:"height"`
}
type commitRecord struct {
	RunID  string        `json:"run_id"`
	Events []commitEvent `json:"events"`
}
type oracleRecord struct {
	RunID        string `json:"run_id"`
	Status       string `json:"status"`
	CommonHeight int64  `json:"common_height"`
	AppHash      string `json:"app_hash"`
}
type replayResult struct {
	Schema            string `json:"schema"`
	RunID             string `json:"run_id"`
	CommonHeight      int64  `json:"common_height"`
	GenesisAccounts   int    `json:"genesis_accounts"`
	InitialSupply     uint64 `json:"initial_supply"`
	FinalSupply       uint64 `json:"final_supply"`
	ValidReplayed     int    `json:"valid_replayed"`
	InvalidNotApplied int    `json:"invalid_not_applied"`
	AppHash           string `json:"app_hash"`
	Status            string `json:"status"`
}

func readJSON(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}

func replay(runDir, chainID string) (replayResult, error) {
	var plan planRecord
	var commits commitRecord
	var oracle oracleRecord
	for _, item := range []struct {
		name string
		into any
	}{{"plan.json", &plan}, {"commit_observations.json", &commits}, {"oracle.json", &oracle}} {
		if err := readJSON(filepath.Join(runDir, item.name), item.into); err != nil {
			return replayResult{}, err
		}
	}
	if plan.RunID == "" || commits.RunID != plan.RunID || oracle.RunID != plan.RunID ||
		oracle.Status != "OK" || oracle.CommonHeight < 2 || !plan.Profile.Valid() {
		return replayResult{}, errors.New("run/oracle identity")
	}
	genesis, err := os.ReadFile(filepath.Join(runDir, "genesis.bin"))
	if err != nil {
		return replayResult{}, err
	}
	state, err := researchv2.ParseGenesis(genesis, chainID)
	if err != nil {
		return replayResult{}, err
	}
	initial, accountCount := state.TotalSupply(), len(state.Base().Accounts())
	byID := make(map[string]plannedEvent, len(plan.Events))
	for _, event := range plan.Events {
		if event.ID == "" || byID[event.ID].ID != "" {
			return replayResult{}, errors.New("duplicate planned event")
		}
		byID[event.ID] = event
	}
	byHeight := make(map[int64][]plannedEvent)
	seen := make(map[string]bool, len(commits.Events))
	for _, committed := range commits.Events {
		event, ok := byID[committed.ID]
		if !ok || event.Traffic != "legitimate" || seen[committed.ID] ||
			committed.Height < 1 || committed.Height >= oracle.CommonHeight ||
			!strings.EqualFold(event.SHA256, committed.SHA256) {
			return replayResult{}, errors.New("invalid/duplicate/missing committed event")
		}
		seen[committed.ID] = true
		byHeight[committed.Height] = append(byHeight[committed.Height], event)
	}
	invalid := 0
	for _, event := range plan.Events {
		if event.Traffic == "legitimate" && !seen[event.ID] {
			return replayResult{}, errors.New("valid event absent from commits")
		}
		if event.Traffic == "adversarial" {
			invalid++
		}
	}
	// Comet's block header at height H carries the application hash from
	// committed state H-1. The live oracle recorded that header at H.
	for height := int64(1); height < oracle.CommonHeight; height++ {
		for _, event := range byHeight[height] {
			wire, err := hex.DecodeString(event.BytesHex)
			if err != nil {
				return replayResult{}, err
			}
			sum := sha256.Sum256(wire)
			if !strings.EqualFold(hex.EncodeToString(sum[:]), event.SHA256) {
				return replayResult{}, errors.New("committed wire hash mismatch")
			}
			next, outcome, err := researchv2.EvaluateTrusted(state, plan.Profile, wire)
			if err != nil || outcome.Code != researchv2.CodeOK {
				return replayResult{}, errors.New("valid replay rejected")
			}
			state = next
		}
		state, err = state.AdvanceBlock()
		if err != nil {
			return replayResult{}, err
		}
	}
	hash := state.AppHash()
	appHash := strings.ToUpper(hex.EncodeToString(hash[:]))
	if appHash != strings.ToUpper(oracle.AppHash) || state.TotalSupply() != initial ||
		len(state.Base().Accounts()) != accountCount {
		return replayResult{}, fmt.Errorf("replayed AppHash/supply/accounts differ: hash=%s oracle=%s supply=%d initial=%d accounts=%d initial_accounts=%d", appHash, oracle.AppHash, state.TotalSupply(), initial, len(state.Base().Accounts()), accountCount)
	}
	return replayResult{Schema: "pluto.file02.q07_replay.v1", RunID: plan.RunID,
		CommonHeight: oracle.CommonHeight, GenesisAccounts: accountCount,
		InitialSupply: initial, FinalSupply: state.TotalSupply(),
		ValidReplayed: len(commits.Events), InvalidNotApplied: invalid,
		AppHash: appHash, Status: "PASS"}, nil
}

func main() {
	runDir := flag.String("run-dir", "", "private frozen run directory")
	chainID := flag.String("chain-id", "", "expected chain ID")
	flag.Parse()
	if *runDir == "" || *chainID == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "run-dir and chain-id required")
		os.Exit(2)
	}
	result, err := replay(*runDir, *chainID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
