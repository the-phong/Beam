package researchv2app_test

import (
	"context"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/experiment/profiles"
	"example.com/pluto/internal/experiment/profiles/app"
	"example.com/pluto/internal/experiment/profiles/testfixture"
	"example.com/pluto/internal/experiment/telemetry"
)

func BenchmarkS05CheckTx(b *testing.B) {
	f, err := researchv2test.New("s05-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	for _, profile := range []struct {
		name string
		p    researchv2.Profile
	}{{"CP", researchv2.ProfileCP}, {"H", researchv2.ProfileH}} {
		raw, err := f.Signed(profile.p, 1, 0)
		if err != nil {
			b.Fatal(err)
		}
		for _, enabled := range []bool{false, true} {
			name := profile.name + "/OFF"
			if enabled {
				name = profile.name + "/ON"
			}
			b.Run(name, func(b *testing.B) {
				a, err := researchv2app.New(f.Genesis, f.ChainID, profile.p)
				if err != nil {
					b.Fatal(err)
				}
				if enabled {
					a.SetRecorder(telemetry.NewRecorder(b.N + 1))
				}
				b.ReportAllocs()
				b.SetBytes(int64(len(raw)))
				ctx := context.Background()
				req := &abci.RequestCheckTx{Tx: raw}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					resp, err := a.CheckTx(ctx, req)
					if err != nil || resp.Code != 0 {
						b.Fatalf("CheckTx: %+v %v", resp, err)
					}
				}
			})
		}
	}
}
