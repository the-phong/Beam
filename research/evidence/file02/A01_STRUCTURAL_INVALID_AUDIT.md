# A01 — structural-invalid workload audit

Date: 2026-11-14. The locked method and measurement source were committed at
`657eb20` before any A01 load. The private session is
`research/artifacts/file02/a01-20261113T170536Z-9bdf0a45`; its immutable
raw manifest SHA-256 is
`e8ea05c17bdb405e0339e895cd0f562a1c55ad7eae29745965fa54124f10c98a`.
The plan SHA-256 is
`692429f760b8ff4d80808c51ef4db992c78f7c2a68332a86063ac47175b7e624`.
Raw contains private genesis and transaction bytes and remains Git-ignored.

All 12 preregistered pairs completed on 24 fresh four-validator, single-host
networks using validator-0 loopback RPC. Each arm offered 20 H-sized
transactions/s, or 51,360 bytes/s, over 12 steady seconds. The 6 legitimate
transactions/s had the same bytes, hashes, offsets and genesis within a pair.
The benign arm added 14 valid H transactions/s; the attack arm added 14
distinct structural-invalid 2,568-byte strings/s. The attack strings covered
wrong magic, declared-length inconsistency and forbidden chain-ID encoding.
Actor A needs no registered key to construct them. All 168 invalid sends per
attack arm returned code 1 at `structural`, had no secp/ML recorder span and
never appeared in a block; all 73 valid transactions committed once.

QC and independent raw rederivation passed for all 24 arms. The registered
endpoint was inclusion of each of 72 fixed legitimate transactions observed
on all four validators by 3,000 ms after the socket write. Every arm achieved
72/72. All 12 paired benign-minus-attack differences were zero; mean effect
**0.000**, paired-bootstrap 95% CI **[0.000, 0.000]**. This is a negative
finding at the registered deadline and tested rate. The all-four observation
is an upper bound on inclusion time, not the exact commit instant.

| Descriptive metric | Benign | Structural-invalid arm |
| --- | ---: | ---: |
| Fixed legitimate inclusion upper bound, mean over 864 tx | 1,137.53 ms | 1,182.85 ms |
| Max fixed legitimate inclusion upper bound | 2,007.27 ms | 2,003.63 ms |
| Validator-0 accepted mempool bytes, median run peak | 86,028 B | 28,248 B |
| Validator-0 CheckTx socket queue, median run mean wait | 46.34 µs | 42.52 µs |
| Shared application-process CPU / 241 offered attempts, mean across runs | 8,734.44 µs | 5,684.65 µs |

The CPU quotient is a **whole-process envelope per offered attempt**: the
integration harness shares the application process, so it is not CPU
attributable to a structural-invalid transaction. That class-specific CPU/tx
is `UNAVAILABLE`; the directly observed invalid CheckTx wall-service mean is
7.25 µs/tx. The queue metric is specifically Comet-to-application CheckTx
socket wait, not the full transaction queue. Accepted mempool and CPU
differences reflect valid background traffic as well as the attack contrast;
the table does not isolate a causal per-class CPU effect. No run-specific
practical SLO or DoS threshold was selected after viewing data.

Commands: `python3 research/tools/file02_attack.py run A01` exited 0 with
`integration_exit=0` and `qc=PASS`;
`python3 research/tools/file02_attack.py verify
research/artifacts/file02/a01-20261113T170536Z-9bdf0a45` exited 0 with
`ATTACK_VERIFY=OK`. Raw plans, network binding, result JSON/CSV, QC,
aggregate, figure status and decision are in the private session. A01 is
**PASS for bounded attack-feasibility and negative service-endpoint test**.
It establishes neither ingress-overload impossibility nor a knee, DoS,
P2P-origin, multi-host or sustainable-throughput result.
