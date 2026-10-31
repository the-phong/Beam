//go:build integration

package integration

import (
	"crypto/sha256"
	"encoding/hex"
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
)

func waitMarker(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("fault marker not reached: %s", path)
}

func TestB0M7AsymmetricApplicationCrash(t *testing.T) {
	if mode := os.Getenv("PLUTO_M7_CHILD"); mode != "" {
		home := os.Getenv("PLUTO_M7_HOME")
		genesis, err := hex.DecodeString(b0GoldenGenesis)
		if err != nil {
			t.Fatal(err)
		}
		config, err := node.ReadConfig(home)
		if err != nil {
			t.Fatal(err)
		}
		a, err := b0app.Open(filepath.Join(home, "data", "pluto-b0-v2"), "pluto-b0-v1", genesis)
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
		if err := os.WriteFile(os.Getenv("PLUTO_M7_READY"), []byte("server-ready"), 0600); err != nil {
			t.Fatal(err)
		}
		go func() {
			<-gate.blocked
			if err := m7WriteMarker(a, mode, os.Getenv("PLUTO_M7_BLOCKED")); err != nil {
				panic(err)
			}
		}()
		select {} // The parent SIGKILLs this disposable application process.
	}
	for _, mode := range []string{"before-proposal", "before-process", "before-finalize", "before-commit", "post-sync-commit"} {
		t.Run(mode, func(t *testing.T) {
			plutod, _ := filepath.Abs("../../build/plutod")
			comet, _ := filepath.Abs("../../build/cometbft")
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
				for _, name := range []string{"app.log", "comet-first.log", "comet-restart.log", "app.ready", "app.blocked"} {
					if raw, err := os.ReadFile(filepath.Join(root, name)); err == nil {
						_ = os.WriteFile(filepath.Join(artifactDir, "failed-m7-"+mode+"-"+name), raw, 0600)
					}
				}
			})
			home := filepath.Join(root, "home")
			genesisFile := filepath.Join(root, "genesis.bin")
			genesis, _ := hex.DecodeString(b0GoldenGenesis)
			if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
				t.Fatal(err)
			}
			rpc, p2p, abci := 0, 0, 0
			for attempt := 0; attempt < 10; attempt++ {
				rpc, p2p, abci = freePort(t), freePort(t), freePort(t)
				if rpc != p2p && rpc != abci && p2p != abci {
					break
				}
			}
			if rpc == p2p || rpc == abci || p2p == abci {
				t.Fatal("could not allocate distinct ports")
			}
			args := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile,
				"--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abci)}
			if out, err := exec.Command(plutod, args...).CombinedOutput(); err != nil {
				t.Fatalf("init: %v %s", err, out)
			}
			ready, blocked := filepath.Join(root, "app.ready"), filepath.Join(root, "app.blocked")
			appLog, err := os.Create(filepath.Join(root, "app.log"))
			if err != nil {
				t.Fatal(err)
			}
			appChild := exec.Command(os.Args[0], "-test.run=^TestB0M7AsymmetricApplicationCrash$")
			appChild.Env = append(os.Environ(), "PLUTO_M7_CHILD="+mode, "PLUTO_M7_HOME="+home, "PLUTO_M7_READY="+ready, "PLUTO_M7_BLOCKED="+blocked)
			appChild.Stdout, appChild.Stderr = appLog, appLog
			if err := appChild.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if appChild.ProcessState == nil {
					_ = appChild.Process.Kill()
					_ = appChild.Wait()
				}
				_ = appLog.Close()
			})
			waitMarker(t, ready)
			firstLog, err := os.Create(filepath.Join(root, "comet-first.log"))
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
				_ = firstLog.Close()
			})
			b0WaitHeight(t, rpc, 2)
			var sender, receiver b0state.Address
			aBytes, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
			bBytes, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
			copy(sender[:], aBytes)
			copy(receiver[:], bBytes)
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
				t.Fatalf("admission=%d", admission.Result.Code)
			}
			waitMarker(t, blocked)
			marked := m7ReadMarker(t, blocked, mode)
			if !m7ProcessAlive(cmd.Process.Pid) {
				t.Fatal("Comet was not alive when the application callback was reached")
			}
			statusAtMarker, statusErr := m7StatusHeight(t, rpc)
			if statusErr != nil {
				t.Fatalf("Comet status unavailable at callback: %v", statusErr)
			}
			if err := appChild.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			if err := appChild.Wait(); err == nil {
				t.Fatal("application child was not killed")
			}
			// Comet remains the independent process at the instant of app kill.
			// Observe its RPC again after the lost callback, then coordinate stop.
			time.Sleep(1200 * time.Millisecond)
			statusAfterOutage, rpcErr := m7StatusHeight(t, rpc)
			if rpcErr == nil && statusAfterOutage > statusAtMarker {
				t.Fatalf("Comet advanced while application was unavailable: status %d -> %d", statusAtMarker, statusAfterOutage)
			}
			cometAlive := m7ProcessAlive(cmd.Process.Pid)
			if cometAlive {
				if err := cmd.Process.Kill(); err != nil && err != os.ErrProcessDone {
					t.Fatal(err)
				}
			}
			cometExit := cmd.Wait()
			_ = firstLog.Close()
			beforeB, beforeC := cometHeights(t, home)
			expectedC := marked.Height
			if mode == "post-sync-commit" {
				expectedC--
			}
			if !marked.Exists || beforeC != expectedC {
				t.Fatalf("durable consensus height advanced during application outage: marker=%+v expected C=%d actual C=%d", marked, expectedC, beforeC)
			}
			dbDir := filepath.Join(home, "data", "pluto-b0-v2")
			a, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
			if err != nil {
				t.Fatal(err)
			}
			before, _, err := a.Committed()
			if err != nil {
				t.Fatal(err)
			}
			beforeHash := before.AppHash()
			if before.Height() != marked.Height || hex.EncodeToString(beforeHash[:]) != marked.AppHash {
				t.Fatalf("durable application height/hash advanced during outage: marker=%+v reopened A=%d H=%x", marked, before.Height(), beforeHash)
			}
			account, _ := before.Account(sender)
			decided := mode == "before-finalize" || mode == "before-commit" || mode == "post-sync-commit"
			expectedA, expectedBalance, expectedNonce := beforeC, uint64(100), uint64(0)
			if mode == "post-sync-commit" {
				expectedA, expectedBalance, expectedNonce = beforeB, 75, 1
			}
			if (decided && beforeB != beforeC+1) || (!decided && beforeB != beforeC) || before.Height() != expectedA || account.Balance != expectedBalance || account.Nonce != expectedNonce {
				t.Fatalf("pre-recovery B=%d C=%d A=%d sender=%+v", beforeB, beforeC, before.Height(), account)
			}
			beforeReceiver, beforeFound := before.Account(receiver)
			if mode == "post-sync-commit" {
				if !beforeFound || beforeReceiver.Balance != 25 || beforeReceiver.Nonce != 0 {
					t.Fatalf("post-sync recipient=%+v found=%t", beforeReceiver, beforeFound)
				}
			} else if beforeFound {
				t.Fatalf("recipient exists before transaction commit: %+v", beforeReceiver)
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
			b0WaitHeight(t, rpc, beforeB+1)
			if mode == "before-proposal" {
				var readmission struct {
					Result struct {
						Code uint32 `json:"code"`
					} `json:"result"`
				}
				if err := rpcGet(t, rpc, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &readmission); err != nil || readmission.Result.Code != 0 {
					t.Fatalf("readmission after undecided proposal: %v code=%d", err, readmission.Result.Code)
				}
				b0WaitHeight(t, rpc, beforeB+2)
			}
			if mode == "before-process" {
				// Comet has accepted this proposal and retains the transaction
				// in its WAL/mempool; a second submission can be a duplicate.
				b0WaitHeight(t, rpc, beforeB+2)
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
			if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			if err := restart.Wait(); err != nil {
				t.Fatal(err)
			}
			after, _, err := a.Committed()
			if err != nil {
				t.Fatal(err)
			}
			account, _ = after.Account(sender)
			receiverAccount, found := after.Account(receiver)
			if account.Balance != 75 || account.Nonce != 1 || !found || receiverAccount.Balance != 25 || receiverAccount.Nonce != 0 {
				t.Fatalf("duplicate/missing effect: A=%+v B=%+v", account, receiverAccount)
			}
			finalB, finalC := cometHeights(t, home)
			if finalB != finalC || finalC != after.Height() {
				t.Fatalf("post-recovery B=%d C=%d A=%d", finalB, finalC, after.Height())
			}
			if artifactDir := os.Getenv("PLUTO_RECOVERY_ARTIFACTS"); artifactDir != "" {
				if err := os.MkdirAll(artifactDir, 0700); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"app.log", "comet-first.log", "comet-restart.log", "app.ready", "app.blocked"} {
					raw, err := os.ReadFile(filepath.Join(root, name))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(artifactDir, "m7-"+mode+"-"+name), raw, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			t.Logf("M7 %s: marker A=%d H=%s status=%d; SIGKILL app; Comet alive after 1.2s=%t exit=%v status=%d/%v; before coordinated recovery B=%d C=%d A=%d hash=%x sender=%d/%d receiver=%d/%d found=%t; decided=%t; indexed execution Code=%d; after B=%d C=%d A=%d hash=%x sender=75/1 receiver=25/0; logs=%s", mode, marked.Height, marked.AppHash, statusAtMarker, cometAlive, cometExit, statusAfterOutage, rpcErr, beforeB, beforeC, before.Height(), before.AppHash(), expectedBalance, expectedNonce, beforeReceiver.Balance, beforeReceiver.Nonce, beforeFound, decided, indexed.Result.TxResult.Code, finalB, finalC, after.Height(), after.AppHash(), root)
		})
	}
}
