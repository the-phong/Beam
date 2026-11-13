//go:build integration && fourval

package integration

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
	"example.com/pluto/internal/experiment/telemetry"
)

// TestS07ReproSmoke is the real-network execution stage of the S07 command.
// S07_RUN_ROOT is created exclusively by the orchestrator beneath ignored
// research/artifacts/s07. No private raw file is copied into tracked evidence.
func TestS07ReproSmoke(t *testing.T) {
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	sessionRoot := os.Getenv("S07_RUN_ROOT")
	if sessionRoot == "" {
		sessionRoot = t.TempDir()
	} else {
		allowed, err := filepath.Abs("../../research/artifacts/s07")
		if err != nil {
			t.Fatal(err)
		}
		abs, err := filepath.Abs(sessionRoot)
		if err != nil {
			t.Fatal(err)
		}
		allowedInfo, err := os.Stat(allowed)
		if err != nil {
			t.Fatal(err)
		}
		parentInfo, err := os.Stat(filepath.Dir(abs))
		if err != nil || !os.SameFile(allowedInfo, parentInfo) {
			t.Fatal("S07_RUN_ROOT must be beneath ignored research/artifacts/s07")
		}
		if info, err := os.Lstat(abs); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("S07_RUN_ROOT must be an existing private directory")
		}
		sessionRoot = abs
	}
	profiles := []struct {
		name string
		code researchv2.Profile
	}{
		{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP},
		{"P", researchv2.ProfileP}, {"H", researchv2.ProfileH},
	}
	for i, profile := range profiles {
		i, profile := i, profile
		t.Run(profile.name, func(t *testing.T) {
			phaseOrigin := time.Now()
			runRoot := filepath.Join(sessionRoot, "profile-"+profile.name)
			if err := os.Mkdir(runRoot, 0700); err != nil {
				t.Fatalf("run ID already exists or cannot be created: %v", err)
			}
			phase, err := os.OpenFile(filepath.Join(runRoot, "phases.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer phase.Close()
			mark := func(name, detail string) {
				t.Helper()
				kind := "observed_transition"
				if name == "warm-up" || name == "execute" || name == "drain" {
					kind = "delegated_schedule"
				}
				if err := json.NewEncoder(phase).Encode(map[string]any{"phase": name, "detail": detail, "event_kind": kind,
					"observed_utc":         time.Now().UTC().Format(time.RFC3339Nano),
					"elapsed_monotonic_ns": time.Since(phaseOrigin).Nanoseconds()}); err != nil {
					t.Fatal(err)
				}
			}
			mark("setup", "fresh four-validator network and private fixture")
			fixture, err := load.NewPilotFixture("pluto-s07-smoke", 16)
			if err != nil {
				t.Fatal(err)
			}
			runID := os.Getenv("S07_SESSION_ID")
			if runID == "" {
				runID = fmt.Sprintf("s07-direct-%d", time.Now().UnixNano())
			}
			runID += "-" + profile.name
			seed := int64(711 + i)
			plan, err := load.BuildPilotPlan(fixture, profile.code, load.FixedLegitimatePlusAdversary, "constant", runID, seed, c, configHash, 0)
			if err != nil {
				t.Fatal(err)
			}
			planPath := filepath.Join(runRoot, "plan.json")
			if err := load.WritePlan(planPath, plan, c, configHash); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(runRoot, "genesis.bin"), fixture.Genesis, 0600); err != nil {
				t.Fatal(err)
			}
			pilot := startS06NetworkAt(t, c, fixture, profile.code, runRoot)
			pilot.startObservers(t, c)
			nodes, err := json.Marshal(pilot.nodes)
			if err != nil {
				t.Fatal(err)
			}
			if err := s07WriteExclusive(filepath.Join(runRoot, "nodes.json"), nodes); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(10 * time.Second)
			for {
				ready := true
				for _, watched := range pilot.nodes {
					info, err := os.Stat(watched.ObserverFile)
					if err != nil || info.Size() == 0 {
						ready = false
					}
				}
				if ready {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("four observers did not become ready")
				}
				time.Sleep(100 * time.Millisecond)
			}
			host, _ := os.Hostname()
			meta := map[string]any{
				"schema": "pluto.s07.run.v1", "run_id": runID, "profile": profile.name,
				"seed": seed, "source_commit": os.Getenv("S07_SOURCE_COMMIT"),
				"source_tree": os.Getenv("S07_SOURCE_TREE"), "config_sha256": configHash,
				"corpus_sha256": os.Getenv("S07_CORPUS_SHA256"), "genesis_sha256": plan.GenesisSHA256,
				"host": host, "go_os": runtime.GOOS, "go_arch": runtime.GOARCH,
				"logical_cpus": runtime.NumCPU(), "placement": "four_validators_one_host_smoke",
				"regime": plan.Regime, "target_legitimate_tx_per_second": plan.TargetLegitTPS,
				"target_invalid_tx_per_second": plan.TargetInvalidTPS,
			}
			if err := s07WriteJSON(filepath.Join(runRoot, "metadata.json"), meta); err != nil {
				t.Fatal(err)
			}
			mark("warm-up", "load process executes pinned warm-up schedule")
			mark("execute", "load process executes pinned open-loop steady schedule")
			mark("drain", "load process waits for pinned drain interval")
			observed := make(chan []s07CommitObservation, 1)
			stopObservation := make(chan struct{})
			var stopOnce sync.Once
			stopObserve := func() { stopOnce.Do(func() { close(stopObservation) }) }
			t.Cleanup(stopObserve)
			go s07ObserveCommits(pilot, plan, stopObservation, observed)
			loadBinary, _ := filepath.Abs("../../build/profile-load")
			cmd := exec.Command(loadBinary, "-config", configPath, "-plan", planPath,
				"-genesis", filepath.Join(runRoot, "genesis.bin"), "-nodes", filepath.Join(runRoot, "nodes.json"),
				"-rpc", pilot.nodes[0].RPC, "-data-root", pilot.n.root, "-kill-file", filepath.Join(runRoot, "STOP"),
				"-result", filepath.Join(runRoot, "result.json"), "-allocated-cores", fmt.Sprint(runtime.NumCPU()))
			output, runErr := cmd.CombinedOutput()
			if err := s07WriteExclusive(filepath.Join(runRoot, "load.log"), output); err != nil {
				t.Fatal(err)
			}
			if runErr != nil {
				t.Fatalf("bounded load failed: %v (private load.log)", runErr)
			}
			var result load.Result
			raw, err := os.ReadFile(filepath.Join(runRoot, "result.json"))
			if err != nil || json.Unmarshal(raw, &result) != nil || result.Status != load.StatusOK || result.ActualSendCount != len(plan.Events) {
				t.Fatalf("load result invalid: %v status=%s", err, result.Status)
			}
			mark("oracle", "independent common-height block and account checks")
			pilot.n.waitAll(8, 30*time.Second)
			oracle := s07Oracle(t, pilot, fixture, plan, result)
			var commits []s07CommitObservation
			select {
			case commits = <-observed:
			case <-time.After(10 * time.Second):
				stopObserve()
				commits = <-observed
			}
			if len(commits) != oracle.ValidCommitted {
				t.Fatalf("all-four inclusion observations missing: %d/%d", len(commits), oracle.ValidCommitted)
			}
			if err := s07WriteJSON(filepath.Join(runRoot, "commit_observations.json"), map[string]any{
				"schema": "pluto.s07.commits.v1", "run_id": plan.RunID, "poll_interval_ms": 100,
				"observation_semantics": "first RPC tx-index observation on all four validators; upper bound on inclusion time",
				"events":                commits,
			}); err != nil {
				t.Fatal(err)
			}
			if err := s07WriteJSON(filepath.Join(runRoot, "oracle.json"), oracle); err != nil {
				t.Fatal(err)
			}
			mark("collect", "four observer streams retained as private raw telemetry")
			var p2pSeen bool
			for _, watched := range pilot.nodes {
				file, err := os.Open(watched.ObserverFile)
				if err != nil {
					t.Fatal(err)
				}
				scan := bufio.NewScanner(file)
				var count int
				for scan.Scan() {
					var sample telemetry.NodeSample
					if err := json.Unmarshal(scan.Bytes(), &sample); err != nil || sample.Schema != telemetry.Schema {
						t.Fatalf("invalid observer stream: %v", err)
					}
					count++
					if sample.P2PIngressTxTotal != nil && *sample.P2PIngressTxTotal > 0 {
						p2pSeen = true
					}
				}
				_ = file.Close()
				if err := scan.Err(); err != nil || count == 0 {
					t.Fatalf("observer stream empty or invalid: %v", err)
				}
			}
			if !p2pSeen {
				t.Fatal("no direct P2P relay observation")
			}
			if rawCooldown := os.Getenv("S07_FILE02_COOLDOWN_MS"); rawCooldown != "" {
				cooldownMS, err := strconv.Atoi(rawCooldown)
				if err != nil || cooldownMS < 1 || cooldownMS > 5000 {
					t.Fatal("invalid bounded File 02 cooldown")
				}
				startNS := time.Since(phaseOrigin).Nanoseconds()
				start := time.Now()
				time.Sleep(time.Duration(cooldownMS) * time.Millisecond)
				pilot.n.waitAll(oracle.CommonHeight, 5*time.Second)
				endNS := time.Since(phaseOrigin).Nanoseconds()
				if err := s07WriteJSON(filepath.Join(runRoot, "cooldown.json"), map[string]any{
					"schema": "pluto.file02.cooldown_qa.v1", "run_id": runID,
					"planned_ms": cooldownMS, "elapsed_ns": time.Since(start).Nanoseconds(),
					"start_monotonic_ns": startNS, "end_monotonic_ns": endNS,
					"network_live": true,
				}); err != nil {
					t.Fatal(err)
				}
				mark("cooldown", "separate monotonic cooldown after drain, oracle and telemetry collection")
			}
			t.Logf("S07 smoke profile=%s run_id=%s valid=%d invalid=%d height=%d", profile.name, runID, oracle.ValidCommitted, oracle.InvalidRejected, oracle.CommonHeight)
		})
	}
}

