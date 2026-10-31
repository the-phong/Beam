// Package app implements Pluto's Phase 02 transaction-free ABCI++ state machine.
package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/store"
	"example.com/pluto/internal/version"
)

const CodeUnsupportedTx uint32 = 1

// App serializes consensus and query access to committed and pending state.
// The Pebble database is owned by the App and must be closed after the ABCI
// server stops.
type App struct {
	mu        sync.RWMutex
	db        *store.DB
	committed *store.State
	pending   *store.State
}

var _ abci.Application = (*App)(nil)

func Open(dbDir string) (*App, error) {
	db, err := store.Open(dbDir)
	if err != nil {
		return nil, err
	}
	s, err := db.Load()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &App{db: db, committed: s}, nil
}

func (a *App) Close() error {
	return a.db.Close()
}

// Committed returns a copy of durable state, excluding any pending block.
func (a *App) Committed() *store.State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.committed == nil {
		return nil
	}
	copyState := *a.committed
	return &copyState
}

func (a *App) Info(context.Context, *abci.RequestInfo) (*abci.ResponseInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	resp := &abci.ResponseInfo{Data: "Pluto", Version: version.Number, AppVersion: 1}
	if a.committed != nil {
		resp.LastBlockHeight = a.committed.Height
		resp.LastBlockAppHash = append([]byte(nil), a.committed.AppHash[:]...)
	}
	return resp, nil
}

func (a *App) InitChain(_ context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	if req == nil || req.InitialHeight != 1 || len(req.AppStateBytes) != 0 {
		return nil, errors.New("Phase 02 requires initial height 1 and empty genesis app_state")
	}
	genesis, err := store.Genesis(req.ChainId)
	if err != nil {
		return nil, fmt.Errorf("invalid genesis chain ID: %w", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.committed != nil {
		if a.committed.Height != 0 || *a.committed != genesis {
			return nil, errors.New("InitChain conflicts with already committed application state")
		}
		return &abci.ResponseInitChain{AppHash: append([]byte(nil), genesis.AppHash[:]...)}, nil
	}
	if err := a.db.Save(genesis); err != nil {
		return nil, fmt.Errorf("persist genesis state: %w", err)
	}
	a.committed = &genesis
	// Nil validator and consensus updates retain the validated genesis values.
	return &abci.ResponseInitChain{AppHash: append([]byte(nil), genesis.AppHash[:]...)}, nil
}

func (a *App) CheckTx(context.Context, *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	return &abci.ResponseCheckTx{Code: CodeUnsupportedTx, Log: "transactions are unsupported in Phase 02"}, nil
}

func (a *App) PrepareProposal(_ context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if req == nil || a.committed == nil || req.Height != a.committed.Height+1 {
		return nil, errors.New("unexpected proposal height or uninitialized application")
	}
	// A correct proposer produces only empty blocks in this profile.
	return &abci.ResponsePrepareProposal{Txs: nil}, nil
}

func (a *App) ProcessProposal(_ context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	status := abci.ResponseProcessProposal_REJECT
	if req != nil && a.committed != nil && req.Height == a.committed.Height+1 && len(req.Txs) == 0 {
		status = abci.ResponseProcessProposal_ACCEPT
	}
	return &abci.ResponseProcessProposal{Status: status}, nil
}

func (a *App) FinalizeBlock(_ context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if req == nil || a.committed == nil {
		return nil, errors.New("cannot finalize without initialized application")
	}
	if a.pending != nil {
		return nil, errors.New("previous finalized block is not committed")
	}
	if len(req.Txs) != 0 {
		return nil, errors.New("cannot finalize unsupported transactions")
	}
	if req.Height != a.committed.Height+1 {
		return nil, fmt.Errorf("unexpected finalized height %d; expected %d", req.Height, a.committed.Height+1)
	}
	next, err := a.committed.Next()
	if err != nil {
		return nil, err
	}
	a.pending = &next
	return &abci.ResponseFinalizeBlock{AppHash: append([]byte(nil), next.AppHash[:]...)}, nil
}

func (a *App) Commit(context.Context, *abci.RequestCommit) (*abci.ResponseCommit, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pending == nil {
		return nil, errors.New("no finalized block to commit")
	}
	if err := a.db.Save(*a.pending); err != nil {
		return nil, fmt.Errorf("commit application state: %w", err)
	}
	a.committed = a.pending
	a.pending = nil
	return &abci.ResponseCommit{}, nil
}

func (a *App) Query(context.Context, *abci.RequestQuery) (*abci.ResponseQuery, error) {
	return &abci.ResponseQuery{Code: CodeUnsupportedTx, Log: "application queries are unsupported in Phase 02"}, nil
}

func (a *App) ExtendVote(context.Context, *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	return nil, errors.New("vote extensions are unsupported in Phase 02")
}

func (a *App) VerifyVoteExtension(context.Context, *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
}

func (a *App) ListSnapshots(context.Context, *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	return &abci.ResponseListSnapshots{}, nil
}

func (a *App) OfferSnapshot(context.Context, *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	return &abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_REJECT}, nil
}

func (a *App) LoadSnapshotChunk(context.Context, *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	return nil, errors.New("state sync snapshots are unsupported in Phase 02")
}

func (a *App) ApplySnapshotChunk(context.Context, *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	return &abci.ResponseApplySnapshotChunk{Result: abci.ResponseApplySnapshotChunk_ABORT}, nil
}
