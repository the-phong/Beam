//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"example.com/pluto/internal/classical/app"
	"example.com/pluto/internal/classical/state"
	"example.com/pluto/internal/node"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/abci/server"
	abci "github.com/cometbft/cometbft/abci/types"
	sm "github.com/cometbft/cometbft/state"
	blockstore "github.com/cometbft/cometbft/store"
)

// The wrapper is confined to a disposable real-Comet integration test. It
// loses the Commit acknowledgment after the application sync has completed.
type postSyncCommitGate struct {
	*b0app.App
	target  atomic.Bool
	once    sync.Once
	synced  chan struct{}
	release chan struct{}
}

type preExecutionGate struct {
	*b0app.App
	mode      string // "before-finalize" or "before-commit"
	target    atomic.Bool
	cancelled atomic.Bool
	once      sync.Once
	blocked   chan struct{}
	release   chan struct{}
}

func (g *preExecutionGate) PrepareProposal(ctx context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	if len(req.Txs) > 0 && g.mode == "before-proposal" {
		g.once.Do(func() { close(g.blocked) })
		<-g.release
	}
	return g.App.PrepareProposal(ctx, req)
}

func (g *preExecutionGate) ProcessProposal(ctx context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	if len(req.Txs) > 0 && g.mode == "before-process" {
		g.once.Do(func() { close(g.blocked) })
		<-g.release
	}
	return g.App.ProcessProposal(ctx, req)
}

func (g *preExecutionGate) FinalizeBlock(ctx context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	if len(req.Txs) > 0 && g.mode == "before-finalize" {
		g.once.Do(func() { close(g.blocked) })
		<-g.release
		if g.cancelled.Load() {
			return nil, context.Canceled
		}
	}
	resp, err := g.App.FinalizeBlock(ctx, req)
	if err == nil && len(req.Txs) > 0 {
		g.target.Store(true)
	}
	return resp, err
}

func (g *preExecutionGate) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	if g.target.Load() && g.mode == "before-commit" {
		g.once.Do(func() { close(g.blocked) })
		<-g.release
		if g.cancelled.Load() {
			return nil, context.Canceled
		}
	}
	resp, err := g.App.Commit(ctx, req)
	if err == nil && g.target.Load() && g.mode == "post-sync-commit" {
		g.once.Do(func() { close(g.blocked) })
		<-g.release
	}
	return resp, err
}

func (g *postSyncCommitGate) FinalizeBlock(ctx context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	resp, err := g.App.FinalizeBlock(ctx, req)
	if err == nil && len(req.Txs) > 0 {
		g.target.Store(true)
	}
	return resp, err
}

func (g *postSyncCommitGate) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	resp, err := g.App.Commit(ctx, req)
	if err == nil && g.target.Load() {
		g.once.Do(func() { close(g.synced) })
		<-g.release
	}
	return resp, err
}

