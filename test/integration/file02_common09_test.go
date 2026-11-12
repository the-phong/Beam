//go:build integration && fourval

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// TestFile02Common09ForcedStop intentionally fails after the bounded
// generator honors its private kill file; its orchestrator audits the raw.
func TestFile02Common09ForcedStop(t *testing.T) {
	root := os.Getenv("FILE02_COMMON09_ROOT")
	if root == "" {
		t.Fatal("FILE02_COMMON09_ROOT required")
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
		t.Fatal("COMMON09_ROOT must be private and ignored")
	}
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := load.NewPilotFixture("pluto-file02-common09", 90)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := load.BuildPilotPlan(fixture, researchv2.ProfileH, load.EqualTxRate,
		"constant", "common09-forced-stop-h", 1909, c, configHash, 2)
	if err != nil {
		t.Fatal(err)
	}
	file02ExecuteQA(t, c, configPath, configHash, fixture, researchv2.ProfileH,
		filepath.Join(root, "run"), plan, map[string]any{
			"schema": "pluto.file02.common09_run.v1", "profile": "H",
			"case_id": "COMMON09_QA_STOP", "actor_capability": "C",
			"inject_stop_before_load": true,
		})
}
