# X09 — instrumentation ON/OFF and safety QA

Status: **PASS for correctness, capture and bounded safety QA**, with
observer overhead precision limited. The X09 plan was committed at
`f5f767a` before two fresh dedicated-process single-host loopback QA
networks. ON raw manifest:
`2430a42ae6d3cc53a6977a02ce5c747d69d825c79c665482164683ffe62cd5f3`.
OFF raw manifest:
`e6c84b998cc7be42ed70835a95bf3c9152a34b65735a3c80993a6fe6f9d44925`.
Both private ignored raw trees were frozen after processes exited.

Each arm had four app PIDs distinct from the harness, positive CPU/RSS,
complete per-validator block-index observations, a valid H CheckTx and
DeliverTx, and four-validator block ID/AppHash agreement. ON emitted
CheckTx events; OFF emitted none; recorder drop counters were zero in all
resource samples. The fixtures were independently generated, so their
AppHashes are **not** a matched cross-arm performance comparison. The
public `TestS05CheckTxTelemetryONOFFCorpus` ran with same-fixture ON/OFF
and passed identical response code/stage and AppHash for all corpus cases
including NEW/RECHECK. The new timestamps are only on recorder-ON code
paths and do not influence deterministic validation.

The frozen T05 eight-network matched recorder study was reverified
(`T05_VERIFY=OK`, four pairs; manifest
`60aa6e38bde61b6ac619c44f4ecaa1b681fce3e0e1271edb6916ef5502f467bc`).
Its ON−OFF RPC p50 effects by C/CP/P/H were +0.053/+0.046/−0.192/+0.077
ms, with mixed CPU/RSS signs, zero drops and equal correctness. One pair
per profile cannot prove negligible overhead at 20 tx/s. X06 profiled
primitive perturbation was likewise order-confounded. New extension
latency claims must disclose recorder mode and retain process CPU and
observer-drop QC.

The accepted S06 config remains 25 tx/s, 65,536 B/s, 16 outstanding and
60 s/run with loopback `127.0.0.1`, CPU/RSS/disk/observer guards.
`go test ./internal/experiment/load -run 'Safety|Guard|Abort' -count=1`
passed safety abort and kill-switch coverage. The historical Q04
socket-write fidelity audit rechecked 248 QC-passing networks and 2,976
steady one-second windows, with no >±5% target deviations. The fresh X09
one-transaction QA is not a generator-rate test; X10 must independently
check offered windows and abort/exclusion records. No safety cap changed.
