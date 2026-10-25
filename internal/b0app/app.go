// Package b0app connects the accepted B0 state machine to CometBFT ABCI++.
package b0app

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"slices"
	"sync"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/b0genesis"
	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/b0store"
	"example.com/pluto/internal/tx"
	"example.com/pluto/internal/version"
)

const (
	maxBlockTxs     = 6144
	maxBlockBytes   = 1048576
	resultCodespace = "pluto.b0.v1"
)

var ErrGenesisConflict = errors.New("B0 genesis conflicts with configured or committed state")

// App owns the B0 store and serializes ABCI calls. D-026 requires explicit
// H0 creation by init-b0; no ABCI callback creates a missing store.
type App struct {
	mu              sync.RWMutex
	dbDir           string
	chainID         string
	expectedGenesis []byte
	expectedState   b0state.State
	store           *b0store.Store
	closed          bool
}

var _ abci.Application = (*App)(nil)

func Open(dbDir, chainID string, canonicalGenesis []byte) (*App, error) {
	return OpenWithStoreOptions(dbDir, chainID, canonicalGenesis, b0store.Options{})
}

// OpenWithStoreOptions exposes Pebble filesystem fault injection to recovery
// tests. The production node calls Open with zero options.
func OpenWithStoreOptions(dbDir, chainID string, canonicalGenesis []byte, options b0store.Options) (*App, error) {
	expected, err := b0state.ParseGenesis(canonicalGenesis, chainID)
	if err != nil {
		return nil, err
	}
	a := &App{dbDir: dbDir, chainID: chainID, expectedGenesis: bytes.Clone(canonicalGenesis), expectedState: expected}
	if _, err := os.Stat(dbDir); err == nil {
		a.store, err = b0store.OpenExistingWithOptions(dbDir, chainID, options)
		if err != nil {
			return nil, err
		}
		state, err := a.store.Committed()
		if err != nil {
			_ = a.store.Close()
			return nil, err
		}
		// At height zero the whole logical snapshot must equal genesis. After
		// transfers, accounts can differ; supply remains conserved. CometBFT's
		// replay handshake checks the committed height/hash against its history.
		// Neither check alone proves complete historical genesis provenance.
		if (state.Height() == 0 && (state.AppHash() != expected.AppHash() ||
			!slices.Equal(state.Accounts(), expected.Accounts()))) ||
			state.TotalSupply() != expected.TotalSupply() {
			_ = a.store.Close()
			return nil, ErrGenesisConflict
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return a, nil
}

func (a *App) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return b0store.ErrClosed
	}
	a.closed = true
	if a.store != nil {
		return a.store.Close()
	}
	return nil
}

func (a *App) Committed() (b0state.State, bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return b0state.State{}, false, b0store.ErrClosed
	}
	if a.store == nil {
		return b0state.State{}, false, nil
	}
	state, err := a.store.Committed()
	return state, err == nil, err
}

func (a *App) Info(context.Context, *abci.RequestInfo) (*abci.ResponseInfo, error) {
	state, exists, err := a.Committed()
	if err != nil {
		return nil, err
	}
	resp := &abci.ResponseInfo{Data: "Pluto B0 V2", Version: version.Number, AppVersion: 2}
	if exists {
		resp.LastBlockHeight = state.Height()
		hash := state.AppHash()
		resp.LastBlockAppHash = append([]byte(nil), hash[:]...)
	}
	return resp, nil
}

func (a *App) InitChain(_ context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	if req == nil || req.InitialHeight != 1 || req.ChainId != a.chainID {
		return nil, ErrGenesisConflict
	}
	binary, state, err := b0genesis.Decode(req.AppStateBytes, req.ChainId)
	if err != nil || !bytes.Equal(binary, a.expectedGenesis) || state.AppHash() != a.expectedState.AppHash() {
		return nil, ErrGenesisConflict
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil, b0store.ErrClosed
	}
	if a.store == nil {
		return nil, b0store.ErrMissing
	}
	committed, err := a.store.Committed()
	if err != nil {
		return nil, err
	}
	if committed.Height() != 0 || committed.AppHash() != state.AppHash() ||
		!slices.Equal(committed.Accounts(), state.Accounts()) ||
		committed.TotalSupply() != state.TotalSupply() {
		return nil, ErrGenesisConflict
	}
	hash := state.AppHash()
	return &abci.ResponseInitChain{AppHash: append([]byte(nil), hash[:]...)}, nil
}

func resultLog(code tx.Code) string {
	switch code {
	case 0:
		return "ok"
	case 1:
		return "malformed"
	case 2:
		return "wrong_chain"
	case 3:
		return "invalid_key"
	case 4:
		return "invalid_signature"
	case 5:
		return "invalid_amount"
	case 6:
		return "invalid_nonce"
	case 7:
		return "insufficient_balance"
	case 8:
		return "balance_overflow"
	case 9:
		return "account_limit"
	case 10:
		return "block_limit"
	default:
		panic("unrecognized B0 result code")
	}
}

func (a *App) CheckTx(_ context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	state, exists, err := a.Committed()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("B0 application is not initialized")
	}
	var raw []byte
	if req != nil {
		raw = req.Tx
	}
	_, code, err := b0state.EvaluateTrusted(state, raw)
	if err != nil {
		return nil, err
	}
	return &abci.ResponseCheckTx{Code: uint32(code), Log: resultLog(code), Codespace: resultCodespace}, nil
}

// committedAtHeightLocked must be called with the application read or write
// lock held. Holding that lock through proposal simulation prevents a Commit
// from changing the reference snapshot during height validation.
func (a *App) committedAtHeightLocked(height int64) (b0state.State, error) {
	if a.closed || a.store == nil {
		return b0state.State{}, errors.New("B0 application is not initialized")
	}
	state, err := a.store.Committed()
	if err != nil {
		return b0state.State{}, err
	}
	if state.Height() == math.MaxInt64 || height != state.Height()+1 {
		return b0state.State{}, errors.New("unexpected B0 height")
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
		return nil, errors.New("unexpected B0 proposal height")
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
		next, code, err := b0state.EvaluateTrusted(candidate, raw)
		if err != nil {
			return nil, err
		}
		if code != tx.CodeOK {
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
		return nil, errors.New("missing B0 proposal")
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
		next, code, err := b0state.EvaluateTrusted(candidate, raw)
		if err != nil {
			return nil, err
		}
		if code != tx.CodeOK {
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}
		candidate = next
	}
	return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
}

func (a *App) FinalizeBlock(_ context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	if req == nil {
		return nil, errors.New("missing B0 finalized block")
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
		code := tx.Code(10)
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
	if a.closed || a.store == nil {
		return nil, errors.New("B0 application is not initialized")
	}
	if err := a.store.Commit(); err != nil {
		return nil, err
	}
	return &abci.ResponseCommit{}, nil
}

func (a *App) Query(_ context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	state, exists, err := a.Committed()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("B0 application is not initialized")
	}
	if req == nil || req.Path != "b0/account/v1" || len(req.Data) != 20 || req.Height != 0 || req.Prove {
		return &abci.ResponseQuery{Code: 1, Codespace: "pluto.query.v1", Log: "unsupported_query"}, nil
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
