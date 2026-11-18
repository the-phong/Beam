#!/usr/bin/env python3
"""Build a source-checked, disposable CometBFT copy for S05 telemetry.

The instrumented copy stays under ignored build/ and leaves the pinned module
cache and go.mod untouched. A temporary modfile selects it for one binary.
"""

import hashlib
import os
import pathlib
import shutil
import subprocess
import sys


VERSION = "v0.38.26"
EXPECTED = {
    "mempool/reactor.go": "f5644ce6a55fb97c84739d6293c4b551660ac639cd3be2d51a7f9532cb6475a0",
    "mempool/clist_mempool.go": "87c360cf4ff176893d0925ec7214203ab7f5f9bffd2a0376b343a1996ec5e458",
    "rpc/core/mempool.go": "23aa63b395a2622b2baf1045f7ebeb1f96ce24ef6b9aafed414d15009ba13313",
    "abci/client/socket_client.go": "e2aa962874c2226993a805617b32a6ac6acb998c80c019e439da6e467b2de426",
    "abci/client/client.go": "03f68615e6baf423547c503550a031c7edcfb822713bca7dac4134b57f64ca2c",
    "abci/server/socket_server.go": "5f47f1fa08e2e7a01b46d0f47b9afb53eca6e9443a421c2b76c381f9e8b17c35",
    "p2p/peer.go": "de9d3744d2aa1bca3edc43eafdba326ddf34a24c1c6813c0298a71ffb6009ff2",
    "p2p/conn/connection.go": "40e6548229f0b0adb484cfa6b56ce88e9d3215a9403eebad00f046096925af26",
}


def replace_once(source: str, old: str, new: str) -> str:
    if source.count(old) != 1:
        raise ValueError(f"expected exactly one patch anchor: {old[:80]!r}")
    return source.replace(old, new, 1)


def patch_reactor(source: str) -> str:
    source = replace_once(
        source,
        '\t"fmt"\n',
        '\t"fmt"\n\n\t"github.com/prometheus/client_golang/prometheus"\n\t"github.com/cometbft/cometbft/internal/plutotrace"\n',
    )
    source = replace_once(
        source,
        "var _ p2p.MsgBytesFilter = (*Reactor)(nil)\n",
        '''// Counts decoded transaction payloads at the P2P reactor boundary.
// This is distinct from Comet's encoded channel-envelope byte counters.
var plutoP2PIngressTxs = prometheus.NewCounter(prometheus.CounterOpts{
    Name: "pluto_comet_p2p_ingress_txs_total",
    Help: "Decoded transactions received by the mempool P2P reactor.",
})
var plutoP2PIngressBytes = prometheus.NewCounter(prometheus.CounterOpts{
    Name: "pluto_comet_p2p_ingress_tx_bytes_total",
    Help: "Decoded transaction payload bytes received by the mempool P2P reactor.",
})

func init() {
    prometheus.MustRegister(plutoP2PIngressTxs, plutoP2PIngressBytes)
}

var _ p2p.MsgBytesFilter = (*Reactor)(nil)
''',
    )
    source = replace_once(
        source,
        '\t\ttxInfo := TxInfo{SenderID: memR.ids.GetForPeer(e.Src)}\n',
        '''        for _, tx := range protoTxs {
            plutoP2PIngressTxs.Inc()
            plutoP2PIngressBytes.Add(float64(len(tx)))
            plutotrace.Emit("p2p_receive", tx, -1)
        }
        txInfo := TxInfo{SenderID: memR.ids.GetForPeer(e.Src)}
''',
    )
    source = replace_once(
        source,
        '\t\t\tif !success {\n\t\t\t\ttime.Sleep(PeerCatchupSleepIntervalMS * time.Millisecond)\n\t\t\t\tcontinue\n\t\t\t}\n',
        '\t\t\tif !success {\n\t\t\t\ttime.Sleep(PeerCatchupSleepIntervalMS * time.Millisecond)\n\t\t\t\tcontinue\n\t\t\t}\n\t\t\tplutotrace.Emit("p2p_send_handoff", memTx.tx, -1)\n',
    )
    return source


