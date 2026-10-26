// Package b1store persists one complete B1 snapshot with a distinct record
// key and magic. It cannot open a B0/Phase 02 store or initialize on start.
package b1store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cockroachdb/pebble"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/b1state"
	"example.com/pluto/internal/b1tx"
)

const recordMagic = "PLUTOS3!"
const maxRecordSize = 8 + 32 + 4 + 2 + 50 + 8 + 8 + 4 + b1state.MaxGenesisAccounts*36 + 64

var committedKey = []byte("pluto/committed-state/b1-v1")
var b0Key = []byte("pluto/committed-state/v2")
var v1Key = []byte("pluto/committed-state/v1")

var ErrExisting = errors.New("B1 database path already exists")
var ErrMissing = errors.New("B1 committed snapshot missing")
var ErrForeign = errors.New("foreign B0 or Phase 02 state; no B1 migration")
var ErrPending = errors.New("B1 candidate already pending")
var ErrNoPending = errors.New("no B1 candidate pending")
var ErrUncertain = errors.New("B1 commit outcome uncertain; close and reopen")
var ErrClosed = errors.New("B1 store closed")

type Options struct{ BeforeSync, AfterSync func() error }
type Store struct {
	mu                sync.RWMutex
	db                *pebble.DB
	committed         b1state.State
	pending           *b1state.State
	options           Options
	uncertain, closed bool
}

func Create(dir string, genesis []byte, chainID string) (*Store, error) {
	if dir == "" {
		return nil, ErrMissing
	}
	state, err := b1state.ParseGenesis(genesis, chainID)
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
	if err := os.Mkdir(dir, 0700); err != nil {
		return nil, fmt.Errorf("claim B1 database directory: %w", err)
	}
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfExists: true})
	if err != nil {
		return nil, fmt.Errorf("open new B1 database: %w", err)
	}
	record, err := encodeRecord(state)
	if err == nil {
		err = db.Set(committedKey, record, pebble.Sync)
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sync B1 H0 snapshot: %w", err)
	}
	return &Store{db: db, committed: state}, nil
}

func inspectDir(dir string) error {
	if dir == "" {
		return ErrMissing
	}
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return ErrMissing
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrMissing
	}
	current, err := os.Stat(filepath.Join(dir, "CURRENT"))
	if errors.Is(err, os.ErrNotExist) {
		return ErrMissing
	}
	if err != nil {
		return err
	}
	if !current.Mode().IsRegular() {
		return ErrMissing
	}
	return nil
}

// ProbeExisting uses Pebble read-only mode before a node creates its ABCI
// socket. A refused home is never initialized or rewritten by startup.
func ProbeExisting(dir, chainID string, genesis []byte) (b1state.State, error) {
	if err := inspectDir(dir); err != nil {
		return b1state.State{}, err
	}
	expected, err := b1state.ParseGenesis(genesis, chainID)
	if err != nil {
		return b1state.State{}, err
	}
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true, ReadOnly: true})
	if err != nil {
		return b1state.State{}, fmt.Errorf("open B1 snapshot read-only: %w", err)
	}
	state, loadErr := load(db, expected)
	closeErr := db.Close()
	if loadErr != nil {
		return b1state.State{}, loadErr
	}
	if closeErr != nil {
		return b1state.State{}, closeErr
	}
	return state, nil
}

func OpenExisting(dir, chainID string, genesis []byte) (*Store, error) {
	return OpenExistingWithOptions(dir, chainID, genesis, Options{})
}
func OpenExistingWithOptions(dir, chainID string, genesis []byte, options Options) (*Store, error) {
	if err := inspectDir(dir); err != nil {
		return nil, err
	}
	expected, err := b1state.ParseGenesis(genesis, chainID)
	if err != nil {
		return nil, err
	}
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true})
	if err != nil {
		return nil, fmt.Errorf("open existing B1 database: %w", err)
	}
	state, err := load(db, expected)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, committed: state, options: options}, nil
}

