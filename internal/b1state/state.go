// Package b1state binds immutable genesis keys to the accepted B0 balance
// transition while giving B1 its own genesis and application commitment.
package b1state

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/b1tx"
)

const (
	GenesisMagic              = "PLUTOG3!"
	GenesisVersion     uint32 = 3
	MaxGenesisAccounts        = 1024
	entrySize                 = 20 + 8 + 33 + mldsa44.PublicKeySize
	MaxGenesisSize            = 8 + 4 + 2 + b1tx.MaxChainID + 4 + entrySize*MaxGenesisAccounts
	registryDomain            = "PLUTO/B1/REGISTRY/V1\x00"
	stateDomain               = "PLUTO/B1/STATE/V1\x00"
)

type Binding struct {
	Address b0state.Address
	SecpKey [33]byte
	MLKey   [mldsa44.PublicKeySize]byte
}

type Allocation struct {
	Binding Binding
	Balance uint64
}

// State carries an immutable, genesis-bound registry. The B0 logical state
// remains an internal balance/nonce primitive; B1 commits to both it and the
// registry under a separate domain and persists under a separate record key.
type State struct {
	base         b0state.State
	bindings     []Binding
	registryHash [32]byte
	hash         [32]byte
}

var ErrGenesis = errors.New("invalid B1 hybrid genesis")
var ErrState = errors.New("invalid B1 hybrid state")

func validBindings(bindings []Binding) bool {
	if len(bindings) < 1 || len(bindings) > MaxGenesisAccounts {
		return false
	}
	for i, entry := range bindings {
		if i > 0 && bytes.Compare(bindings[i-1].Address[:], entry.Address[:]) >= 0 {
			return false
		}
		address, err := b1tx.SenderAddress(entry.SecpKey, entry.MLKey)
		if err != nil || entry.Address != b0state.Address(address) {
			return false
		}
	}
	return true
}

func registryDigest(bindings []Binding) [32]byte {
	buf := make([]byte, len(registryDomain)+4+len(bindings)*(20+33+mldsa44.PublicKeySize))
	copy(buf, registryDomain)
	binary.BigEndian.PutUint32(buf[len(registryDomain):], uint32(len(bindings)))
	p := len(registryDomain) + 4
	for _, entry := range bindings {
		copy(buf[p:p+20], entry.Address[:])
		p += 20
		copy(buf[p:p+33], entry.SecpKey[:])
		p += 33
		copy(buf[p:p+mldsa44.PublicKeySize], entry.MLKey[:])
		p += mldsa44.PublicKeySize
	}
	return sha256.Sum256(buf)
}

func commitment(base b0state.State, registryHash [32]byte) ([32]byte, error) {
	logical, err := base.LogicalBytes()
	if err != nil {
		return [32]byte{}, err
	}
	buf := make([]byte, len(stateDomain)+32+len(logical))
	copy(buf, stateDomain)
	copy(buf[len(stateDomain):], registryHash[:])
	copy(buf[len(stateDomain)+32:], logical)
	return sha256.Sum256(buf), nil
}

func NewState(base b0state.State, bindings []Binding) (State, error) {
	if err := base.Validate(); err != nil || !validBindings(bindings) {
		return State{}, ErrState
	}
	accounts := base.Accounts()
	if len(accounts) != len(bindings) {
		return State{}, ErrState
	}
	for i := range accounts {
		if accounts[i].Address != bindings[i].Address {
			return State{}, ErrState
		}
	}
	copyBindings := append([]Binding(nil), bindings...)
	rh := registryDigest(copyBindings)
	hash, err := commitment(base, rh)
	if err != nil {
		return State{}, ErrState
	}
	return State{base: base, bindings: copyBindings, registryHash: rh, hash: hash}, nil
}