def patch_client(source: str) -> str:
    source = replace_once(source, '\t"sync"\n', '\t"sync"\n\t"time"\n')
    source = replace_once(
        source,
        '\tcb              func(*types.Response) // A single callback that may be set.\n',
        '\tcb              func(*types.Response) // A single callback that may be set.\n\tplutoQueuedAt time.Time // Monotonic timestamp used only by S05 socket-client overlay.\n',
    )
    return source


def patch_socket(source: str) -> str:
    source = replace_once(
        source,
        '\t"github.com/cometbft/cometbft/abci/types"\n',
        '\t"github.com/cometbft/cometbft/abci/types"\n\t"github.com/cometbft/cometbft/internal/plutotrace"\n\t"github.com/prometheus/client_golang/prometheus"\n',
    )
    source = replace_once(
        source,
        'const (\n\treqQueueSize    = 256',
        '''var plutoCheckTxSocketQueue = prometheus.NewHistogram(prometheus.HistogramOpts{
    Name: "pluto_comet_checktx_socket_queue_seconds",
    Help: "CheckTx request wait from socket-client enqueue attempt to writer dequeue, excluding ABCI service.",
    Buckets: []float64{0.000001, 0.00001, 0.0001, 0.001, 0.01, 0.1, 1, 10},
})

func init() {
    prometheus.MustRegister(plutoCheckTxSocketQueue)
}

const (
    reqQueueSize    = 256''',
    )
    source = replace_once(
        source,
        '\t\tcase reqres := <-cli.reqQueue:\n\t\t\t// N.B. We must enqueue before sending out the request',
        '''        case reqres := <-cli.reqQueue:
            if !reqres.plutoQueuedAt.IsZero() {
                plutoCheckTxSocketQueue.Observe(time.Since(reqres.plutoQueuedAt).Seconds())
                plutotrace.Emit("socket_dequeue", reqres.Request.GetCheckTx().Tx, -1)
            }
            // N.B. We must enqueue before sending out the request''',
    )
    source = replace_once(
        source,
        'func (cli *socketClient) queueRequest(ctx context.Context, req *types.Request) (*ReqRes, error) {\n\treqres := NewReqRes(req)\n',
        '''func (cli *socketClient) queueRequest(ctx context.Context, req *types.Request) (*ReqRes, error) {
    reqres := NewReqRes(req)
    if _, ok := req.Value.(*types.Request_CheckTx); ok {
        reqres.plutoQueuedAt = time.Now()
        plutotrace.Emit("socket_queue_attempt", req.GetCheckTx().Tx, -1)
    }
''',
    )
    return source


def patch_mempool(source: str) -> str:
    source = replace_once(source, '\t"github.com/cometbft/cometbft/config"\n',
        '\t"github.com/cometbft/cometbft/config"\n\t"github.com/cometbft/cometbft/internal/plutotrace"\n')
    source = replace_once(source, ') error {\n\tmem.updateMtx.RLock()\n',
        ') error {\n\tplutotrace.Emit("mempool_checktx_entry", tx, -1)\n\tmem.updateMtx.RLock()\n')
    source = replace_once(source, '\tif mem.preCheck != nil {\n\t\tif err := mem.preCheck(tx); err != nil {\n',
        '\tif mem.preCheck != nil {\n\t\tif err := mem.preCheck(tx); err != nil {\n\t\t\tplutotrace.Emit("precheck_reject", tx, -1)\n')
    source = replace_once(source, '\t// NOTE: proxyAppConn may error if tx buffer is full\n',
        '\tplutotrace.Emit("precheck_pass", tx, -1)\n\t// NOTE: proxyAppConn may error if tx buffer is full\n')
    source = replace_once(source, '\tif !mem.cache.Push(tx) { // if the transaction already exists in the cache\n',
        '\tif !mem.cache.Push(tx) { // if the transaction already exists in the cache\n\t\tplutotrace.Emit("cache_duplicate", tx, -1)\n')
    source = replace_once(source, '\treqRes, err := mem.proxyAppConn.CheckTxAsync(context.TODO(), &abci.RequestCheckTx{Tx: tx})\n',
        '\tplutotrace.Emit("abci_call", tx, -1)\n\treqRes, err := mem.proxyAppConn.CheckTxAsync(context.TODO(), &abci.RequestCheckTx{Tx: tx})\n')
    source = replace_once(source, '\t\tmem.resCbFirstTime(tx, txInfo, res)\n',
        '\t\tif check := res.GetCheckTx(); check != nil {\n\t\t\tplutotrace.Emit("checktx_callback", tx, int64(check.Code))\n\t\t}\n\t\tmem.resCbFirstTime(tx, txInfo, res)\n')
    source = replace_once(source, '\tmem.txsBytes.Add(int64(len(memTx.tx)))\n',
        '\tmem.txsBytes.Add(int64(len(memTx.tx)))\n\tplutotrace.Emit("mempool_accepted", memTx.tx, 0)\n')
    source = replace_once(source, '\t\tmem.txsBytes.Add(int64(-len(tx)))\n',
        '\t\tmem.txsBytes.Add(int64(-len(tx)))\n\t\tplutotrace.Emit("mempool_removed", tx, -1)\n')
    source = replace_once(source, '\t\t\tmem.metrics.FailedTxs.Add(1)\n',
        '\t\t\tplutotrace.Emit("mempool_app_reject", tx, int64(r.CheckTx.Code))\n\t\t\tmem.metrics.FailedTxs.Add(1)\n')
    source = replace_once(source, '\t\t// Remove committed tx from the mempool.\n',
        '\t\tplutotrace.Emit("committed_update", tx, int64(txResults[i].Code))\n\t\t// Remove committed tx from the mempool.\n')
    return source


