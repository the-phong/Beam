package store

import (
	"errors"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
)

var stateKey = []byte("pluto/committed-state/v1")

// DB owns one writable Pebble directory. Pebble's lock prevents a second
// process from opening the same directory for writing.
type DB struct {
	db *pebble.DB
}

func Open(dir string) (*DB, error) {
	if dir == "" {
		return nil, errors.New("empty application database directory")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create application database: %w", err)
	}
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("open application database: %w", err)
	}
	return &DB{db: db}, nil
}

func (s *DB) Load() (*State, error) {
	value, closer, err := s.db.Get(stateKey)
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load committed state: %w", err)
	}
	copyValue := append([]byte(nil), value...)
	if err := closer.Close(); err != nil {
		return nil, fmt.Errorf("close state value: %w", err)
	}
	state, err := decode(copyValue)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// Save atomically replaces the one complete committed-state record. Sync
// requests an fsync of Pebble's WAL before successful return.
func (s *DB) Save(state State) error {
	record, err := encode(state)
	if err != nil {
		return err
	}
	if err := s.db.Set(stateKey, record, pebble.Sync); err != nil {
		return fmt.Errorf("persist committed state: %w", err)
	}
	return nil
}

func (s *DB) Close() error {
	return s.db.Close()
}
