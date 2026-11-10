package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

func TestCollectorKeepsAcceptedOccupancySeparateFromFailuresAndRelay(t *testing.T) {
	var delta atomic.Int64
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/num_unconfirmed_txs" {
			t.Fatal("unexpected RPC path")
		}
		fmt.Fprint(w, `{"result":{"total":"2","total_bytes":"2575"}}`)
	}))
	defer rpc.Close()
	metrics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "cometbft_mempool_failed_txs 7\ncometbft_mempool_rejected_txs 3\ncometbft_mempool_recheck_times 1\n")
		fmt.Fprint(w, "cometbft_p2p_peer_send_bytes_total{peer_id=\"a\",chID=\"0x30\"} 100\n")
		fmt.Fprint(w, "cometbft_p2p_peer_send_bytes_total{peer_id=\"b\",chID=\"0x20\"} 800\n")
		fmt.Fprint(w, "cometbft_p2p_peer_receive_bytes_total{peer_id=\"b\",chID=\"0x30\"} 200\n")
		fmt.Fprintf(w, "pluto_comet_p2p_ingress_txs_total %d\npluto_comet_p2p_ingress_tx_bytes_total %d\n", 4+delta.Load(), 321+100*delta.Load())
		fmt.Fprint(w, "pluto_comet_checktx_socket_queue_seconds_count 5\npluto_comet_checktx_socket_queue_seconds_sum 0.025\n")
	}))
	defer metrics.Close()
	c, err := NewCollector(rpc.URL, metrics.URL, os.Getpid(), os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	s := c.Sample(context.Background())
	if s.SampleStartedUTC == "" || s.ObservedUTC == "" || s.SampleDurationNS < 0 || s.AcceptedMempoolCount == nil || *s.AcceptedMempoolCount != 2 || s.AcceptedMempoolBytes == nil || *s.AcceptedMempoolBytes != 2575 || s.FailedTxTotal == nil || *s.FailedTxTotal != 7 || s.ResourceRejectedTxTotal == nil || *s.ResourceRejectedTxTotal != 3 || s.RelaySendEnvelopeBytes == nil || *s.RelaySendEnvelopeBytes != 100 || s.RelayReceiveEnvelopeBytes == nil || *s.RelayReceiveEnvelopeBytes != 200 || s.P2PIngressTxTotal == nil || *s.P2PIngressTxTotal != 4 || s.P2PIngressTxBytesTotal == nil || *s.P2PIngressTxBytesTotal != 321 || s.CheckTxSocketQueueCount == nil || *s.CheckTxSocketQueueCount != 5 || s.CheckTxSocketQueueSeconds == nil || *s.CheckTxSocketQueueSeconds != 0.025 || s.Comet == nil || s.Application == nil || len(s.Unavailable) != 0 || s.DroppedSamples != 0 {
		t.Fatalf("incorrect sample: %+v", s)
	}
	if s.P2PIngressTxPerSecond != nil || s.P2PIngressBytesPerSecond != nil {
		t.Fatal("first sample cannot have a rate")
	}
	delta.Store(2)
	s = c.Sample(context.Background())
	if s.P2PIngressTxPerSecond == nil || *s.P2PIngressTxPerSecond <= 0 || s.P2PIngressBytesPerSecond == nil || *s.P2PIngressBytesPerSecond <= 0 {
		t.Fatalf("missing positive same-process P2P rates: %+v", s)
	}
	delta.Store(-3)
	s = c.Sample(context.Background())
	if s.P2PIngressTxPerSecond != nil || s.P2PIngressBytesPerSecond != nil {
		t.Fatalf("counter reset must make rates unavailable: %+v", s)
	}
	rpc.Close()
	s = c.Sample(context.Background())
	if s.AcceptedMempoolCount != nil || s.AcceptedMempoolBytes != nil || s.DroppedSamples != 1 || len(s.Unavailable) == 0 {
		t.Fatalf("missing RPC was converted to zero: %+v", s)
	}
}

func TestCollectorRejectsNonLocalEndpoints(t *testing.T) {
	for _, endpoint := range []string{"https://127.0.0.1:26657", "http://example.com:26657", "http://127.0.0.1:26657/path", "http://user:password@127.0.0.1:26657"} {
		if _, err := NewCollector(endpoint, "http://127.0.0.1:26660", 1, 1); err == nil {
			t.Fatalf("accepted endpoint %q", endpoint)
		}
	}
}
