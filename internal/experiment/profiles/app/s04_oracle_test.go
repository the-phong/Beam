package researchv2app_test

import (
	"context"
	"encoding/hex"
	"path/filepath"
	"reflect"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/classical/state"
	"example.com/pluto/internal/experiment/corpus/profiles"
	"example.com/pluto/internal/experiment/profiles"
	"example.com/pluto/internal/experiment/profiles/app"
	"example.com/pluto/internal/experiment/profiles/store"
)

func TestS04CorpusABCIOracle(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "research", "corpus", "s04_public_corpus.json")
	c, err := researchv2corpus.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	genesis, _ := hex.DecodeString(c.GenesisHex)
	ctx := context.Background()
	for _, item := range c.Cases {
		t.Run(item.CaseID, func(t *testing.T) {
			raw, _ := hex.DecodeString(item.BytesHex)
			initial, err := researchv2.ParseGenesis(genesis, c.ChainID)
			if err != nil {
				t.Fatal(err)
			}
			state := initial
			if item.Precondition == "valid_nonce0_committed" {
				for _, candidate := range c.Cases {
					if candidate.Profile == item.Profile && candidate.ExpectedCode == researchv2.CodeOK {
						valid, _ := hex.DecodeString(candidate.BytesHex)
						var result researchv2.Result
						state, result, err = researchv2.EvaluateTrusted(state, item.Profile, valid)
						if err != nil || result.Code != researchv2.CodeOK {
							t.Fatalf("precondition: %v %+v", err, result)
						}
						state, err = state.AdvanceBlock()
						if err != nil {
							t.Fatal(err)
						}
						break
					}
				}
			}
			var visited []researchv2.Stage
			recorded, recordResult, err := researchv2.EvaluateTrustedRecorded(state, item.Profile, raw, func(s researchv2.Stage) { visited = append(visited, s) })
			if err != nil || recordResult.Code != item.ExpectedCode || recordResult.LastStage != item.Stage {
				t.Fatalf("recorder: %+v %v", recordResult, err)
			}
			plain, plainResult, err := researchv2.EvaluateTrusted(state, item.Profile, raw)
			if err != nil || plainResult != recordResult || plain.AppHash() != recorded.AppHash() || len(visited) == 0 || visited[len(visited)-1] != item.Stage {
				t.Fatalf("recorder changes decision: %+v %+v %v", plainResult, recordResult, err)
			}
			for _, stage := range visited {
				if (item.Stage == researchv2.StageStructural || item.Stage == researchv2.StageProfile || item.Stage == researchv2.StageChain || item.Stage == researchv2.StageAmount || item.Stage == researchv2.StageBinding) && (stage == researchv2.StageSecp || stage == researchv2.StageML) {
					t.Fatalf("early reject reached expensive verifier: %v", visited)
				}
			}
			app, err := researchv2app.New(genesis, c.ChainID, item.Profile)
			if err != nil {
				t.Fatal(err)
			}
			if item.Precondition == "valid_nonce0_committed" {
				var valid []byte
				for _, candidate := range c.Cases {
					if candidate.Profile == item.Profile && candidate.ExpectedCode == researchv2.CodeOK {
						valid, _ = hex.DecodeString(candidate.BytesHex)
						break
					}
				}
				if _, err := app.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{valid}}); err != nil {
					t.Fatal(err)
				}
				if _, err := app.Commit(ctx, &abci.RequestCommit{}); err != nil {
					t.Fatal(err)
				}
			}
			before := app.Committed()
			check, err := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw, Type: abci.CheckTxType_New})
			if err != nil || check.Code != uint32(item.ExpectedCode) || check.Log != string(item.Stage) {
				t.Fatalf("CheckTx: %+v %v", check, err)
			}
			recheck, err := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw, Type: abci.CheckTxType_Recheck})
			if err != nil || recheck.Code != check.Code || recheck.Log != check.Log {
				t.Fatalf("Recheck: %+v %v", recheck, err)
			}
			height := before.Height() + 1
			proposal, err := app.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: height, Txs: [][]byte{raw}})
			want := abci.ResponseProcessProposal_REJECT
			if item.ExpectedCode == researchv2.CodeOK {
				want = abci.ResponseProcessProposal_ACCEPT
			}
			if err != nil || proposal.Status != want {
				t.Fatalf("ProcessProposal: %+v %v", proposal, err)
			}
			if app.Committed().AppHash() != before.AppHash() {
				t.Fatal("proposal mutated state")
			}
			final, err := app.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: height, Txs: [][]byte{raw}})
			if err != nil || len(final.TxResults) != 1 || final.TxResults[0].Code != uint32(item.ExpectedCode) {
				t.Fatalf("FinalizeBlock: %+v %v", final, err)
			}
			if app.Committed().AppHash() != before.AppHash() {
				t.Fatal("finalize mutated committed state")
			}
			if _, err := app.Commit(ctx, &abci.RequestCommit{}); err != nil {
				t.Fatal(err)
			}
			after := app.Committed()
			if item.ExpectedCode != researchv2.CodeOK {
				if after.TotalSupply() != before.TotalSupply() || !reflect.DeepEqual(after.Base().Accounts(), before.Base().Accounts()) {
					t.Fatal("rejected input mutated accounts")
				}
			} else if after.AppHash() == before.AppHash() {
				t.Fatal("valid input not committed")
			}
		})
	}
}

