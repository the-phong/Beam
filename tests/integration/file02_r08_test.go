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

func TestFile02R08(t *testing.T) {
	root := os.Getenv("FILE02_R08_ROOT")
	if root == "" {
		t.Fatal("FILE02_R08_ROOT is required")
	}
	allowed, err := filepath.Abs("../../research/artifacts/file02")
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	parent, parentErr := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, baseErr := os.Stat(allowed)
	info, rootErr := os.Lstat(abs)
	if parentErr != nil || baseErr != nil || rootErr != nil || !os.SameFile(parent, base) || filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("R08 private raw root")
	}
	raw, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schedule struct {
		Schema string `json:"schema"`
		Jobs   []struct {
			Block   int    `json:"block"`
			Order   int    `json:"order"`
			Profile string `json:"profile"`
			Arm     string `json:"arm"`
			Seed    int64  `json:"seed"`
			RunID   string `json:"run_id"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(raw, &schedule); err != nil || schedule.Schema != "pluto.file02.r08.schedule.v1" || len(schedule.Jobs) != 32 {
		t.Fatal("invalid R08 schedule")
	}
	profiles := map[string]researchv2.Profile{"C": researchv2.ProfileC, "CP": researchv2.ProfileCP, "P": researchv2.ProfileP, "H": researchv2.ProfileH}
	for block := 0; block < 4; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-r08", 81)
		if err != nil {
			t.Fatal(err)
		}
		for order := 0; order < 8; order++ {
			job := schedule.Jobs[8*block+order]
			p, ok := profiles[job.Profile]
			if !ok || job.Block != block || job.Order != order || job.Seed <= 0 || job.RunID == "" || (job.Arm != "baseline" && job.Arm != "count1") {
				t.Fatal("invalid R08 job")
			}
			if passed := t.Run(fmt.Sprintf("b%02d-%s-%s", block, job.Profile, job.Arm), func(t *testing.T) {
				configPath := filepath.Join(root, "config-"+job.Arm+".json")
				c, configHash, err := load.ReadConfig(configPath)
				if err != nil {
					t.Fatal(err)
				}
				plan, err := load.BuildPilotPlan(fixture, p, load.EqualTxRate, "constant", job.RunID, job.Seed, c, configHash, 20)
				if err != nil {
					t.Fatal(err)
				}
				file02ExecuteQA(t, c, configPath, configHash, fixture, p, filepath.Join(root, "runs", job.RunID), plan, map[string]any{
					"schema": "pluto.file02.r08_run.v1", "scope": "R_CONFIRMATORY", "case_id": "R08",
					"profile": job.Profile, "arm": job.Arm, "block": job.Block, "order": job.Order, "seed": job.Seed,
					"recorder_mode": "ON", "capture_commit_observations": false,
				})
			}); !passed {
				t.Fatal("R08 stopped after failed arm; no replacement")
			}
			time.Sleep(2 * time.Second)
		}
	}
}
