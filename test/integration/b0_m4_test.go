//go:build integration

package integration

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	b0app "example.com/pluto/internal/classical/app"
	b0state "example.com/pluto/internal/classical/state"
	b0store "example.com/pluto/internal/classical/store"
	"example.com/pluto/internal/node"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/cockroachdb/pebble/vfs/errorfs"
	"github.com/cometbft/cometbft/abci/server"
	abci "github.com/cometbft/cometbft/abci/types"
)

type m4SyncFS struct {
	vfs.FS
	enabled *atomic.Bool
	mark    string
	kill    bool
}
type m4SyncFile struct {
	vfs.File
	fs   *m4SyncFS
	path string
}

func (fs *m4SyncFS) wrap(path string, f vfs.File, err error) (vfs.File, error) {
	if err != nil {
		return nil, err
	}
	return &m4SyncFile{File: f, fs: fs, path: path}, nil
}
func (fs *m4SyncFS) Create(path string) (vfs.File, error) {
	f, e := fs.FS.Create(path)
	return fs.wrap(path, f, e)
}
func (fs *m4SyncFS) Open(path string, opts ...vfs.OpenOption) (vfs.File, error) {
	f, e := fs.FS.Open(path, opts...)
	return fs.wrap(path, f, e)
}
func (fs *m4SyncFS) OpenReadWrite(path string, opts ...vfs.OpenOption) (vfs.File, error) {
	f, e := fs.FS.OpenReadWrite(path, opts...)
	return fs.wrap(path, f, e)
}
func (fs *m4SyncFS) ReuseForWrite(oldPath, newPath string) (vfs.File, error) {
	f, e := fs.FS.ReuseForWrite(oldPath, newPath)
	return fs.wrap(newPath, f, e)
}
func (f *m4SyncFile) fail(method string) bool {
	if !f.fs.enabled.Load() || !strings.HasSuffix(f.path, ".log") {
		return false
	}
	if err := os.WriteFile(f.fs.mark, []byte(method+" "+f.path), 0600); err != nil {
		panic(err)
	}
	if f.fs.kill {
		select {}
	}
	return true
}
func (f *m4SyncFile) Sync() error {
	if f.fail("Sync") {
		return errorfs.ErrInjected
	}
	return f.File.Sync()
}
func (f *m4SyncFile) SyncData() error {
	if f.fail("SyncData") {
		return errorfs.ErrInjected
	}
	return f.File.SyncData()
}
func (f *m4SyncFile) SyncTo(n int64) (bool, error) {
	if f.fail(fmt.Sprintf("SyncTo(%d)", n)) {
		return false, errorfs.ErrInjected
	}
	return f.File.SyncTo(n)
}

type m4FaultApp struct {
	*b0app.App
	target  atomic.Bool
	enabled *atomic.Bool
}

func (a *m4FaultApp) FinalizeBlock(ctx context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	r, e := a.App.FinalizeBlock(ctx, req)
	if e == nil && len(req.Txs) > 0 {
		a.target.Store(true)
	}
	return r, e
}
func (a *m4FaultApp) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	if a.target.Load() {
		a.enabled.Store(true)
	}
	return a.App.Commit(ctx, req)
}