func (s State) Base() b0state.State                               { return s.base }
func (s State) ChainID() string                                   { return s.base.ChainID() }
func (s State) Height() int64                                     { return s.base.Height() }
func (s State) TotalSupply() uint64                               { return s.base.TotalSupply() }
func (s State) AppHash() [32]byte                                 { return s.hash }
func (s State) RegistryHash() [32]byte                            { return s.registryHash }
func (s State) Bindings() []Binding                               { return append([]Binding(nil), s.bindings...) }
func (s State) Account(a b0state.Address) (b0state.Account, bool) { return s.base.Account(a) }
func (s State) lookup(address b0state.Address) (Binding, bool) {
	lo, hi := 0, len(s.bindings)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if bytes.Compare(s.bindings[mid].Address[:], address[:]) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(s.bindings) && s.bindings[lo].Address == address {
		return s.bindings[lo], true
	}
	return Binding{}, false
}

func (s State) Validate() error {
	verified, err := NewState(s.base, s.bindings)
	if err != nil || verified.hash != s.hash || verified.registryHash != s.registryHash {
		return ErrState
	}
	return nil
}

func (s State) withBase(base b0state.State) (State, error) {
	// Every successful B1 transfer uses only genesis-registered recipients;
	// account membership and the registry cannot change during execution.
	accounts := base.Accounts()
	if len(accounts) != len(s.bindings) {
		return State{}, ErrState
	}
	for i := range accounts {
		if accounts[i].Address != s.bindings[i].Address {
			return State{}, ErrState
		}
	}
	hash, err := commitment(base, s.registryHash)
	if err != nil {
		return State{}, err
	}
	return State{base: base, bindings: s.bindings, registryHash: s.registryHash, hash: hash}, nil
}

func (s State) AdvanceBlock() (State, error) {
	base, err := s.base.AdvanceBlock()
	if err != nil {
		return State{}, err
	}
	return s.withBase(base)
}

func EncodeGenesis(chainID string, allocations []Allocation) ([]byte, error) {
	if !b1tx.ValidChainID(chainID) || len(allocations) < 1 || len(allocations) > MaxGenesisAccounts {
		return nil, ErrGenesis
	}
	bindings := make([]Binding, len(allocations))
	accounts := make([]b0state.Account, len(allocations))
	var sum uint64
	for i, a := range allocations {
		if a.Balance > math.MaxUint64-sum {
			return nil, ErrGenesis
		}
		sum += a.Balance
		bindings[i] = a.Binding
		accounts[i] = b0state.Account{Address: a.Binding.Address, Balance: a.Balance}
	}
	if sum == 0 || !validBindings(bindings) {
		return nil, ErrGenesis
	}
	if _, err := b0state.NewState(chainID, 0, sum, accounts); err != nil {
		return nil, ErrGenesis
	}
	n := len(chainID)
	out := make([]byte, 8+4+2+n+4+entrySize*len(allocations))
	copy(out[:8], GenesisMagic)
	binary.BigEndian.PutUint32(out[8:12], GenesisVersion)
	binary.BigEndian.PutUint16(out[12:14], uint16(n))
	copy(out[14:14+n], chainID)
	p := 14 + n
	binary.BigEndian.PutUint32(out[p:p+4], uint32(len(allocations)))
	p += 4
	for _, a := range allocations {
		copy(out[p:p+20], a.Binding.Address[:])
		p += 20
		binary.BigEndian.PutUint64(out[p:p+8], a.Balance)
		p += 8
		copy(out[p:p+33], a.Binding.SecpKey[:])
		p += 33
		copy(out[p:p+mldsa44.PublicKeySize], a.Binding.MLKey[:])
		p += mldsa44.PublicKeySize
	}
	return out, nil
}

