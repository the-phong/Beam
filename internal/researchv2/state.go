package researchv2

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"

	"example.com/pluto/internal/b0state"
)

const (
	GenesisMagic              = "PLUTOG4!"
	GenesisVersion     uint32 = 4
	MaxGenesisAccounts        = 1024
	genesisEntrySize          = 20 + 8 + 33 + mldsa44.PublicKeySize
	registryDomain            = "PLUTO/RV2/REGISTRY/V1\x00"
	stateDomain               = "PLUTO/RV2/STATE/V1\x00"
)

type Binding struct {
	Account b0state.Address
	SecpKey [33]byte
	MLKey   [mldsa44.PublicKeySize]byte
}
type Allocation struct {
	Binding Binding
	Balance uint64
}

// State is immutable after construction. All profiles use the same registry,
// B0 transfer primitive, genesis bytes and AppHash rule.
type State struct {
	base         b0state.State
	bindings     []Binding
	registryHash [32]byte
	hash         [32]byte
	trusted      bool
}

var ErrGenesis = errors.New("invalid research-v2 genesis")
var ErrState = errors.New("invalid research-v2 state")

func validBindings(bindings []Binding) bool {
	if len(bindings) < 1 || len(bindings) > MaxGenesisAccounts {
		return false
	}
	seenSecp := make(map[[33]byte]struct{}, len(bindings))
	seenML := make(map[[mldsa44.PublicKeySize]byte]struct{}, len(bindings))
	for i, b := range bindings {
		if i > 0 && bytes.Compare(bindings[i-1].Account[:], b.Account[:]) >= 0 {
			return false
		}
		if _, err := parseSecpKey(b.SecpKey); err != nil {
			return false
		}
		if _, err := parseMLKey(b.MLKey); err != nil {
			return false
		}
		if _, exists := seenSecp[b.SecpKey]; exists {
			return false
		}
		if _, exists := seenML[b.MLKey]; exists {
			return false
		}
		seenSecp[b.SecpKey] = struct{}{}
		seenML[b.MLKey] = struct{}{}
	}
	return true
}

func registryDigest(bindings []Binding) [32]byte {
	buf := make([]byte, len(registryDomain)+4+len(bindings)*(20+33+mldsa44.PublicKeySize))
	copy(buf, registryDomain)
	binary.BigEndian.PutUint32(buf[len(registryDomain):], uint32(len(bindings)))
	o := len(registryDomain) + 4
	for _, b := range bindings {
		copy(buf[o:o+20], b.Account[:])
		o += 20
		copy(buf[o:o+33], b.SecpKey[:])
		o += 33
		copy(buf[o:o+mldsa44.PublicKeySize], b.MLKey[:])
		o += mldsa44.PublicKeySize
	}
	return sha256.Sum256(buf)
}

func stateDigest(base b0state.State, registryHash [32]byte) ([32]byte, error) {
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
		if accounts[i].Address != bindings[i].Account {
			return State{}, ErrState
		}
	}
	copyBindings := append([]Binding(nil), bindings...)
	rh := registryDigest(copyBindings)
	h, err := stateDigest(base, rh)
	if err != nil {
		return State{}, ErrState
	}
	return State{base: base, bindings: copyBindings, registryHash: rh, hash: h, trusted: true}, nil
}

func (s State) Validate() error {
	if !s.trusted {
		return ErrState
	}
	rebuilt, err := NewState(s.base, s.bindings)
	if err != nil || rebuilt.registryHash != s.registryHash || rebuilt.hash != s.hash {
		return ErrState
	}
	return nil
}
func (s State) ChainID() string                                   { return s.base.ChainID() }
func (s State) Height() int64                                     { return s.base.Height() }
func (s State) TotalSupply() uint64                               { return s.base.TotalSupply() }
func (s State) AppHash() [32]byte                                 { return s.hash }
func (s State) RegistryHash() [32]byte                            { return s.registryHash }
func (s State) Bindings() []Binding                               { return append([]Binding(nil), s.bindings...) }
func (s State) Account(a b0state.Address) (b0state.Account, bool) { return s.base.Account(a) }
func (s State) lookup(a b0state.Address) (Binding, bool) {
	lo, hi := 0, len(s.bindings)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if bytes.Compare(s.bindings[mid].Account[:], a[:]) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(s.bindings) && s.bindings[lo].Account == a {
		return s.bindings[lo], true
	}
	return Binding{}, false
}