func cometHeights(t *testing.T, home string) (int64, int64) {
	t.Helper()
	dir := filepath.Join(home, "data")
	blockDB, err := dbm.NewDB("blockstore", dbm.GoLevelDBBackend, dir)
	if err != nil {
		t.Fatal(err)
	}
	b := blockstore.NewBlockStore(blockDB).Height()
	if err := blockDB.Close(); err != nil {
		t.Fatal(err)
	}
	stateDB, err := dbm.NewDB("state", dbm.GoLevelDBBackend, dir)
	if err != nil {
		t.Fatal(err)
	}
	state, err := sm.NewStore(stateDB, sm.StoreOptions{}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateDB.Close(); err != nil {
		t.Fatal(err)
	}
	return b, state.LastBlockHeight
}

func cometFinalizedResponse(t *testing.T, home string, height int64) (*abci.ResponseFinalizeBlock, error) {
	t.Helper()
	db, err := dbm.NewDB("state", dbm.GoLevelDBBackend, filepath.Join(home, "data"))
	if err != nil {
		t.Fatal(err)
	}
	response, loadErr := sm.NewStore(db, sm.StoreOptions{}).LoadLastFinalizeBlockResponse(height)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return response, loadErr
}

func TestB0PinnedMockReplayAfterDurableCommitBeforeAcknowledgment(t *testing.T) {
	plutod, _ := filepath.Abs("../../build/plutod")
	comet, _ := filepath.Abs("../../build/cometbft")
	home := filepath.Join(t.TempDir(), "home")
	genesisFile := filepath.Join(t.TempDir(), "genesis.bin")
	genesis, err := hex.DecodeString(b0GoldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
		t.Fatal(err)
	}
	rpc, p2p, abciPort := freePort(t), freePort(t), freePort(t)
	if rpc == p2p || rpc == abciPort || p2p == abciPort {
		t.Skip("ephemeral port collision")
	}
	args := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile,
		"--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abciPort)}
	if out, err := exec.Command(plutod, args...).CombinedOutput(); err != nil {
		t.Fatalf("init: %v: %s", err, out)
	}
	config, err := node.ReadConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	dbDir := filepath.Join(home, "data", "pluto-b0-v2")
	a, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	gate := &postSyncCommitGate{App: a, synced: make(chan struct{}), release: make(chan struct{})}
	srv, err := server.NewServer(config.ABCIAddress, "socket", gate)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "comet-first.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(comet, "node", "--home", home)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		_ = srv.Stop()
		_ = a.Close()
		_ = logFile.Close()
	})
	b0WaitHeight(t, rpc, 2)
	var receiver b0state.Address
	rawReceiver, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
	copy(receiver[:], rawReceiver)
	wire := realNodeSigned(t, receiver, 25, 0)
	var admission struct {
		Result struct {
			Code uint32 `json:"code"`
		} `json:"result"`
	}
	if err := rpcGet(t, rpc, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &admission); err != nil {
		t.Fatal(err)
	}
	if admission.Result.Code != 0 {
		t.Fatalf("CheckTx admission: %+v", admission)
	}
	select {
	case <-gate.synced:
	case <-time.After(30 * time.Second):
		t.Fatal("transaction Commit did not reach post-sync gate")
	}
	// The child is killed while its ABCI Commit response is withheld.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("Comet child was not killed")
	}
	close(gate.release)
	if err := srv.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	_ = logFile.Close()
	blockHeight, stateHeight := cometHeights(t, home)
	recovered, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	state, exists, err := recovered.Committed()
	if err != nil || !exists {
		t.Fatalf("reopen app: %v", err)
	}
	appHeight := state.Height()
	var sender b0state.Address
	rawSender, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
	copy(sender[:], rawSender)
	account, _ := state.Account(sender)
	if blockHeight != appHeight || stateHeight != appHeight-1 || account.Balance != 75 || account.Nonce != 1 {
		t.Fatalf("pre-replay B=%d C=%d A=%d sender=%+v", blockHeight, stateHeight, appHeight, account)
	}
	storedResponse, err := cometFinalizedResponse(t, home, blockHeight)
	committedHash := state.AppHash()
	if err != nil || !bytes.Equal(storedResponse.AppHash, committedHash[:]) || len(storedResponse.TxResults) != 1 || storedResponse.TxResults[0].Code != 0 {
		t.Fatalf("M5 persisted Finalize response: %+v %v", storedResponse, err)
	}
	secondServer, err := server.NewServer(config.ABCIAddress, "socket", recovered)
	if err != nil {
		t.Fatal(err)
	}
	if err := secondServer.Start(); err != nil {
		t.Fatal(err)
	}
	defer secondServer.Stop()
	defer recovered.Close()
	secondLogPath := filepath.Join(t.TempDir(), "comet-restart.log")
	secondLog, err := os.Create(secondLogPath)
	if err != nil {
		t.Fatal(err)
	}
	defer secondLog.Close()
	restart := exec.Command(comet, "node", "--home", home)
	restart.Stdout, restart.Stderr = secondLog, secondLog
	if err := restart.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if restart.ProcessState == nil {
			_ = restart.Process.Kill()
			_ = restart.Wait()
		}
	}()
	b0WaitHeight(t, rpc, appHeight+1)
	if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := restart.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := secondServer.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
	secondLogBytes, err := os.ReadFile(secondLogPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(secondLogBytes, []byte("Replay last block using mock app")) {
		t.Fatalf("pinned mock replay path not observed: %s", secondLogBytes)
	}
	after, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	defer after.Close()
	afterState, _, err := after.Committed()
	if err != nil {
		t.Fatal(err)
	}
	account, _ = afterState.Account(sender)
	if account.Balance != 75 || account.Nonce != 1 {
		t.Fatalf("duplicate transfer after mock replay: %+v", account)
	}
	finalB, finalC := cometHeights(t, home)
	if finalB != finalC || finalC != afterState.Height() {
		t.Fatalf("replay heights B=%d C=%d A=%d", finalB, finalC, afterState.Height())
	}
	t.Logf("M5/M6: real Comet SIGKILL while Commit response withheld; before B=%d C=%d A=%d; mock replay observed; after B=%d C=%d A=%d; sender balance=%d nonce=%d; app hash=%x; log=%s", blockHeight, stateHeight, appHeight, finalB, finalC, afterState.Height(), account.Balance, account.Nonce, afterState.AppHash(), strings.TrimSpace(secondLogPath))
}