def patch_rpc(source: str) -> str:
    source = replace_once(source, '\tabci "github.com/cometbft/cometbft/abci/types"\n',
        '\tabci "github.com/cometbft/cometbft/abci/types"\n\t"github.com/cometbft/cometbft/internal/plutotrace"\n')
    source = replace_once(source,
        'func (env *Environment) BroadcastTxSync(ctx *rpctypes.Context, tx types.Tx) (*ctypes.ResultBroadcastTx, error) {\n',
        'func (env *Environment) BroadcastTxSync(ctx *rpctypes.Context, tx types.Tx) (*ctypes.ResultBroadcastTx, error) {\n\tplutotrace.Emit("rpc_sync_entry", tx, -1)\n\tdefer plutotrace.Emit("rpc_sync_return", tx, -1)\n')
    return source


def patch_server(source: str) -> str:
    source = replace_once(source,
        '\t"github.com/cometbft/cometbft/abci/types"\n',
        '\t"github.com/cometbft/cometbft/abci/types"\n\t"github.com/cometbft/cometbft/internal/plutotrace"\n')
    source = replace_once(source,
        '\t\ts.appMtx.Lock()\n\t\tlocked = true\n\t\tresp, err := s.handleRequest(context.TODO(), req)\n',
        '''        if check := req.GetCheckTx(); check != nil {
            plutotrace.Emit("server_socket_read", check.Tx, -1)
            plutotrace.Emit("server_mutex_attempt", check.Tx, -1)
        }
        s.appMtx.Lock()
        locked = true
        if check := req.GetCheckTx(); check != nil {
            plutotrace.Emit("server_mutex_acquired", check.Tx, -1)
        }
        resp, err := s.handleRequest(context.TODO(), req)
        if check := req.GetCheckTx(); check != nil {
            code := int64(-1)
            if resp != nil && resp.GetCheckTx() != nil { code = int64(resp.GetCheckTx().Code) }
            plutotrace.Emit("server_app_return", check.Tx, code)
        }
''')
    source = replace_once(source,
        '\t\ts.appMtx.Unlock()\n\t\tlocked = false\n',
        '''            if check := req.GetCheckTx(); check != nil {
                plutotrace.Emit("server_response_handoff", check.Tx, -1)
            }
            s.appMtx.Unlock()
            locked = false
''')
    return source


def patch_peer(source: str) -> str:
    source = replace_once(source,
        '\t"github.com/cometbft/cometbft/libs/cmap"\n',
        '\t"github.com/cometbft/cometbft/internal/plutotrace"\n\t"github.com/cometbft/cometbft/libs/cmap"\n')
    source = replace_once(source,
        '\tmetricLabelValue := p.mlc.ValueToMetricLabel(msg)\n',
        '''    metricLabelValue := p.mlc.ValueToMetricLabel(msg)
    // The mempool Txs type exposes GetTxs without importing its package
    // (which would create a p2p import cycle).
    var plutoTxs [][]byte
    if txs, ok := msg.(interface{ GetTxs() [][]byte }); ok {
        plutoTxs = txs.GetTxs()
    }
''')
    source = replace_once(source,
        '\tres := sendFunc(chID, msgBytes)\n',
        '''    if chID == 0x30 {
        for _, tx := range plutoTxs {
            plutotrace.EmitEnvelopeMap(tx, msgBytes)
        }
    }
    res := sendFunc(chID, msgBytes)
''')
    return source