func (s State) withBase(base b0state.State) (State, error) {
	if !s.trusted {
		return State{}, ErrState
	}
	accounts := base.Accounts()
	if len(accounts) != len(s.bindings) {
		return State{}, ErrState
	}
	for i := range accounts {
		if accounts[i].Address != s.bindings[i].Account {
			return State{}, ErrState
		}
	}
	h, err := stateDigest(base, s.registryHash)
	if err != nil {
		return State{}, err
	}
	return State{base: base, bindings: s.bindings, registryHash: s.registryHash, hash: h, trusted: true}, nil
}
func (s State) AdvanceBlock() (State, error) {
	if !s.trusted {
		return State{}, ErrState
	}
	base, err := s.base.AdvanceBlock()
	if err != nil {
		return State{}, err
	}
	return s.withBase(base)
}

func EncodeGenesis(chainID string, allocations []Allocation) ([]byte, error) {
	if !ValidChainID(chainID) || len(allocations) < 1 || len(allocations) > MaxGenesisAccounts {
		return nil, ErrGenesis
	}
	bindings := make([]Binding, len(allocations))
	accounts := make([]b0state.Account, len(allocations))
	var supply uint64
	for i, a := range allocations {
		if a.Balance > math.MaxUint64-supply {
			return nil, ErrGenesis
		}
		supply += a.Balance
		bindings[i] = a.Binding
		accounts[i] = b0state.Account{Address: a.Binding.Account, Balance: a.Balance}
	}
	if supply == 0 || !validBindings(bindings) {
		return nil, ErrGenesis
	}
	if _, err := b0state.NewState(chainID, 0, supply, accounts); err != nil {
		return nil, ErrGenesis
	}
	n := len(chainID)
	out := make([]byte, 8+4+2+n+4+genesisEntrySize*len(allocations))
	copy(out[:8], GenesisMagic)
	binary.BigEndian.PutUint32(out[8:12], GenesisVersion)
	binary.BigEndian.PutUint16(out[12:14], uint16(n))
	copy(out[14:14+n], chainID)
	o := 14 + n
	binary.BigEndian.PutUint32(out[o:o+4], uint32(len(allocations)))
	o += 4
	for _, a := range allocations {
		copy(out[o:o+20], a.Binding.Account[:])
		o += 20
		binary.BigEndian.PutUint64(out[o:o+8], a.Balance)
		o += 8
		copy(out[o:o+33], a.Binding.SecpKey[:])
		o += 33
		copy(out[o:o+mldsa44.PublicKeySize], a.Binding.MLKey[:])
		o += mldsa44.PublicKeySize
	}
	return out, nil
}