func TestB0PinnedRealReplayBeforeApplicationCommit(t *testing.T) {
	for _, mode := range []string{"before-finalize", "before-commit"} {
		t.Run(mode, func(t *testing.T) {
			plutod, _ := filepath.Abs("../../build/plutod")
			comet, _ := filepath.Abs("../../build/cometbft")
			home := filepath.Join(t.TempDir(), "home")
			genesisFile := filepath.Join(t.TempDir(), "genesis.bin")
			genesis, err := hex.DecodeString(b0GoldenGenesis)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
				t.Fatal(err)
			}
			rpc, p2p, abciPort := freePort(t), freePort(t), freePort(t)
			if rpc == p2p || rpc == abciPort || p2p == abciPort {
				t.Skip("ephemeral port collision")
			}
			args := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile,
				"--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abciPort)}
			if out, err := exec.Command(plutod, args...).CombinedOutput(); err != nil {
				t.Fatalf("init: %v: %s", err, out)
			}
			config, err := node.ReadConfig(home)
			if err != nil {
				t.Fatal(err)
			}
			dbDir := filepath.Join(home, "data", "pluto-b0-v2")
			a, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
			if err != nil {
				t.Fatal(err)
			}
			gate := &preExecutionGate{App: a, mode: mode, blocked: make(chan struct{}), release: make(chan struct{})}
			srv, err := server.NewServer(config.ABCIAddress, "socket", gate)
			if err != nil {
				t.Fatal(err)
			}
			if err := srv.Start(); err != nil {
				t.Fatal(err)
			}
			firstLog, err := os.Create(filepath.Join(t.TempDir(), "first.log"))
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(comet, "node", "--home", home)
			cmd.Stdout, cmd.Stderr = firstLog, firstLog
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
				_ = srv.Stop()
				_ = a.Close()
				_ = firstLog.Close()
			})
			b0WaitHeight(t, rpc, 2)
			var receiver b0state.Address
			rawReceiver, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
			copy(receiver[:], rawReceiver)
			wire := realNodeSigned(t, receiver, 25, 0)
			var admission struct {
				Result struct {
					Code uint32 `json:"code"`
				} `json:"result"`
			}
			if err := rpcGet(t, rpc, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &admission); err != nil {
				t.Fatal(err)
			}
			if admission.Result.Code != 0 {
				t.Fatalf("admission: %+v", admission)
			}
			select {
			case <-gate.blocked:
			case <-time.After(30 * time.Second):
				t.Fatal("target callback did not block")
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Wait(); err == nil {
				t.Fatal("Comet child was not killed")
			}
			gate.cancelled.Store(true)
			close(gate.release)
			if err := srv.Stop(); err != nil {
				t.Fatal(err)
			}
			if err := a.Close(); err != nil {
				t.Fatal(err)
			}
			_ = firstLog.Close()
			blockHeight, stateHeight := cometHeights(t, home)
			recovered, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
			if err != nil {
				t.Fatal(err)
			}
			before, _, err := recovered.Committed()
			if err != nil {
				t.Fatal(err)
			}
			if blockHeight != stateHeight+1 || before.Height() != stateHeight {
				t.Fatalf("pre-replay B=%d C=%d A=%d", blockHeight, stateHeight, before.Height())
			}
			if mode == "before-commit" {
				storedResponse, err := cometFinalizedResponse(t, home, blockHeight)
				if err != nil || len(storedResponse.TxResults) != 1 || storedResponse.TxResults[0].Code != 0 {
					t.Fatalf("M3 persisted Finalize response: %+v %v", storedResponse, err)
				}
				var sender, receiver b0state.Address
				ab, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
				bb, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
				copy(sender[:], ab)
				copy(receiver[:], bb)
				expected, err := b0state.NewState("pluto-b0-v1", blockHeight, 100, []b0state.Account{{Address: sender, Balance: 75, Nonce: 1}, {Address: receiver, Balance: 25}})
				if err != nil {
					t.Fatal(err)
				}
				hash := expected.AppHash()
				if !bytes.Equal(storedResponse.AppHash, hash[:]) {
					t.Fatalf("M3 stored AppHash %x expected %x", storedResponse.AppHash, hash)
				}
			}
			secondServer, err := server.NewServer(config.ABCIAddress, "socket", recovered)
			if err != nil {
				t.Fatal(err)
			}
			if err := secondServer.Start(); err != nil {
				t.Fatal(err)
			}
			secondLogPath := filepath.Join(t.TempDir(), "restart.log")
			secondLog, err := os.Create(secondLogPath)
			if err != nil {
				t.Fatal(err)
			}
			restart := exec.Command(comet, "node", "--home", home)
			restart.Stdout, restart.Stderr = secondLog, secondLog
			if err := restart.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if restart.ProcessState == nil {
					_ = restart.Process.Kill()
					_ = restart.Wait()
				}
				_ = secondServer.Stop()
				_ = recovered.Close()
				_ = secondLog.Close()
			})
			b0WaitHeight(t, rpc, blockHeight+1)
			if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			if err := restart.Wait(); err != nil {
				t.Fatal(err)
			}
			if err := secondServer.Stop(); err != nil {
				t.Fatal(err)
			}
			if err := recovered.Close(); err != nil {
				t.Fatal(err)
			}
			_ = secondLog.Close()
			logBytes, err := os.ReadFile(secondLogPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(logBytes, []byte("Replay last block using real app")) {
				t.Fatalf("real replay path absent: %s", logBytes)
			}
			after, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
			if err != nil {
				t.Fatal(err)
			}
			defer after.Close()
			state, _, err := after.Committed()
			if err != nil {
				t.Fatal(err)
			}
			var sender b0state.Address
			rawSender, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
			copy(sender[:], rawSender)
			account, _ := state.Account(sender)
			if account.Balance != 75 || account.Nonce != 1 {
				t.Fatalf("replayed transfer state: %+v", account)
			}
			finalB, finalC := cometHeights(t, home)
			if finalB != finalC || finalC != state.Height() {
				t.Fatalf("heights after real replay B=%d C=%d A=%d", finalB, finalC, state.Height())
			}
			t.Logf("%s: Comet SIGKILL before application write; before B=%d C=%d A=%d; real replay observed; after B=%d C=%d A=%d; balance=%d nonce=%d hash=%x", mode, blockHeight, stateHeight, before.Height(), finalB, finalC, state.Height(), account.Balance, account.Nonce, state.AppHash())
		})
	}
}