type s07OracleRecord struct {
	Schema                   string           `json:"schema"`
	RunID                    string           `json:"run_id"`
	Status                   string           `json:"status"`
	ObservedUTC              string           `json:"observed_utc"`
	CommonHeight             int64            `json:"common_height"`
	BlockID                  string           `json:"block_id"`
	AppHash                  string           `json:"app_hash"`
	ValidCommitted           int              `json:"valid_committed"`
	InvalidRejected          int              `json:"invalid_rejected"`
	CommittedHashes          map[string]int   `json:"committed_hashes,omitempty"`
	ValidatorCommittedHashes []map[string]int `json:"validator_committed_hashes,omitempty"`
}

type s07CommitObservation struct {
	ID                     string `json:"id"`
	SHA256                 string `json:"sha256"`
	AllFourSeenUTC         string `json:"all_four_seen_utc"`
	AllFourSeenMonotonicNS *int64 `json:"all_four_seen_monotonic_ns,omitempty"`
	Height                 int64  `json:"height"`
}

// Polling observes the first indexed result on each real validator. It is a
// conservative same-host upper bound, not an exact consensus commit instant.
func s07ObserveCommits(pilot *s06Network, plan load.Plan, stop <-chan struct{}, output chan<- []s07CommitObservation) {
	type pending struct {
		event  load.PlannedTx
		seen   [4]bool
		height int64
		latest time.Time
	}
	items := make([]*pending, 0)
	for _, event := range plan.Events {
		if event.Traffic == "legitimate" {
			items = append(items, &pending{event: event})
		}
	}
	client := http.Client{Timeout: 350 * time.Millisecond, Transport: &http.Transport{Proxy: nil}}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	completed := map[string]s07CommitObservation{}
	ordered := func() []s07CommitObservation {
		out := make([]s07CommitObservation, 0, len(completed))
		for _, item := range items {
			if record, ok := completed[item.event.ID]; ok {
				out = append(out, record)
			}
		}
		return out
	}
	for {
		select {
		case <-stop:
			output <- ordered()
			return
		case <-ticker.C:
			for _, item := range items {
				if _, ok := completed[item.event.ID]; ok {
					continue
				}
				for node := range pilot.n.nodes {
					if item.seen[node] {
						continue
					}
					height, ok := s07LookupTx(&client, pilot.nodes[node].RPC, item.event.SHA256)
					if ok && (item.height == 0 || item.height == height) {
						item.seen[node] = true
						item.height = height
						item.latest = time.Now()
					}
				}
				if item.seen[0] && item.seen[1] && item.seen[2] && item.seen[3] {
					completed[item.event.ID] = s07CommitObservation{ID: item.event.ID, SHA256: item.event.SHA256,
						AllFourSeenUTC: item.latest.UTC().Format(time.RFC3339Nano), Height: item.height}
				}
			}
			if len(completed) == len(items) {
				output <- ordered()
				return
			}
		}
	}
}

