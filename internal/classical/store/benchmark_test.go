package b0store

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"example.com/pluto/internal/classical/state"
)

func benchmarkGenesis(k int) []byte {
	allocations := make([]b0state.Allocation, k)
	for i := range allocations {
		binary.BigEndian.PutUint32(allocations[i].Address[16:], uint32(i))
		allocations[i].Balance = 1_000_000
	}
	sort.Slice(allocations, func(i, j int) bool { return string(allocations[i].Address[:]) < string(allocations[j].Address[:]) })
	genesis, err := b0state.EncodeGenesis("pluto-b0-v1", allocations)
	if err != nil {
		panic(err)
	}
	return genesis
}

func BenchmarkB0SyncSnapshot(b *testing.B) {
	for _, k := range []int{1, 64, 4096, 65536} {
		b.Run(fmt.Sprintf("K=%d", k), func(b *testing.B) {
			dir := filepath.Join(b.TempDir(), "store")
			store, err := Create(dir, benchmarkGenesis(k), "pluto-b0-v1")
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				if _, _, err := store.PrepareTrusted(nil, false); err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				if err := store.Commit(); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(8+26+len("pluto-b0-v1")+36*k+32), "record-bytes/op")
		})
	}
}

// Enable explicitly for raw, independent sync latency samples. The test
// prints every observation; no percentile is inferred from a tiny sample.
func TestB0SyncLatencySamples(t *testing.T) {
	if os.Getenv("PLUTO_B0_SYNC_SAMPLES") != "1" {
		t.Skip("set PLUTO_B0_SYNC_SAMPLES=1 for diagnostic samples")
	}
	for _, k := range []int{1, 64, 4096, 65536} {
		store, err := Create(filepath.Join(t.TempDir(), "store"), benchmarkGenesis(k), "pluto-b0-v1")
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 30; i++ {
			if _, _, err := store.PrepareTrusted(nil, false); err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			if err := store.Commit(); err != nil {
				t.Fatal(err)
			}
			t.Logf("sync_sample k=%d trial=%d elapsed_ns=%d record_bytes=%d", k, i, time.Since(start).Nanoseconds(), 8+26+len("pluto-b0-v1")+36*k+32)
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
