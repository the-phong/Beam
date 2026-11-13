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

func TestFile02T06FormalCalibration(t *testing.T) {
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("FILE02_T06_FORMAL_ROOT")
	if root == "" {
		t.Fatal("FILE02_T06_FORMAL_ROOT is required")
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
		t.Fatal("FILE02_T06_FORMAL_ROOT must be a private directory directly below ignored research/artifacts/file02")
	}
	profiles := []struct {
		name string
		code researchv2.Profile
	}{
		{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP},
		{"P", researchv2.ProfileP}, {"H", researchv2.ProfileH},
	}
	for _, rate := range []int{2, 6, 12, 20} {
		rate := rate
		t.Run(fmt.Sprintf("rate-%02d", rate), func(t *testing.T) {
			for i, profile := range profiles {
				i, profile := i, profile
				t.Run(profile.name, func(t *testing.T) {
					fixture, err := load.NewPilotFixture("pluto-file02-t06-formal", 90)
					if err != nil {
						t.Fatal(err)
					}
					runID := fmt.Sprintf("t06-formal-r%02d-%s", rate, profile.name)
					plan, err := load.BuildPilotPlan(fixture, profile.code, load.EqualTxRate,
						"constant", runID, int64(3600+rate*10+i), c, configHash, rate)
					if err != nil {
						t.Fatal(err)
					}
					file02ExecuteQA(t, c, configPath, configHash, fixture, profile.code,
						filepath.Join(root, fmt.Sprintf("rate-%02d", rate), profile.name), plan, map[string]any{
							"schema": "pluto.file02.t06_formal.v1", "scope": "T_PREFLIGHT",
							"case_id": "T06", "profile": profile.name, "target_tx_s": rate,
							"capture_commit_observations": false,
						})
				})
			}
		})
	}
}