func TestB0PinnedCrashBeforeBlockPersistence(t *testing.T) {
	plutod, _ := filepath.Abs("../../build/plutod")
	comet, _ := filepath.Abs("../../build/cometbft")
	home := filepath.Join(t.TempDir(), "home")
	genesisFile := filepath.Join(t.TempDir(), "genesis.bin")
	genesis, err := hex.DecodeString(b0GoldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
		t.Fatal(err)
	}
	rpc, p2p, abciPort := freePort(t), freePort(t), freePort(t)
	if rpc == p2p || rpc == abciPort || p2p == abciPort {
		t.Skip("ephemeral port collision")
	}
	args := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile,
		"--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abciPort)}
	if out, err := exec.Command(plutod, args...).CombinedOutput(); err != nil {
		t.Fatalf("init: %v: %s", err, out)
	}
	config, err := node.ReadConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	dbDir := filepath.Join(home, "data", "pluto-b0-v2")
	a, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	gate := &preExecutionGate{App: a, mode: "before-proposal", blocked: make(chan struct{}), release: make(chan struct{})}
	srv, err := server.NewServer(config.ABCIAddress, "socket", gate)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	logFile, err := os.Create(filepath.Join(t.TempDir(), "first.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(comet, "node", "--home", home)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		_ = srv.Stop()
		_ = a.Close()
		_ = logFile.Close()
	})
	b0WaitHeight(t, rpc, 2)
	var receiver b0state.Address
	rawReceiver, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
	copy(receiver[:], rawReceiver)
	wire := realNodeSigned(t, receiver, 25, 0)
	var admission struct {
		Result struct {
			Code uint32 `json:"code"`
		} `json:"result"`
	}
	if err := rpcGet(t, rpc, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &admission); err != nil {
		t.Fatal(err)
	}
	if admission.Result.Code != 0 {
		t.Fatalf("admission: %+v", admission)
	}
	select {
	case <-gate.blocked:
	case <-time.After(30 * time.Second):
		t.Fatal("proposal callback did not block")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("Comet child was not killed")
	}
	gate.cancelled.Store(true)
	close(gate.release)
	if err := srv.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	_ = logFile.Close()
	blockHeight, stateHeight := cometHeights(t, home)
	recovered, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err := recovered.Committed()
	if err != nil {
		t.Fatal(err)
	}
	if blockHeight != stateHeight || stateHeight != state.Height() {
		t.Fatalf("unexpected block persistence B=%d C=%d A=%d", blockHeight, stateHeight, state.Height())
	}
	var sender b0state.Address
	rawSender, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
	copy(sender[:], rawSender)
	account, _ := state.Account(sender)
	if account.Balance != 100 || account.Nonce != 0 {
		t.Fatalf("pre-restart state: %+v", account)
	}
	secondServer, err := server.NewServer(config.ABCIAddress, "socket", recovered)
	if err != nil {
		t.Fatal(err)
	}
	if err := secondServer.Start(); err != nil {
		t.Fatal(err)
	}
	secondLog, err := os.Create(filepath.Join(t.TempDir(), "restart.log"))
	if err != nil {
		t.Fatal(err)
	}
	restart := exec.Command(comet, "node", "--home", home)
	restart.Stdout, restart.Stderr = secondLog, secondLog
	if err := restart.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if restart.ProcessState == nil {
			_ = restart.Process.Kill()
			_ = restart.Wait()
		}
		_ = secondServer.Stop()
		_ = recovered.Close()
		_ = secondLog.Close()
	})
	b0WaitHeight(t, rpc, blockHeight+1)
	// The pre-persistence proposal was not available for replay. Rebroadcast
	// it explicitly and confirm one transfer is committed.
	var committed broadcastResult
	if err := rpcGetLong(t, rpc, "/broadcast_tx_commit?tx=0x"+hex.EncodeToString(wire), &committed); err != nil {
		t.Fatal(err)
	}
	if committed.Result.CheckTx.Code != 0 || committed.Result.TxResult.Code != 0 {
		t.Fatalf("rebroadcast result: %+v", committed)
	}
	state, _, err = recovered.Committed()
	if err != nil {
		t.Fatal(err)
	}
	account, _ = state.Account(sender)
	if account.Balance != 75 || account.Nonce != 1 {
		t.Fatalf("rebroadcast state: %+v", account)
	}
	if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := restart.Wait(); err != nil {
		t.Fatal(err)
	}
	t.Logf("M1: killed Comet in PrepareProposal before block persistence; B=C=A=%d; after explicit rebroadcast A=%d balance=%d nonce=%d", blockHeight, state.Height(), account.Balance, account.Nonce)
}

