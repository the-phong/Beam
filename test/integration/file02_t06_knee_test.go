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

func TestFile02T06KneePilot(t *testing.T) {
	root := os.Getenv("FILE02_T06_KNEE_ROOT")
	allowed, err := filepath.Abs("../../research/artifacts/file02")
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	parent, e1 := os.Stat(filepath.Dir(abs))
	base, e2 := os.Stat(allowed)
	info, e3 := os.Lstat(abs)
	if root == "" || e1 != nil || e2 != nil || e3 != nil || !os.SameFile(parent, base) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("private ignored knee session required")
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
	}{{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP}, {"P", researchv2.ProfileP}, {"H", researchv2.ProfileH}}
	for _, rate := range []int{20, 23, 25} {
		for repetition := 0; repetition < 3; repetition++ {
			for position := 0; position < 4; position++ {
				profile := profiles[(position+repetition)%4]
				runID := fmt.Sprintf("t06k-r%02d-n%d-%s", rate, repetition, profile.name)
				if passed := t.Run(runID, func(t *testing.T) {
					fixture, err := load.NewPilotFixture("pluto-file02-t06-knee", 301)
					if err != nil {
						t.Fatal(err)
					}
					plan, err := load.BuildKneePilotPlan(fixture, profile.code, runID, int64(20000+rate*100+repetition*4+position), c, configHash, rate)
					if err != nil {
						t.Fatal(err)
					}
					file02ExecuteQA(t, c, configPath, configHash, fixture, profile.code, filepath.Join(root, "runs", runID), plan, map[string]any{
						"schema": "pluto.file02.t06_knee_run.v1", "profile": profile.name, "rate": rate, "repetition": repetition,
						"position": position, "seed": plan.Seed, "commit_observer_mode": "block_scan",
					})
				}); !passed {
					t.Fatal("knee pilot stopped at first failed run")
				}
			}
		}
	}
}
