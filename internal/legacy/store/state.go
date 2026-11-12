// Package store defines and persists Pluto's minimal committed application state.
package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const FormatVersion uint32 = 1

const (
	hashDomain  = "PLUTO/APP_STATE/V1\x00"
	recordMagic = "PLUTO02!"
)

// State is the entire consensus-relevant application state in Phase 02.
// AppHash is a digest of Version, ChainID, and Height, including empty blocks.
type State struct {
	Version uint32
	ChainID string
	Height  int64
	AppHash [sha256.Size]byte
}

func Genesis(chainID string) (State, error) {
	s := State{Version: FormatVersion, ChainID: chainID}
	if err := validateFields(s); err != nil {
		return State{}, err
	}
	s.AppHash = hashState(s)
	return s, nil
}

func (s State) Next() (State, error) {
	if err := s.Validate(); err != nil {
		return State{}, err
	}
	if s.Height == math.MaxInt64 {
		return State{}, errors.New("application height overflow")
	}
	next := s
	next.Height++
	next.AppHash = hashState(next)
	return next, nil
}

func (s State) Validate() error {
	if err := validateFields(s); err != nil {
		return err
	}
	if s.AppHash != hashState(s) {
		return errors.New("application state hash mismatch")
	}
	return nil
}

func validateFields(s State) error {
	if s.Version != FormatVersion {
		return fmt.Errorf("unsupported state format version %d", s.Version)
	}
	if s.Height < 0 {
		return fmt.Errorf("negative application height %d", s.Height)
	}
	if len(s.ChainID) == 0 || len(s.ChainID) > 50 {
		return errors.New("chain ID must be 1 to 50 bytes")
	}
	for _, b := range []byte(s.ChainID) {
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '-' || b == '_' || b == '.') {
			return errors.New("chain ID contains unsupported characters")
		}
	}
	return nil
}

// hashState specifies the canonical commitment: fixed domain, big-endian
// uint32 version, uint16 chain-ID length, chain-ID bytes, uint64 height.
func hashState(s State) [sha256.Size]byte {
	var input bytes.Buffer
	input.WriteString(hashDomain)
	_ = binary.Write(&input, binary.BigEndian, s.Version)
	_ = binary.Write(&input, binary.BigEndian, uint16(len(s.ChainID)))
	input.WriteString(s.ChainID)
	_ = binary.Write(&input, binary.BigEndian, uint64(s.Height))
	return sha256.Sum256(input.Bytes())
}

func encode(s State) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	var record bytes.Buffer
	record.WriteString(recordMagic)
	_ = binary.Write(&record, binary.BigEndian, s.Version)
	_ = binary.Write(&record, binary.BigEndian, uint16(len(s.ChainID)))
	record.WriteString(s.ChainID)
	_ = binary.Write(&record, binary.BigEndian, uint64(s.Height))
	record.Write(s.AppHash[:])
	return record.Bytes(), nil
}

func decode(record []byte) (State, error) {
	const fixed = len(recordMagic) + 4 + 2 + 8 + sha256.Size
	if len(record) < fixed || string(record[:len(recordMagic)]) != recordMagic {
		return State{}, errors.New("invalid application state record")
	}
	pos := len(recordMagic)
	s := State{Version: binary.BigEndian.Uint32(record[pos : pos+4])}
	pos += 4
	chainLen := int(binary.BigEndian.Uint16(record[pos : pos+2]))
	pos += 2
	if chainLen == 0 || chainLen > 50 || len(record) != fixed+chainLen {
		return State{}, errors.New("invalid application state record length")
	}
	s.ChainID = string(record[pos : pos+chainLen])
	pos += chainLen
	height := binary.BigEndian.Uint64(record[pos : pos+8])
	if height > math.MaxInt64 {
		return State{}, errors.New("application state height overflow")
	}
	s.Height = int64(height)
	pos += 8
	copy(s.AppHash[:], record[pos:])
	if err := s.Validate(); err != nil {
		return State{}, fmt.Errorf("invalid persisted state: %w", err)
	}
	return s, nil
}