def patch_connection(source: str) -> str:
    source = replace_once(source,
        '\t"github.com/cometbft/cometbft/config"\n',
        '\t"github.com/cometbft/cometbft/config"\n\t"github.com/cometbft/cometbft/internal/plutotrace"\n')
    source = replace_once(source,
        '\tsendQueue     chan []byte\n',
        '\tsendQueue     chan []byte\n\tplutoTraceID uint64 // private overlay-only per-channel correlation token\n')
    source = replace_once(source,
        '\t\tsendQueue:               make(chan []byte, desc.SendQueueCapacity),\n',
        '\t\tsendQueue:               make(chan []byte, desc.SendQueueCapacity),\n'
        '\t\tplutoTraceID:            plutotrace.NewChannelID(),\n')
    source = replace_once(source,
        'func (ch *Channel) sendBytes(bytes []byte) bool {\n\tselect {\n\tcase ch.sendQueue <- bytes:\n',
        '''func (ch *Channel) sendBytes(bytes []byte) bool {
    if ch.desc.ID == 0x30 { plutotrace.EmitEnvelope("p2p_queue_attempt", bytes, ch.plutoTraceID) }
    select {
    case ch.sendQueue <- bytes:
        if ch.desc.ID == 0x30 { plutotrace.EmitEnvelope("p2p_queue_accepted", bytes, ch.plutoTraceID) }
''')
    source = replace_once(source,
        'func (ch *Channel) trySendBytes(bytes []byte) bool {\n\tselect {\n\tcase ch.sendQueue <- bytes:\n',
        '''func (ch *Channel) trySendBytes(bytes []byte) bool {
    if ch.desc.ID == 0x30 { plutotrace.EmitEnvelope("p2p_queue_attempt", bytes, ch.plutoTraceID) }
    select {
    case ch.sendQueue <- bytes:
        if ch.desc.ID == 0x30 { plutotrace.EmitEnvelope("p2p_queue_accepted", bytes, ch.plutoTraceID) }
''')
    source = replace_once(source,
        '\t\tch.sending = <-ch.sendQueue\n',
        '''        ch.sending = <-ch.sendQueue
        if ch.desc.ID == 0x30 { plutotrace.EmitEnvelope("p2p_queue_dequeue", ch.sending, ch.plutoTraceID) }
''')
    return source


