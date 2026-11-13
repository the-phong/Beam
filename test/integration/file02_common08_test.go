//go:build integration && fourval

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// TestFile02Common08QCEvidence records independent observer streams on one
// disposable bounded QA network for the fault-exclusion contract.
func TestFile02Common08QCEvidence(t *testing.T) {
	root := os.Getenv("FILE02_COMMON08_ROOT")
	if root == "" {
		t.Fatal("FILE02_COMMON08_ROOT is required")
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
		t.Fatal("COMMON08_ROOT must be a private directory directly below ignored research/artifacts/file02")
	}
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := load.NewPilotFixture("pluto-file02-common08", 90)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := load.BuildPilotPlan(fixture, researchv2.ProfileH, load.EqualTxRate,
		"constant", "common08-qa-h", 1808, c, configHash, 2)
	if err != nil {
		t.Fatal(err)
	}
	file02ExecuteQA(t, c, configPath, configHash, fixture, researchv2.ProfileH,
		filepath.Join(root, "run"), plan, map[string]any{
			"schema": "pluto.file02.common08_run.v1", "profile": "H", "case_id": "COMMON08_QA",
			"actor_capability": "C", "capture_commit_observations": true,
		})
}
