//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"example.com/pluto/internal/classical/app"
	"example.com/pluto/internal/classical/genesis"
	"example.com/pluto/internal/node"
	"github.com/cometbft/cometbft/abci/server"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/types"
)

const b0GoldenGenesis = "504c55544f47322100000002000b706c75746f2d62302d763100000001b9631c0f8d9d2d9d56bd6151559f3339e14eb36a0000000000000064"

type capturedB0App struct {
	*b0app.App
	observed chan []byte
}

func (c *capturedB0App) InitChain(ctx context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	select {
	case c.observed <- bytes.Clone(req.AppStateBytes):
	default:
	}
	return c.App.InitChain(ctx, req)
}

func b0WaitHeight(t *testing.T, port int, minimum int64) {
	t.Helper()
	deadline := time.Now().Add(35 * time.Second)
	for time.Now().Before(deadline) {
		var status statusResponse
		if err := rpcGet(t, port, "/status", &status); err == nil {
			h, _ := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64)
			if h >= minimum && status.Result.NodeInfo.Network == "pluto-b0-v1" {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("B0 node did not reach expected height")
}

func TestPinnedCometGenesisTransportAndB0Restart(t *testing.T) {
	plutod, err := filepath.Abs("../../build/plutod")
	if err != nil {
		t.Fatal(err)
	}
	comet, err := filepath.Abs("../../build/cometbft")
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(t.TempDir(), "b0-home")
	genesisFile := filepath.Join(t.TempDir(), "genesis-v2.bin")
	binary, err := hex.DecodeString(b0GoldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(genesisFile, binary, 0600); err != nil {
		t.Fatal(err)
	}
	rpc, p2p, abciPort := 0, 0, 0
	for attempt := 0; attempt < 10; attempt++ {
		rpc, p2p, abciPort = freePort(t), freePort(t), freePort(t)
		if rpc != p2p && rpc != abciPort && p2p != abciPort {
			break
		}
	}
	if rpc == p2p || rpc == abciPort || p2p == abciPort {
		t.Fatal("could not allocate distinct ports")
	}
	args := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile,
		"--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abciPort)}
	if out, err := exec.Command(plutod, args...).CombinedOutput(); err != nil {
		t.Fatalf("init-b0: %v: %s", err, out)
	}
	genesis, err := types.GenesisDocFromFile(filepath.Join(home, "config", "genesis.json"))
	if err != nil {
		t.Fatal(err)
	}
	expectedTransport, err := b0genesis.Encode(binary, "pluto-b0-v1")
	if err != nil || !bytes.Equal(genesis.AppState, expectedTransport) {
		t.Fatalf("Comet genesis app_state mismatch: %v", err)
	}
	config, err := node.ReadConfig(home)
	if err != nil || config.Profile != node.ProfileB0 {
		t.Fatalf("B0 profile: %+v %v", config, err)
	}
	dbDir := filepath.Join(home, "data", "pluto-b0-v2")
	repeatOutput, repeatErr := exec.Command(plutod, args...).CombinedOutput()
	if repeatErr == nil || !bytes.Contains(repeatOutput, []byte("node home already exists")) {
		t.Fatalf("repeated init-b0 changed existing home: err=%v output=%s", repeatErr, repeatOutput)
	}
	preStartBackup := filepath.Join(t.TempDir(), "preserved-h0")
	if err := os.Rename(dbDir, preStartBackup); err != nil {
		t.Fatal(err)
	}
	missingBeforeStart, missingErr := exec.Command(plutod, "start", "--home", home).CombinedOutput()
	if missingErr == nil || !bytes.Contains(missingBeforeStart, []byte("B0 snapshot missing; start requires an existing initialized B0 store")) {
		t.Fatalf("missing H0 before first start: err=%v output=%s", missingErr, missingBeforeStart)
	}
	if err := os.Mkdir(dbDir, 0700); err != nil {
		t.Fatal(err)
	}
	incompleteBeforeStart, incompleteErr := exec.Command(plutod, "start", "--home", home).CombinedOutput()
	if incompleteErr == nil || !bytes.Contains(incompleteBeforeStart, []byte("validate existing B0 snapshot read-only")) {
		t.Fatalf("incomplete H0 before first start: err=%v output=%s", incompleteErr, incompleteBeforeStart)
	}
	if err := os.Remove(dbDir); err != nil {
		entries, _ := os.ReadDir(dbDir)
		t.Fatalf("remove incomplete directory: %v entries=%v", err, entries)
	}
	if err := os.Rename(preStartBackup, dbDir); err != nil {
		t.Fatal(err)
	}
	t.Logf("D-026 pre-start refusals: repeated init-b0=%q missing H0=%q incomplete directory=%q", repeatOutput, missingBeforeStart, incompleteBeforeStart)
	a, err := b0app.Open(dbDir, "pluto-b0-v1", binary)
	if err != nil {
		t.Fatal(err)
	}
	capture := &capturedB0App{App: a, observed: make(chan []byte, 1)}
	srv, err := server.NewServer(config.ABCIAddress, "socket", capture)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	logFile, err := os.Create(filepath.Join(t.TempDir(), "comet.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(comet, "node", "--home", home)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stop := func() {
		if cmd == nil {
			return
		}
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("stop CometBFT: %v", err)
			}
		case <-time.After(15 * time.Second):
			_ = cmd.Process.Kill()
			<-done
			t.Error("CometBFT stop timeout")
		}
		cmd = nil
	}
	t.Cleanup(stop)
	select {
	case received := <-capture.observed:
		if !bytes.Equal(received, expectedTransport) || bytes.Equal(received, binary) {
			t.Fatalf("actual CometBFT InitChain bytes differ from D-025 JSON transport")
		}
		t.Logf("pinned CometBFT delivered exact JSON/Base64 InitChain bytes: length=%d", len(received))
	case <-time.After(35 * time.Second):
		t.Fatal("real CometBFT did not call InitChain")
	}
	b0WaitHeight(t, rpc, 2)
	stop()
	_ = logFile.Close()
	if err := srv.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	committed, err := b0app.Open(dbDir, "pluto-b0-v1", binary)
	if err != nil {
		t.Fatal(err)
	}
	state, exists, err := committed.Committed()
	if err != nil || !exists || state.Height() < 2 || state.TotalSupply() != 100 || len(state.Accounts()) != 1 || state.Accounts()[0].Balance != 100 {
		t.Fatalf("recovered B0 state: exists=%v height=%d err=%v", exists, state.Height(), err)
	}
	if err := committed.Close(); err != nil {
		t.Fatal(err)
	}
	// At height > 0 the node metadata still detects a changed genesis file,
	// even though the durable state record has no independent genesis digest.
	genesisPath := filepath.Join(home, "config", "genesis.json")
	originalGenesisJSON, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatal(err)
	}
	changedBinary := bytes.Clone(binary)
	changedBinary[len(changedBinary)-1]++
	changedTransport, err := b0genesis.Encode(changedBinary, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	genesis.AppState = changedTransport
	if err := genesis.SaveAs(genesisPath); err != nil {
		t.Fatal(err)
	}
	genesisProbe, genesisProbeErr := exec.Command(plutod, "start", "--home", home).CombinedOutput()
	if err := os.WriteFile(genesisPath, originalGenesisJSON, 0600); err != nil {
		t.Fatal(err)
	}
	if genesisProbeErr == nil || !bytes.Contains(genesisProbe, []byte("genesis digest conflicts")) {
		t.Fatalf("changed genesis was accepted after height zero: err=%v output=%s", genesisProbeErr, genesisProbe)
	}
	// The ordinary CLI start path must recognize the B0 profile on restart.
	logRestart, err := os.Create(filepath.Join(t.TempDir(), "restart.log"))
	if err != nil {
		t.Fatal(err)
	}
	restart := exec.Command(plutod, "start", "--home", home)
	restart.Stdout, restart.Stderr = logRestart, logRestart
	if err := restart.Start(); err != nil {
		t.Fatal(err)
	}
	defer logRestart.Close()
	restartRunning := true
	t.Cleanup(func() {
		if restartRunning {
			_ = restart.Process.Kill()
			_ = restart.Wait()
		}
	})
	b0WaitHeight(t, rpc, state.Height()+1)
	if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := restart.Wait(); err != nil {
		t.Fatal(err)
	}
	restartRunning = false
	// A missing B0 store beside persisted CometBFT data must fail closed.
	backup := filepath.Join(home, "data", "b0-test-backup")
	if err := os.Rename(dbDir, backup); err != nil {
		t.Fatal(err)
	}
	probeOut, probeErr := exec.Command(plutod, "start", "--home", home).CombinedOutput()
	if err := os.Rename(backup, dbDir); err != nil {
		t.Fatal(err)
	}
	if probeErr == nil || !bytes.Contains(probeOut, []byte("B0 snapshot missing; start requires an existing initialized B0 store")) {
		t.Fatalf("missing B0 store accepted: err=%v output=%s", probeErr, probeOut)
	}
	if _, err := os.Stat(filepath.Join(home, "data", "pluto")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Phase 02 database was created: %v", err)
	}
	t.Logf("B0 restart preserved genesis balance and refused implicit reset at height >= %d", state.Height()+1)

	// D-026 rejects complete local data loss under retained configuration
	// before CometBFT starts, regardless of its own validator-state error.
	oldData := filepath.Join(t.TempDir(), "previous-data")
	if err := os.Rename(filepath.Join(home, "data"), oldData); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(home, "data"), 0700); err != nil {
		t.Fatal(err)
	}
	probeContext, cancelProbe := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelProbe()
	allLossOutput, allLossErr := exec.CommandContext(probeContext, plutod, "start", "--home", home).CombinedOutput()
	if probeContext.Err() != nil || allLossErr == nil || !bytes.Contains(allLossOutput, []byte("B0 snapshot missing; start requires an existing initialized B0 store")) {
		t.Fatalf("all-data-loss outcome: context=%v err=%v output=%s", probeContext.Err(), allLossErr, allLossOutput)
	}
	if _, err := os.Stat(dbDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("all-data-loss startup created B0 database: %v", err)
	}
	t.Log("all local data lost under retained config: Pluto refused startup for missing B0 snapshot before starting CometBFT; no B0 snapshot was created")
}
