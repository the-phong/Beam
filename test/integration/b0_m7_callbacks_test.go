//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"example.com/pluto/internal/b0app"
	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/node"
	"github.com/cometbft/cometbft/abci/server"
	abci "github.com/cometbft/cometbft/abci/types"
)

// This gate runs in a separate application process. Its marker is written by
// the callback itself, so a timed-out marker can never be reported as a hit.
type m7CallbackGate struct {
	*b0app.App
	mode, marker, arm string
}

type m7CallbackMarker struct {
	Callback string `json:"callback"`
	Exists   bool   `json:"exists"`
	Height   int64  `json:"height"`
	AppHash  string `json:"app_hash"`
}

func m7WriteMarker(a *b0app.App, mode, path string) error {
	state, exists, err := a.Committed()
	if err != nil {
		return err
	}
	marker := m7CallbackMarker{Callback: mode, Exists: exists}
	if exists {
		marker.Height = state.Height()
		hash := state.AppHash()
		marker.AppHash = hex.EncodeToString(hash[:])
	}
	raw, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0600)
}

func m7ReadMarker(t *testing.T, path, mode string) m7CallbackMarker {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var marker m7CallbackMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		t.Fatal(err)
	}
	if marker.Callback != mode {
		t.Fatalf("wrong callback marker %q for %q", marker.Callback, mode)
	}
	return marker
}

func m7StatusHeight(t *testing.T, port int) (int64, error) {
	t.Helper()
	var status statusResponse
	if err := rpcGet(t, port, "/status", &status); err != nil {
		return 0, err
	}
	return strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64)
}

func (g *m7CallbackGate) block() {
	if err := m7WriteMarker(g.App, g.mode, g.marker); err != nil {
		panic(err)
	}
	select {}
}

func (g *m7CallbackGate) InitChain(ctx context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	if g.mode == "init-pre-write" || g.mode == "init-h0-pre" {
		g.block()
	}
	resp, err := g.App.InitChain(ctx, req)
	if err == nil && g.mode == "init-post-sync" {
		g.block()
	}
	return resp, err
}

func (g *m7CallbackGate) Info(ctx context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	if g.mode == "info-initial" || g.mode == "info-h0" {
		g.block()
	}
	if g.mode == "info" {
		if _, err := os.Stat(g.arm); err == nil {
			g.block()
		}
	}
	return g.App.Info(ctx, req)
}

func (g *m7CallbackGate) CheckTx(ctx context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	if g.mode == "check-new" && req.Type == abci.CheckTxType_New {
		g.block()
	}
	if g.mode == "check-recheck" && req.Type == abci.CheckTxType_Recheck {
		g.block()
	}
	return g.App.CheckTx(ctx, req)
}

func (g *m7CallbackGate) PrepareProposal(ctx context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	if g.mode == "check-recheck" && len(req.Txs) > 0 {
		// Test-only legal omission keeps an admitted tx in Comet's mempool
		// across one empty block, causing Comet to issue a real RECHECK.
		return &abci.ResponsePrepareProposal{}, nil
	}
	return g.App.PrepareProposal(ctx, req)
}

func (g *m7CallbackGate) Query(ctx context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	if g.mode == "query" {
		g.block()
	}
	return g.App.Query(ctx, req)
}