func ParseGenesis(raw []byte, expectedChainID string) (State, error) {
	if len(raw) < 8+4+2+1+4+entrySize || len(raw) > MaxGenesisSize ||
		string(raw[:8]) != GenesisMagic || binary.BigEndian.Uint32(raw[8:12]) != GenesisVersion {
		return State{}, ErrGenesis
	}
	n := int(binary.BigEndian.Uint16(raw[12:14]))
	if n < 1 || n > b1tx.MaxChainID || len(raw) < 18+n+entrySize {
		return State{}, ErrGenesis
	}
	chain := string(raw[14 : 14+n])
	if !b1tx.ValidChainID(chain) || chain != expectedChainID {
		return State{}, ErrGenesis
	}
	p := 14 + n
	k := uint64(binary.BigEndian.Uint32(raw[p : p+4]))
	p += 4
	if k < 1 || k > MaxGenesisAccounts || uint64(len(raw)) != uint64(p)+entrySize*k {
		return State{}, ErrGenesis
	}
	allocations := make([]Allocation, int(k))
	for i := range allocations {
		copy(allocations[i].Binding.Address[:], raw[p:p+20])
		p += 20
		allocations[i].Balance = binary.BigEndian.Uint64(raw[p : p+8])
		p += 8
		copy(allocations[i].Binding.SecpKey[:], raw[p:p+33])
		p += 33
		copy(allocations[i].Binding.MLKey[:], raw[p:p+mldsa44.PublicKeySize])
		p += mldsa44.PublicKeySize
	}
	canonical, err := EncodeGenesis(chain, allocations)
	if err != nil || !bytes.Equal(canonical, raw) {
		return State{}, ErrGenesis
	}
	bindings := make([]Binding, len(allocations))
	accounts := make([]b0state.Account, len(allocations))
	var supply uint64
	for i, a := range allocations {
		bindings[i] = a.Binding
		accounts[i] = b0state.Account{Address: a.Binding.Address, Balance: a.Balance}
		supply += a.Balance
	}
	base, err := b0state.NewState(chain, 0, supply, accounts)
	if err != nil {
		return State{}, ErrGenesis
	}
	return NewState(base, bindings)
}

// EvaluateTrusted applies one strict-AND authenticated transfer. The stage
// order is structural, chain, amount, key encoding/binding, secp proof,
// ML-DSA proof, then account state. No local resource signal affects a code.
func EvaluateTrusted(s State, raw []byte) (State, b1tx.Code, error) {
	if len(s.bindings) == 0 {
		return State{}, 0, ErrState
	}
	t, err := b1tx.Parse(raw)
	if err != nil {
		return s, b1tx.CodeMalformed, nil
	}
	if t.ChainID != s.ChainID() {
		return s, b1tx.CodeWrongChain, nil
	}
	if t.Amount == 0 {
		return s, b1tx.CodeZeroAmount, nil
	}
	if _, err := b1tx.ParseSecpKey(t.SecpKey); err != nil {
		return s, b1tx.CodeBadSecpKey, nil
	}
	if _, err := b1tx.ParseMLKey(t.MLKey); err != nil {
		return s, b1tx.CodeBadMLKey, nil
	}
	derived, err := b1tx.SenderAddress(t.SecpKey, t.MLKey)
	if err != nil || t.Sender != derived {
		return s, b1tx.CodeBadBinding, nil
	}
	registered, found := s.lookup(b0state.Address(t.Sender))
	if !found || registered.SecpKey != t.SecpKey || registered.MLKey != t.MLKey {
		return s, b1tx.CodeBadBinding, nil
	}
	if _, found := s.lookup(b0state.Address(t.Receiver)); !found {
		return s, b1tx.CodeBadBinding, nil
	}
	digest, _ := b1tx.SigningDigest(t)
	if code := b1tx.VerifySecp(t, digest); code != b1tx.CodeOK {
		return s, code, nil
	}
	if code := b1tx.VerifyML(t, digest); code != b1tx.CodeOK {
		return s, code, nil
	}
	base, code, err := b0state.ApplyAuthenticatedTrusted(s.base, b0state.Address(t.Sender), b0state.Address(t.Receiver), t.Amount, t.Nonce)
	if err != nil {
		return State{}, 0, err
	}
	if code != 0 {
		return s, b1tx.Code(code), nil
	}
	next, err := s.withBase(base)
	return next, b1tx.CodeOK, err
}

func SimulateOrderedTrusted(s State, raws [][]byte) (State, []b1tx.Code, error) {
	candidate := s
	codes := make([]b1tx.Code, 0, len(raws))
	for _, raw := range raws {
		next, code, err := EvaluateTrusted(candidate, raw)
		if err != nil {
			return State{}, nil, err
		}
		candidate = next
		codes = append(codes, code)
	}
	return candidate, codes, nil
}