func TestB0ApplicationProcessKilledWhileCometChildAlive(t *testing.T) {
	plutod, _ := filepath.Abs("../../build/plutod")
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
		for _, name := range []string{"first.log", "restart.log", "post-commit.ack"} {
			if raw, err := os.ReadFile(filepath.Join(root, name)); err == nil {
				_ = os.WriteFile(filepath.Join(artifactDir, "failed-m7-post-commit-"+name), raw, 0600)
			}
		}
	})
	home := filepath.Join(root, "home")
	genesisFile := filepath.Join(root, "genesis.bin")
	genesis, err := hex.DecodeString(b0GoldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
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
		t.Fatalf("init: %v: %s", err, out)
	}
	logFile, err := os.Create(filepath.Join(root, "first.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(plutod, "start", "--home", home)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	group := cmd.Process.Pid
	t.Cleanup(func() {
		_ = syscall.Kill(-group, syscall.SIGKILL)
		if cmd.ProcessState == nil {
			_ = cmd.Wait()
		}
		_ = logFile.Close()
	})
	b0WaitHeight(t, rpc, 2)
	var receiver b0state.Address
	rawReceiver, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
	copy(receiver[:], rawReceiver)
	wire := realNodeSigned(t, receiver, 25, 0)
	var committed broadcastResult
	if err := rpcGetLong(t, rpc, "/broadcast_tx_commit?tx=0x"+hex.EncodeToString(wire), &committed); err != nil {
		t.Fatal(err)
	}
	if committed.Result.CheckTx.Code != 0 || committed.Result.TxResult.Code != 0 {
		t.Fatalf("pre-crash transfer: %+v", committed)
	}
	if err := os.WriteFile(filepath.Join(root, "post-commit.ack"), []byte("check_code=0 tx_code=0 height="+committed.Result.Height+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	statusAtKill, err := m7StatusHeight(t, rpc)
	if err != nil {
		t.Fatalf("Comet status before app kill: %v", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("application process was not killed")
	}
	// The Comet child inherited this isolated process group. Confirm the
	// child remains after the application parent was terminated, then stop it.
	childAlive := syscall.Kill(-group, 0) == nil
	if !childAlive {
		t.Fatal("Comet child did not remain alive after application kill")
	}
	time.Sleep(1200 * time.Millisecond)
	statusAfterOutage, statusErr := m7StatusHeight(t, rpc)
	if statusErr == nil && statusAfterOutage > statusAtKill {
		t.Fatalf("Comet advanced while app was unavailable: status %d -> %d", statusAtKill, statusAfterOutage)
	}
	_ = syscall.Kill(-group, syscall.SIGKILL)
	_ = logFile.Close()
	dbDir := filepath.Join(home, "data", "pluto-b0-v2")
	var reopened *b0app.App
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		reopened, err = b0app.Open(dbDir, "pluto-b0-v1", genesis)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("reopen after app/Comet stop: %v", err)
	}
	state, _, err := reopened.Committed()
	if err != nil {
		t.Fatal(err)
	}
	var sender b0state.Address
	rawSender, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
	copy(sender[:], rawSender)
	account, _ := state.Account(sender)
	if account.Balance != 75 || account.Nonce != 1 {
		t.Fatalf("post-crash transfer state: %+v", account)
	}
	beforeReceiver, beforeFound := state.Account(receiver)
	if !beforeFound || beforeReceiver.Balance != 25 || beforeReceiver.Nonce != 0 {
		t.Fatalf("post-crash receiver: %+v found=%t", beforeReceiver, beforeFound)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	beforeB, beforeC := cometHeights(t, home)
	if beforeC != statusAtKill || state.Height() != statusAtKill {
		t.Fatalf("durable height advanced during app outage: at kill C/A=%d; before recovery B=%d C=%d A=%d", statusAtKill, beforeB, beforeC, state.Height())
	}
	restartLog, err := os.Create(filepath.Join(root, "restart.log"))
	if err != nil {
		t.Fatal(err)
	}
	restart := exec.Command(plutod, "start", "--home", home)
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
	b0WaitHeight(t, rpc, state.Height()+1)
	if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := restart.Wait(); err != nil {
		t.Fatal(err)
	}
	_ = restartLog.Close()
	check, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	after, _, err := check.Committed()
	if err != nil {
		t.Fatal(err)
	}
	account, _ = after.Account(sender)
	if account.Balance != 75 || account.Nonce != 1 {
		t.Fatalf("duplicate execution after app kill: %+v", account)
	}
	afterReceiver, afterFound := after.Account(receiver)
	if !afterFound || afterReceiver.Balance != 25 || afterReceiver.Nonce != 0 {
		t.Fatalf("duplicate/missing receiver effect: %+v found=%t", afterReceiver, afterFound)
	}
	finalB, finalC := cometHeights(t, home)
	if finalB != finalC || finalC != after.Height() {
		t.Fatalf("post-restart B=%d C=%d A=%d", finalB, finalC, after.Height())
	}
	copyM7CallbackLogs(t, root, "post-commit-ack", "first.log", "restart.log", "post-commit.ack")
	t.Logf("M7 post-Commit: SIGKILL application parent; Comet child remained alive; status at kill=%d after 1.2s=%d/%v; before coordinated restart B=%d C=%d A=%d H=%x sender=75/1 receiver=25/0; after B=%d C=%d A=%d H=%x sender balance=%d nonce=%d", statusAtKill, statusAfterOutage, statusErr, beforeB, beforeC, state.Height(), state.AppHash(), finalB, finalC, after.Height(), after.AppHash(), account.Balance, account.Nonce)
}
