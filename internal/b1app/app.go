// Package b1app connects the isolated hybrid state machine to ABCI++.
package b1app

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/b1genesis"
	"example.com/pluto/internal/b1state"
	"example.com/pluto/internal/b1store"
	"example.com/pluto/internal/b1tx"
	"example.com/pluto/internal/version"
)

const (
	maxBlockTxs     = 6144
	maxBlockBytes   = 1048576
	resultCodespace = "pluto.b1.v1"
)

var ErrGenesisConflict = errors.New("B1 genesis conflicts with configured or committed hybrid state")

type App struct {
	mu              sync.RWMutex
	chainID         string
	expectedGenesis []byte
	expectedState   b1state.State
	store           *b1store.Store
	closed          bool
}

var _ abci.Application = (*App)(nil)

func Open(dbDir, chainID string, genesis []byte) (*App, error) {
	return OpenWithStoreOptions(dbDir, chainID, genesis, b1store.Options{})
}
func OpenWithStoreOptions(dbDir, chainID string, genesis []byte, options b1store.Options) (*App, error) {
	expected, err := b1state.ParseGenesis(genesis, chainID)
	if err != nil {
		return nil, err
	}
	store, err := b1store.OpenExistingWithOptions(dbDir, chainID, genesis, options)
	if err != nil {
		return nil, err
	}
	return &App{chainID: chainID, expectedGenesis: bytes.Clone(genesis), expectedState: expected, store: store}, nil
}
func (a *App) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return b1store.ErrClosed
	}
	a.closed = true
	return a.store.Close()
}
func (a *App) Committed() (b1state.State, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return b1state.State{}, b1store.ErrClosed
	}
	return a.store.Committed()
}
func (a *App) Info(context.Context, *abci.RequestInfo) (*abci.ResponseInfo, error) {
	state, err := a.Committed()
	if err != nil {
		return nil, err
	}
	hash := state.AppHash()
	return &abci.ResponseInfo{Data: "Pluto B1 hybrid V1", Version: version.Number, AppVersion: 3, LastBlockHeight: state.Height(), LastBlockAppHash: append([]byte(nil), hash[:]...)}, nil
}
func (a *App) InitChain(_ context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	if req == nil || req.InitialHeight != 1 || req.ChainId != a.chainID {
		return nil, ErrGenesisConflict
	}
	binary, state, err := b1genesis.Decode(req.AppStateBytes, req.ChainId)
	if err != nil || !bytes.Equal(binary, a.expectedGenesis) || state.AppHash() != a.expectedState.AppHash() {
		return nil, ErrGenesisConflict
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil, b1store.ErrClosed
	}
	committed, err := a.store.Committed()
	if err != nil {
		return nil, err
	}
	if committed.Height() != 0 || committed.AppHash() != state.AppHash() {
		return nil, ErrGenesisConflict
	}
	hash := state.AppHash()
	return &abci.ResponseInitChain{AppHash: append([]byte(nil), hash[:]...)}, nil
}
func resultLog(code b1tx.Code) string {
	switch code {
	case b1tx.CodeOK:
		return "ok"
	case b1tx.CodeMalformed:
		return "malformed"
	case b1tx.CodeWrongChain:
		return "wrong_chain"
	case b1tx.CodeBadSecpKey:
		return "invalid_classical_key"
	case b1tx.CodeBadSecpSignature:
		return "invalid_classical_signature"
	case b1tx.CodeZeroAmount:
		return "invalid_amount"
	case b1tx.CodeInvalidNonce:
		return "invalid_nonce"
	case b1tx.CodeInsufficientBalance:
		return "insufficient_balance"
	case b1tx.CodeBalanceOverflow:
		return "balance_overflow"
	case b1tx.CodeAccountLimit:
		return "account_limit"
	case b1tx.CodeBlockLimit:
		return "block_limit"
	case b1tx.CodeBadBinding:
		return "unregistered_or_conflicting_key_binding"
	case b1tx.CodeBadMLKey:
		return "invalid_ml_dsa_key"
	case b1tx.CodeBadMLSignature:
		return "invalid_ml_dsa_signature"
	default:
		panic("unrecognized B1 result code")
	}
}
func (a *App) CheckTx(_ context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	state, err := a.Committed()
	if err != nil {
		return nil, err
	}
	var raw []byte
	if req != nil {
		raw = req.Tx
	}
	_, code, err := b1state.EvaluateTrusted(state, raw)
	if err != nil {
		return nil, err
	}
	return &abci.ResponseCheckTx{Code: uint32(code), Log: resultLog(code), Codespace: resultCodespace}, nil
}
func (a *App) committedAtHeightLocked(height int64) (b1state.State, error) {
	if a.closed {
		return b1state.State{}, errors.New("B1 application is closed")
	}
	state, err := a.store.Committed()
	if err != nil {
		return b1state.State{}, err
	}
	if state.Height() == math.MaxInt64 || height != state.Height()+1 {
		return b1state.State{}, errors.New("unexpected B1 height")
	}
	return state, nil
}
func blockOverLimit(raws [][]byte) bool {
	if len(raws) > maxBlockTxs {
		return true
	}
	var size uint64
	for _, raw := range raws {
		if uint64(len(raw)) > maxBlockBytes-size {
			return true
		}
		size += uint64(len(raw))
	}
	return false
}
func (a *App) PrepareProposal(_ context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	if req == nil {
		return nil, errors.New("missing B1 proposal")
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	candidate, err := a.committedAtHeightLocked(req.Height)
	if err != nil {
		return nil, err
	}
	resp := &abci.ResponsePrepareProposal{}
	if req.MaxTxBytes <= 0 {
		return resp, nil
	}
	budget := uint64(req.MaxTxBytes)
	if budget > maxBlockBytes {
		budget = maxBlockBytes
	}
	var used uint64
	for _, raw := range req.Txs {
		if len(resp.Txs) == maxBlockTxs {
			break
		}
		if uint64(len(raw)) > budget-used {
			continue
		}
		next, code, err := b1state.EvaluateTrusted(candidate, raw)
		if err != nil {
			return nil, err
		}
		if code != b1tx.CodeOK {
			continue
		}
		resp.Txs = append(resp.Txs, raw)
		candidate = next
		used += uint64(len(raw))
	}
	return resp, nil
}
func (a *App) ProcessProposal(_ context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	if req == nil {
		return nil, errors.New("missing B1 proposal")
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	candidate, err := a.committedAtHeightLocked(req.Height)
	if err != nil {
		return nil, err
	}
	if blockOverLimit(req.Txs) {
		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
	}
	for _, raw := range req.Txs {
		next, code, err := b1state.EvaluateTrusted(candidate, raw)
		if err != nil {
			return nil, err
		}
		if code != b1tx.CodeOK {
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}
		candidate = next
	}
	return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
}
func (a *App) FinalizeBlock(_ context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	if req == nil {
		return nil, errors.New("missing B1 finalized block")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, err := a.committedAtHeightLocked(req.Height); err != nil {
		return nil, err
	}
	overLimit := blockOverLimit(req.Txs)
	codes, hash, err := a.store.PrepareTrusted(req.Txs, overLimit)
	if err != nil {
		return nil, err
	}
	resp := &abci.ResponseFinalizeBlock{AppHash: append([]byte(nil), hash[:]...), TxResults: make([]*abci.ExecTxResult, len(req.Txs))}
	for i := range req.Txs {
		code := b1tx.CodeBlockLimit
		if !overLimit {
			code = codes[i]
		}
		resp.TxResults[i] = &abci.ExecTxResult{Code: uint32(code), Log: resultLog(code), Codespace: resultCodespace}
	}
	return resp, nil
}
func (a *App) Commit(context.Context, *abci.RequestCommit) (*abci.ResponseCommit, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil, b1store.ErrClosed
	}
	if err := a.store.Commit(); err != nil {
		return nil, err
	}
	return &abci.ResponseCommit{}, nil
}
func (a *App) Query(_ context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	state, err := a.Committed()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Path != "b1/account/v1" || len(req.Data) != 20 || req.Height != 0 || req.Prove {
		return &abci.ResponseQuery{Code: 1, Codespace: "pluto.b1.query.v1", Log: "unsupported_query"}, nil
	}
	var address b0state.Address
	copy(address[:], req.Data)
	account, found := state.Account(address)
	value := []byte{0}
	if found {
		value = make([]byte, 17)
		value[0] = 1
		binary.BigEndian.PutUint64(value[1:9], account.Balance)
		binary.BigEndian.PutUint64(value[9:17], account.Nonce)
	}
	return &abci.ResponseQuery{Code: 0, Height: state.Height(), Value: value}, nil
}
func (*App) ExtendVote(context.Context, *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	return nil, errors.New("vote extensions are disabled")
}
func (*App) VerifyVoteExtension(context.Context, *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
}
func (*App) ListSnapshots(context.Context, *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	return &abci.ResponseListSnapshots{}, nil
}
func (*App) OfferSnapshot(context.Context, *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	return &abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_REJECT}, nil
}
func (*App) LoadSnapshotChunk(context.Context, *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	return nil, errors.New("snapshots are disabled")
}
func (*App) ApplySnapshotChunk(context.Context, *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	return &abci.ResponseApplySnapshotChunk{Result: abci.ResponseApplySnapshotChunk_ABORT}, nil
}
