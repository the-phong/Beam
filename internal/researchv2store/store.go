// Package researchv2store persists one isolated four-profile snapshot. It
// refuses foreign stores and never initializes on ordinary startup.
package researchv2store

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cockroachdb/pebble"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/researchv2"
)

const recordMagic = "PLUTOS4!"
const maxRecordSize = 8 + 1 + 32 + 32 + 4 + 2 + 50 + 8 + 8 + 4 + researchv2.MaxGenesisAccounts*36 + 64

var committedKey = []byte("pluto/committed-state/rv2-v1")
var b1Key = []byte("pluto/committed-state/b1-v1")
var b0Key = []byte("pluto/committed-state/v2")
var v1Key = []byte("pluto/committed-state/v1")

var ErrExisting = errors.New("research-v2 database path already exists")
var ErrMissing = errors.New("research-v2 committed snapshot missing")
var ErrForeign = errors.New("foreign B0, B1 or Phase 02 state; no research-v2 migration")
var ErrPending = errors.New("research-v2 candidate already pending")
var ErrNoPending = errors.New("no research-v2 candidate pending")
var ErrUncertain = errors.New("research-v2 commit outcome uncertain; close and reopen")
var ErrClosed = errors.New("research-v2 store closed")

type Options struct{ BeforeSync, AfterSync func() error }
type Store struct {
	mu                sync.RWMutex
	db                *pebble.DB
	committed         researchv2.State
	pending           *researchv2.State
	profile           researchv2.Profile
	genesisHash       [32]byte
	options           Options
	uncertain, closed bool
}

func Create(dir string, genesis []byte, chainID string, profile researchv2.Profile) (*Store, error) {
	if dir == "" || !profile.Valid() {
		return nil, ErrMissing
	}
	state, err := researchv2.ParseGenesis(genesis, chainID)
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
		return nil, fmt.Errorf("claim research-v2 database directory: %w", err)
	}
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfExists: true})
	if err != nil {
		return nil, fmt.Errorf("open new research-v2 database: %w", err)
	}
	genesisHash := sha256.Sum256(genesis)
	record, err := encodeRecord(state, profile, genesisHash)
	if err == nil {
		err = db.Set(committedKey, record, pebble.Sync)
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sync research-v2 H0 snapshot: %w", err)
	}
	return &Store{db: db, committed: state, profile: profile, genesisHash: genesisHash}, nil
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
func ProbeExisting(dir, chainID string, genesis []byte, profile researchv2.Profile) (researchv2.State, error) {
	if !profile.Valid() {
		return researchv2.State{}, researchv2.ErrState
	}
	if err := inspectDir(dir); err != nil {
		return researchv2.State{}, err
	}
	expected, err := researchv2.ParseGenesis(genesis, chainID)
	if err != nil {
		return researchv2.State{}, err
	}
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true, ReadOnly: true})
	if err != nil {
		return researchv2.State{}, fmt.Errorf("open research-v2 snapshot read-only: %w", err)
	}
	state, loadErr := load(db, expected, profile, sha256.Sum256(genesis))
	closeErr := db.Close()
	if loadErr != nil {
		return researchv2.State{}, loadErr
	}
	if closeErr != nil {
		return researchv2.State{}, closeErr
	}
	return state, nil
}

func OpenExisting(dir, chainID string, genesis []byte, profile researchv2.Profile) (*Store, error) {
	return OpenExistingWithOptions(dir, chainID, genesis, profile, Options{})
}
func OpenExistingWithOptions(dir, chainID string, genesis []byte, profile researchv2.Profile, options Options) (*Store, error) {
	if !profile.Valid() {
		return nil, researchv2.ErrState
	}
	if err := inspectDir(dir); err != nil {
		return nil, err
	}
	expected, err := researchv2.ParseGenesis(genesis, chainID)
	if err != nil {
		return nil, err
	}
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true})
	if err != nil {
		return nil, fmt.Errorf("open existing research-v2 database: %w", err)
	}
	genesisHash := sha256.Sum256(genesis)
	state, err := load(db, expected, profile, genesisHash)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, committed: state, profile: profile, genesisHash: genesisHash, options: options}, nil
}

