package b0store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"example.com/pluto/internal/b0state"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/cockroachdb/pebble/vfs/errorfs"
)

// walFault reaches a specific Pebble WAL filesystem call. It rejects that
// call before it executes. This models a returned I/O error, not a power loss.
func walFault(target errorfs.Op) (vfs.FS, *atomic.Bool, *atomic.Bool) {
	enabled, reached := new(atomic.Bool), new(atomic.Bool)
	fs := errorfs.Wrap(vfs.Default, errorfs.InjectorFunc(func(op errorfs.Op, path string) error {
		if enabled.Load() && op == target && strings.HasSuffix(path, ".log") && reached.CompareAndSwap(false, true) {
			return errorfs.ErrInjected
		}
		return nil
	}))
	return fs, enabled, reached
}

// Pebble v1.1.5 uses File.SyncTo for WAL durability. Its bundled errorfs
// forwards SyncTo without injection, so this wrapper targets that method.
type walSyncFS struct {
	vfs.FS
	enabled *atomic.Bool
	mark    string
	kill    bool
}
type walSyncFile struct {
	vfs.File
	fs   *walSyncFS
	path string
}

func (fs *walSyncFS) wrap(path string, f vfs.File, err error) (vfs.File, error) {
	if err != nil {
		return nil, err
	}
	return &walSyncFile{File: f, fs: fs, path: path}, nil
}
func (fs *walSyncFS) Create(path string) (vfs.File, error) {
	f, err := fs.FS.Create(path)
	return fs.wrap(path, f, err)
}
func (fs *walSyncFS) Open(path string, opts ...vfs.OpenOption) (vfs.File, error) {
	f, err := fs.FS.Open(path, opts...)
	return fs.wrap(path, f, err)
}
func (fs *walSyncFS) OpenReadWrite(path string, opts ...vfs.OpenOption) (vfs.File, error) {
	f, err := fs.FS.OpenReadWrite(path, opts...)
	return fs.wrap(path, f, err)
}
func (fs *walSyncFS) ReuseForWrite(oldPath, newPath string) (vfs.File, error) {
	f, err := fs.FS.ReuseForWrite(oldPath, newPath)
	return fs.wrap(newPath, f, err)
}
func (f *walSyncFile) SyncTo(length int64) (bool, error) {
	if f.inject("SyncTo", length) {
		return false, errorfs.ErrInjected
	}
	return f.File.SyncTo(length)
}
func (f *walSyncFile) Sync() error {
	if f.inject("Sync", 0) {
		return errorfs.ErrInjected
	}
	return f.File.Sync()
}
func (f *walSyncFile) SyncData() error {
	if f.inject("SyncData", 0) {
		return errorfs.ErrInjected
	}
	return f.File.SyncData()
}
func (f *walSyncFile) inject(method string, length int64) bool {
	if !f.fs.enabled.Load() || !strings.HasSuffix(f.path, ".log") {
		return false
	}
	if err := os.WriteFile(f.fs.mark, []byte(fmt.Sprintf("%s %s length=%d", method, f.path, length)), 0600); err != nil {
		panic(err)
	}
	if f.fs.kill {
		select {}
	}
	return true
}

func recoveredTransfer(t *testing.T, dir string, oldHash, newHash [32]byte, sender, receiver b0state.Address) string {
	t.Helper()
	s, err := OpenExisting(dir, chain)
	if err != nil {
		return fmt.Sprintf("FAIL_CLOSED:%v", err)
	}
	defer s.Close()
	state, err := s.Committed()
	if err != nil || state.Validate() != nil {
		t.Fatalf("reopened invalid committed state: %v", err)
	}
	a, _ := state.Account(sender)
	b, found := state.Account(receiver)
	switch state.AppHash() {
	case oldHash:
		if state.Height() != 0 || a.Balance != 100 || a.Nonce != 0 || found {
			t.Fatalf("old hash with incorrect effects: A=%+v B=%+v", a, b)
		}
		return fmt.Sprintf("OLD A=0 hash=%x sender=100/0 receiver=absent", oldHash)
	case newHash:
		if state.Height() != 1 || a.Balance != 75 || a.Nonce != 1 || !found || b.Balance != 25 || b.Nonce != 0 {
			t.Fatalf("new hash with incorrect effects: A=%+v B=%+v", a, b)
		}
		return fmt.Sprintf("NEW A=1 hash=%x sender=75/1 receiver=25/0", newHash)
	default:
		t.Fatalf("reopened unexpected AppHash=%x height=%d", state.AppHash(), state.Height())
		return ""
	}
}

