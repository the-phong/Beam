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

func TestFile02R15(t *testing.T) {
	root := os.Getenv("FILE02_R15_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	parent, pe := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, be := os.Stat(allowed)
	info, ie := os.Lstat(abs)
	if root == "" || pe != nil || be != nil || ie != nil || !os.SameFile(parent, base) ||
		filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("private ignored R15 raw root required")
	}
	configPath, _ := filepath.Abs("../../research/config/s06_pilot.json")
	c, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	profiles := []struct {
		name string
		code researchv2.Profile
	}{{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP}, {"P", researchv2.ProfileP}, {"H", researchv2.ProfileH}}
	for _, rate := range []int{20, 23, 25} {
		for repetition := 0; repetition < 3; repetition++ {
			for position := 0; position < 4; position++ {
				profile := profiles[(position+repetition)%4]
				runID := fmt.Sprintf("r15-r%02d-n%d-%s", rate, repetition, profile.name)
				if passed := t.Run(runID, func(t *testing.T) {
					fixture, err := load.NewPilotFixture("pluto-file02-r15", 301)
					if err != nil {
						t.Fatal(err)
					}
					plan, err := load.BuildKneePilotPlan(fixture, profile.code, runID,
						int64(20000+rate*100+repetition*4+position), c, hash, rate)
					if err != nil {
						t.Fatal(err)
					}
					file02ExecuteQA(t, c, configPath, hash, fixture, profile.code,
						filepath.Join(root, "runs", runID), plan, map[string]any{
							"schema": "pluto.file02.r15_run.v1", "scope": "R_CONFIRMATORY", "case_id": "R15",
							"profile": profile.name, "rate": rate, "repetition": repetition,
							"position": position, "order": position, "seed": plan.Seed, "recorder_mode": "ON",
							"commit_observer_mode": "block_scan",
						})
				}); !passed {
					t.Fatal("R15 stopped after failed run; no replacement")
				}
			}
		}
	}
}