func load(db *pebble.DB, expected researchv2.State, profile researchv2.Profile, genesisHash [32]byte) (researchv2.State, error) {
	for _, key := range [][]byte{v1Key, b0Key, b1Key} {
		_, closer, err := db.Get(key)
		if err == nil {
			_ = closer.Close()
			return researchv2.State{}, ErrForeign
		}
		if !errors.Is(err, pebble.ErrNotFound) {
			return researchv2.State{}, err
		}
	}
	value, closer, err := db.Get(committedKey)
	if errors.Is(err, pebble.ErrNotFound) {
		return researchv2.State{}, ErrMissing
	}
	if err != nil {
		return researchv2.State{}, err
	}
	if len(value) < 8+1+32+32+4+2+1+8+8+4+36+64 || len(value) > maxRecordSize {
		_ = closer.Close()
		return researchv2.State{}, researchv2.ErrState
	}
	data := bytes.Clone(value)
	if err := closer.Close(); err != nil {
		return researchv2.State{}, err
	}
	return decodeRecord(data, expected, profile, genesisHash)
}

func encodeRecord(state researchv2.State, profile researchv2.Profile, genesisHash [32]byte) ([]byte, error) {
	if !profile.Valid() {
		return nil, researchv2.ErrState
	}
	if err := state.Validate(); err != nil {
		return nil, err
	}
	logical, err := state.Base().LogicalBytes()
	if err != nil {
		return nil, err
	}
	out := make([]byte, 8+1+32+32+len(logical)+64)
	copy(out[:8], recordMagic)
	out[8] = byte(profile)
	copy(out[9:41], genesisHash[:])
	rh := state.RegistryHash()
	copy(out[41:73], rh[:])
	copy(out[73:73+len(logical)], logical)
	baseHash := state.Base().AppHash()
	copy(out[73+len(logical):73+len(logical)+32], baseHash[:])
	hash := state.AppHash()
	copy(out[len(out)-32:], hash[:])
	return out, nil
}

func decodeRecord(data []byte, expected researchv2.State, profile researchv2.Profile, genesisHash [32]byte) (researchv2.State, error) {
	if len(data) < 8+1+32+32+4+2+1+8+8+4+36+64 || len(data) > maxRecordSize || string(data[:8]) != recordMagic || data[8] != byte(profile) || !bytes.Equal(data[9:41], genesisHash[:]) {
		return researchv2.State{}, researchv2.ErrState
	}
	rh := expected.RegistryHash()
	if !bytes.Equal(data[41:73], rh[:]) {
		return researchv2.State{}, researchv2.ErrState
	}
	var baseHash, hash [32]byte
	copy(baseHash[:], data[len(data)-64:len(data)-32])
	copy(hash[:], data[len(data)-32:])
	base, err := b0state.DecodeLogical(data[73:len(data)-64], baseHash, expected.ChainID())
	if err != nil {
		return researchv2.State{}, researchv2.ErrState
	}
	state, err := researchv2.NewState(base, expected.Bindings())
	if err != nil || state.AppHash() != hash || state.TotalSupply() != expected.TotalSupply() {
		return researchv2.State{}, researchv2.ErrState
	}
	if state.Height() == 0 && state.AppHash() != expected.AppHash() {
		return researchv2.State{}, researchv2.ErrState
	}
	return state, nil
}

func (s *Store) Committed() (researchv2.State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return researchv2.State{}, ErrClosed
	}
	if s.uncertain {
		return researchv2.State{}, ErrUncertain
	}
	return s.committed, nil
}
func (s *Store) Pending() (researchv2.State, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return researchv2.State{}, false, ErrClosed
	}
	if s.uncertain {
		return researchv2.State{}, false, ErrUncertain
	}
	if s.pending == nil {
		return researchv2.State{}, false, nil
	}
	return *s.pending, true, nil
}
func (s *Store) PrepareTrusted(raws [][]byte, overLimit bool) ([]researchv2.Result, [32]byte, error) {
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
	var results []researchv2.Result
	var err error
	if !overLimit {
		candidate, results, err = researchv2.SimulateOrderedTrusted(candidate, s.profile, raws)
		if err != nil {
			return nil, [32]byte{}, err
		}
	}
	candidate, err = candidate.AdvanceBlock()
	if err != nil {
		return nil, [32]byte{}, err
	}
	s.pending = &candidate
	return results, candidate.AppHash(), nil
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
	record, err := encodeRecord(*s.pending, s.profile, s.genesisHash)
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
		return fmt.Errorf("sync research-v2 committed snapshot: %w", err)
	}
	if s.options.AfterSync != nil {
		if err := s.options.AfterSync(); err != nil {
			s.uncertain = true
			return fmt.Errorf("post-sync research-v2 response failure: %w", err)
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
