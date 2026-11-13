//go:build integration && fourval

package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"example.com/pluto/internal/experiment/load"
)

type file02A14Observation struct {
	ID                   string    `json:"id"`
	SHA256               string    `json:"sha256"`
	Height               int64     `json:"height"`
	FirstSeenMonotonicNS [4]int64  `json:"first_seen_monotonic_ns"`
	FirstSeenUTC         [4]string `json:"first_seen_utc"`
}

func file02A14ObservePerNode(pilot *s06Network, plan load.Plan, stop <-chan struct{}, output chan<- []file02A14Observation) {
	items := make(map[string]*file02A14Observation, len(plan.Events))
	order := make([]string, 0, len(plan.Events))
	for _, event := range plan.Events {
		items[event.SHA256] = &file02A14Observation{ID: event.ID, SHA256: event.SHA256}
		order = append(order, event.SHA256)
	}
	result := func() []file02A14Observation {
		rows := make([]file02A14Observation, 0, len(order))
		for _, hash := range order {
			item := items[hash]
			if item.FirstSeenMonotonicNS == [4]int64{} {
				continue
			}
			rows = append(rows, *item)
		}
		return rows
	}
	var next [4]int64
	for i := range next {
		next[i] = 1
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			output <- result()
			return
		case <-ticker.C:
			for node := range next {
				top, _, err := pilot.n.height(node)
				if err != nil {
					continue
				}
				for height := next[node]; height <= top; height++ {
					_, _, txs, err := pilot.n.block(node, height)
					if err != nil {
						break
					}
					var ts unix.Timespec
					_ = unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts)
					stamp := time.Now().UTC().Format(time.RFC3339Nano)
					for _, encoded := range txs {
						wire, err := base64.StdEncoding.DecodeString(encoded)
						if err != nil {
							continue
						}
						sum := sha256.Sum256(wire)
						item := items[hex.EncodeToString(sum[:])]
						if item != nil && item.FirstSeenMonotonicNS[node] == 0 {
							item.FirstSeenMonotonicNS[node] = ts.Nano()
							item.FirstSeenUTC[node] = stamp
							item.Height = height
						}
					}
					next[node] = height + 1
				}
			}
			all := true
			for _, item := range items {
				for _, ns := range item.FirstSeenMonotonicNS {
					if ns == 0 {
						all = false
						break
					}
				}
			}
			if all {
				output <- result()
				return
			}
		}
	}
}

func file02A14Snapshot(pilot *s06Network) [4]*int64 {
	var heights [4]*int64
	client := http.Client{Timeout: 250 * time.Millisecond, Transport: &http.Transport{Proxy: nil}}
	for i, watched := range pilot.nodes {
		response, err := client.Get(watched.RPC + "/status")
		if err != nil {
			continue
		}
		var body statusResponse
		decodeErr := json.NewDecoder(response.Body).Decode(&body)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK || decodeErr != nil {
			continue
		}
		height, err := strconv.ParseInt(body.Result.SyncInfo.LatestBlockHeight, 10, 64)
		if err == nil {
			heights[i] = &height
		}
	}
	return heights
}

func file02A14RunWithFault(cmd *exec.Cmd, pilot *s06Network, run string, arm any) ([]byte, error) {
	condition, ok := arm.(string)
	if !ok || (condition != "control" && condition != "fault") {
		return nil, fmt.Errorf("invalid A14 fault condition")
	}
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	started := time.Now()
	trace := map[string]any{"schema": "pluto.file02.a14.fault_trace.v1", "condition": condition,
		"fault_validator": 3, "process_scope": "validator-3 CometBFT process only",
		"started_utc": started.UTC().Format(time.RFC3339Nano), "pause_target_ms": 2000,
		"pause_duration_target_ms": 1500, "source": "local process SIGSTOP/SIGCONT"}
	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(time.Until(started.Add(2 * time.Second)))
		trace["before_heights"] = file02A14Snapshot(pilot)
		if condition == "fault" {
			trace["pause_elapsed_ms"] = time.Since(started).Milliseconds()
			err := pilot.n.nodes[3].cmd.Process.Signal(syscall.SIGSTOP)
			trace["pause_status"] = "OK"
			if err != nil {
				trace["pause_status"] = err.Error()
				return
			}
			paused := time.Now()
			defer func() {
				trace["resume_elapsed_ms"] = time.Since(started).Milliseconds()
				err := pilot.n.nodes[3].cmd.Process.Signal(syscall.SIGCONT)
				trace["resume_status"] = "OK"
				if err != nil {
					trace["resume_status"] = err.Error()
				}
			}()
			time.Sleep(250 * time.Millisecond)
			trace["during_heights"] = file02A14Snapshot(pilot)
			time.Sleep(time.Until(paused.Add(1500 * time.Millisecond)))
		} else {
			time.Sleep(500 * time.Millisecond)
			trace["during_heights"] = file02A14Snapshot(pilot)
			trace["pause_status"] = "NOT_APPLICABLE_CONTROL"
			trace["resume_status"] = "NOT_APPLICABLE_CONTROL"
		}
	}()
	err := cmd.Wait()
	<-done
	trace["after_heights"] = file02A14Snapshot(pilot)
	trace["load_exit_ok"] = err == nil
	if writeErr := s07WriteJSON(filepath.Join(run, "fault_trace.json"), trace); writeErr != nil {
		return output.Bytes(), writeErr
	}
	return output.Bytes(), err
}
