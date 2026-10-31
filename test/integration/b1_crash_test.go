//go:build integration && fourval

package integration

import (
	"context"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/cometbft/cometbft/abci/server"
	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"

	"example.com/pluto/internal/b1app"
	"example.com/pluto/internal/b1genesis"
	"example.com/pluto/internal/b1store"
	"example.com/pluto/internal/b1test"
	"example.com/pluto/internal/node"
)

// This gate runs only in a disposable test process. A marker proves that the
// target callback boundary was reached before the parent sends SIGKILL.
type hybridCommitGate struct {
	*b1app.App
	mode, marker string
	armed        atomic.Bool
}

func (g *hybridCommitGate) FinalizeBlock(ctx context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	resp, err := g.App.FinalizeBlock(ctx, req)
	if err == nil && len(req.Txs) > 0 {
		g.armed.Store(true)
	}
	return resp, err
}
func (g *hybridCommitGate) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	if g.armed.Load() && g.mode == "pre-write" {
		g.block()
	}
	resp, err := g.App.Commit(ctx, req)
	if err == nil && g.armed.Load() && g.mode == "post-sync" {
		g.block()
	}
	return resp, err
}
func (g *hybridCommitGate) block() {
	if err := os.WriteFile(g.marker, []byte(g.mode), 0600); err != nil {
		panic(err)
	}
	select {} // Parent SIGKILLs only this owned test process.
}

