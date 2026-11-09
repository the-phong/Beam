// Package researchv2app exposes one direct ABCI++ application path for all
// four research profiles. S04 will add durable storage and real node wiring.
package researchv2app

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/researchv2"
	"example.com/pluto/internal/version"
)

const (
	maxBlockTxs   = 6144
	maxBlockBytes = 1048576
	codespace     = "pluto.rv2.v1"
)

var ErrState = errors.New("invalid research-v2 ABCI state or callback order")

type App struct {
	mu          sync.RWMutex
	chainID     string
	profile     researchv2.Profile
	genesis     []byte
	initialHash [32]byte
	committed   researchv2.State
	pending     *researchv2.State
}

var _ abci.Application = (*App)(nil)

func New(genesis []byte, chainID string, profile researchv2.Profile) (*App, error) {
	if !profile.Valid() {
		return nil, ErrState
	}
	s, err := researchv2.ParseGenesis(genesis, chainID)
	if err != nil {
		return nil, err
	}
	return &App{chainID: chainID, profile: profile, genesis: bytes.Clone(genesis), initialHash: s.AppHash(), committed: s}, nil
}
func (a *App) Committed() researchv2.State { a.mu.RLock(); defer a.mu.RUnlock(); return a.committed }
func (a *App) Profile() researchv2.Profile { return a.profile }

func (a *App) Info(context.Context, *abci.RequestInfo) (*abci.ResponseInfo, error) {
	s := a.Committed()
	hash := s.AppHash()
	return &abci.ResponseInfo{Data: "Pluto research-v2 four-profile app", Version: version.Number, AppVersion: 4, LastBlockHeight: s.Height(), LastBlockAppHash: append([]byte(nil), hash[:]...)}, nil
}
func (a *App) InitChain(_ context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	if req == nil || req.InitialHeight != 1 || req.ChainId != a.chainID || !bytes.Equal(req.AppStateBytes, a.genesis) {
		return nil, ErrState
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.committed.Height() != 0 || a.pending != nil || a.committed.AppHash() != a.initialHash {
		return nil, ErrState
	}
	h := a.committed.AppHash()
	return &abci.ResponseInitChain{AppHash: append([]byte(nil), h[:]...)}, nil
}
func (a *App) CheckTx(_ context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var raw []byte
	if req != nil {
		raw = req.Tx
	}
	_, result, err := researchv2.EvaluateTrusted(a.committed, a.profile, raw)
	if err != nil {
		return nil, err
	}
	return &abci.ResponseCheckTx{Code: uint32(result.Code), Log: string(result.LastStage), Codespace: codespace}, nil
}
func (a *App) nextHeight(height int64) error {
	if a.committed.Height() == math.MaxInt64 || height != a.committed.Height()+1 {
		return ErrState
	}
	return nil
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
		return nil, ErrState
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.pending != nil || a.nextHeight(req.Height) != nil {
		return nil, ErrState
	}
	resp := &abci.ResponsePrepareProposal{}
	if req.MaxTxBytes <= 0 {
		return resp, nil
	}
	budget := uint64(req.MaxTxBytes)
	if budget > maxBlockBytes {
		budget = maxBlockBytes
	}
	candidate := a.committed
	var used uint64
	for _, raw := range req.Txs {
		if len(resp.Txs) == maxBlockTxs {
			break
		}
		if uint64(len(raw)) > budget-used {
			continue
		}
		next, result, err := researchv2.EvaluateTrusted(candidate, a.profile, raw)
		if err != nil {
			return nil, err
		}
		if result.Code != researchv2.CodeOK {
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
		return nil, ErrState
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.pending != nil || a.nextHeight(req.Height) != nil {
		return nil, ErrState
	}
	if blockOverLimit(req.Txs) {
		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
	}
	candidate := a.committed
	for _, raw := range req.Txs {
		next, result, err := researchv2.EvaluateTrusted(candidate, a.profile, raw)
		if err != nil {
			return nil, err
		}
		if result.Code != researchv2.CodeOK {
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}
		candidate = next
	}
	return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
}
func (a *App) FinalizeBlock(_ context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	if req == nil {
		return nil, ErrState
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pending != nil || a.nextHeight(req.Height) != nil {
		return nil, ErrState
	}
	if blockOverLimit(req.Txs) {
		return nil, ErrState
	}
	candidate, results, err := researchv2.SimulateOrderedTrusted(a.committed, a.profile, req.Txs)
	if err != nil {
		return nil, err
	}
	next, err := candidate.AdvanceBlock()
	if err != nil {
		return nil, err
	}
	a.pending = &next
	hash := next.AppHash()
	resp := &abci.ResponseFinalizeBlock{AppHash: append([]byte(nil), hash[:]...), TxResults: make([]*abci.ExecTxResult, len(results))}
	for i, r := range results {
		resp.TxResults[i] = &abci.ExecTxResult{Code: uint32(r.Code), Log: string(r.LastStage), Codespace: codespace}
	}
	return resp, nil
}
func (a *App) Commit(context.Context, *abci.RequestCommit) (*abci.ResponseCommit, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pending == nil {
		return nil, ErrState
	}
	a.committed = *a.pending
	a.pending = nil
	return &abci.ResponseCommit{}, nil
}
func (a *App) Query(_ context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if req == nil || req.Path != "rv2/account/v1" || len(req.Data) != 20 || req.Height != 0 || req.Prove {
		return &abci.ResponseQuery{Code: 1, Codespace: codespace, Log: "unsupported_query"}, nil
	}
	var address b0state.Address
	copy(address[:], req.Data)
	account, found := a.committed.Account(address)
	value := []byte{0}
	if found {
		value = make([]byte, 17)
		value[0] = 1
		binary.BigEndian.PutUint64(value[1:9], account.Balance)
		binary.BigEndian.PutUint64(value[9:17], account.Nonce)
	}
	return &abci.ResponseQuery{Code: 0, Height: a.committed.Height(), Value: value}, nil
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
