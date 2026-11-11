package b0state

import (
	"bytes"
	"encoding/binary"
	"math"
)

type Allocation struct {
	Address Address
	Balance uint64
}

// EncodeGenesis emits only canonical, strictly ordered, positive allocations.
func EncodeGenesis(chainID string, allocations []Allocation) ([]byte, error) {
	if !validChainID(chainID) || len(allocations) < 1 || len(allocations) > MaxAccounts {
		return nil, ErrGenesis
	}
	var sum uint64
	for i, a := range allocations {
		if a.Balance == 0 || a.Balance > math.MaxUint64-sum ||
			(i > 0 && bytes.Compare(allocations[i-1].Address[:], a.Address[:]) >= 0) {
			return nil, ErrGenesis
		}
		sum += a.Balance
	}
	n := len(chainID)
	out := make([]byte, 8+4+2+n+4+28*len(allocations))
	copy(out[:8], genesisMagic)
	binary.BigEndian.PutUint32(out[8:12], Version)
	binary.BigEndian.PutUint16(out[12:14], uint16(n))
	copy(out[14:14+n], chainID)
	p := 14 + n
	binary.BigEndian.PutUint32(out[p:p+4], uint32(len(allocations)))
	p += 4
	for _, a := range allocations {
		copy(out[p:p+20], a.Address[:])
		p += 20
		binary.BigEndian.PutUint64(out[p:p+8], a.Balance)
		p += 8
	}
	return out, nil
}

// ParseGenesis verifies exact wire length, order, supply and explicit chain
// context. It creates the height-zero V2 logical state without touching disk.
func ParseGenesis(data []byte, expectedChainID string) (State, error) {
	const fixed = 8 + 4 + 2 + 4
	if len(data) < fixed+1+28 || len(data) > fixed+50+28*MaxAccounts ||
		string(data[:8]) != genesisMagic || binary.BigEndian.Uint32(data[8:12]) != Version {
		return State{}, ErrGenesis
	}
	n := int(binary.BigEndian.Uint16(data[12:14]))
	if n < 1 || n > 50 || len(data) < fixed+n+28 {
		return State{}, ErrGenesis
	}
	chain := string(data[14 : 14+n])
	if !validChainID(chain) || chain != expectedChainID {
		return State{}, ErrGenesis
	}
	p := 14 + n
	k := uint64(binary.BigEndian.Uint32(data[p : p+4]))
	p += 4
	if k < 1 || k > MaxAccounts || uint64(len(data)) != uint64(p)+28*k {
		return State{}, ErrGenesis
	}
	accounts := make([]Account, int(k))
	var sum uint64
	for i := range accounts {
		copy(accounts[i].Address[:], data[p:p+20])
		p += 20
		accounts[i].Balance = binary.BigEndian.Uint64(data[p : p+8])
		p += 8
		if accounts[i].Balance == 0 || accounts[i].Balance > math.MaxUint64-sum ||
			(i > 0 && bytes.Compare(accounts[i-1].Address[:], accounts[i].Address[:]) >= 0) {
			return State{}, ErrGenesis
		}
		sum += accounts[i].Balance
	}
	state, err := NewState(chain, 0, sum, accounts)
	if err != nil {
		return State{}, ErrGenesis
	}
	return state, nil
}
