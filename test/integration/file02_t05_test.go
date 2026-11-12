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

func TestFile02T05RecorderPairs(t *testing.T) {
	root := os.Getenv("FILE02_T05_ROOT")
	if root == "" {
		t.Fatal("FILE02_T05_ROOT is required")
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
		t.Fatal("FILE02_T05_ROOT must be directly below ignored research/artifacts/file02")
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
			fixture, err := load.NewPilotFixture("pluto-file02-t05", 16)
			if err != nil {
				t.Fatal(err)
			}
			basePlan, err := load.BuildPilotPlan(fixture, profile.code, load.EqualTxRate,
				"constant", fmt.Sprintf("t05-%02d-base", index), int64(3501+index),
				c, configHash, 2)
			if err != nil {
				t.Fatal(err)
			}
			modes := []string{"ON", "OFF"}
			if index%2 == 1 {
				modes = []string{"OFF", "ON"}
			}
			for order, mode := range modes {
				mode := mode
				t.Run(mode, func(t *testing.T) {
					plan := basePlan
					plan.RunID = fmt.Sprintf("t05-%02d-%s", index, mode)
					file02ExecuteQA(t, c, configPath, configHash, fixture, profile.code,
						filepath.Join(root, "runs", plan.RunID), plan, map[string]any{
							"schema": "pluto.file02.t05_run.v1", "scope": "T_PREFLIGHT",
							"case_id": "T05", "profile": profile.name, "seed": 3501 + index,
							"recorder_mode": mode, "pair_order": order,
						})
				})
			}
		})
	}
}
