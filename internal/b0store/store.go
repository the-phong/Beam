// Package b0store persists isolated B0 V2 snapshots in Pebble. It never
// opens or migrates the Phase 02 V1 application store on behalf of the node.
package b0store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/tx"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

const (
	recordMagic   = "PLUTOS2!"
	maxRecordSize = 8 + 4 + 2 + 50 + 8 + 8 + 4 + b0state.MaxAccounts*36 + 32
)

var committedKey = []byte("pluto/committed-state/v2")
var v1Key = []byte("pluto/committed-state/v1")

var ErrExisting = errors.New("B0 database path already exists")
var ErrMissing = errors.New("B0 committed state missing")
var ErrV1 = errors.New("Phase 02 V1 record present; no implicit migration")
var ErrPending = errors.New("B0 candidate already pending")
var ErrNoPending = errors.New("no B0 candidate pending")
var ErrUncertain = errors.New("B0 commit outcome uncertain; close and reopen")
var ErrClosed = errors.New("B0 store closed")

// Options supplies deterministic fault hooks for tests. The production
// constructor passes zero Options. An AfterSync error models a response error
// after durable bytes may already have changed; reopening decides recovery.
type Options struct {
	BeforeSync func() error
	AfterSync  func() error
	// FS and the genesis hooks are used by recovery tests. A nil FS uses the
	// production filesystem. No hook changes the persisted record format.
	FS                 vfs.FS
	BeforeGenesisMkdir func() error
	AfterGenesisMkdir  func() error
	BeforeGenesisSync  func() error
	AfterGenesisSync   func() error
}

// Store owns one Pebble lock and serializes candidate and committed access.
type Store struct {
	mu        sync.RWMutex
	db        *pebble.DB
	syncWrite func([]byte, []byte) error // Internal fault-injection seam; production uses pebble.Sync.
	committed b0state.State
	pending   *b0state.State
	options   Options
	uncertain bool
	closed    bool
}

// Create initializes only a previously absent directory. A failed attempt
// leaves its directory for explicit operator inspection; it never resets it.
func Create(dir string, genesis []byte, expectedChainID string) (*Store, error) {
	return createWithOptions(dir, genesis, expectedChainID, Options{})
}

// CreateWithOptions is the same D-024 creation path with deterministic
// filesystem fault hooks for D-026 initialization tests. Production passes
// zero Options and writes the identical single committed record.
func CreateWithOptions(dir string, genesis []byte, expectedChainID string, options Options) (*Store, error) {
	return createWithOptions(dir, genesis, expectedChainID, options)
}

func createWithOptions(dir string, genesis []byte, expectedChainID string, options Options) (*Store, error) {
	if dir == "" {
		return nil, ErrMissing
	}
	state, err := b0state.ParseGenesis(genesis, expectedChainID)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dir); err == nil {
		return nil, ErrExisting
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		return nil, err
	}
	if options.BeforeGenesisMkdir != nil {
		if err := options.BeforeGenesisMkdir(); err != nil {
			return nil, fmt.Errorf("before B0 directory creation: %w", err)
		}
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		return nil, fmt.Errorf("create B0 database directory: %w", err)
	}
	if options.AfterGenesisMkdir != nil {
		if err := options.AfterGenesisMkdir(); err != nil {
			return nil, fmt.Errorf("after B0 directory creation: %w", err)
		}
	}
	fs := options.FS
	if fs == nil {
		fs = vfs.Default
	}
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfExists: true, FS: fs})
	if err != nil {
		return nil, fmt.Errorf("open new B0 Pebble database: %w", err)
	}
	record, err := encodeRecord(state)
	if err == nil {
		if options.BeforeGenesisSync != nil {
			err = options.BeforeGenesisSync()
		}
		if err == nil {
			err = db.Set(committedKey, record, pebble.Sync)
		}
		if err == nil && options.AfterGenesisSync != nil {
			err = options.AfterGenesisSync()
		}
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("persist B0 genesis: %w", err)
	}
	return &Store{db: db, syncWrite: func(key, value []byte) error { return db.Set(key, value, pebble.Sync) }, committed: state}, nil
}