func TestB0M7OtherApplicationCallbacks(t *testing.T) {
	if mode := os.Getenv("PLUTO_M7_CALLBACK_CHILD"); mode != "" {
		home := os.Getenv("PLUTO_M7_CALLBACK_HOME")
		genesis, _ := hex.DecodeString(b0GoldenGenesis)
		config, err := node.ReadConfig(home)
		if err != nil {
			t.Fatal(err)
		}
		a, err := b0app.Open(filepath.Join(home, "data", "pluto-b0-v2"), "pluto-b0-v1", genesis)
		if err != nil {
			t.Fatal(err)
		}
		gate := &m7CallbackGate{App: a, mode: mode, marker: os.Getenv("PLUTO_M7_CALLBACK_MARKER"), arm: os.Getenv("PLUTO_M7_CALLBACK_ARM")}
		srv, err := server.NewServer(config.ABCIAddress, "socket", gate)
		if err != nil {
			t.Fatal(err)
		}
		if err := srv.Start(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(os.Getenv("PLUTO_M7_CALLBACK_READY"), []byte("ready"), 0600); err != nil {
			t.Fatal(err)
		}
		select {}
	}
	for _, mode := range []string{"info-initial", "init-pre-write", "info-h0", "init-h0-pre", "init-post-sync", "check-new", "check-recheck", "info", "query"} {
		t.Run(mode, func(t *testing.T) { runM7Callback(t, mode) })
	}
}

func runM7Callback(t *testing.T, mode string) {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		artifactDir := os.Getenv("PLUTO_RECOVERY_ARTIFACTS")
		if artifactDir == "" {
			return
		}
		_ = os.MkdirAll(artifactDir, 0700)
		for _, name := range []string{"app.log", "comet-first.log", "comet-restart.log", "ready", "blocked", "armed"} {
			if raw, err := os.ReadFile(filepath.Join(root, name)); err == nil {
				_ = os.WriteFile(filepath.Join(artifactDir, "failed-m7-"+mode+"-"+name), raw, 0600)
			}
		}
	})
	home := filepath.Join(root, "home")
	plutod, _ := filepath.Abs("../../build/plutod")
	comet, _ := filepath.Abs("../../build/cometbft")
	genesis, _ := hex.DecodeString(b0GoldenGenesis)
	genesisFile := filepath.Join(root, "genesis.bin")
	if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
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
		t.Fatalf("init: %v %s", err, out)
	}
	// Preserve the historical M7 missing-H0 callback fixture. D-026 init-b0
	// normally creates H0; these two callbacks exercise an incomplete home
	// through a separately launched test application, never production start.
	if mode == "info-initial" || mode == "init-pre-write" {
		if err := os.Rename(filepath.Join(home, "data", "pluto-b0-v2"), filepath.Join(root, "preserved-h0")); err != nil {
			t.Fatal(err)
		}
	}
	ready, marker, arm := filepath.Join(root, "ready"), filepath.Join(root, "blocked"), filepath.Join(root, "armed")
	appLog, err := os.Create(filepath.Join(root, "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	app := exec.Command(os.Args[0], "-test.run=^TestB0M7OtherApplicationCallbacks$")
	app.Env = append(os.Environ(), "PLUTO_M7_CALLBACK_CHILD="+mode, "PLUTO_M7_CALLBACK_HOME="+home,
		"PLUTO_M7_CALLBACK_READY="+ready, "PLUTO_M7_CALLBACK_MARKER="+marker, "PLUTO_M7_CALLBACK_ARM="+arm)
	app.Stdout, app.Stderr = appLog, appLog
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if app.ProcessState == nil {
			_ = app.Process.Kill()
			_ = app.Wait()
		}
		_ = appLog.Close()
	})
	waitMarker(t, ready)
	cometLog, err := os.Create(filepath.Join(root, "comet-first.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(comet, "node", "--home", home)
	cmd.Stdout, cmd.Stderr = cometLog, cometLog
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		_ = cometLog.Close()
	})
	var sender, receiver b0state.Address
	senderBytes, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
	receiverBytes, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
	copy(sender[:], senderBytes)
	copy(receiver[:], receiverBytes)
	var wire []byte
	if mode != "init-post-sync" && mode != "init-pre-write" && mode != "info-initial" && mode != "info-h0" && mode != "init-h0-pre" {
		b0WaitHeight(t, rpc, 2)
		switch mode {
		case "check-new":
			wire = realNodeSigned(t, receiver, 25, 0)
			go func() {
				var result any
				_ = rpcGet(t, rpc, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &result)
			}()
		case "check-recheck":
			wire = realNodeSigned(t, receiver, 25, 0)
			var admission struct {
				Result struct {
					Code uint32 `json:"code"`
				} `json:"result"`
			}
			if err := rpcGet(t, rpc, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &admission); err != nil || admission.Result.Code != 0 {
				t.Fatalf("initial admission for RECHECK: %v code=%d", err, admission.Result.Code)
			}
		case "info":
			if err := os.WriteFile(arm, []byte("armed"), 0600); err != nil {
				t.Fatal(err)
			}
			go func() { var result any; _ = rpcGet(t, rpc, "/abci_info", &result) }()
		case "query":
			go func() {
				var result any
				_ = rpcGet(t, rpc, "/abci_query?path=%22b0/account/v1%22&data=0x"+hex.EncodeToString(sender[:]), &result)
			}()
		}
	}
	waitMarker(t, marker)
	marked := m7ReadMarker(t, marker, mode)
	if !m7ProcessAlive(cmd.Process.Pid) {
		t.Fatal("Comet was not alive when the application callback was reached")
	}
	statusAtMarker, statusAtMarkerErr := m7StatusHeight(t, rpc)
	if mode != "info-initial" && mode != "init-pre-write" && mode != "info-h0" && mode != "init-h0-pre" && mode != "init-post-sync" && statusAtMarkerErr != nil {
		t.Fatalf("Comet status unavailable at callback: %v", statusAtMarkerErr)
	}
	if err := app.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := app.Wait(); err == nil {
		t.Fatal("application process was not SIGKILLed")
	}
	// Check the independent Comet process before any coordinated stop.
	time.Sleep(1200 * time.Millisecond)
	statusAfterOutage, rpcErr := m7StatusHeight(t, rpc)
	if statusAtMarkerErr == nil && rpcErr == nil && statusAfterOutage > statusAtMarker {
		t.Fatalf("Comet advanced while application was unavailable: status %d -> %d", statusAtMarker, statusAfterOutage)
	}
	cometAlive := m7ProcessAlive(cmd.Process.Pid)
	if cometAlive {
		_ = cmd.Process.Kill()
	}
	cometExit := cmd.Wait()
	_ = cometLog.Close()
	beforeB, beforeC := cometHeights(t, home)
	if (marked.Exists && beforeC != marked.Height) || (!marked.Exists && beforeC != 0) {
		t.Fatalf("durable consensus height advanced during application outage: marker A=%d exists=%t; C after outage=%d", marked.Height, marked.Exists, beforeC)
	}
	a, err := b0app.Open(filepath.Join(home, "data", "pluto-b0-v2"), "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	before, exists, err := a.Committed()
	if err != nil {
		t.Fatalf("reopen pre-restart: exists=%t err=%v", exists, err)
	}
	if mode == "info-initial" || mode == "init-pre-write" {
		if marked.Exists || exists || beforeB != 0 || beforeC != 0 {
			t.Fatalf("initial Info interruption B=%d C=%d exists=%t", beforeB, beforeC, exists)
		}
		_ = a.Close()
		probeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		out, startErr := exec.CommandContext(probeCtx, plutod, "start", "--home", home).CombinedOutput()
		const missingB0 = "B0 snapshot missing; start requires an existing initialized B0 store"
		if startErr == nil || probeCtx.Err() != nil || !bytes.Contains(out, []byte(missingB0)) {
			t.Fatalf("expected missing-B0 startup refusal, err=%v ctx=%v output=%s", startErr, probeCtx.Err(), out)
		}
		copyM7CallbackLogs(t, root, mode, "app.log", "comet-first.log", "ready", "blocked")
		t.Logf("M7 %s: callback marker A=absent status=%d/%v; SIGKILL app; Comet alive after 1.2s=%t exit=%v status=%d/%v; before coordinated recovery B=%d C=%d A=absent sender=absent receiver=absent; production restart refused missing B0: %s; no tx requested; logs=%s", mode, statusAtMarker, statusAtMarkerErr, cometAlive, cometExit, statusAfterOutage, rpcErr, beforeB, beforeC, out, root)
		return
	}
	if !exists {
		t.Fatal("initialized state absent")
	}
	beforeHash := before.AppHash()
	if !marked.Exists || before.Height() != marked.Height || hex.EncodeToString(beforeHash[:]) != marked.AppHash {
		t.Fatalf("durable application height/hash advanced during outage: marker=%+v reopened A=%d H=%x", marked, before.Height(), before.AppHash())
	}
	beforeSender, _ := before.Account(sender)
	if beforeSender.Balance != 100 || beforeSender.Nonce != 0 {
		t.Fatalf("unexpected pre-restart state: %+v", beforeSender)
	}
	if before.Height() != beforeC || beforeB < beforeC || beforeB > beforeC+1 {
		t.Fatalf("unexpected pre-restart B/C/A relation: B=%d C=%d A=%d", beforeB, beforeC, before.Height())
	}
	if (mode == "init-post-sync" || mode == "info-h0" || mode == "init-h0-pre") && (beforeB != 0 || beforeC != 0 || before.Height() != 0) {
		t.Fatalf("init response loss B=%d C=%d A=%d", beforeB, beforeC, before.Height())
	}
	config, err := node.ReadConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := server.NewServer(config.ABCIAddress, "socket", a)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()
	defer a.Close()
	restartLog, err := os.Create(filepath.Join(root, "comet-restart.log"))
	if err != nil {
		t.Fatal(err)
	}
	restart := exec.Command(comet, "node", "--home", home)
	restart.Stdout, restart.Stderr = restartLog, restartLog
	if err := restart.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if restart.ProcessState == nil {
			_ = restart.Process.Kill()
			_ = restart.Wait()
		}
		_ = restartLog.Close()
	})
	target := beforeB + 1
	if target < 2 {
		target = 2
	}
	b0WaitHeight(t, rpc, target)
	if mode == "check-new" || mode == "check-recheck" {
		var admission struct {
			Result struct {
				Code uint32 `json:"code"`
			} `json:"result"`
		}
		if err := rpcGet(t, rpc, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &admission); err != nil || admission.Result.Code != 0 {
			if mode == "check-new" {
				t.Fatalf("readmit after lost CheckTx: %v code=%d", err, admission.Result.Code)
			}
			// A tx retained by Comet's mempool can report duplicate on re-submit.
		}
		txHash := sha256.Sum256(wire)
		var indexed struct {
			Result struct {
				TxResult struct {
					Code uint32 `json:"code"`
				} `json:"tx_result"`
			} `json:"result"`
		}
		deadline := time.Now().Add(15 * time.Second)
		var lookupErr error
		for time.Now().Before(deadline) {
			lookupErr = rpcGet(t, rpc, "/tx?hash=0x"+hex.EncodeToString(txHash[:]), &indexed)
			if lookupErr == nil {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if lookupErr != nil || indexed.Result.TxResult.Code != 0 {
			t.Fatalf("committed tx lookup: %v code=%d", lookupErr, indexed.Result.TxResult.Code)
		}
		t.Logf("M7 %s: final indexed transaction result Code=%d", mode, indexed.Result.TxResult.Code)
	}
	if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := restart.Wait(); err != nil {
		t.Fatal(err)
	}
	_ = restartLog.Close()
	after, _, err := a.Committed()
	if err != nil {
		t.Fatal(err)
	}
	afterSender, _ := after.Account(sender)
	afterReceiver, found := after.Account(receiver)
	if mode == "check-new" || mode == "check-recheck" {
		if afterSender.Balance != 75 || afterSender.Nonce != 1 || !found || afterReceiver.Balance != 25 || afterReceiver.Nonce != 0 {
			t.Fatalf("transaction not exactly once: sender=%+v receiver=%+v found=%t", afterSender, afterReceiver, found)
		}
	} else if afterSender.Balance != 100 || afterSender.Nonce != 0 || found {
		t.Fatalf("unexpected transaction effect: sender=%+v receiver=%+v found=%t", afterSender, afterReceiver, found)
	}
	finalB, finalC := cometHeights(t, home)
	if finalB != finalC || finalC != after.Height() {
		t.Fatalf("post-restart B=%d C=%d A=%d", finalB, finalC, after.Height())
	}
	copyM7CallbackLogs(t, root, mode, "app.log", "comet-first.log", "comet-restart.log", "ready", "blocked")
	t.Logf("M7 %s: marker A=%d H=%s status=%d/%v; SIGKILL app; Comet alive after 1.2s=%t exit=%v status=%d/%v; before coordinated recovery B=%d C=%d A=%d H=%x sender=%d/%d receiver=absent; actual Comet restart after B=%d C=%d A=%d H=%x sender=%d/%d receiver=%d/%d found=%t; logs=%s", mode, marked.Height, marked.AppHash, statusAtMarker, statusAtMarkerErr, cometAlive, cometExit, statusAfterOutage, rpcErr, beforeB, beforeC, before.Height(), before.AppHash(), beforeSender.Balance, beforeSender.Nonce, finalB, finalC, after.Height(), after.AppHash(), afterSender.Balance, afterSender.Nonce, afterReceiver.Balance, afterReceiver.Nonce, found, root)
}

func m7ProcessAlive(pid int) bool {
	if err := syscall.Kill(pid, 0); err != nil {
		return false
	}
	procStatus, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	return err == nil && !bytes.Contains(procStatus, []byte("State:\tZ"))
}

func copyM7CallbackLogs(t *testing.T, root, mode string, names ...string) {
	t.Helper()
	artifactDir := os.Getenv("PLUTO_RECOVERY_ARTIFACTS")
	if artifactDir == "" {
		return
	}
	if err := os.MkdirAll(artifactDir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(artifactDir, "m7-"+mode+"-"+name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if raw, err := os.ReadFile(filepath.Join(root, "armed")); err == nil {
		if err := os.WriteFile(filepath.Join(artifactDir, "m7-"+mode+"-armed"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