func TestB0M4RealCrossStoreWALFault(t *testing.T) {
	if mode := os.Getenv("PLUTO_M4_REAL_CHILD"); mode != "" {
		home := os.Getenv("PLUTO_M4_REAL_HOME")
		genesis, _ := hex.DecodeString(b0GoldenGenesis)
		config, err := node.ReadConfig(home)
		if err != nil {
			t.Fatal(err)
		}
		enabled := new(atomic.Bool)
		mark := os.Getenv("PLUTO_M4_REAL_MARK")
		killMode := strings.HasSuffix(mode, "-kill")
		var fs vfs.FS
		if strings.HasPrefix(mode, "wal-write") {
			fs = errorfs.Wrap(vfs.Default, errorfs.InjectorFunc(func(op errorfs.Op, path string) error {
				if enabled.Load() && op == errorfs.OpFileWrite && strings.HasSuffix(path, ".log") {
					if err := os.WriteFile(mark, []byte("Write "+path), 0600); err != nil {
						panic(err)
					}
					if killMode {
						select {}
					}
					return errorfs.ErrInjected
				}
				return nil
			}))
		} else {
			fs = &m4SyncFS{FS: vfs.Default, enabled: enabled, mark: mark, kill: killMode}
		}
		a, err := b0app.OpenWithStoreOptions(filepath.Join(home, "data", "pluto-b0-v2"), "pluto-b0-v1", genesis, b0store.Options{FS: fs})
		if err != nil {
			t.Fatal(err)
		}
		wrapper := &m4FaultApp{App: a, enabled: enabled}
		srv, err := server.NewServer(config.ABCIAddress, "socket", wrapper)
		if err != nil {
			t.Fatal(err)
		}
		if err := srv.Start(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(os.Getenv("PLUTO_M4_REAL_READY"), []byte("ready"), 0600); err != nil {
			t.Fatal(err)
		}
		select {} // Pebble fatal exits this disposable application process.
	}
	for _, mode := range []string{"wal-write", "wal-sync", "wal-write-kill", "wal-sync-kill"} {
		t.Run(mode, func(t *testing.T) {
			plutod, _ := filepath.Abs("../../build/plutod")
			comet, _ := filepath.Abs("../../build/cometbft")
			root := t.TempDir()
			home := filepath.Join(root, "home")
			genesisFile := filepath.Join(root, "genesis.bin")
			genesis, _ := hex.DecodeString(b0GoldenGenesis)
			if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
				t.Fatal(err)
			}
			rpc, p2p, abciPort := freePort(t), freePort(t), freePort(t)
			if rpc == p2p || rpc == abciPort || p2p == abciPort {
				t.Skip("ephemeral port collision")
			}
			args := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile, "--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abciPort)}
			if out, err := exec.Command(plutod, args...).CombinedOutput(); err != nil {
				t.Fatalf("init: %v %s", err, out)
			}
			initialLog, err := os.Create(filepath.Join(root, "comet-initial.log"))
			if err != nil {
				t.Fatal(err)
			}
			initial := exec.Command(plutod, "start", "--home", home)
			initial.Stdout, initial.Stderr = initialLog, initialLog
			if err := initial.Start(); err != nil {
				t.Fatal(err)
			}
			b0WaitHeight(t, rpc, 2)
			if err := initial.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			if err := initial.Wait(); err != nil {
				t.Fatal(err)
			}
			_ = initialLog.Close()
			ready, mark := filepath.Join(root, "app.ready"), filepath.Join(root, "fault.mark")
			appLog, err := os.Create(filepath.Join(root, "app.log"))
			if err != nil {
				t.Fatal(err)
			}
			appChild := exec.Command(os.Args[0], "-test.run=^TestB0M4RealCrossStoreWALFault$")
			appChild.Env = append(os.Environ(), "PLUTO_M4_REAL_CHILD="+mode, "PLUTO_M4_REAL_HOME="+home, "PLUTO_M4_REAL_READY="+ready, "PLUTO_M4_REAL_MARK="+mark)
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
			deadline := time.Now().Add(30 * time.Second)
			for {
				var status statusResponse
				if err := rpcGet(t, rpc, "/status", &status); err == nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("Comet RPC not ready")
				}
				time.Sleep(100 * time.Millisecond)
			}
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
			waitMarker(t, mark)
			if strings.HasSuffix(mode, "-kill") {
				if err := appChild.Process.Kill(); err != nil {
					t.Fatal(err)
				}
			}
			childDone := make(chan error, 1)
			go func() { childDone <- appChild.Wait() }()
			select {
			case <-childDone:
			case <-time.After(30 * time.Second):
				t.Fatal("fault child did not terminate")
			}
			if err := cmd.Process.Kill(); err != nil && err != os.ErrProcessDone {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			_ = firstLog.Close()
			_ = appLog.Close()
			beforeB, beforeC := cometHeights(t, home)
			dbDir := filepath.Join(home, "data", "pluto-b0-v2")
			a, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
			if err != nil {
				t.Fatal(err)
			}
			before, _, err := a.Committed()
			if err != nil {
				t.Fatal(err)
			}
			account, _ := before.Account(sender)
			receiverAccount, found := before.Account(receiver)
			if beforeB != beforeC+1 || (before.Height() != beforeC && before.Height() != beforeC+1) {
				t.Fatalf("fault cross-store heights B=%d C=%d A=%d", beforeB, beforeC, before.Height())
			}
			if before.Height() == beforeC && (account.Balance != 100 || account.Nonce != 0 || found) {
				t.Fatalf("old state effects=%+v %+v", account, receiverAccount)
			}
			if before.Height() == beforeC+1 && (account.Balance != 75 || account.Nonce != 1 || !found || receiverAccount.Balance != 25) {
				t.Fatalf("new state effects=%+v %+v", account, receiverAccount)
			}
			if err := a.Close(); err != nil {
				t.Fatal(err)
			}
			restartLog, err := os.Create(filepath.Join(root, "comet-restart.log"))
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
			b0WaitHeight(t, rpc, beforeB+1)
			if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			if err := restart.Wait(); err != nil {
				t.Fatal(err)
			}
			_ = restartLog.Close()
			finalB, finalC := cometHeights(t, home)
			afterApp, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
			if err != nil {
				t.Fatal(err)
			}
			defer afterApp.Close()
			after, _, err := afterApp.Committed()
			if err != nil {
				t.Fatal(err)
			}
			account, _ = after.Account(sender)
			receiverAccount, found = after.Account(receiver)
			if finalB != finalC || finalC != after.Height() || account.Balance != 75 || account.Nonce != 1 || !found || receiverAccount.Balance != 25 || receiverAccount.Nonce != 0 {
				t.Fatalf("recovery B=%d C=%d A=%d sender=%+v receiver=%+v", finalB, finalC, after.Height(), account, receiverAccount)
			}
			marker, _ := os.ReadFile(mark)
			if artifactDir := os.Getenv("PLUTO_RECOVERY_ARTIFACTS"); artifactDir != "" {
				if err := os.MkdirAll(artifactDir, 0700); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"app.log", "comet-initial.log", "comet-first.log", "comet-restart.log"} {
					raw, err := os.ReadFile(filepath.Join(root, name))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(artifactDir, "m4-"+mode+"-"+name), raw, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			replay := "real"
			if before.Height() == beforeC+1 {
				replay = "mock"
			}
			method := "Pebble process-fatal I/O error"
			if strings.HasSuffix(mode, "-kill") {
				method = "SIGKILL application while WAL call blocked"
			}
			t.Logf("M4 %s at %q, %s; before B=%d C=%d A=%d hash=%x sender=%d/%d; %s replay; after B=%d C=%d A=%d hash=%x sender=75/1 receiver=25/0", mode, marker, method, beforeB, beforeC, before.Height(), before.AppHash(), func() uint64 {
				if before.Height() == beforeC {
					return 100
				}
				return 75
			}(), func() uint64 {
				if before.Height() == beforeC {
					return 0
				}
				return 1
			}(), replay, finalB, finalC, after.Height(), after.AppHash())
		})
	}
}
