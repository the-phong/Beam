package researchv2app_test

import (
	"context"
	"encoding/hex"
	"path/filepath"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/experiment/corpus/profiles"
	"example.com/pluto/internal/experiment/profiles"
	"example.com/pluto/internal/experiment/profiles/app"
	"example.com/pluto/internal/experiment/telemetry"
)

func TestS05CheckTxTelemetryONOFFCorpus(t *testing.T) {
	c, err := researchv2corpus.Read(filepath.Join("..", "..", "..", "..", "research", "corpus", "s04_public_corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := hex.DecodeString(c.GenesisHex)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, item := range c.Cases {
		t.Run(item.CaseID, func(t *testing.T) {
			on, err := researchv2app.New(genesis, c.ChainID, item.Profile)
			if err != nil {
				t.Fatal(err)
			}
			off, err := researchv2app.New(genesis, c.ChainID, item.Profile)
			if err != nil {
				t.Fatal(err)
			}
			if item.Precondition == "valid_nonce0_committed" {
				for _, candidate := range c.Cases {
					if candidate.Profile == item.Profile && candidate.ExpectedCode == researchv2.CodeOK {
						valid, _ := hex.DecodeString(candidate.BytesHex)
						for _, a := range []*researchv2app.App{on, off} {
							if _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{valid}}); err != nil {
								t.Fatal(err)
							}
							if _, err := a.Commit(ctx, &abci.RequestCommit{}); err != nil {
								t.Fatal(err)
							}
						}
						break
					}
				}
			}
			r := telemetry.NewRecorder(2)
			on.SetRecorder(r)
			raw, _ := hex.DecodeString(item.BytesHex)
			for _, checkType := range []abci.CheckTxType{abci.CheckTxType_New, abci.CheckTxType_Recheck} {
				req := &abci.RequestCheckTx{Tx: raw, Type: checkType}
				a, errA := on.CheckTx(ctx, req)
				b, errB := off.CheckTx(ctx, req)
				if errA != nil || errB != nil || a.Code != b.Code || a.Log != b.Log || a.Code != uint32(item.ExpectedCode) || on.Committed().AppHash() != off.Committed().AppHash() {
					t.Fatalf("ON/OFF divergence: %+v %+v %v %v", a, b, errA, errB)
				}
			}
			events, dropped := r.Drain()
			if len(events) != 2 || dropped != 0 || events[0].CheckType != "new" || events[1].CheckType != "recheck" {
				t.Fatalf("wrong events: %+v dropped=%d", events, dropped)
			}
			for _, e := range events {
				if e.Code != uint32(item.ExpectedCode) || e.LastStage != string(item.Stage) || e.WireBytes != len(raw) || e.ServiceNS < 0 || e.AppLockWaitNS < 0 {
					t.Fatalf("wrong measurement: %+v", e)
				}
				for _, span := range e.Spans {
					if span.DurationNS < 0 || (item.Stage == researchv2.StageStructural && span.Stage != string(researchv2.StageStructural)) {
						t.Fatalf("invalid span: %+v", span)
					}
				}
			}
		})
	}
}