func TestS04PersistentRestartAndUncertainCommit(t *testing.T) {
	c, err := researchv2corpus.Read(filepath.Join("..", "..", "..", "..", "research", "corpus", "s04_public_corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	genesis, _ := hex.DecodeString(c.GenesisHex)
	dir := filepath.Join(t.TempDir(), "store")
	s, err := researchv2store.Create(dir, genesis, c.ChainID, researchv2.ProfileH)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := researchv2store.OpenExisting(dir, c.ChainID, genesis, researchv2.ProfileC); err == nil {
		t.Fatal("wrong profile opened")
	}
	app, err := researchv2app.Open(dir, genesis, c.ChainID, researchv2.ProfileH)
	if err != nil {
		t.Fatal(err)
	}
	var raw []byte
	for _, item := range c.Cases {
		if item.CaseID == "p4-valid" {
			raw, _ = hex.DecodeString(item.BytesHex)
			break
		}
	}
	ctx := context.Background()
	if _, err := app.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{raw}}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Commit(ctx, &abci.RequestCommit{}); err != nil {
		t.Fatal(err)
	}
	before := app.Committed()
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
	app, err = researchv2app.Open(dir, genesis, c.ChainID, researchv2.ProfileH)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if app.Committed().AppHash() != before.AppHash() || app.Committed().Height() != 1 {
		t.Fatal("restart lost committed snapshot")
	}
}

func TestS04ProposalDuplicateAndRecheck(t *testing.T) {
	c, err := researchv2corpus.Read(filepath.Join("..", "..", "..", "..", "research", "corpus", "s04_public_corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	genesis, _ := hex.DecodeString(c.GenesisHex)
	ctx := context.Background()
	for _, profile := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		var raw []byte
		for _, item := range c.Cases {
			if item.Profile == profile && item.ExpectedCode == researchv2.CodeOK {
				raw, _ = hex.DecodeString(item.BytesHex)
				break
			}
		}
		app, err := researchv2app.New(genesis, c.ChainID, profile)
		if err != nil {
			t.Fatal(err)
		}
		initial := app.Committed().AppHash()
		prepared, err := app.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 100000, Txs: [][]byte{raw, raw}})
		if err != nil || len(prepared.Txs) != 1 {
			t.Fatalf("profile %d prepare dedup: %+v %v", profile, prepared, err)
		}
		proposal, err := app.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: [][]byte{raw, raw}})
		if err != nil || proposal.Status != abci.ResponseProcessProposal_REJECT {
			t.Fatalf("profile %d duplicate proposal: %+v %v", profile, proposal, err)
		}
		if app.Committed().AppHash() != initial {
			t.Fatal("duplicate proposal mutated state")
		}
		final, err := app.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{raw, raw}})
		if err != nil || len(final.TxResults) != 2 || final.TxResults[0].Code != 0 || final.TxResults[1].Code != uint32(researchv2.CodeInvalidNonce) {
			t.Fatalf("profile %d finalize dedup: %+v %v", profile, final, err)
		}
		if _, err := app.Commit(ctx, &abci.RequestCommit{}); err != nil {
			t.Fatal(err)
		}
		state := app.Committed()
		account, _ := state.Account(corpusSender(t, genesis, c.ChainID))
		if account.Balance != 93 || account.Nonce != 1 {
			t.Fatalf("profile %d duplicate execution: %+v", profile, account)
		}
		recheck, err := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw, Type: abci.CheckTxType_Recheck})
		if err != nil || recheck.Code != uint32(researchv2.CodeInvalidNonce) {
			t.Fatalf("profile %d post-commit recheck: %+v %v", profile, recheck, err)
		}
	}
}

func corpusSender(t *testing.T, genesis []byte, chainID string) b0state.Address {
	t.Helper()
	state, err := researchv2.ParseGenesis(genesis, chainID)
	if err != nil {
		t.Fatal(err)
	}
	for _, account := range state.Base().Accounts() {
		if account.Balance == 100 {
			return account.Address
		}
	}
	t.Fatal("funded sender missing")
	return b0state.Address{}
}
