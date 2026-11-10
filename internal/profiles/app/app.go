// Package researchv2app exposes one direct ABCI++ application path for all
// four research profiles, in memory or backed by one durable snapshot.
package researchv2app

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/profiles"
	"example.com/pluto/internal/profiles/genesis"
	"example.com/pluto/internal/profiles/store"
	"example.com/pluto/internal/telemetry"
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
	store       *researchv2store.Store
	recorder    atomic.Pointer[telemetry.Recorder]
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
func Open(dbDir string, genesis []byte, chainID string, profile researchv2.Profile) (*App, error) {
	a, err := New(genesis, chainID, profile)
	if err != nil {
		return nil, err
	}
	s, err := researchv2store.OpenExisting(dbDir, chainID, genesis, profile)
	if err != nil {
		return nil, err
	}
	a.store = s
	return a, nil
}
func (a *App) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store != nil {
		return a.store.Close()
	}
	return nil
}
func (a *App) stateLocked() (researchv2.State, error) {
	if a.store != nil {
		return a.store.Committed()
	}
	return a.committed, nil
}
func (a *App) pendingLocked() (bool, error) {
	if a.store != nil {
		_, pending, err := a.store.Pending()
		return pending, err
	}
	return a.pending != nil, nil
}
func (a *App) CommittedChecked() (researchv2.State, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stateLocked()
}

// Committed is retained for direct S03 callers. Node callbacks use the
// checked getter so an uncertain disk commit fails closed.
func (a *App) Committed() researchv2.State { s, _ := a.CommittedChecked(); return s }
func (a *App) Profile() researchv2.Profile { return a.profile }

// SetRecorder may be called while the node is running. Disabling it restores
// the unobserved CheckTx path; neither mode changes the validation result.
func (a *App) SetRecorder(r *telemetry.Recorder) { a.recorder.Store(r) }

func (a *App) Info(context.Context, *abci.RequestInfo) (*abci.ResponseInfo, error) {
	s, err := a.CommittedChecked()
	if err != nil {
		return nil, err
	}
	hash := s.AppHash()
	return &abci.ResponseInfo{Data: "Pluto research-v2 four-profile app", Version: version.Number, AppVersion: 4, LastBlockHeight: s.Height(), LastBlockAppHash: append([]byte(nil), hash[:]...)}, nil
}
func (a *App) InitChain(_ context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	if req == nil || req.InitialHeight != 1 || req.ChainId != a.chainID {
		return nil, ErrState
	}
	if !bytes.Equal(req.AppStateBytes, a.genesis) {
		binary, _, err := researchv2genesis.Decode(req.AppStateBytes, a.chainID)
		if err != nil || !bytes.Equal(binary, a.genesis) {
			return nil, ErrState
		}
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	s, err := a.stateLocked()
	if err != nil {
		return nil, err
	}
	pending, err := a.pendingLocked()
	if err != nil {
		return nil, err
	}
	if s.Height() != 0 || pending || s.AppHash() != a.initialHash {
		return nil, ErrState
	}
	h := s.AppHash()
	return &abci.ResponseInitChain{AppHash: append([]byte(nil), h[:]...)}, nil
}
func (a *App) CheckTx(_ context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	recorder := a.recorder.Load()
	var started time.Time
	if recorder != nil {
		started = time.Now()
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	var lockWait time.Duration
	if recorder != nil {
		lockWait = time.Since(started)
	}
	s, err := a.stateLocked()
	if err != nil {
		return nil, err
	}
	var raw []byte
	if req != nil {
		raw = req.Tx
	}
	var result researchv2.Result
	if recorder == nil {
		_, result, err = researchv2.EvaluateTrusted(s, a.profile, raw)
	} else {
		spans := make([]telemetry.Span, 0, 3)
		_, result, err = researchv2.EvaluateTrustedObserved(s, a.profile, raw, nil, func(stage researchv2.Stage, elapsed time.Duration) {
			spans = append(spans, telemetry.Span{Stage: string(stage), DurationNS: elapsed.Nanoseconds()})
		})
		if err == nil {
			checkType := "new"
			if req == nil {
				checkType = "unknown"
			} else if req.Type == abci.CheckTxType_Recheck {
				checkType = "recheck"
			}
			recorder.Record(telemetry.CheckTxEvent{
				Profile: uint8(a.profile), CheckType: checkType, WireBytes: len(raw),
				Code: uint32(result.Code), LastStage: string(result.LastStage),
				ServiceNS:     (time.Since(started) - lockWait).Nanoseconds(),
				AppLockWaitNS: lockWait.Nanoseconds(), Spans: spans,
			})
		}
	}
	if err != nil {
		return nil, err
	}
	return &abci.ResponseCheckTx{Code: uint32(result.Code), Log: string(result.LastStage), Codespace: codespace}, nil
}
func (a *App) nextHeight(s researchv2.State, height int64) error {
	if s.Height() == math.MaxInt64 || height != s.Height()+1 {
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
	s, err := a.stateLocked()
	if err != nil {
		return nil, err
	}
	pending, err := a.pendingLocked()
	if err != nil {
		return nil, err
	}
	if pending || a.nextHeight(s, req.Height) != nil {
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
	candidate := s
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
	s, err := a.stateLocked()
	if err != nil {
		return nil, err
	}
	pending, err := a.pendingLocked()
	if err != nil {
		return nil, err
	}
	if pending || a.nextHeight(s, req.Height) != nil {
		return nil, ErrState
	}
	if blockOverLimit(req.Txs) {
		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
	}
	candidate := s
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
	s, err := a.stateLocked()
	if err != nil {
		return nil, err
	}
	pending, err := a.pendingLocked()
	if err != nil {
		return nil, err
	}
	if pending || a.nextHeight(s, req.Height) != nil {
		return nil, ErrState
	}
	if blockOverLimit(req.Txs) {
		return nil, ErrState
	}
	if a.store != nil {
		results, hash, err := a.store.PrepareTrusted(req.Txs, false)
		if err != nil {
			return nil, err
		}
		resp := &abci.ResponseFinalizeBlock{AppHash: append([]byte(nil), hash[:]...), TxResults: make([]*abci.ExecTxResult, len(results))}
		for i, r := range results {
			resp.TxResults[i] = &abci.ExecTxResult{Code: uint32(r.Code), Log: string(r.LastStage), Codespace: codespace}
		}
		return resp, nil
	}
	candidate, results, err := researchv2.SimulateOrderedTrusted(s, a.profile, req.Txs)
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
	if a.store != nil {
		if err := a.store.Commit(); err != nil {
			return nil, err
		}
		return &abci.ResponseCommit{}, nil
	}
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
	s, err := a.stateLocked()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Path != "rv2/account/v1" || len(req.Data) != 20 || req.Height != 0 || req.Prove {
		return &abci.ResponseQuery{Code: 1, Codespace: codespace, Log: "unsupported_query"}, nil
	}
	var address b0state.Address
	copy(address[:], req.Data)
	account, found := s.Account(address)
	value := []byte{0}
	if found {
		value = make([]byte, 17)
		value[0] = 1
		binary.BigEndian.PutUint64(value[1:9], account.Balance)
		binary.BigEndian.PutUint64(value[9:17], account.Nonce)
	}
	return &abci.ResponseQuery{Code: 0, Height: s.Height(), Value: value}, nil
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
