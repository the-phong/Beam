package corpus

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/b0app"
	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/b0store"
	"example.com/pluto/internal/b1app"
	"example.com/pluto/internal/b1state"
	"example.com/pluto/internal/b1store"
)

// CheckOracle executes the accepted state evaluator and real ABCI CheckTx and
// ProcessProposal on disposable, independent application stores. It never
// derives an expected rejection code from the implementation under test.
func CheckOracle(c Corpus) error {
	if err := ValidateStructure(c); err != nil {
		return err
	}
	for _, p := range c.Profiles {
		if err := checkProfile(p); err != nil {
			return fmt.Errorf("%s: %w", p.Name, err)
		}
	}
	return nil
}

func checkProfile(p Profile) error {
	genesis, err := p.Genesis()
	if err != nil {
		return err
	}
	root, err := os.MkdirTemp("", "pluto-b0b1-oracle-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	ctx := context.Background()
	var check func([]byte) (uint32, error)
	var propose func([][]byte) (abci.ResponseProcessProposal_ProposalStatus, error)
	var closeApp func() error
	var after func() error
	if p.Name == "b0" {
		initial, err := b0state.ParseGenesis(genesis, p.ChainID)
		if err != nil {
			return err
		}
		store, err := b0store.Create(filepath.Join(root, "b0"), genesis, p.ChainID)
		if err != nil {
			return err
		}
		if err := store.Close(); err != nil {
			return err
		}
		app, err := b0app.Open(filepath.Join(root, "b0"), p.ChainID, genesis)
		if err != nil {
			return err
		}
		closeApp = app.Close
		check = func(raw []byte) (uint32, error) {
			r, e := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw})
			if e != nil {
				return 0, e
			}
			return r.Code, nil
		}
		propose = func(raws [][]byte) (abci.ResponseProcessProposal_ProposalStatus, error) {
			r, e := app.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: raws})
			if e != nil {
				return 0, e
			}
			return r.Status, nil
		}
		after = func() error {
			raw, _ := p.Valid[0].Bytes()
			next, code, e := b0state.EvaluateTrusted(initial, raw)
			if e != nil || code != 0 {
				return fmt.Errorf("valid B0 transition: code=%d err=%v", code, e)
			}
			if err := checkCasesB0(p, initial, next, check); err != nil {
				return err
			}
			if len(p.Valid) > SenderCount {
				state := initial
				for _, v := range p.Valid {
					wire, _ := v.Bytes()
					state, code, e = b0state.EvaluateTrusted(state, wire)
					if e != nil || code != 0 {
						return fmt.Errorf("sequenced B0 %s: code=%d err=%v", v.ID, code, e)
					}
				}
			}
			return nil
		}
	} else {
		initial, err := b1state.ParseGenesis(genesis, p.ChainID)
		if err != nil {
			return err
		}
		store, err := b1store.Create(filepath.Join(root, "b1"), genesis, p.ChainID)
		if err != nil {
			return err
		}
		if err := store.Close(); err != nil {
			return err
		}
		app, err := b1app.Open(filepath.Join(root, "b1"), p.ChainID, genesis)
		if err != nil {
			return err
		}
		closeApp = app.Close
		check = func(raw []byte) (uint32, error) {
			r, e := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw})
			if e != nil {
				return 0, e
			}
			return r.Code, nil
		}
		propose = func(raws [][]byte) (abci.ResponseProcessProposal_ProposalStatus, error) {
			r, e := app.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: raws})
			if e != nil {
				return 0, e
			}
			return r.Status, nil
		}
		after = func() error {
			raw, _ := p.Valid[0].Bytes()
			next, code, e := b1state.EvaluateTrusted(initial, raw)
			if e != nil || code != 0 {
				return fmt.Errorf("valid B1 transition: code=%d err=%v", code, e)
			}
			if err := checkCasesB1(p, initial, next, check); err != nil {
				return err
			}
			if len(p.Valid) > SenderCount {
				state := initial
				for _, v := range p.Valid {
					wire, _ := v.Bytes()
					state, code, e = b1state.EvaluateTrusted(state, wire)
					if e != nil || code != 0 {
						return fmt.Errorf("sequenced B1 %s: code=%d err=%v", v.ID, code, e)
					}
				}
			}
			return nil
		}
	}
	defer closeApp()
	for _, v := range p.Valid {
		if v.Nonce > 0 {
			continue // future nonces are verified by sequential evaluator replay
		}
		raw, _ := v.Bytes()
		code, err := check(raw)
		if err != nil || code != 0 {
			return fmt.Errorf("valid %s: CheckTx code=%d err=%v", v.ID, code, err)
		}
	}
	if err := after(); err != nil {
		return err
	}
	for _, q := range p.Proposals {
		var raws [][]byte
		for _, id := range q.TxIDs {
			raw, _ := p.Tx(id)
			raws = append(raws, raw)
		}
		result, err := propose(raws)
		if err != nil || result != abci.ResponseProcessProposal_REJECT {
			return fmt.Errorf("proposal %s: status=%s err=%v", q.ID, result, err)
		}
	}
	for _, gc := range p.GenesisCases {
		if p.Name != "b1" {
			return errors.New("unexpected B0 genesis case")
		}
		// Public genesis corruption must be rejected before a validator starts.
		raw, err := hex.DecodeString(gc.GenesisHex)
		if err != nil {
			return err
		}
		if _, err := b1state.ParseGenesis(raw, p.ChainID); err == nil {
			return fmt.Errorf("genesis %s accepted", gc.ID)
		}
	}
	return nil
}

func checkCasesB0(p Profile, initial, post b0state.State, check func([]byte) (uint32, error)) error {
	for _, cs := range p.Cases {
		raw, _ := cs.Bytes()
		state := initial
		if cs.Context == "after_V000" {
			state = post
		}
		next, code, err := b0state.EvaluateTrusted(state, raw)
		if err != nil || uint32(code) != cs.ExpectedCode {
			return fmt.Errorf("case %s evaluator expected %d got %d: %v", cs.ID, cs.ExpectedCode, code, err)
		}
		if code != 0 && next.AppHash() != state.AppHash() {
			return fmt.Errorf("case %s mutated rejected state", cs.ID)
		}
		if cs.Context == "initial" {
			actual, e := check(raw)
			if e != nil || actual != cs.ExpectedCode {
				return fmt.Errorf("case %s CheckTx expected %d got %d: %v", cs.ID, cs.ExpectedCode, actual, e)
			}
		}
	}
	return nil
}

func checkCasesB1(p Profile, initial, post b1state.State, check func([]byte) (uint32, error)) error {
	for _, cs := range p.Cases {
		raw, _ := cs.Bytes()
		state := initial
		if cs.Context == "after_V000" {
			state = post
		}
		next, code, err := b1state.EvaluateTrusted(state, raw)
		if err != nil || uint32(code) != cs.ExpectedCode {
			return fmt.Errorf("case %s evaluator expected %d got %d: %v", cs.ID, cs.ExpectedCode, code, err)
		}
		if code != 0 && next.AppHash() != state.AppHash() {
			return fmt.Errorf("case %s mutated rejected state", cs.ID)
		}
		if cs.Context == "initial" {
			actual, e := check(raw)
			if e != nil || actual != cs.ExpectedCode {
				return fmt.Errorf("case %s CheckTx expected %d got %d: %v", cs.ID, cs.ExpectedCode, actual, e)
			}
		}
	}
	return nil
}