func TestM4ReturnedWALWriteAndSyncErrors(t *testing.T) {
	genesis, wire, sender, receiver := fixture(t)
	if mode := os.Getenv("PLUTO_M4_CHILD"); mode != "" {
		op := errorfs.OpFileWrite
		if mode == "wal-sync" {
			op = errorfs.OpFileSync
		}
		fs, enabled, reached := walFault(op)
		// Pebble treats some WAL write failures as fatal and may exit this
		// process before DB.Set returns. The marker records the reached call.
		fs = errorfs.Wrap(fs, errorfs.InjectorFunc(func(current errorfs.Op, path string) error {
			if enabled.Load() && current == op && strings.HasSuffix(path, ".log") && !reached.Load() {
				if err := os.WriteFile(os.Getenv("PLUTO_FAULT_MARK"), []byte(mode+" "+path), 0600); err != nil {
					panic(err)
				}
			}
			return nil
		}))
		var faultFS vfs.FS = fs
		if mode == "wal-sync" {
			faultFS = &walSyncFS{FS: vfs.Default, enabled: enabled, mark: os.Getenv("PLUTO_FAULT_MARK")}
		}
		s, err := OpenExistingWithOptions(os.Getenv("PLUTO_FAULT_DIR"), chain, Options{FS: faultFS})
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := s.Prepare([][]byte{wire}); err != nil {
			t.Fatal(err)
		}
		enabled.Store(true)
		err = s.Commit()
		enabled.Store(false)
		fmt.Printf("child Commit returned: %v\n", err)
		_ = s.Close()
		return
	}
	for _, tc := range []struct {
		name string
		op   errorfs.Op
	}{{"wal-write", errorfs.OpFileWrite}, {"wal-sync", errorfs.OpFileSync}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "b0")
			created, err := Create(dir, genesis, chain)
			if err != nil {
				t.Fatal(err)
			}
			old, err := created.Committed()
			if err != nil {
				t.Fatal(err)
			}
			if err := created.Close(); err != nil {
				t.Fatal(err)
			}
			s, err := OpenExisting(dir, chain)
			if err != nil {
				t.Fatal(err)
			}
			_, newHash, err := s.Prepare([][]byte{wire})
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			mark := filepath.Join(t.TempDir(), "reached")
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestM4ReturnedWALWriteAndSyncErrors$")
			cmd.Env = append(os.Environ(), "PLUTO_M4_CHILD="+tc.name, "PLUTO_FAULT_DIR="+dir, "PLUTO_FAULT_MARK="+mark)
			output, childErr := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatal(ctx.Err())
			}
			marker, err := os.ReadFile(mark)
			if err != nil {
				t.Fatalf("target WAL %s not reached: %v child=%v output=%s", tc.name, err, childErr, output)
			}
			outcome := recoveredTransfer(t, dir, old.AppHash(), newHash, sender, receiver)
			t.Logf("M4 simulated I/O %s reached at %q; child_exit=%v output=%q; reopened %s", tc.name, marker, childErr, output, outcome)
		})
	}
}

