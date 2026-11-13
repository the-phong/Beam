//go:build integration && fourval

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// TestFile02R14 runs the frozen valid-only recheck schedule. Each run has
// independent state and a fresh four-validator Comet network.
func TestFile02R14(t *testing.T) {
	root := os.Getenv("FILE02_R14_ROOT")
	if root == "" {
		t.Fatal("FILE02_R14_ROOT required")
	}
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	parent, pe := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, be := os.Stat(allowed)
	info, ie := os.Lstat(abs)
	if pe != nil || be != nil || ie != nil || !os.SameFile(parent, base) || filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("private R14 raw root required")
	}
	var schedule struct {
		Schema string `json:"schema"`
		Jobs   []struct {
			Block   int    `json:"block"`
			Order   int    `json:"order"`
			Profile string `json:"profile"`
			Seed    int64  `json:"seed"`
			RunID   string `json:"run_id"`
		} `json:"jobs"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil || json.Unmarshal(raw, &schedule) != nil || schedule.Schema != "pluto.file02.r14.schedule.v1" || len(schedule.Jobs) != 16 {
		t.Fatal("invalid R14 schedule")
	}
	configPath := filepath.Join(root, "config.json")
	c, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	profiles := map[string]researchv2.Profile{"C": researchv2.ProfileC, "CP": researchv2.ProfileCP, "P": researchv2.ProfileP, "H": researchv2.ProfileH}
	for block := 0; block < 4; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-r14", 81)
		if err != nil {
			t.Fatal(err)
		}
		for order := 0; order < 4; order++ {
			job := schedule.Jobs[4*block+order]
			p, ok := profiles[job.Profile]
			if !ok || job.Block != block || job.Order != order || job.Seed <= 0 || job.RunID == "" {
				t.Fatal("invalid R14 job")
			}
			if passed := t.Run(fmt.Sprintf("b%02d-%s", block, job.Profile), func(t *testing.T) {
				plan, err := load.BuildPilotPlan(fixture, p, load.EqualTxRate, "constant", job.RunID, job.Seed, c, hash, 20)
				if err != nil {
					t.Fatal(err)
				}
				file02ExecuteQA(t, c, configPath, hash, fixture, p, filepath.Join(root, "runs", job.RunID), plan, map[string]any{
					"schema": "pluto.file02.r14_run.v1", "scope": "R_CONFIRMATORY", "case_id": "R14",
					"profile": job.Profile, "block": job.Block, "order": job.Order, "seed": job.Seed,
					"recorder_mode": "ON", "capture_commit_observations": false,
				})
			}); !passed {
				t.Fatal("R14 stopped after failed run; no replacement")
			}
			time.Sleep(2 * time.Second)
		}
	}
}
