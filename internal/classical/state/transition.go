package b0state

import (
	"math"

	"example.com/pluto/internal/classical/tx"
)

const (
	CodeInvalidNonce        tx.Code = 6
	CodeInsufficientBalance tx.Code = 7
	CodeBalanceOverflow     tx.Code = 8
	CodeAccountLimit        tx.Code = 9
)

// Evaluate runs the accepted stateless validator first, then the ordered
// state checks. A nonzero code returns the original State without mutation.
// An error indicates an invalid internal state, never an ordinary bad tx.
func Evaluate(s State, raw []byte) (State, tx.Code, error) {
	if err := s.Validate(); err != nil {
		return State{}, 0, err
	}
	return EvaluateTrusted(s, raw)
}

// EvaluateTrusted is for immutable states produced by NewState, ParseGenesis,
// DecodeLogical, or a prior successful transition. The store verifies its
// complete snapshot on load; this path avoids an O(K) hash for cheap-invalid
// wire input. A malformed in-memory State is an internal error, not a tx code.
func EvaluateTrusted(s State, raw []byte) (State, tx.Code, error) {
	if !s.trusted {
		return State{}, 0, ErrState
	}
	if code := tx.Validate(raw, s.chainID); code != tx.CodeOK {
		return s, code, nil
	}
	record, err := tx.Parse(raw)
	if err != nil {
		return State{}, 0, err
	} // unreachable after CodeOK
	sender20, err := tx.SenderAddress(record.Sender)
	if err != nil {
		return State{}, 0, err
	} // unreachable after CodeOK
	var sender, receiver Address
	copy(sender[:], sender20[:])
	copy(receiver[:], record.Receiver[:])
	return ApplyAuthenticatedTrusted(s, sender, receiver, record.Amount, record.Nonce)
}

// ApplyAuthenticatedTrusted implements the accepted B0 balance/nonce
// transition after profile-specific authentication has succeeded. B1 uses
// this same business-state primitive with its separately bound hybrid keys.
// A failed result always returns the unchanged candidate state.
func ApplyAuthenticatedTrusted(s State, sender, receiver Address, amount, nonce uint64) (State, tx.Code, error) {
	if !s.trusted {
		return State{}, 0, ErrState
	}
	if amount == 0 {
		return s, tx.CodeZeroAmount, nil
	}
	senderIndex, senderFound := s.lookup(sender)
	senderAccount := Account{Address: sender}
	if senderFound {
		senderAccount = s.accounts[senderIndex]
	}
	if nonce != senderAccount.Nonce || senderAccount.Nonce == math.MaxUint64 {
		return s, CodeInvalidNonce, nil
	}
	if senderAccount.Balance < amount {
		return s, CodeInsufficientBalance, nil
	}
	if sender == receiver {
		accounts := append([]Account(nil), s.accounts...)
		accounts[senderIndex].Nonce++ // sufficient balance implies sender exists
		next, err := NewState(s.chainID, s.height, s.supply, accounts)
		return next, tx.CodeOK, err
	}
	receiverIndex, receiverFound := s.lookup(receiver)
	receiverAccount := Account{Address: receiver}
	if receiverFound {
		receiverAccount = s.accounts[receiverIndex]
	}
	if receiverAccount.Balance > math.MaxUint64-amount {
		return s, CodeBalanceOverflow, nil
	}
	if !receiverFound && len(s.accounts) == MaxAccounts {
		return s, CodeAccountLimit, nil
	}
	accounts := make([]Account, len(s.accounts))
	copy(accounts, s.accounts)
	accounts[senderIndex].Balance -= amount
	accounts[senderIndex].Nonce++
	if receiverFound {
		accounts[receiverIndex].Balance += amount
	} else {
		accounts = append(accounts, Account{})
		copy(accounts[receiverIndex+1:], accounts[receiverIndex:len(accounts)-1])
		accounts[receiverIndex] = Account{Address: receiver, Balance: amount}
	}
	next, err := NewState(s.chainID, s.height, s.supply, accounts)
	return next, tx.CodeOK, err
}

// SimulateOrderedTrusted starts from a verified immutable snapshot and uses
// the same transition logic as Evaluate without repeated full-state checks.
func SimulateOrderedTrusted(s State, raws [][]byte) (State, []tx.Code, error) {
	if !s.trusted {
		return State{}, nil, ErrState
	}
	candidate := s
	codes := make([]tx.Code, 0, len(raws))
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

// SimulateOrdered evaluates every byte string in supplied order against all
// prior successes; failures leave the candidate unchanged and do not stop it.
// It does not enforce an ABCI block envelope or advance height.
func SimulateOrdered(s State, raws [][]byte) (State, []tx.Code, error) {
	if err := s.Validate(); err != nil {
		return State{}, nil, err
	}
	candidate := s
	codes := make([]tx.Code, 0, len(raws))
	for _, raw := range raws {
		next, code, err := Evaluate(candidate, raw)
		if err != nil {
			return State{}, nil, err
		}
		candidate = next
		codes = append(codes, code)
	}
	return candidate, codes, nil
}