func TestM11GenesisCreationFaultBoundaries(t *testing.T) {
	genesis, _, sender, receiver := fixture(t)
	if mode := os.Getenv("PLUTO_M11_CHILD"); mode != "" {
		op := errorfs.OpFileWrite
		if strings.HasPrefix(mode, "wal-sync") {
			op = errorfs.OpFileSync
		}
		killMode := strings.HasSuffix(mode, "-kill")
		fs, enabled, _ := walFault(op)
		fs = errorfs.Wrap(fs, errorfs.InjectorFunc(func(current errorfs.Op, path string) error {
			if enabled.Load() && current == op && strings.HasSuffix(path, ".log") {
				if err := os.WriteFile(os.Getenv("PLUTO_FAULT_MARK"), []byte(mode+" "+path), 0600); err != nil {
					panic(err)
				}
				if killMode {
					select {}
				}
			}
			return nil
		}))
		var faultFS vfs.FS = fs
		if strings.HasPrefix(mode, "wal-sync") {
			faultFS = &walSyncFS{FS: vfs.Default, enabled: enabled, mark: os.Getenv("PLUTO_FAULT_MARK"), kill: killMode}
		}
		_, err := createWithOptions(os.Getenv("PLUTO_FAULT_DIR"), genesis, chain, Options{FS: faultFS, BeforeGenesisSync: func() error { enabled.Store(true); return nil }})
		fmt.Printf("child genesis returned: %v\n", err)
		return
	}
	for _, mode := range []string{"before-mkdir", "after-mkdir", "before-record", "wal-write", "wal-sync", "wal-write-kill", "wal-sync-kill", "after-sync"} {
		t.Run(mode, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "b0")
			var reached atomic.Bool
			opts := Options{}
			switch mode {
			case "before-mkdir":
				opts.BeforeGenesisMkdir = func() error { reached.Store(true); return errorfs.ErrInjected }
			case "after-mkdir":
				opts.AfterGenesisMkdir = func() error { reached.Store(true); return errorfs.ErrInjected }
			case "before-record":
				opts.BeforeGenesisSync = func() error { reached.Store(true); return errorfs.ErrInjected }
			case "after-sync":
				opts.AfterGenesisSync = func() error { reached.Store(true); return errorfs.ErrInjected }
			}
			var createErr error
			if strings.HasPrefix(mode, "wal-write") || strings.HasPrefix(mode, "wal-sync") {
				mark := filepath.Join(t.TempDir(), "reached")
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestM11GenesisCreationFaultBoundaries$")
				cmd.Env = append(os.Environ(), "PLUTO_M11_CHILD="+mode, "PLUTO_FAULT_DIR="+dir, "PLUTO_FAULT_MARK="+mark)
				var output []byte
				var err error
				if strings.HasSuffix(mode, "-kill") {
					var buf bytes.Buffer
					cmd.Stdout, cmd.Stderr = &buf, &buf
					if startErr := cmd.Start(); startErr != nil {
						t.Fatal(startErr)
					}
					deadline := time.Now().Add(15 * time.Second)
					for {
						if _, statErr := os.Stat(mark); statErr == nil {
							break
						}
						if time.Now().After(deadline) {
							_ = cmd.Process.Kill()
							_ = cmd.Wait()
							t.Fatal("genesis WAL kill marker not reached")
						}
						time.Sleep(10 * time.Millisecond)
					}
					if killErr := cmd.Process.Kill(); killErr != nil {
						t.Fatal(killErr)
					}
					err = cmd.Wait()
					output = buf.Bytes()
				} else {
					output, err = cmd.CombinedOutput()
				}
				if ctx.Err() != nil {
					t.Fatal(ctx.Err())
				}
				if _, statErr := os.Stat(mark); statErr != nil {
					t.Fatalf("genesis %s not reached: %v exit=%v output=%s", mode, statErr, err, output)
				}
				createErr = fmt.Errorf("child exit=%v output=%q", err, output)
				reached.Store(true)
			} else {
				_, createErr = createWithOptions(dir, genesis, chain, opts)
			}
			if !reached.Load() || createErr == nil {
				t.Fatalf("genesis %s target not reached: %v", mode, createErr)
			}
			if mode == "before-mkdir" {
				if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("directory unexpectedly created: %v", err)
				}
				if _, err := OpenExisting(dir, chain); !errors.Is(err, ErrMissing) {
					t.Fatalf("absent store opened: %v", err)
				}
				if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("OpenExisting created missing path: %v", err)
				}
				retry, err := Create(dir, genesis, chain)
				if err != nil {
					t.Fatalf("fresh retry failed: %v", err)
				}
				state, err := retry.Committed()
				if err != nil || state.Height() != 0 {
					t.Fatalf("fresh retry state: %v", err)
				}
				t.Logf("M11 before-mkdir reached; no B0 directory; fresh retry A=0 hash=%x sender=100/0", state.AppHash())
				_ = retry.Close()
				return
			}
			if _, err := Create(dir, genesis, chain); !errors.Is(err, ErrExisting) {
				t.Fatalf("incomplete path reset: %v", err)
			}
			opened, err := OpenExisting(dir, chain)
			if mode == "after-sync" && err != nil {
				t.Fatalf("post-sync genesis not recoverable: %v", err)
			}
			if err == nil {
				state, stateErr := opened.Committed()
				if stateErr != nil || state.Height() != 0 || state.Validate() != nil {
					t.Fatalf("post-sync genesis invalid: %v", stateErr)
				}
				a, _ := state.Account(sender)
				_, receiverFound := state.Account(receiver)
				if a.Balance != 100 || a.Nonce != 0 || receiverFound {
					t.Fatal("genesis effects incorrect")
				}
				t.Logf("M11 %s reached; A=0 hash=%x sender=100/0 receiver=absent; identical record recoverable", mode, state.AppHash())
				_ = opened.Close()
				return
			}
			t.Logf("M11 simulated %s reached; create=%v; incomplete store fails closed: %v", mode, createErr, err)
		})
	}
}