func load(db *pebble.DB, expected b1state.State) (b1state.State, error) {
	for _, key := range [][]byte{v1Key, b0Key} {
		_, closer, err := db.Get(key)
		if err == nil {
			_ = closer.Close()
			return b1state.State{}, ErrForeign
		}
		if !errors.Is(err, pebble.ErrNotFound) {
			return b1state.State{}, err
		}
	}
	value, closer, err := db.Get(committedKey)
	if errors.Is(err, pebble.ErrNotFound) {
		return b1state.State{}, ErrMissing
	}
	if err != nil {
		return b1state.State{}, err
	}
	if len(value) < 8+32+4+2+1+8+8+4+36+64 || len(value) > maxRecordSize {
		_ = closer.Close()
		return b1state.State{}, b1state.ErrState
	}
	data := bytes.Clone(value)
	if err := closer.Close(); err != nil {
		return b1state.State{}, err
	}
	return decodeRecord(data, expected)
}

func encodeRecord(state b1state.State) ([]byte, error) {
	if err := state.Validate(); err != nil {
		return nil, err
	}
	logical, err := state.Base().LogicalBytes()
	if err != nil {
		return nil, err
	}
	out := make([]byte, 8+32+len(logical)+64)
	copy(out[:8], recordMagic)
	rh := state.RegistryHash()
	copy(out[8:40], rh[:])
	copy(out[40:40+len(logical)], logical)
	baseHash := state.Base().AppHash()
	copy(out[40+len(logical):40+len(logical)+32], baseHash[:])
	hash := state.AppHash()
	copy(out[len(out)-32:], hash[:])
	return out, nil
}

func decodeRecord(data []byte, expected b1state.State) (b1state.State, error) {
	if len(data) < 8+32+4+2+1+8+8+4+36+64 || len(data) > maxRecordSize || string(data[:8]) != recordMagic {
		return b1state.State{}, b1state.ErrState
	}
	rh := expected.RegistryHash()
	if !bytes.Equal(data[8:40], rh[:]) {
		return b1state.State{}, b1state.ErrState
	}
	var baseHash, hash [32]byte
	copy(baseHash[:], data[len(data)-64:len(data)-32])
	copy(hash[:], data[len(data)-32:])
	base, err := b0state.DecodeLogical(data[40:len(data)-64], baseHash, expected.ChainID())
	if err != nil {
		return b1state.State{}, b1state.ErrState
	}
	state, err := b1state.NewState(base, expected.Bindings())
	if err != nil || state.AppHash() != hash || state.TotalSupply() != expected.TotalSupply() {
		return b1state.State{}, b1state.ErrState
	}
	if state.Height() == 0 && state.AppHash() != expected.AppHash() {
		return b1state.State{}, b1state.ErrState
	}
	return state, nil
}

func (s *Store) Committed() (b1state.State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return b1state.State{}, ErrClosed
	}
	if s.uncertain {
		return b1state.State{}, ErrUncertain
	}
	return s.committed, nil
}
func (s *Store) Pending() (b1state.State, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return b1state.State{}, false, ErrClosed
	}
	if s.uncertain {
		return b1state.State{}, false, ErrUncertain
	}
	if s.pending == nil {
		return b1state.State{}, false, nil
	}
	return *s.pending, true, nil
}
func (s *Store) PrepareTrusted(raws [][]byte, overLimit bool) ([]b1tx.Code, [32]byte, error) {
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
	candidate := s.committed
	var codes []b1tx.Code
	var err error
	if !overLimit {
		candidate, codes, err = b1state.SimulateOrderedTrusted(candidate, raws)
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
	if err := s.db.Set(committedKey, record, pebble.Sync); err != nil {
		s.uncertain = true
		return fmt.Errorf("sync B1 committed snapshot: %w", err)
	}
	if s.options.AfterSync != nil {
		if err := s.options.AfterSync(); err != nil {
			s.uncertain = true
			return fmt.Errorf("post-sync B1 response failure: %w", err)
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