func OpenExisting(dir, expectedChainID string) (*Store, error) {
	return OpenExistingWithOptions(dir, expectedChainID, Options{})
}

// ProbeExisting validates a committed snapshot without granting Pebble a
// writable handle. Startup uses this before opening the application so a
// refused incomplete or corrupt home is not modified by database recovery.
func ProbeExisting(dir, expectedChainID string) (b0state.State, error) {
	if dir == "" {
		return b0state.State{}, ErrMissing
	}
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return b0state.State{}, ErrMissing
	}
	if err != nil {
		return b0state.State{}, err
	}
	if !info.IsDir() {
		return b0state.State{}, ErrMissing
	}
	// Pebble's read-only Open still creates LOCK. Reject directories that lack
	// Pebble's CURRENT manifest pointer before opening them at all.
	current, err := os.Stat(filepath.Join(dir, "CURRENT"))
	if errors.Is(err, os.ErrNotExist) {
		return b0state.State{}, ErrMissing
	}
	if err != nil {
		return b0state.State{}, err
	}
	if !current.Mode().IsRegular() {
		return b0state.State{}, ErrMissing
	}
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true, ReadOnly: true})
	if err != nil {
		return b0state.State{}, fmt.Errorf("open B0 snapshot read-only: %w", err)
	}
	state, loadErr := load(db, expectedChainID)
	closeErr := db.Close()
	if loadErr != nil {
		return b0state.State{}, loadErr
	}
	if closeErr != nil {
		return b0state.State{}, closeErr
	}
	return state, nil
}

// OpenExistingWithOptions is intended for fault-injection tests. It requires
// an existing Pebble database and a complete, integrity-valid V2 record.
func OpenExistingWithOptions(dir, expectedChainID string, options Options) (*Store, error) {
	if dir == "" {
		return nil, ErrMissing
	}
	if info, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil, ErrMissing
	} else if err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, ErrMissing
	}
	fs := options.FS
	if fs == nil {
		fs = vfs.Default
	}
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true, FS: fs})
	if err != nil {
		return nil, fmt.Errorf("open existing B0 Pebble database: %w", err)
	}
	state, err := load(db, expectedChainID)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, syncWrite: func(key, value []byte) error { return db.Set(key, value, pebble.Sync) }, committed: state, options: options}, nil
}

func load(db *pebble.DB, expectedChainID string) (b0state.State, error) {
	if _, closer, err := db.Get(v1Key); err == nil {
		if closeErr := closer.Close(); closeErr != nil {
			return b0state.State{}, closeErr
		}
		return b0state.State{}, ErrV1
	} else if !errors.Is(err, pebble.ErrNotFound) {
		return b0state.State{}, err
	}
	value, closer, err := db.Get(committedKey)
	if errors.Is(err, pebble.ErrNotFound) {
		return b0state.State{}, ErrMissing
	}
	if err != nil {
		return b0state.State{}, err
	}
	if len(value) < 8+4+2+1+8+8+4+36+32 || len(value) > maxRecordSize {
		_ = closer.Close()
		return b0state.State{}, b0state.ErrState
	}
	copyValue := bytes.Clone(value)
	if err := closer.Close(); err != nil {
		return b0state.State{}, fmt.Errorf("close B0 value: %w", err)
	}
	return decodeRecord(copyValue, expectedChainID)
}

func encodeRecord(s b0state.State) ([]byte, error) {
	logical, err := s.LogicalBytes()
	if err != nil {
		return nil, err
	}
	out := make([]byte, 8+len(logical)+32)
	copy(out[:8], recordMagic)
	copy(out[8:8+len(logical)], logical)
	hash := s.AppHash()
	copy(out[8+len(logical):], hash[:])
	return out, nil
}

