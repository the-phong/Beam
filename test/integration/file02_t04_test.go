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

func TestFile02T04MixedConsensus(t *testing.T) {
	root := os.Getenv("FILE02_T04_ROOT")
	if root == "" {
		t.Fatal("FILE02_T04_ROOT is required")
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
		t.Fatal("FILE02_T04_ROOT must be directly below ignored research/artifacts/file02")
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
			fixture, err := load.NewPilotFixture("pluto-file02-t04", 16)
			if err != nil {
				t.Fatal(err)
			}
			runID := fmt.Sprintf("t04-%02d-%s", index, profile.name)
			plan, err := load.BuildPilotPlan(fixture, profile.code,
				load.FixedLegitimatePlusAdversary, "constant", runID,
				int64(3401+index), c, configHash, 0)
			if err != nil {
				t.Fatal(err)
			}
			file02ExecuteQA(t, c, configPath, configHash, fixture, profile.code,
				filepath.Join(root, "runs", runID), plan, map[string]any{
					"schema": "pluto.file02.t04_run.v1", "scope": "T_PREFLIGHT",
					"case_id": "T04", "profile": profile.name, "seed": 3401 + index,
					"oracle_contract": "valid_invalid_exact_commit_same_height_state_supply",
				})
		})
	}
}
