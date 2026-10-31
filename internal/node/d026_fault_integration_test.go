//go:build integration

package node

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	b0app "example.com/pluto/internal/classical/app"
	b0state "example.com/pluto/internal/classical/state"
	b0store "example.com/pluto/internal/classical/store"
	"github.com/cockroachdb/pebble/vfs"
)

const d026GenesisHex = "504c55544f47322100000002000b706c75746f2d62302d763100000001b9631c0f8d9d2d9d56bd6151559f3339e14eb36a0000000000000064"

var d026Injected = errors.New("D-026 injected I/O error")

// This wrapper reaches the actual Pebble WAL File.Write and File.SyncTo calls.
// The child is SIGKILLed only after the selected call writes its marker.
type d026WALFS struct {
	vfs.FS
	mode, marker string
	enabled      atomic.Bool
	reached      atomic.Bool
}
type d026WALFile struct {
	vfs.File
	fs   *d026WALFS
	path string
}

func (f *d026WALFS) wrap(path string, file vfs.File, err error) (vfs.File, error) {
	if err != nil {
		return nil, err
	}
	return &d026WALFile{File: file, fs: f, path: path}, nil
}
func (f *d026WALFS) Create(path string) (vfs.File, error) {
	file, err := f.FS.Create(path)
	return f.wrap(path, file, err)
}
func (f *d026WALFS) Open(path string, opts ...vfs.OpenOption) (vfs.File, error) {
	file, err := f.FS.Open(path, opts...)
	return f.wrap(path, file, err)
}
func (f *d026WALFS) OpenReadWrite(path string, opts ...vfs.OpenOption) (vfs.File, error) {
	file, err := f.FS.OpenReadWrite(path, opts...)
	return f.wrap(path, file, err)
}
func (f *d026WALFS) ReuseForWrite(oldPath, newPath string) (vfs.File, error) {
	file, err := f.FS.ReuseForWrite(oldPath, newPath)
	return f.wrap(newPath, file, err)
}
func (f *d026WALFile) inject(call string) error {
	if !f.fs.enabled.Load() || !strings.HasSuffix(f.path, ".log") || !strings.HasPrefix(f.fs.mode, "wal-"+call) || !f.fs.reached.CompareAndSwap(false, true) {
		return nil
	}
	if err := os.WriteFile(f.fs.marker, []byte(call+" "+f.path+"\n"), 0600); err != nil {
		panic(err)
	}
	if strings.HasSuffix(f.fs.mode, "-kill") {
		select {}
	}
	return d026Injected
}
func (f *d026WALFile) Write(p []byte) (int, error) {
	if err := f.inject("write"); err != nil {
		return 0, err
	}
	return f.File.Write(p)
}
func (f *d026WALFile) SyncTo(n int64) (bool, error) {
	if err := f.inject("sync"); err != nil {
		return false, err
	}
	return f.File.SyncTo(n)
}
func (f *d026WALFile) Sync() error {
	if err := f.inject("sync"); err != nil {
		return err
	}
	return f.File.Sync()
}
func (f *d026WALFile) SyncData() error {
	if err := f.inject("sync"); err != nil {
		return err
	}
	return f.File.SyncData()
}

