package main

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestReplayAcceptsFullStateAndRejectsCorruption(t *testing.T) {
	c, configHash, err := load.ReadConfig("../../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := load.NewPilotFixture("pluto-q07-test", 5)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := load.BuildPilotPlan(fixture, researchv2.ProfileC, load.EqualTxRate,
		"constant", "Q07-test", 1, c, configHash, 1)
	if err != nil {
		t.Fatal(err)
	}
	state, err := researchv2.ParseGenesis(fixture.Genesis, fixture.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	commits := commitRecord{RunID: plan.RunID}
	for _, event := range plan.Events {
		wire, err := hex.DecodeString(event.BytesHex)
		if err != nil {
			t.Fatal(err)
		}
		state, _, err = researchv2.EvaluateTrusted(state, plan.Profile, wire)
		if err != nil {
			t.Fatal(err)
		}
		commits.Events = append(commits.Events, commitEvent{ID: event.ID,
			SHA256: event.SHA256, Height: 1})
	}
	for i := 0; i < 2; i++ {
		state, err = state.AdvanceBlock()
		if err != nil {
			t.Fatal(err)
		}
	}
	hash := state.AppHash()
	oracle := oracleRecord{RunID: plan.RunID, Status: "OK", CommonHeight: 3,
		AppHash: hex.EncodeToString(hash[:])}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "genesis.bin"), fixture.Genesis, 0600); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(dir, "plan.json"), plan)
	writeJSON(t, filepath.Join(dir, "commit_observations.json"), commits)
	writeJSON(t, filepath.Join(dir, "oracle.json"), oracle)
	result, err := replay(dir, fixture.ChainID)
	if err != nil || result.Status != "PASS" || result.InitialSupply != 10 ||
		result.FinalSupply != 10 || result.ValidReplayed != len(plan.Events) {
		t.Fatalf("valid replay: %+v %v", result, err)
	}
	commits.Events[0].SHA256 = "corrupted"
	writeJSON(t, filepath.Join(dir, "commit_observations.json"), commits)
	if _, err := replay(dir, fixture.ChainID); err == nil {
		t.Fatal("tampered commit hash accepted")
	}
	commits.Events[0].SHA256 = plan.Events[0].SHA256
	writeJSON(t, filepath.Join(dir, "commit_observations.json"), commits)
	oracle.AppHash = "corrupted"
	writeJSON(t, filepath.Join(dir, "oracle.json"), oracle)
	if _, err := replay(dir, fixture.ChainID); err == nil {
		t.Fatal("tampered AppHash accepted")
	}
}