TRACE_SOURCE = '''// Measurement-only, bounded per-validator transaction-hash trace.
// Built only in the disposable source-checked Comet overlay.
package plutotrace

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "sync/atomic"
    "time"
    "golang.org/x/sys/unix"
)

const maxEvents = 100000
var origin = time.Now()
var mu sync.Mutex
var output *os.File
var count uint64
var enabled atomic.Bool
var channelCounter uint64

type event struct {
    Schema string `json:"schema"`
    Event string `json:"event"`
    TxSHA256 string `json:"tx_sha256"`
    WireBytes int `json:"wire_bytes"`
    EnvelopeSHA256 string `json:"envelope_sha256,omitempty"`
    ChannelID uint64 `json:"channel_id,omitempty"`
    Code int64 `json:"code"`
    PID int `json:"pid"`
    Sequence uint64 `json:"sequence"`
    MonotonicNS int64 `json:"monotonic_ns"`
    HostMonotonicNS int64 `json:"host_monotonic_ns"`
    UTC string `json:"utc"`
}

func init() {
    path := os.Getenv("PLUTO_X07_TRACE_FILE")
    if path == "" { return }
    if !filepath.IsAbs(path) {
        fmt.Fprintln(os.Stderr, "X07 trace requires absolute private path")
        return
    }
    f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
    if err != nil {
        fmt.Fprintln(os.Stderr, "X07 trace open failed:", err)
        return
    }
    output = f
    enabled.Store(true)
}

func Emit(name string, tx []byte, code int64) {
    if !enabled.Load() { return }
    digest := sha256.Sum256(tx)
    emit(name, hex.EncodeToString(digest[:]), len(tx), code, "", 0)
}

func NewChannelID() uint64 { return atomic.AddUint64(&channelCounter, 1) }

// EmitEnvelopeMap correlates a wrapped mempool message with its tx without
// writing either payload. The peer supplies the unwrapped tx before Send.
func EmitEnvelopeMap(tx, envelope []byte) {
    if !enabled.Load() { return }
    txDigest := sha256.Sum256(tx)
    envelopeDigest := sha256.Sum256(envelope)
    emit("p2p_envelope_map", hex.EncodeToString(txDigest[:]), len(tx), -1,
        hex.EncodeToString(envelopeDigest[:]), 0)
}

// Connection queue events carry the envelope hash as both the trace key and
// envelope identifier. A separate peer marker supplies its transaction hash.
func EmitEnvelope(name string, envelope []byte, channelID uint64) {
    if !enabled.Load() { return }
    digest := sha256.Sum256(envelope)
    key := hex.EncodeToString(digest[:])
    emit(name, key, len(envelope), -1, key, channelID)
}

func emit(name, key string, wireBytes int, code int64, envelope string, channelID uint64) {
    if !enabled.Load() { return }
    now := time.Now()
    var hostClock unix.Timespec
    if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &hostClock); err != nil { return }
    mu.Lock()
    defer mu.Unlock()
    if output == nil { return }
    if count >= maxEvents { return }
    count++
    item := event{Schema:"pluto.file02.x07_comet_trace.v1", Event:name,
        TxSHA256:key, WireBytes:wireBytes, Code:code,
        EnvelopeSHA256:envelope, ChannelID:channelID,
        PID:os.Getpid(), Sequence:count, MonotonicNS:now.Sub(origin).Nanoseconds(),
        HostMonotonicNS:hostClock.Sec*1000000000+hostClock.Nsec,
        UTC:now.UTC().Format(time.RFC3339Nano)}
    raw, err := json.Marshal(item)
    if err != nil { return }
    raw = append(raw, '\\n')
    if _, err = output.Write(raw); err != nil {
        fmt.Fprintln(os.Stderr, "X07 trace write failed:", err)
        _ = output.Close()
        output = nil
        enabled.Store(false)
    }
}
'''


PATCHERS = {
    "mempool/reactor.go": patch_reactor,
    "mempool/clist_mempool.go": patch_mempool,
    "rpc/core/mempool.go": patch_rpc,
    "abci/client/socket_client.go": patch_socket,
    "abci/client/client.go": patch_client,
    "abci/server/socket_server.go": patch_server,
    "p2p/peer.go": patch_peer,
    "p2p/conn/connection.go": patch_connection,
}


def main() -> None:
    output = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "build/s05-comet-overlay").resolve()
    project = pathlib.Path(__file__).resolve().parents[2]
    module_cache = pathlib.Path(subprocess.check_output(["go", "env", "GOMODCACHE"], text=True).strip())
    module = module_cache / "github.com/cometbft" / f"cometbft@{VERSION}"
    if not module.is_dir():
        raise SystemExit(f"pinned CometBFT source missing: {module}")
    module_copy = output / "module"
    if not module_copy.exists():
        shutil.copytree(module, module_copy)
    for relative, expected_hash in EXPECTED.items():
        original = module / relative
        raw = original.read_bytes()
        actual = hashlib.sha256(raw).hexdigest()
        if actual != expected_hash:
            raise SystemExit(f"pinned CometBFT source changed at {relative}: {actual}")
        patched = PATCHERS[relative](raw.decode("utf-8"))
        target = module_copy / relative
        os.chmod(target, 0o644)
        target.write_text(patched)
    trace = module_copy / "internal" / "plutotrace" / "trace.go"
    os.chmod(trace.parent.parent, 0o755)
    trace.parent.mkdir(parents=True, exist_ok=True)
    trace.write_text(TRACE_SOURCE)
    modfile = project / "build" / "s05-comet.mod"
    sumfile = project / "build" / "s05-comet.sum"
    modfile.write_text((project / "go.mod").read_text() + f"\nreplace github.com/cometbft/cometbft => {module_copy}\n")
    shutil.copyfile(project / "go.sum", sumfile)
    print(modfile)


if __name__ == "__main__":
    main()
