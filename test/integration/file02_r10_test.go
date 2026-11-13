//go:build integration && fourval

package integration

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
	"example.com/pluto/internal/experiment/telemetry"
)

func TestFile02R10(t *testing.T) {
	root := os.Getenv("FILE02_R10_ROOT")
	if root == "" {
		t.Fatal("FILE02_R10_ROOT required")
	}
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	parent, pe := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, be := os.Stat(allowed)
	info, ie := os.Lstat(abs)
	if pe != nil || be != nil || ie != nil || !os.SameFile(parent, base) || filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("private R10 raw root required")
	}
	configPath, _ := filepath.Abs("../../research/config/s06_pilot.json")
	c, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	chainID := "pluto-r10-" + "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	if len(chainID) != researchv2.MaxChainID {
		t.Fatalf("chain length %d", len(chainID))
	}
	fixture, err := load.NewPilotFixture(chainID, 9)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := load.BuildPilotPlan(fixture, researchv2.ProfileH, load.EqualTxRate, "constant", "r10-max-bound", 20261113, c, hash, 2)
	if err != nil {
		t.Fatal(err)
	}
	run := filepath.Join(root, "run")
	if err := os.Mkdir(run, 0700); err != nil {
		t.Fatal(err)
	}
	if err := load.WritePlan(filepath.Join(run, "signed_corpus_plan.json"), plan, c, hash); err != nil {
		t.Fatal(err)
	}
	if err := s07WriteExclusive(filepath.Join(run, "genesis.bin"), fixture.Genesis); err != nil {
		t.Fatal(err)
	}
	valid, err := hex.DecodeString(plan.Events[0].BytesHex)
	if err != nil || len(valid) != 2602 {
		t.Fatalf("valid H max wire %d: %v", len(valid), err)
	}
	malformed := append(append([]byte(nil), valid...), make([]byte, 8192-len(valid))...)
	oversize := append(append([]byte(nil), valid...), make([]byte, 8193-len(valid))...)
	pilot := startS06NetworkAt(t, c, fixture, researchv2.ProfileH, t.TempDir())
	pilot.startObservers(t, c)
	deadline := time.Now().Add(10 * time.Second)
	for {
		ready := true
		for _, node := range pilot.nodes {
			stat, err := os.Stat(node.ObserverFile)
			if err != nil || stat.Size() == 0 {
				ready = false
				break
			}
		}
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("observers not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	var configs [4]string
	for i := range configs {
		path := filepath.Join(pilot.n.root, fmt.Sprintf("validator-%d/config/config.toml", i))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		configs[i] = hex.EncodeToString(sum[:])
	}
	lock := map[string]any{"schema": "pluto.file02.r10.run_plan.v1", "status": "LOCKED", "case_id": "R10",
		"source_commit": os.Getenv("FILE02_SOURCE_COMMIT"), "source_tree": os.Getenv("FILE02_SOURCE_TREE"),
		"config_sha256": hash, "genesis_sha256": plan.GenesisSHA256, "signed_corpus_plan_sha256": file02HashFile(t, filepath.Join(run, "signed_corpus_plan.json")),
		"rpc_endpoint": pilot.nodes[0].RPC, "validator_config_sha256": configs,
		"sizes_bytes": []int{2602, 8192, 8193}, "spacing_seconds": 1, "origin": "RPC_LOOPBACK",
		"safety_caps": c.Safety, "topology": "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC"}
	if err := s07WriteJSON(filepath.Join(run, "research_plan.json"), lock); err != nil {
		t.Fatal(err)
	}
	guard, err := load.NewHostGuard(c, pilot.nodes, pilot.n.root, filepath.Join(run, "STOP"), runtime.NumCPU())
	if err != nil {
		t.Fatal(err)
	}
	sender, err := load.NewHTTPSender(pilot.nodes[0].RPC, time.Duration(c.Safety.RequestTimeoutMS)*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	raws := [][]byte{valid, malformed, oversize}
	labels := []string{"valid_max_canonical", "invalid_at_limit", "invalid_over_limit"}
	rows := make([]map[string]any, 0, 3)
	for i, wire := range raws {
		if err := guard.Check(context.Background()); err != nil {
			t.Fatalf("safety before %s: %v", labels[i], err)
		}
		start := time.Now()
		result := sender.Send(context.Background(), wire)
		duration := time.Since(start)
		sum := sha256.Sum256(wire)
		var code any
		if result.Code != nil {
			code = *result.Code
		}
		rows = append(rows, map[string]any{"label": labels[i], "sha256": hex.EncodeToString(sum[:]), "wire_bytes": len(wire),
			"rpc_code": code, "rpc_reason": result.Reason, "rpc_error": result.Err != nil,
			"rpc_roundtrip_ns": duration.Nanoseconds(), "written_utc": result.WrittenAt.UTC().Format(time.RFC3339Nano),
			"completed_utc": result.Completed.UTC().Format(time.RFC3339Nano)})
		if i < 2 {
			time.Sleep(time.Second)
		}
	}
	if err := s07WriteJSON(filepath.Join(run, "rpc_results.json"), map[string]any{"schema": "pluto.file02.r10.rpc.v1", "rows": rows}); err != nil {
		t.Fatal(err)
	}
	pilot.n.waitAll(8, 30*time.Second)
	for i, watched := range pilot.nodes {
		raw, err := os.ReadFile(watched.ObserverFile)
		if err != nil {
			t.Fatal(err)
		}
		if err := s07WriteExclusive(filepath.Join(run, fmt.Sprintf("observer-%d.jsonl", i)), raw); err != nil {
			t.Fatal(err)
		}
	}
	var ingress []telemetry.CheckTxEvent
	for i, rec := range pilot.recorders {
		events, dropped := rec.Drain()
		if dropped != 0 {
			t.Fatal("recorder drop")
		}
		if i == 0 {
			ingress = events
		}
		if err := s07WriteJSON(filepath.Join(run, fmt.Sprintf("recorder-%d.json", i)), map[string]any{"events": events, "dropped": dropped}); err != nil {
			t.Fatal(err)
		}
	}
	newEvents := map[string]telemetry.CheckTxEvent{}
	for _, event := range ingress {
		if event.CheckType == "new" {
			newEvents[event.TxSHA256] = event
		}
	}
	if len(newEvents) != 2 {
		t.Fatalf("expected valid and malformed application CheckTx, got %d", len(newEvents))
	}
	validHash := rows[0]["sha256"].(string)
	malformedHash := rows[1]["sha256"].(string)
	oversizeHash := rows[2]["sha256"].(string)
	if rows[0]["rpc_code"] != uint32(0) || rows[0]["rpc_error"] != false || newEvents[validHash].Code != 0 || newEvents[validHash].LastStage != "accepted" {
		t.Fatal("maximal canonical valid transaction failed")
	}
	if rows[1]["rpc_code"] == nil || rows[1]["rpc_code"] == uint32(0) || newEvents[malformedHash].Code == 0 || newEvents[malformedHash].LastStage != "structural" {
		t.Fatal("at-limit malformed transaction did not fail structurally")
	}
	if rows[2]["rpc_reason"] != "mempool_tx_too_large" || rows[2]["rpc_error"] != true {
		t.Fatal("over-limit transaction did not fail at Comet size gate")
	}
	if _, ok := newEvents[oversizeHash]; ok {
		t.Fatal("over-limit transaction reached application CheckTx")
	}
	common := int64(1 << 62)
	for i := range pilot.n.nodes {
		height, _, err := pilot.n.height(i)
		if err != nil || height < 8 {
			t.Fatalf("node %d height: %v", i, err)
		}
		if height < common {
			common = height
		}
	}
	var reference map[string]int
	var blockID, appHash string
	for i := range pilot.n.nodes {
		seen := map[string]int{}
		for h := int64(1); h <= common; h++ {
			block, app, txs, err := pilot.n.block(i, h)
			if err != nil {
				t.Fatal(err)
			}
			if h == common {
				if i == 0 {
					blockID, appHash = block, app
				} else if blockID != block || appHash != app {
					t.Fatal("BlockID/AppHash mismatch")
				}
			}
			for _, encoded := range txs {
				wire, err := base64.StdEncoding.DecodeString(encoded)
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(wire)
				seen[hex.EncodeToString(sum[:])]++
			}
		}
		if i == 0 {
			reference = seen
		} else {
			if len(seen) != len(reference) {
				t.Fatal("commit set differs")
			}
			for hash, count := range reference {
				if seen[hash] != count {
					t.Fatal("commit count differs")
				}
			}
		}
	}
	if len(reference) != 1 || reference[validHash] != 1 || reference[malformedHash] != 0 || reference[oversizeHash] != 0 {
		t.Fatal("R10 inclusion oracle")
	}
	if err := s07WriteJSON(filepath.Join(run, "oracle.json"), map[string]any{"schema": "pluto.file02.r10.oracle.v1", "status": "OK", "common_height": common, "block_id": blockID, "app_hash": appHash,
		"valid_max_committed": 1, "invalid_at_limit_committed": 0, "invalid_over_limit_committed": 0,
		"valid_checktx_service_ns": newEvents[validHash].ServiceNS, "invalid_at_limit_checktx_service_ns": newEvents[malformedHash].ServiceNS,
		"invalid_at_limit_stage": newEvents[malformedHash].LastStage, "over_limit_checktx_reached": false,
		"comet_precheck_cpu_ns": nil, "comet_precheck_cpu_status": "UNAVAILABLE"}); err != nil {
		t.Fatal(err)
	}
}

func file02HashFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
