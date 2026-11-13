//go:build integration && fourval

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// TestFile02T01Baseline creates new, bounded four-validator correctness runs.
// The oracle in file02ExecuteQA verifies each valid transfer exactly once,
// sender nonce/balance, receiver balance and same-height block/AppHash.
func TestFile02T01Baseline(t *testing.T) {
	root := os.Getenv("FILE02_T01_ROOT")
	if root == "" {
		t.Fatal("FILE02_T01_ROOT is required")
	}
	allowed, err := filepath.Abs("../../research/artifacts/file02")
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	parent, parentErr := os.Stat(filepath.Dir(abs))
	base, baseErr := os.Stat(allowed)
	info, rootErr := os.Lstat(abs)
	if parentErr != nil || baseErr != nil || rootErr != nil || !os.SameFile(parent, base) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("FILE02_T01_ROOT must be directly below ignored research/artifacts/file02")
	}
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	profiles := []struct {
		name string
		code researchv2.Profile
	}{
		{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP},
		{"P", researchv2.ProfileP}, {"H", researchv2.ProfileH},
	}
	for index, profile := range profiles {
		index, profile := index, profile
		t.Run(profile.name, func(t *testing.T) {
			fixture, err := load.NewPilotFixture("pluto-file02-t01", 16)
			if err != nil {
				t.Fatal(err)
			}
			runID := fmt.Sprintf("t01-%02d-%s", index, profile.name)
			plan, err := load.BuildPilotPlan(fixture, profile.code, load.EqualTxRate,
				"constant", runID, int64(3101+index), c, configHash, 2)
			if err != nil {
				t.Fatal(err)
			}
			file02ExecuteQA(t, c, configPath, configHash, fixture, profile.code,
				filepath.Join(root, "runs", runID), plan, map[string]any{
					"schema": "pluto.file02.t01_run.v1", "scope": "T_PREFLIGHT",
					"case_id": "T01", "profile": profile.name, "seed": 3101 + index,
					"oracle_contract": "valid_exactly_once_account_state_same_height_block_apphash",
				})
		})
	}
}
