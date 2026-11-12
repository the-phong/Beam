package b1store_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"

	"example.com/pluto/internal/hybrid/store"
	"example.com/pluto/internal/hybrid/testutil"
	"example.com/pluto/internal/hybrid/tx"
)

func TestB1StoreFailClosedAndRecovery(t *testing.T) {
	f, err := b1test.New("pluto-b1-store")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	dir := filepath.Join(root, "b1")
	if _, err := b1store.ProbeExisting(dir, f.ChainID, f.Genesis); !errors.Is(err, b1store.ErrMissing) {
		t.Fatalf("missing store: %v", err)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := b1store.ProbeExisting(dir, f.ChainID, f.Genesis); !errors.Is(err, b1store.ErrMissing) {
		t.Fatalf("incomplete directory: %v", err)
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	s, err := b1store.Create(dir, f.Genesis, f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b1store.Create(dir, f.Genesis, f.ChainID); !errors.Is(err, b1store.ErrExisting) {
		t.Fatalf("reinitialization: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := b1store.ProbeExisting(dir, "foreign-chain", f.Genesis); err == nil {
		t.Fatal("wrong chain accepted")
	}
	if _, err := b1store.ProbeExisting(dir, f.ChainID, f.Genesis); err != nil {
		t.Fatal(err)
	}
	other, err := b1test.New(f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b1store.ProbeExisting(dir, f.ChainID, other.Genesis); err == nil {
		t.Fatal("conflicting B1 genesis accepted")
	}
	for _, boundary := range []struct {
		name      string
		options   b1store.Options
		uncertain bool
		durable   bool
	}{
		{"pre-write", b1store.Options{BeforeSync: func() error { return errors.New("pre-write injected") }}, false, false},
		{"post-sync", b1store.Options{AfterSync: func() error { return errors.New("post-sync injected") }}, true, true},
	} {
		t.Run(boundary.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "b1")
			created, err := b1store.Create(path, f.Genesis, f.ChainID)
			if err != nil {
				t.Fatal(err)
			}
			if err := created.Close(); err != nil {
				t.Fatal(err)
			}
			store, err := b1store.OpenExistingWithOptions(path, f.ChainID, f.Genesis, boundary.options)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.Signed(25, 0)
			if err != nil {
				t.Fatal(err)
			}
			codes, _, err := store.PrepareTrusted([][]byte{raw}, false)
			if err != nil || len(codes) != 1 || codes[0] != b1tx.CodeOK {
				t.Fatalf("prepare: %v %v", codes, err)
			}
			if err := store.Commit(); err == nil {
				t.Fatal("fault boundary not reached")
			}
			_, readErr := store.Committed()
			if boundary.uncertain && !errors.Is(readErr, b1store.ErrUncertain) {
				t.Fatalf("uncertain result: %v", readErr)
			}
			if !boundary.uncertain && readErr != nil {
				t.Fatalf("pre-write read: %v", readErr)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := b1store.ProbeExisting(path, f.ChainID, f.Genesis)
			if err != nil {
				t.Fatal(err)
			}
			want := int64(0)
			if boundary.durable {
				want = 1
			}
			if reopened.Height() != want {
				t.Fatalf("A=%d want=%d", reopened.Height(), want)
			}
			sender, _ := reopened.Account(f.Sender.Binding.Address)
			receiver, _ := reopened.Account(f.Receiver.Binding.Address)
			if boundary.durable {
				if sender.Balance != 75 || sender.Nonce != 1 || receiver.Balance != 25 {
					t.Fatalf("post-sync effects: %+v %+v", sender, receiver)
				}
			} else if sender.Balance != 100 || sender.Nonce != 0 || receiver.Balance != 0 {
				t.Fatalf("pre-write effects: %+v %+v", sender, receiver)
			}
			t.Logf("H23 %s A=%d AppHash=%x sender=%d/%d receiver=%d/%d", boundary.name, reopened.Height(), reopened.AppHash(), sender.Balance, sender.Nonce, receiver.Balance, receiver.Nonce)
		})
	}
	// A correct Pebble DB with a malformed B1 value cannot be treated as a
	// fresh H0 or rewritten by the production probe.
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Set([]byte("pluto/committed-state/b1-v1"), []byte("corrupt"), pebble.Sync); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := b1store.ProbeExisting(dir, f.ChainID, f.Genesis); err == nil {
		t.Fatal("corrupt snapshot accepted")
	}
	foreignPath := filepath.Join(root, "foreign-record")
	foreignStore, err := b1store.Create(foreignPath, f.Genesis, f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if err := foreignStore.Close(); err != nil {
		t.Fatal(err)
	}
	foreignDB, err := pebble.Open(foreignPath, &pebble.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := foreignDB.Set([]byte("pluto/committed-state/v2"), []byte("foreign"), pebble.Sync); err != nil {
		t.Fatal(err)
	}
	if err := foreignDB.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := b1store.ProbeExisting(foreignPath, f.ChainID, f.Genesis); !errors.Is(err, b1store.ErrForeign) {
		t.Fatalf("foreign B0 record: %v", err)
	}
}
