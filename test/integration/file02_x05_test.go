//go:build integration && fourval

package integration

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

type x05ProcessSample struct {
	Schema         string `json:"schema"`
	PID            int    `json:"pid"`
	UserCPUNS      int64  `json:"user_cpu_ns"`
	SystemCPUNS    int64  `json:"system_cpu_ns"`
	RSSPeakBytes   int64  `json:"rss_peak_bytes"`
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`
	GCCount        uint32 `json:"gc_count"`
	GCPauseNS      uint64 `json:"gc_pause_total_ns"`
	Threads        int    `json:"threads"`
	ThreadStatus   string `json:"thread_status"`
	RecorderDrops  uint64 `json:"recorder_drops"`
}

func TestFile02X05DedicatedAppProcesses(t *testing.T) {
	runRoot := os.Getenv("FILE02_X05_ROOT")
	recorderMode := os.Getenv("FILE02_X05_RECORDER_MODE")
	if recorderMode == "" {
		recorderMode = "ON"
	}
	if recorderMode != "ON" && recorderMode != "OFF" {
		t.Fatal("FILE02_X05_RECORDER_MODE must be ON or OFF")
	}
	if runRoot == "" {
		runRoot = t.TempDir()
	} else {
		allowed, err := filepath.Abs("../../research/artifacts/file02")
		if err != nil {
			t.Fatal(err)
		}
		absolute, err := filepath.Abs(runRoot)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(absolute) != "raw" || filepath.Dir(filepath.Dir(absolute)) != allowed {
			t.Fatal("FILE02_X05_ROOT must be session/raw under ignored research/artifacts/file02")
		}
		info, err := os.Lstat(absolute)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("FILE02_X05_ROOT must be a real private directory")
		}
		runRoot = absolute
	}
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := load.NewPilotFixture("pluto-file02-x05-qa", 16)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := load.BuildPilotPlan(fixture, researchv2.ProfileH, load.EqualTxRate,
		"constant", "x05-app-process-qa", 20261115, c, configHash, 2)
	if err != nil {
		t.Fatal(err)
	}
	var valid []byte
	var validEvent load.PlannedTx
	for _, event := range plan.Events {
		if event.ExpectedCode == researchv2.CodeOK {
			validEvent = event
			valid, err = hex.DecodeString(event.BytesHex)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if len(valid) == 0 {
		t.Fatal("no valid pre-signed H transaction")
	}
	pilot := startX05NetworkAtRecorder(t, c, fixture, researchv2.ProfileH, runRoot, recorderMode == "ON")
	seenPID := map[int]bool{}
	for i, cmd := range pilot.appProcesses {
		if cmd == nil || cmd.Process == nil || cmd.Process.Pid == os.Getpid() || seenPID[cmd.Process.Pid] || pilot.apps[i] != nil {
			t.Fatalf("validator %d lacks an independent app PID", i)
		}
		seenPID[cmd.Process.Pid] = true
	}
	observationPlan := plan
	observationPlan.Events = []load.PlannedTx{validEvent}
	observationStop := make(chan struct{})
	observations := make(chan []s07CommitObservation, 1)
	go file02ObserveCommitsByBlock(pilot, observationPlan, observationStop, observations)
	t.Cleanup(func() { close(observationStop) })
	admission, err := pilot.n.broadcast(valid)
	if err != nil || admission.Result.CheckTx.Code != 0 || admission.Result.TxResult.Code != 0 {
		t.Fatalf("valid H transaction failed: %v check=%d deliver=%d", err,
			admission.Result.CheckTx.Code, admission.Result.TxResult.Code)
	}
	height, err := strconv.ParseInt(admission.Result.Height, 10, 64)
	if err != nil || height < 1 {
		t.Fatalf("invalid commit height %q: %v", admission.Result.Height, err)
	}
	pilot.n.waitAll(height+1, 30*time.Second)
	var observation s07CommitObservation
	select {
	case entries := <-observations:
		if len(entries) != 1 {
			t.Fatalf("expected one X08 per-validator observation, got %d", len(entries))
		}
		observation = entries[0]
	case <-time.After(10 * time.Second):
		t.Fatal("X08 per-validator observation timed out")
	}
	if observation.Height != height || observation.AllFourSeenMonotonicNS == nil {
		t.Fatalf("wrong X08 observation height/timestamp: %+v", observation)
	}
	for i := 0; i < 4; i++ {
		if observation.PerValidatorSeenUTC[i] == "" || observation.PerValidatorSeenMonotonicNS[i] <= 0 ||
			observation.PerValidatorSeenMonotonicNS[i] > *observation.AllFourSeenMonotonicNS {
			t.Fatalf("validator %d missing or late X08 observation", i)
		}
	}
	var blockID, appHash string
	for i := 0; i < 4; i++ {
		id, hash, _, err := pilot.n.block(i, height)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			blockID, appHash = id, hash
		} else if id != blockID || hash != appHash {
			t.Fatalf("four-validator block/AppHash divergence at height %d", height)
		}
	}
	time.Sleep(1100 * time.Millisecond) // one complete resource and event export tick
	resources := make([]map[string]any, 0, 4)
	for i, cmd := range pilot.appProcesses {
		raw, err := os.ReadFile(pilot.appResources[i])
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
		if len(lines) < 2 {
			t.Fatalf("validator %d needs two resource samples", i)
		}
		var first, last x05ProcessSample
		if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
			t.Fatal(err)
		}
		if first.PID != cmd.Process.Pid || last.PID != first.PID ||
			last.Schema != "pluto.file02.app_process_resources.v1" ||
			last.UserCPUNS+last.SystemCPUNS <= first.UserCPUNS+first.SystemCPUNS ||
			last.RSSPeakBytes <= 0 || last.Threads < 1 || last.ThreadStatus != "AVAILABLE" ||
			last.HeapAllocBytes == 0 || last.GCCount < first.GCCount || last.GCPauseNS < first.GCPauseNS {
			t.Fatalf("validator %d invalid dedicated-app resource identity/delta", i)
		}
		for _, line := range lines {
			var item x05ProcessSample
			if err := json.Unmarshal([]byte(line), &item); err != nil || item.RecorderDrops != 0 {
				t.Fatalf("validator %d recorder sample invalid/drop: %v", i, err)
			}
		}
		events, err := os.ReadFile(pilot.appEvents[i])
		if err != nil || (recorderMode == "ON" && i == 0 && len(events) == 0) ||
			(recorderMode == "OFF" && len(events) != 0) {
			t.Fatalf("validator %d recorder mode %s unexpected events: %v", i, recorderMode, err)
		}
		resources = append(resources, map[string]any{
			"validator": i, "pid": last.PID, "cpu_delta_ns": last.UserCPUNS + last.SystemCPUNS - first.UserCPUNS - first.SystemCPUNS,
			"rss_peak_bytes": last.RSSPeakBytes, "gc_count_delta": last.GCCount - first.GCCount,
			"gc_pause_delta_ns": last.GCPauseNS - first.GCPauseNS,
			"threads":           last.Threads, "thread_status": last.ThreadStatus,
			"resource_sample_count": len(lines),
		})
	}
	if os.Getenv("FILE02_X05_ROOT") != "" {
		summary := map[string]any{
			"schema": "pluto.file02.x05_process_qa.v1", "status": "PASS",
			"topology":              "four_comet_plus_four_dedicated_app_processes_single_host_loopback",
			"valid_h_commit_height": height, "block_id": blockID, "header_app_hash": appHash,
			"processes": resources, "per_validator_observation": observation,
			"recorder_mode": recorderMode, "recorder_drop_count": 0,
		}
		encoded, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(filepath.Join(runRoot, "x05-qa.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(append(encoded, '\n')); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