func d026Child(t *testing.T, mode, home, marker string) {
	t.Helper()
	genesis, _ := hex.DecodeString(d026GenesisHex)
	comet, err := filepath.Abs("../../build/cometbft")
	if err != nil {
		t.Fatal(err)
	}
	if mode == "after-success-kill" {
		if err := InitB0(home, comet, "pluto-b0-v1", genesis, 26657, 26656, 26658, os.Stdout); err != nil {
			t.Fatalf("production init-b0 did not return success: %v", err)
		}
		if err := os.WriteFile(marker, []byte("after successful init-b0 before first start\n"), 0600); err != nil {
			t.Fatal(err)
		}
		select {}
	}
	hit := func(stage string) func() error {
		return func() error {
			if err := os.WriteFile(marker, []byte(stage+"\n"), 0600); err != nil {
				return err
			}
			if strings.HasSuffix(mode, "-kill") {
				select {}
			}
			return d026Injected
		}
	}
	hooks := &b0InitHooks{}
	switch mode {
	case "before-home-kill":
		hooks.BeforeHome = hit("before home creation")
	case "after-home-kill":
		hooks.AfterHome = hit("after home creation")
	case "after-comet-kill":
		hooks.AfterCometInit = hit("after CometBFT init")
	case "before-store-kill":
		hooks.BeforeStore = hit("before B0 snapshot")
	case "before-mkdir-kill":
		hooks.StoreOptions.BeforeGenesisMkdir = hit("before B0 directory")
	case "after-mkdir-kill":
		hooks.StoreOptions.AfterGenesisMkdir = hit("after B0 directory")
	case "before-record-kill":
		hooks.StoreOptions.BeforeGenesisSync = hit("before record write")
	case "after-sync-kill":
		hooks.AfterStore = hit("after synced H0")
	case "before-metadata-kill":
		hooks.BeforeMetadata = hit("before metadata")
	case "after-metadata-kill":
		hooks.AfterMetadata = hit("after metadata")
	case "before-return-kill":
		hooks.BeforeReturn = hit("after all sync before success response")
	case "wal-write-error", "wal-sync-error", "wal-write-kill", "wal-sync-kill":
		fs := &d026WALFS{FS: vfs.Default, mode: mode, marker: marker}
		hooks.StoreOptions = b0store.Options{FS: fs, BeforeGenesisSync: func() error { fs.enabled.Store(true); return nil }}
	default:
		t.Fatalf("unknown D-026 child mode %q", mode)
	}
	err = initB0WithHooks(home, comet, "pluto-b0-v1", genesis, 26657, 26656, 26658, hooks, os.Stdout)
	if err == nil {
		t.Fatalf("fault %s returned success", mode)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("fault %s missed marker: %v; init error: %v", mode, statErr, err)
	}
	if strings.HasSuffix(mode, "-error") && !errors.Is(err, d026Injected) {
		t.Fatalf("wrong injected error: %v", err)
	}
	t.Logf("reached=%s outcome=%v", mode, err)
}

