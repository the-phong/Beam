//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

type statusResponse struct {
	Result struct {
		NodeInfo struct {
			Network string `json:"network"`
		} `json:"node_info"`
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
			LatestAppHash     string `json:"latest_app_hash"`
		} `json:"sync_info"`
	} `json:"result"`
}

type appInfoResponse struct {
	Result struct {
		Response struct {
			LastBlockHeight  string `json:"last_block_height"`
			LastBlockAppHash string `json:"last_block_app_hash"`
		} `json:"response"`
	} `json:"result"`
}

type blockResponse struct {
	Result struct {
		Block struct {
			Header struct {
				Height  string `json:"height"`
				AppHash string `json:"app_hash"`
			} `json:"header"`
			Data struct {
				Txs []string `json:"txs"`
			} `json:"data"`
		} `json:"block"`
	} `json:"result"`
}

type validatorsResponse struct {
	Result struct {
		Count string `json:"count"`
	} `json:"result"`
}

type stateResponse struct {
	ChainID string `json:"chain_id"`
	Version uint32 `json:"state_version"`
	Height  int64  `json:"height"`
	AppHash string `json:"app_hash"`
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func rpcGet(t *testing.T, port int, path string, target any) error {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	r, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("RPC HTTP %d", r.StatusCode)
	}
	return json.NewDecoder(r.Body).Decode(target)
}

func waitHeight(t *testing.T, port int, minimum int64) (int64, string) {
	t.Helper()
	deadline := time.Now().Add(35 * time.Second)
	for time.Now().Before(deadline) {
		var status statusResponse
		if err := rpcGet(t, port, "/status", &status); err == nil {
			h, _ := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64)
			if h >= minimum {
				if status.Result.NodeInfo.Network != "pluto-integration" {
					t.Fatalf("wrong network %q", status.Result.NodeInfo.Network)
				}
				return h, status.Result.SyncInfo.LatestAppHash
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("node did not reach height %d", minimum)
	return 0, ""
}

func TestSingleValidatorRestart(t *testing.T) {
	bin, err := filepath.Abs("../../build/plutod")
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(t.TempDir(), "node")
	rpc, p2p, abci := freePort(t), freePort(t), freePort(t)
	if rpc == p2p || rpc == abci || p2p == abci {
		t.Skip("ephemeral port collision")
	}
	init := exec.Command(bin, "init", "--home", home, "--chain-id", "pluto-integration", "--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abci))
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	var running *exec.Cmd
	var logFile *os.File
	logPaths := make(map[string]string)
	stop := func() {
		if running == nil {
			return
		}
		_ = running.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- running.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("stop: %v", err)
			}
		case <-time.After(15 * time.Second):
			_ = running.Process.Kill()
			<-done
			t.Error("stop timed out")
		}
		running = nil
		if logFile != nil {
			_ = logFile.Close()
			logFile = nil
		}
	}
	t.Cleanup(stop)
	start := func(name string) {
		logPath := filepath.Join(t.TempDir(), name+".log")
		f, err := os.Create(logPath)
		if err != nil {
			t.Fatal(err)
		}
		logFile = f
		logPaths[name] = logPath
		running = exec.Command(bin, "start", "--home", home)
		running.Stdout, running.Stderr = f, f
		if err := running.Start(); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s log: %s", name, logPath)
	}
	readState := func() stateResponse {
		out, err := exec.Command(bin, "state", "--home", home).CombinedOutput()
		if err != nil {
			t.Fatalf("state: %v\n%s", err, out)
		}
		var s stateResponse
		// Pebble may log before JSON; use the final line.
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if err := json.Unmarshal([]byte(lines[len(lines)-1]), &s); err != nil {
			t.Fatalf("decode state: %v\n%s", err, out)
		}
		return s
	}
	start("first")
	h1, hash1 := waitHeight(t, rpc, 2)
	var info appInfoResponse
	if err := rpcGet(t, rpc, "/abci_info", &info); err != nil {
		t.Fatal(err)
	}
	var block blockResponse
	if err := rpcGet(t, rpc, "/block?height=2", &block); err != nil {
		t.Fatal(err)
	}
	if block.Result.Block.Header.Height != "2" || len(block.Result.Block.Data.Txs) != 0 {
		t.Fatalf("invalid RPC block: %+v", block)
	}
	var validators validatorsResponse
	if err := rpcGet(t, rpc, "/validators?height=2", &validators); err != nil {
		t.Fatal(err)
	}
	if validators.Result.Count != "1" {
		t.Fatalf("validator count: %+v", validators)
	}
	t.Logf("first RPC: height=%d header_app_hash=%s application_height=%s application_app_hash_base64=%s", h1, hash1, info.Result.Response.LastBlockHeight, info.Result.Response.LastBlockAppHash)
	t.Logf("block RPC: height=%s app_hash=%s tx_count=%d validator_count=%s", block.Result.Block.Header.Height, block.Result.Block.Header.AppHash, len(block.Result.Block.Data.Txs), validators.Result.Count)
	stop()
	assertLog := func(name string) {
		t.Helper()
		data, err := os.ReadFile(logPaths[name])
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, marker := range []string{"Completed ABCI Handshake", "Started node", "finalized block"} {
			if !strings.Contains(text, marker) {
				t.Fatalf("%s log lacks %q", name, marker)
			}
			for _, line := range strings.Split(text, "\n") {
				if strings.Contains(line, marker) {
					t.Logf("%s node log: %s", name, line)
					break
				}
			}
		}
	}
	assertLog("first")
	before := readState()
	if before.ChainID != "pluto-integration" || before.Version != 1 || before.Height < 2 {
		t.Fatalf("invalid committed state: %+v", before)
	}
	start("restart")
	h2, hash2 := waitHeight(t, rpc, before.Height+2)
	t.Logf("restart RPC: height=%d header_app_hash=%s recovered_height=%d recovered_app_hash=%s", h2, hash2, before.Height, before.AppHash)
	stop()
	assertLog("restart")
	after := readState()
	if after.ChainID != before.ChainID || after.Version != before.Version || after.Height < before.Height+2 || after.AppHash == before.AppHash {
		t.Fatalf("restart did not advance: before=%+v after=%+v", before, after)
	}
	t.Logf("durable after restart: %+v", after)
	// A missing application database beside committed CometBFT state must not
	// silently initialize a new application. This mutates only t.TempDir data.
	dbDir := filepath.Join(home, "data", "pluto")
	backup := filepath.Join(home, "data", "pluto-test-backup")
	if err := os.Rename(dbDir, backup); err != nil {
		t.Fatal(err)
	}
	probe := exec.Command(bin, "start", "--home", home)
	output, probeErr := probe.CombinedOutput()
	if err := os.Rename(backup, dbDir); err != nil {
		t.Fatal(err)
	}
	if probeErr == nil || !strings.Contains(string(output), "refusing implicit reset") {
		t.Fatalf("missing application database not rejected: err=%v output=%s", probeErr, output)
	}
	t.Log("missing application database with existing CometBFT state was rejected")
}