func decodeRecord(data []byte, expectedChainID string) (b0state.State, error) {
	if len(data) < 8+4+2+1+8+8+4+36+32 || len(data) > maxRecordSize || string(data[:8]) != recordMagic {
		return b0state.State{}, b0state.ErrState
	}
	var hash [32]byte
	copy(hash[:], data[len(data)-32:])
	return b0state.DecodeLogical(data[8:len(data)-32], hash, expectedChainID)
}

func (s *Store) Committed() (b0state.State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return b0state.State{}, ErrClosed
	}
	if s.uncertain {
		return b0state.State{}, ErrUncertain
	}
	return s.committed, nil // State exposes no mutable account slice.
}

func (s *Store) Pending() (b0state.State, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return b0state.State{}, false, ErrClosed
	}
	if s.uncertain {
		return b0state.State{}, false, ErrUncertain
	}
	if s.pending == nil {
		return b0state.State{}, false, nil
	}
	return *s.pending, true, nil
}

// Prepare simulates ordered transaction bytes and then advances exactly one
// block. It has no ABCI or block-envelope behavior and performs no disk write.
func (s *Store) Prepare(raws [][]byte) ([]tx.Code, [32]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, [32]byte{}, ErrClosed
	}
	if s.uncertain {
		return nil, [32]byte{}, ErrUncertain
	}
	if s.pending != nil {
		return nil, [32]byte{}, ErrPending
	}
	candidate, codes, err := b0state.SimulateOrdered(s.committed, raws)
	if err != nil {
		return nil, [32]byte{}, err
	}
	candidate, err = candidate.AdvanceBlock()
	if err != nil {
		return nil, [32]byte{}, err
	}
	s.pending = &candidate
	return codes, candidate.AppHash(), nil
}

// PrepareTrusted is the B0 ABCI finalization path. The committed value was
// fully validated on creation/reload and cannot be mutated by callers. It
// preserves the Phase 03C Prepare API as the reference evaluator path.
// An over-limit decided block passes no transactions and advances once.
func (s *Store) PrepareTrusted(raws [][]byte, overLimit bool) ([]tx.Code, [32]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, [32]byte{}, ErrClosed
	}
	if s.uncertain {
		return nil, [32]byte{}, ErrUncertain
	}
	if s.pending != nil {
		return nil, [32]byte{}, ErrPending
	}
	var candidate b0state.State
	var codes []tx.Code
	var err error
	if overLimit {
		candidate = s.committed
	} else {
		candidate, codes, err = b0state.SimulateOrderedTrusted(s.committed, raws)
		if err != nil {
			return nil, [32]byte{}, err
		}
	}
	candidate, err = candidate.AdvanceBlock()
	if err != nil {
		return nil, [32]byte{}, err
	}
	s.pending = &candidate
	return codes, candidate.AppHash(), nil
}

// Commit performs one synchronous complete-record Pebble write. A failed
// pre-sync hook leaves pending for retry. Any Set or post-sync error leaves
// in-memory committed unchanged and requires reopening to resolve ambiguity.
func (s *Store) Commit() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.uncertain {
		return ErrUncertain
	}
	if s.pending == nil {
		return ErrNoPending
	}
	record, err := encodeRecord(*s.pending)
	if err != nil {
		return err
	}
	if s.options.BeforeSync != nil {
		if err := s.options.BeforeSync(); err != nil {
			return err
		}
	}
	if err := s.syncWrite(committedKey, record); err != nil {
		s.uncertain = true
		return fmt.Errorf("sync B0 committed snapshot: %w", err)
	}
	if s.options.AfterSync != nil {
		if err := s.options.AfterSync(); err != nil {
			s.uncertain = true
			return fmt.Errorf("post-sync B0 response failure: %w", err)
		}
	}
	s.committed = *s.pending
	s.pending = nil
	return nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	s.closed = true
	return s.db.Close()
}
