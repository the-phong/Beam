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
    "abci/client/socket_client.go": "e2aa962874c2226993a805617b32a6ac6acb998c80c019e439da6e467b2de426",
    "abci/client/client.go": "03f68615e6baf423547c503550a031c7edcfb822713bca7dac4134b57f64ca2c",
}


def replace_once(source: str, old: str, new: str) -> str:
    if source.count(old) != 1:
        raise ValueError(f"expected exactly one patch anchor: {old[:80]!r}")
    return source.replace(old, new, 1)


def patch_reactor(source: str) -> str:
    source = replace_once(
        source,
        '\t"fmt"\n',
        '\t"fmt"\n\n\t"github.com/prometheus/client_golang/prometheus"\n',
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
        }
        txInfo := TxInfo{SenderID: memR.ids.GetForPeer(e.Src)}
''',
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
        '\t"github.com/cometbft/cometbft/abci/types"\n\t"github.com/prometheus/client_golang/prometheus"\n',
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
    }
''',
    )
    return source


PATCHERS = {
    "mempool/reactor.go": patch_reactor,
    "abci/client/socket_client.go": patch_socket,
    "abci/client/client.go": patch_client,
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
    modfile = project / "build" / "s05-comet.mod"
    sumfile = project / "build" / "s05-comet.sum"
    modfile.write_text((project / "go.mod").read_text() + f"\nreplace github.com/cometbft/cometbft => {module_copy}\n")
    shutil.copyfile(project / "go.sum", sumfile)
    print(modfile)


if __name__ == "__main__":
    main()
