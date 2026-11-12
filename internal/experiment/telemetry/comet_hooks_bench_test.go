package telemetry

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// BenchmarkS05CometHooks estimates the isolated cost of the Prometheus calls
// used by the instrumented Comet build. It does not estimate node overhead.
func BenchmarkS05CometHooks(b *testing.B) {
	txs := prometheus.NewCounter(prometheus.CounterOpts{Name: "pluto_bench_p2p_txs"})
	bytes := prometheus.NewCounter(prometheus.CounterOpts{Name: "pluto_bench_p2p_bytes"})
	queue := prometheus.NewHistogram(prometheus.HistogramOpts{Name: "pluto_bench_queue", Buckets: []float64{0.000001, 0.00001, 0.0001, 0.001, 0.01, 0.1, 1, 10}})
	b.Run("P2PReceive", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			txs.Inc()
			bytes.Add(2575)
		}
	})
	b.Run("SocketQueueObserve", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			queue.Observe(0.00001)
		}
	})
}