func TestD026InitFaultBoundaries(t *testing.T) {
	if mode := os.Getenv("PLUTO_D026_CHILD"); mode != "" {
		d026Child(t, mode, os.Getenv("PLUTO_D026_HOME"), os.Getenv("PLUTO_D026_MARKER"))
		return
	}
	genesis, _ := hex.DecodeString(d026GenesisHex)
	for _, mode := range []string{"before-home-kill", "after-home-kill", "after-comet-kill", "before-store-kill", "before-mkdir-kill", "after-mkdir-kill", "before-record-kill", "wal-write-error", "wal-sync-error", "wal-write-kill", "wal-sync-kill", "after-sync-kill", "before-metadata-kill", "after-metadata-kill", "before-return-kill", "after-success-kill"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "home")
			marker := filepath.Join(root, "reached")
			cmd := exec.Command(os.Args[0], "-test.run=^TestD026InitFaultBoundaries$", "-test.v")
			cmd.Env = append(os.Environ(), "PLUTO_D026_CHILD="+mode, "PLUTO_D026_HOME="+home, "PLUTO_D026_MARKER="+marker)
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			var childErr error
			if strings.HasSuffix(mode, "-kill") {
				deadline := time.Now().Add(25 * time.Second)
				for time.Now().Before(deadline) {
					if _, err := os.Stat(marker); err == nil {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				if _, err := os.Stat(marker); err != nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
					t.Fatalf("target not reached: %v output=%s", err, output.String())
				}
				if err := cmd.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				childErr = cmd.Wait()
				if childErr == nil {
					t.Fatal("SIGKILL child exited successfully")
				}
			} else {
				childErr = cmd.Wait()
				if childErr == nil || !strings.Contains(output.String(), "pebble: fatal commit error: D-026 injected I/O error") {
					t.Fatalf("expected Pebble fatal WAL I/O error: %v output=%s", childErr, output.String())
				}
			}
			markerText, err := os.ReadFile(marker)
			if err != nil {
				t.Fatal(err)
			}
			if artifactDir := os.Getenv("PLUTO_D026_ARTIFACTS"); artifactDir != "" {
				if err := os.MkdirAll(artifactDir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(artifactDir, mode+".marker"), markerText, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(artifactDir, mode+".child.log"), output.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
			dbDir := filepath.Join(home, "data", "pluto-b0-v2")
			a, openErr := b0app.Open(dbDir, "pluto-b0-v1", genesis)
			var height int64 = -1
			var appHash string = "absent"
			var balance uint64
			var nonce uint64
			if openErr == nil {
				state, exists, readErr := a.Committed()
				_ = a.Close()
				if readErr != nil {
					openErr = readErr
				} else if exists {
					height = state.Height()
					hash := state.AppHash()
					appHash = hex.EncodeToString(hash[:])
					account := state.Accounts()[0]
					balance = account.Balance
					nonce = account.Nonce
				}
			}
			complete := mode == "after-sync-kill" || mode == "before-metadata-kill" || mode == "after-metadata-kill" || mode == "before-return-kill" || mode == "after-success-kill"
			expected, err := b0state.ParseGenesis(genesis, "pluto-b0-v1")
			if err != nil {
				t.Fatal(err)
			}
			expectedHash := expected.AppHash()
			if height >= 0 && (height != 0 || appHash != hex.EncodeToString(expectedHash[:]) || balance != 100 || nonce != 0) {
				t.Fatalf("recovered state conflicts with H0: A=%d H=%s balance=%d nonce=%d", height, appHash, balance, nonce)
			}
			if complete && (openErr != nil || height != 0 || balance != 100 || nonce != 0) {
				t.Fatalf("synced H0 not recovered: A=%d H=%s balance=%d nonce=%d err=%v", height, appHash, balance, nonce, openErr)
			}
			ambiguousWAL := mode == "wal-sync-kill" || mode == "wal-sync-error"
			if !complete && !ambiguousWAL && height >= 0 {
				t.Fatalf("incomplete attempt unexpectedly contains H0: A=%d H=%s", height, appHash)
			}
			if _, err := os.Stat(filepath.Join(home, "data", "state.db")); err == nil {
				t.Fatal("Comet consensus state unexpectedly advanced before first start")
			}
			if _, err := os.Stat(filepath.Join(home, "data", "blockstore.db")); err == nil {
				t.Fatal("Comet blockstore unexpectedly advanced before first start")
			}
			if mode == "before-return-kill" || mode == "after-success-kill" {
				if _, err := ReadConfig(home); err != nil {
					t.Fatalf("complete metadata verification: %v", err)
				}
				if err := verifyB0H0(dbDir, "pluto-b0-v1", genesis); err != nil {
					t.Fatalf("complete H0 verification: %v", err)
				}
			}
			restart := "fails closed"
			if mode != "after-metadata-kill" && mode != "before-return-kill" && mode != "after-success-kill" { // Production start must fail before any Comet process is launched.
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				err := Run(ctx, home, "/does/not/launch/comet", &bytes.Buffer{})
				if err == nil || strings.Contains(err.Error(), "start CometBFT") {
					t.Fatalf("incomplete-home start did not fail closed: %v", err)
				}
				restart = err.Error()
			} else {
				comet, err := filepath.Abs("../../build/cometbft")
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				var nodeLog bytes.Buffer
				err = Run(ctx, home, comet, &nodeLog)
				cancel()
				if err != nil {
					t.Fatalf("H0 restart failed: %v log=%s", err, nodeLog.String())
				}
				restarted, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
				if err != nil {
					t.Fatal(err)
				}
				state, exists, err := restarted.Committed()
				_ = restarted.Close()
				if err != nil || !exists || state.Height() < 1 || state.Accounts()[0].Balance != 100 || state.Accounts()[0].Nonce != 0 {
					t.Fatalf("first start did not preserve genesis: exists=%t err=%v state=%+v", exists, err, state)
				}
				restart = fmt.Sprintf("real Comet first-start succeeded; A=%d H=%x sender=100/0", state.Height(), state.AppHash())
				if artifactDir := os.Getenv("PLUTO_D026_ARTIFACTS"); artifactDir != "" {
					if err := os.WriteFile(filepath.Join(artifactDir, mode+".restart.log"), nodeLog.Bytes(), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			t.Logf("mode=%s reached=%q child=%v B=0 C=0 A=%d AppHash=%s sender_balance=%d sender_nonce=%d restart=%s tx_effect=none output=%q", mode, strings.TrimSpace(string(markerText)), childErr, height, appHash, balance, nonce, restart, output.String())
		})
	}
}
