// Package b0state implements the isolated B0 V2 logical account state.
// It is not connected to the Phase 02 ABCI application.
package b0state

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"

	"example.com/pluto/internal/classical/tx"
)

const (
	Version      uint32 = 2
	MaxAccounts         = 65536
	stateDomain         = "PLUTO/B0/STATE/V2\x00"
	genesisMagic        = "PLUTOG2!"
	maxLogical          = 4 + 2 + 50 + 8 + 8 + 4 + MaxAccounts*36
)

type Address [20]byte

type Account struct {
	Address Address
	Balance uint64
	Nonce   uint64
}

// State has no writable exported fields. All transitions return fresh account
// slices, so copying a State value cannot mutate an earlier snapshot.
type State struct {
	chainID  string
	height   int64
	supply   uint64
	accounts []Account
	hash     [32]byte
	// trusted is set only by constructors which check every invariant and
	// compute (or verify) the complete commitment. State is immutable after
	// construction; a zero State cannot enter the fast evaluation path.
	trusted bool
}

var ErrState = errors.New("invalid B0 V2 state")
var ErrGenesis = errors.New("invalid B0 V2 genesis")
var ErrHeightOverflow = errors.New("B0 V2 height overflow")

func validChainID(s string) bool {
	if len(s) < 1 || len(s) > tx.MaxChainID {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.') {
			return false
		}
	}
	return true
}

// NewState validates a canonical, sorted logical snapshot. It is also useful
// for explicit integrity-valid boundary fixtures; genesis uses ParseGenesis.
func NewState(chainID string, height int64, supply uint64, accounts []Account) (State, error) {
	if !validChainID(chainID) || height < 0 || len(accounts) < 1 || len(accounts) > MaxAccounts {
		return State{}, ErrState
	}
	copyAccounts := append([]Account(nil), accounts...)
	var sum uint64
	for i, account := range copyAccounts {
		if i > 0 && bytes.Compare(copyAccounts[i-1].Address[:], account.Address[:]) >= 0 {
			return State{}, ErrState
		}
		if account.Balance > math.MaxUint64-sum {
			return State{}, ErrState
		}
		sum += account.Balance
	}
	if sum != supply {
		return State{}, ErrState
	}
	s := State{chainID: chainID, height: height, supply: supply, accounts: copyAccounts, trusted: true}
	s.hash = sha256.Sum256(append([]byte(stateDomain), s.marshalLogical()...))
	return s, nil
}

func (s State) ChainID() string     { return s.chainID }
func (s State) Height() int64       { return s.height }
func (s State) TotalSupply() uint64 { return s.supply }
func (s State) AppHash() [32]byte   { return s.hash }
func (s State) Accounts() []Account { return append([]Account(nil), s.accounts...) }
func (s State) Account(address Address) (Account, bool) {
	i, found := s.lookup(address)
	if !found {
		return Account{Address: address}, false
	}
	return s.accounts[i], true
}
func (s State) lookup(address Address) (int, bool) {
	lo, hi := 0, len(s.accounts)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if bytes.Compare(s.accounts[mid].Address[:], address[:]) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo, lo < len(s.accounts) && s.accounts[lo].Address == address
}

// LogicalBytes is the exact D-021 state serialization, excluding domain and
// AppHash. A returned slice is independent of State storage.
func (s State) LogicalBytes() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s.marshalLogical(), nil
}

func (s State) marshalLogical() []byte {
	n := len(s.chainID)
	out := make([]byte, 4+2+n+8+8+4+36*len(s.accounts))
	binary.BigEndian.PutUint32(out[:4], Version)
	binary.BigEndian.PutUint16(out[4:6], uint16(n))
	copy(out[6:6+n], s.chainID)
	p := 6 + n
	binary.BigEndian.PutUint64(out[p:p+8], uint64(s.height))
	p += 8
	binary.BigEndian.PutUint64(out[p:p+8], s.supply)
	p += 8
	binary.BigEndian.PutUint32(out[p:p+4], uint32(len(s.accounts)))
	p += 4
	for _, a := range s.accounts {
		copy(out[p:p+20], a.Address[:])
		p += 20
		binary.BigEndian.PutUint64(out[p:p+8], a.Balance)
		p += 8
		binary.BigEndian.PutUint64(out[p:p+8], a.Nonce)
		p += 8
	}
	return out
}

// Validate checks all logical invariants and the stored application digest.
func (s State) Validate() error {
	if !validChainID(s.chainID) || s.height < 0 || len(s.accounts) < 1 || len(s.accounts) > MaxAccounts {
		return ErrState
	}
	var sum uint64
	for i, a := range s.accounts {
		if i > 0 && bytes.Compare(s.accounts[i-1].Address[:], a.Address[:]) >= 0 {
			return ErrState
		}
		if a.Balance > math.MaxUint64-sum {
			return ErrState
		}
		sum += a.Balance
	}
	if sum != s.supply || sha256.Sum256(append([]byte(stateDomain), s.marshalLogical()...)) != s.hash {
		return ErrState
	}
	return nil
}

// DecodeLogical accepts only exact canonical bytes and an independently
// provided AppHash. The chain context is explicit and must match.
func DecodeLogical(data []byte, hash [32]byte, expectedChainID string) (State, error) {
	const fixed = 4 + 2 + 8 + 8 + 4
	if len(data) < fixed+1+36 || len(data) > maxLogical || binary.BigEndian.Uint32(data[:4]) != Version {
		return State{}, ErrState
	}
	n := int(binary.BigEndian.Uint16(data[4:6]))
	if n < 1 || n > 50 || len(data) < fixed+n+36 {
		return State{}, ErrState
	}
	chain := string(data[6 : 6+n])
	if !validChainID(chain) || chain != expectedChainID {
		return State{}, ErrState
	}
	p := 6 + n
	height := binary.BigEndian.Uint64(data[p : p+8])
	p += 8
	if height > math.MaxInt64 {
		return State{}, ErrState
	}
	supply := binary.BigEndian.Uint64(data[p : p+8])
	p += 8
	k := uint64(binary.BigEndian.Uint32(data[p : p+4]))
	p += 4
	if k < 1 || k > MaxAccounts || uint64(len(data)) != uint64(p)+36*k {
		return State{}, ErrState
	}
	accounts := make([]Account, int(k))
	for i := range accounts {
		copy(accounts[i].Address[:], data[p:p+20])
		p += 20
		accounts[i].Balance = binary.BigEndian.Uint64(data[p : p+8])
		p += 8
		accounts[i].Nonce = binary.BigEndian.Uint64(data[p : p+8])
		p += 8
	}
	s, err := NewState(chain, int64(height), supply, accounts)
	if err != nil || s.hash != hash {
		return State{}, ErrState
	}
	return s, nil
}

// AdvanceBlock changes only the height and commitment. MaxInt64 cannot wrap.
func (s State) AdvanceBlock() (State, error) {
	if err := s.Validate(); err != nil {
		return State{}, err
	}
	if s.height == math.MaxInt64 {
		return State{}, ErrHeightOverflow
	}
	next := s
	next.height++
	next.hash = sha256.Sum256(append([]byte(stateDomain), next.marshalLogical()...))
	return next, nil
}