func ParseGenesis(raw []byte, chainID string) (State, error) {
	const min = 8 + 4 + 2 + 1 + 4 + genesisEntrySize
	const max = 8 + 4 + 2 + MaxChainID + 4 + genesisEntrySize*MaxGenesisAccounts
	if len(raw) < min || len(raw) > max || string(raw[:8]) != GenesisMagic || binary.BigEndian.Uint32(raw[8:12]) != GenesisVersion {
		return State{}, ErrGenesis
	}
	n := int(binary.BigEndian.Uint16(raw[12:14]))
	if n < 1 || n > MaxChainID || len(raw) < 18+n+genesisEntrySize {
		return State{}, ErrGenesis
	}
	chain := string(raw[14 : 14+n])
	if !ValidChainID(chain) || chain != chainID {
		return State{}, ErrGenesis
	}
	o := 14 + n
	k := uint64(binary.BigEndian.Uint32(raw[o : o+4]))
	o += 4
	if k < 1 || k > MaxGenesisAccounts || uint64(len(raw)) != uint64(o)+genesisEntrySize*k {
		return State{}, ErrGenesis
	}
	allocations := make([]Allocation, int(k))
	for i := range allocations {
		copy(allocations[i].Binding.Account[:], raw[o:o+20])
		o += 20
		allocations[i].Balance = binary.BigEndian.Uint64(raw[o : o+8])
		o += 8
		copy(allocations[i].Binding.SecpKey[:], raw[o:o+33])
		o += 33
		copy(allocations[i].Binding.MLKey[:], raw[o:o+mldsa44.PublicKeySize])
		o += mldsa44.PublicKeySize
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
		accounts[i] = b0state.Account{Address: a.Binding.Account, Balance: a.Balance}
		supply += a.Balance
	}
	base, err := b0state.NewState(chain, 0, supply, accounts)
	if err != nil {
		return State{}, ErrGenesis
	}
	return NewState(base, bindings)
}

// EvaluateTrusted is deterministic for the same immutable state, profile and
// bytes. Invalid input returns the original state without mutation.
func EvaluateTrusted(s State, profile Profile, raw []byte) (State, Result, error) {
	if !s.trusted || !profile.Valid() {
		return State{}, Result{}, ErrState
	}
	t, err := Parse(raw)
	if err != nil {
		return s, Result{CodeMalformed, StageStructural}, nil
	}
	if t.Profile != profile {
		return s, Result{CodeWrongProfile, StageProfile}, nil
	}
	if t.ChainID != s.ChainID() {
		return s, Result{CodeWrongChain, StageChain}, nil
	}
	if t.Amount == 0 {
		return s, Result{CodeZeroAmount, StageAmount}, nil
	}
	sender, found := s.lookup(b0state.Address(t.Sender))
	if !found {
		return s, Result{CodeBadBinding, StageBinding}, nil
	}
	if _, found = s.lookup(b0state.Address(t.Receiver)); !found {
		return s, Result{CodeBadBinding, StageBinding}, nil
	}
	digest, err := SigningDigest(t)
	if err != nil {
		return State{}, Result{}, err
	}
	if profile.UsesSecp() {
		if code := verifySecp(sender.SecpKey, t.SecpSignature, digest); code != CodeOK {
			return s, Result{code, StageSecp}, nil
		}
	}
	if profile.UsesML() {
		if code := verifyML(sender.MLKey, t.MLSignature, digest); code != CodeOK {
			return s, Result{code, StageML}, nil
		}
	}
	base, code, err := b0state.ApplyAuthenticatedTrusted(s.base, b0state.Address(t.Sender), b0state.Address(t.Receiver), t.Amount, t.Nonce)
	if err != nil {
		return State{}, Result{}, err
	}
	if code != 0 {
		return s, Result{Code(code), StageState}, nil
	}
	next, err := s.withBase(base)
	if err != nil {
		return State{}, Result{}, err
	}
	return next, Result{CodeOK, StageAccepted}, nil
}
func Evaluate(s State, profile Profile, raw []byte) (State, Result, error) {
	if err := s.Validate(); err != nil {
		return State{}, Result{}, err
	}
	return EvaluateTrusted(s, profile, raw)
}
func SimulateOrderedTrusted(s State, profile Profile, raws [][]byte) (State, []Result, error) {
	if !s.trusted || !profile.Valid() {
		return State{}, nil, ErrState
	}
	candidate := s
	results := make([]Result, 0, len(raws))
	for _, raw := range raws {
		next, result, err := EvaluateTrusted(candidate, profile, raw)
		if err != nil {
			return State{}, nil, err
		}
		candidate = next
		results = append(results, result)
	}
	return candidate, results, nil
}
