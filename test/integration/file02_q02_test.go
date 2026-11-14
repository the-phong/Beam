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

func TestFile02Q02Decomposition(t *testing.T) {
	root := os.Getenv("FILE02_Q02_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	parent, pe := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, be := os.Stat(allowed)
	info, ie := os.Lstat(abs)
	if root == "" || pe != nil || be != nil || ie != nil || !os.SameFile(parent, base) ||
		filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("private ignored Q02 raw root required")
	}
	configPath, _ := filepath.Abs("../../research/config/s06_pilot.json")
	c, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var schedule struct {
		Schema string `json:"schema"`
		Jobs   []struct {
			Block     int    `json:"block"`
			Order     int    `json:"order"`
			Profile   string `json:"profile"`
			Condition string `json:"condition"`
			RunID     string `json:"run_id"`
		} `json:"jobs"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil || json.Unmarshal(raw, &schedule) != nil ||
		schedule.Schema != "pluto.file02.q02.schedule.v2" || len(schedule.Jobs) != 96 {
		t.Fatal("invalid locked Q02 schedule")
	}
	profiles := map[string]researchv2.Profile{"C": researchv2.ProfileC,
		"CP": researchv2.ProfileCP, "P": researchv2.ProfileP, "H": researchv2.ProfileH}
	for block := 0; block < 12; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-q02", 241)
		if err != nil {
			t.Fatal(err)
		}
		for order := 0; order < 8; order++ {
			job := schedule.Jobs[8*block+order]
			profile, exists := profiles[job.Profile]
			if !exists || job.Block != block || job.Order != order ||
				(job.Condition != "benign" && job.Condition != "attack") {
				t.Fatal("Q02 schedule block/profile/arm drift")
			}
			benign, attack, err := load.BuildQ02StructuralPairPlans(fixture, profile, block, c, hash)
			if err != nil {
				t.Fatal(err)
			}
			plan := benign
			if job.Condition == "attack" {
				plan = attack
			}
			if job.RunID != plan.RunID {
				t.Fatal("Q02 run ID drift")
			}
			if passed := t.Run(fmt.Sprintf("b%02d-%s-%s", block, job.Profile, job.Condition), func(t *testing.T) {
				file02ExecuteQA(t, c, configPath, hash, fixture, profile,
					filepath.Join(root, "runs", job.RunID), plan, map[string]any{
						"schema": "pluto.file02.q02_run.v1", "scope": "Q_CONFIRMATORY",
						"case_id": "Q02", "profile": job.Profile, "block": block,
						"order": order, "condition": job.Condition, "seed": plan.Seed,
						"recorder_mode": "ON", "commit_observer_mode": "block_scan",
					})
			}); !passed {
				t.Fatal("Q02 stopped after failed arm; no replacement")
			}
			time.Sleep(2 * time.Second)
		}
	}
}