func s07LookupTx(client *http.Client, origin, hash string) (int64, bool) {
	response, err := client.Get(origin + "/tx?hash=0x" + hash)
	if err != nil {
		return 0, false
	}
	defer response.Body.Close()
	var payload struct {
		Result struct {
			Height   string `json:"height"`
			TxResult struct {
				Code uint32 `json:"code"`
			} `json:"tx_result"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload) != nil || payload.Error != nil || payload.Result.TxResult.Code != 0 {
		return 0, false
	}
	height, err := strconv.ParseInt(payload.Result.Height, 10, 64)
	return height, err == nil && height > 0
}

func s07Oracle(t *testing.T, pilot *s06Network, fixture *load.PilotFixture, plan load.Plan, result load.Result) s07OracleRecord {
	t.Helper()
	if result.RunID != plan.RunID || result.ConfigSHA256 != plan.ConfigSHA256 || result.GenesisSHA256 != plan.GenesisSHA256 || len(result.Events) != len(plan.Events) {
		t.Fatal("run identity or event count differs from plan")
	}
	common := int64(1 << 62)
	for i := range pilot.n.nodes {
		height, _, err := pilot.n.height(i)
		if err != nil || height < 8 {
			t.Fatalf("validator %d below oracle height: %d %v", i, height, err)
		}
		if height < common {
			common = height
		}
	}
	var reference map[string]int
	var lastBlock, lastApp string
	for i := range pilot.n.nodes {
		seen := map[string]int{}
		for h := int64(1); h <= common; h++ {
			blockID, appHash, txs, err := pilot.n.block(i, h)
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 && h == common {
				lastBlock, lastApp = blockID, appHash
			} else if i != 0 && h == common && (blockID != lastBlock || appHash != lastApp) {
				t.Fatal("same-height block/AppHash divergence")
			}
			for _, encoded := range txs {
				raw, err := base64.StdEncoding.DecodeString(encoded)
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(raw)
				seen[hex.EncodeToString(sum[:])]++
			}
		}
		if i == 0 {
			reference = seen
		} else if !s07EqualCounts(reference, seen) {
			t.Fatal("validator committed transaction sets differ")
		}
	}
	var valid, invalid int
	for i, event := range plan.Events {
		actual := result.Events[i]
		if actual.ID != event.ID || actual.ActualCode == nil || *actual.ActualCode != uint32(event.ExpectedCode) || actual.ActualOffsetNS == nil {
			t.Fatalf("CheckTx oracle mismatch for %s", event.ID)
		}
		if event.Traffic == "legitimate" {
			valid++
			if reference[event.SHA256] != 1 {
				t.Fatalf("valid event %s committed %d times", event.ID, reference[event.SHA256])
			}
		} else {
			invalid++
			if reference[event.SHA256] != 0 {
				t.Fatalf("invalid event %s committed", event.ID)
			}
		}
	}
	for i := range pilot.n.nodes {
		balance, nonce := s04Account(t, pilot.n, i, fixture.Receiver.Binding.Account)
		if balance != uint64(valid) || nonce != 0 {
			t.Fatalf("validator %d receiver state differs: %d/%d", i, balance, nonce)
		}
		for _, event := range plan.Events {
			balance, nonce = s04Account(t, pilot.n, i, fixture.Senders[event.AccountSlot].Binding.Account)
			if event.Traffic == "legitimate" && (balance != 1 || nonce != 1) || event.Traffic == "adversarial" && (balance != 2 || nonce != 0) {
				t.Fatalf("validator %d sender slot %d state differs: %d/%d", i, event.AccountSlot, balance, nonce)
			}
		}
	}
	return s07OracleRecord{Schema: "pluto.s07.oracle.v1", RunID: plan.RunID, Status: "OK", ObservedUTC: time.Now().UTC().Format(time.RFC3339Nano), CommonHeight: common,
		BlockID: lastBlock, AppHash: lastApp, ValidCommitted: valid, InvalidRejected: invalid}
}

func s07EqualCounts(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if a[key] != b[key] {
			return false
		}
	}
	return true
}

func s07WriteJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return s07WriteExclusive(path, append(raw, '\n'))
}

func s07WriteExclusive(path string, raw []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(raw); err != nil {
		return err
	}
	return f.Sync()
}