func TestB1HybridCrashBoundary(t *testing.T) {
	if mode := os.Getenv("PLUTO_B1_CRASH_CHILD"); mode != "" {
		home := os.Getenv("PLUTO_B1_CRASH_HOME")
		config, err := node.ReadConfig(home)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := cmttypes.GenesisDocFromFile(filepath.Join(home, "config", "genesis.json"))
		if err != nil {
			t.Fatal(err)
		}
		genesis, _, err := b1genesis.Decode(doc.AppState, config.ChainID)
		if err != nil {
			t.Fatal(err)
		}
		a, err := b1app.Open(filepath.Join(home, "data", "pluto-b1-v1"), config.ChainID, genesis)
		if err != nil {
			t.Fatal(err)
		}
		gate := &hybridCommitGate{App: a, mode: mode, marker: os.Getenv("PLUTO_B1_CRASH_MARKER")}
		srv, err := server.NewServer(config.ABCIAddress, "socket", gate)
		if err != nil {
			t.Fatal(err)
		}
		if err := srv.Start(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(os.Getenv("PLUTO_B1_CRASH_READY"), []byte("ready"), 0600); err != nil {
			t.Fatal(err)
		}
		select {}
	}
	for _, mode := range []string{"pre-write", "post-sync"} {
		t.Run(mode, func(t *testing.T) {
			f, err := b1test.New("pluto-b1-crash")
			if err != nil {
				t.Fatal(err)
			}
			n := newHybridNetwork(t, f)
			for i := 0; i < 3; i++ {
				n.start(i)
			}
			n.waitSet([]int{0, 1, 2}, 2, 90*time.Second)
			root := filepath.Dir(n.root)
			ready, marker := filepath.Join(root, "b1-app.ready"), filepath.Join(root, "b1-app.blocked")
			appLogPath, cometLogPath := filepath.Join(root, "b1-app.log"), filepath.Join(root, "b1-comet.log")
			appLog, err := os.Create(appLogPath)
			if err != nil {
				t.Fatal(err)
			}
			app := exec.Command(os.Args[0], "-test.run=^TestB1HybridCrashBoundary$")
			app.Env = append(os.Environ(), "PLUTO_B1_CRASH_CHILD="+mode, "PLUTO_B1_CRASH_HOME="+n.nodes[3].home, "PLUTO_B1_CRASH_READY="+ready, "PLUTO_B1_CRASH_MARKER="+marker)
			app.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			app.Stdout, app.Stderr = appLog, appLog
			if err := app.Start(); err != nil {
				t.Fatal(err)
			}
			appWaited := false
			t.Cleanup(func() {
				if !appWaited {
					_ = app.Process.Kill()
					_ = app.Wait()
				}
				_ = appLog.Close()
			})
			waitMarker(t, ready)
			cometLog, err := os.Create(cometLogPath)
			if err != nil {
				t.Fatal(err)
			}
			cometPath, _ := filepath.Abs("../../build/cometbft")
			comet := exec.Command(cometPath, "node", "--home", n.nodes[3].home)
			comet.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			comet.Stdout, comet.Stderr = cometLog, cometLog
			if err := comet.Start(); err != nil {
				t.Fatal(err)
			}
			cometWaited := false
			t.Cleanup(func() {
				if !cometWaited {
					_ = comet.Process.Kill()
					_ = comet.Wait()
				}
				_ = cometLog.Close()
			})
			t.Cleanup(func() {
				if dir := os.Getenv("PLUTO_4V_ARTIFACTS"); dir != "" {
					_ = os.MkdirAll(dir, 0700)
					for _, path := range []string{appLogPath, cometLogPath, ready, marker} {
						if data, err := os.ReadFile(path); err == nil {
							_ = os.WriteFile(filepath.Join(dir, "b1-"+mode+"-"+filepath.Base(path)), data, 0600)
						}
					}
				}
			})
			n.waitAll(2, 90*time.Second)
			wire, err := f.Signed(25, 0)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("H23 %s submitted_raw=%x", mode, wire)
			var admission struct {
				Result struct {
					Code uint32 `json:"code"`
				} `json:"result"`
			}
			if err := rpcGetLong(t, n.nodes[0].ports.RPC, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &admission); err != nil || admission.Result.Code != 0 {
				t.Fatalf("admission code=%d err=%v", admission.Result.Code, err)
			}
			waitMarker(t, marker)
			if data, err := os.ReadFile(marker); err != nil || string(data) != mode {
				t.Fatalf("fault marker: %q %v", data, err)
			}
			aliveAtMarker := m7ProcessAlive(comet.Process.Pid)
			if !aliveAtMarker {
				t.Fatal("Comet dead before application-only kill")
			}
			statusBefore, err := m7StatusHeight(t, n.nodes[3].ports.RPC)
			if err != nil {
				t.Fatal(err)
			}
			if err := app.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			if err := app.Wait(); err == nil {
				t.Fatal("application was not SIGKILLed")
			}
			appWaited = true
			time.Sleep(1200 * time.Millisecond)
			statusAfter, statusErr := m7StatusHeight(t, n.nodes[3].ports.RPC)
			if statusErr == nil && statusAfter > statusBefore {
				t.Fatalf("Comet advanced without application %d -> %d", statusBefore, statusAfter)
			}
			aliveAfterOutage := m7ProcessAlive(comet.Process.Pid)
			if aliveAfterOutage {
				_ = comet.Process.Kill()
			}
			_ = comet.Wait()
			cometWaited = true
			_ = appLog.Close()
			_ = cometLog.Close()
			B, C := cometHeights(t, n.nodes[3].home)
			state, err := b1store.ProbeExisting(filepath.Join(n.nodes[3].home, "data", "pluto-b1-v1"), f.ChainID, f.Genesis)
			if err != nil {
				t.Fatal(err)
			}
			A := state.Height()
			hash := state.AppHash()
			sender, _ := state.Account(f.Sender.Binding.Address)
			receiver, _ := state.Account(f.Receiver.Binding.Address)
			if mode == "pre-write" && (A != C || sender.Balance != 100 || sender.Nonce != 0 || receiver.Balance != 0) {
				t.Fatalf("pre-write B/C/A=%d/%d/%d sender=%+v receiver=%+v", B, C, A, sender, receiver)
			}
			if mode == "post-sync" && (A != B || C != A-1 || sender.Balance != 75 || sender.Nonce != 1 || receiver.Balance != 25) {
				t.Fatalf("post-sync B/C/A=%d/%d/%d sender=%+v receiver=%+v", B, C, A, sender, receiver)
			}
			if B != C+1 {
				t.Fatalf("unexpected consensus fault relation B/C=%d/%d", B, C)
			}
			n.start(3)
			leader, _, err := n.height(0)
			if err != nil {
				t.Fatal(err)
			}
			if leader < A+1 {
				leader = A + 1
			}
			n.waitAll(leader, 90*time.Second)
			var recoveredBlock string
			for i := range n.nodes {
				block, appHash, _, err := n.block(i, A+1)
				if err != nil || block == "" || !strings.EqualFold(appHash, hex.EncodeToString(hash[:])) {
					t.Fatalf("validator %d matching-height H=%d block=%s AppHash=%s want=%x err=%v", i, A+1, block, appHash, hash, err)
				}
				if i == 0 {
					recoveredBlock = block
				} else if block != recoveredBlock {
					t.Fatalf("validator %d divergent recovered block H=%d", i, A+1)
				}
				sb, sn, err := hybridAccount(n, i, f.Sender.Binding.Address)
				if err != nil || sb != 75 || sn != 1 {
					t.Fatalf("recovered validator %d sender=%d/%d err=%v", i, sb, sn, err)
				}
				rb, rn, err := hybridAccount(n, i, f.Receiver.Binding.Address)
				if err != nil || rb != 25 || rn != 0 {
					t.Fatalf("recovered validator %d receiver=%d/%d err=%v", i, rb, rn, err)
				}
			}
			t.Logf("H23 %s app_pid=%d SIGKILL comet_pid=%d alive_at_marker=%t alive_after_outage=%t B/C/A=%d/%d/%d AppHash(A)=%x recovered_block(H=A+1)=%s sender=%d/%d receiver=%d/%d recovered_exact_once=true status_before=%d status_after=%d status_err=%v", mode, app.Process.Pid, comet.Process.Pid, aliveAtMarker, aliveAfterOutage, B, C, A, hash, recoveredBlock, sender.Balance, sender.Nonce, receiver.Balance, receiver.Nonce, statusBefore, statusAfter, statusErr)
		})
	}
}
