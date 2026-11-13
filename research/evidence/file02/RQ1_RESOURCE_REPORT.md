# Checklist 02 section 4 — RQ1 resource report for Controller review

Date: 2026-11-13. Status: **engineering disposition complete for R01–R16;
Gate RQ1 awaits Controller/Owner review**. This report covers the
four-validator single-physical-host, RPC-origin stratum only. The accepted
RQ1 hypothesis in [RESEARCH_PROTOCOL.md](../../protocol/RESEARCH_PROTOCOL.md)
is unchanged: the primary effect is H minus exact-wire-size CP on per-run
mean valid, accepted application CheckTx service time at matched offered
tx/s. RQ1b legitimate-service degradation is a separate question and no
A/Q/D cases, ML training or attack workload were run here.

## Evidence and interpretation

The most direct bounded network contrast is [R06](R06_EQUAL_RATE_AUDIT.md):
12 matched independent blocks, 48 fresh networks, 20 valid tx/s/profile,
exact 2,568-B CP/H transfers (51,360 B/s), and zero invalid traffic.
H-minus-CP CheckTx service was **+81.47 µs/transaction**, two-sided
95% paired-block bootstrap CI **[+75.16, +87.26] µs**. All 48 network
arms passed rate, exact-once inclusion, four-validator AppHash and
recorder/observer QC. The CI lower bound exceeds zero, supporting the
accepted positive **application service-cost effect in this tested
single-host 20-tx/s stratum**. R06's checklist box remains open because
its explicit *below-knee* condition cannot be demonstrated: no knee was
observed within the unchanged safe range. This is not a claim of
sustainable service capacity or DoS.

[R07](R07_EQUAL_BYTES_AUDIT.md) supplies a distinct 12-block, 48-network
encoded-byte-rate control at 2,516/2,568/2,504/2,568 B/s for C/CP/P/H,
within the locked ±5% equality tolerance. CP and H also match exactly
at one valid tx/s. Their H-minus-CP CheckTx difference was
**+102.13 µs/transaction**, CI **[+66.67, +148.84] µs**. This
low-rate result supports the direction of the H–CP service-cost contrast;
it is not a capacity result. C and P have different tx/s at equal bytes/s,
so their cross-profile comparisons do not isolate a verifier effect.
R07's p99 inclusion estimate is `INSUFFICIENT_PRECISION`.

### Mechanism and cost decomposition

| Component | Evidence | What can be attributed |
| --- | --- | --- |
| Primitive cryptography | [R01](R01_PRIMITIVE_VERIFY_AUDIT.md): same-host valid secp256k1 104,762 ns/op and ML-DSA-44 18,294 ns/op; ML−secp −86,468 ns/op, CI [−88,307, −84,549] ns/op | These are isolated primitive implementation times, not full CheckTx or an H−CP effect. No calibrated cycles/op. |
| Application validation | [R02](R02_FULL_VALIDATION_AUDIT.md): direct-ABCI C/CP/P/H mean valid CheckTx 155.05/157.09/76.80/201.36 µs; H−CP +44.28 µs, CI [+40.15, +48.51] µs. [R04](R04_PQC_PATH_AUDIT.md) independently finds H−CP +44.53 µs, CI [+40.92, +48.53] µs. | H performs a second accepted verification path at the same CP wire size. Stage wall spans show added ML work; they are not stage CPU. Direct ABCI excludes Comet queues, relay and commit. |
| Encoded size | [R03](R03_SIZE_CONTROL_AUDIT.md): CP−C +2,420 B/tx with identical secp policy; structural parse +724 ns, CI [+635, +834] ns. Full CheckTx −1.38 µs, CI [−7.93, +5.23] µs. | The added bytes raise measured structural parse time, while the full direct service effect is inconclusive. P is 64 B shorter than CP/H and cannot be canonically padded under the accepted codec, so P−CP and H−P are confounded. |
| Mempool and Comet boundaries | [R08](R08_MEMPOOL_COUNT_AUDIT.md), [R09](R09_MEMPOOL_BYTES_AUDIT.md), [R10](R10_TX_SIZE_BOUND_AUDIT.md), [R14](R14_RECHECK_AUDIT.md) | Artificially constrained count/byte caps reject valid traffic before application verification as classified; native 8,192-B tx precheck rejects 8,193 B. Comet rechecks are internal validation, not new offered traffic. No per-request Comet CPU decomposition or exact queue residence is available. |
| Benign temporal pressure | [R15](R15_BENIGN_CAPACITY_AUDIT.md), [R16](R16_LEGITIMATE_BURST_AUDIT.md) | A 36-network valid-only ramp found no defensible knee through 25 tx/s; 48 benign waveform networks show transient sampled mempool/relay changes. R16 burst−sustained peak-byte CIs cross zero for all profiles, while ramp peaks are higher in matched blocks. These short runs do not establish steady sustainable throughput. |

R08's `size=1` arm rejected 76–77 of 81 offered valid transactions
per run; the baseline `size=128` accepted all 81. R09's 2,700-B arm
accepted 54–57 C versus 4–5 CP/P/H valid transfers per run before byte
full, while the 8-MiB baseline accepted 81. These are deliberately
changed mempool-limit controls. They show that size and count constraints
can dominate admission independently of cryptographic verification;
they do **not** estimate a natural capacity knee. R10's maximal canonical
valid H transaction was 2,602 B, well below the unchanged 8,192-B Comet
limit; a valid-near-8,192-B arm is
`NOT_FEASIBLE_WITHIN_ACCEPTED_CODEC`.

R15's formal 20/23/25 valid-tx/s ramp used the prospective
time-aligned rule and retained the historical T06 pilot's literal
window-biased flags unchanged. All four profile decisions are
`NOT_OBSERVED_WITHIN_SAFE_RANGE`. At every tested profile/rate cell,
the aligned timely fraction was 1.00 and eventual goodput equaled
offered tx/s; neither observation proves long-run sustainable throughput.
p95 values are 100-ms block-scan inclusion **upper bounds** (roughly
1.76–1.86 s across profile/rate means), not exact commit times. Queue
wait refers only to the Comet CheckTx **socket** interval; total mempool
and P2P residence are unavailable. R16's last-send-to-zero-sample
recovery is likewise a sampled upper bound, not an exact drain instant.

## R01–R16 disposition and Gate scope

| Cases | Checklist state | Reason for any open box |
| --- | --- | --- |
| R01, R02, R07, R08, R09, R14, R15, R16 | Checked after independent QC/raw verification | Pass only in the bounded scopes stated in each audit. R15 is a measured negative knee result, not an exact knee level. |
| R03, R04 | PARTIAL, unchecked | Matched peer propagation latency is unavailable; canonical P byte matching is unsupported, so P causal contrasts remain confounded. |
| R05 | UNSUPPORTED, unchecked | Reverse/early-reject verification policy arms are absent from the accepted protocol. |
| R06 | PARTIAL, unchecked | Primary H−CP network effect measured with positive CI; the checklist's below-knee predicate is unobserved. |
| R10 | PARTIAL, unchecked | Accepted codec cannot create a valid H transfer near 8,192 B; Comet precheck CPU and CI unavailable in the one-network boundary diagnostic. |
| R11, R12 | PARTIAL / UNSUPPORTED, unchecked | RPC-origin relay counters exist, but matched peer propagation timestamps/forwarding counts and a direct P2P-origin injector/oracle do not. |
| R13 | PARTIAL, unchecked | Existing four-validator regression verifies duplicate rejection/state safety, but matched repeated-versus-distinct valid/invalid streams and calibrated cache-hit/Comet CPU capture were not available in an R13 case. |

The Gate RQ1 checklist asks for size-versus-cryptography-versus-
application/consensus decomposition, explicit tx-rate/byte-rate
comparisons, run-level CIs and safe onset if measurable. The first
three are supplied within the measurement limits above. A defensible
safe capacity onset was **not observed** below the unraised 25-tx/s and
65,536-B/s caps. Several R cases remain explicitly PARTIAL or
UNSUPPORTED and their checklist boxes remain empty. The report is an
engineering claim for independent review, **not** Owner/Controller
acceptance of Gate RQ1. A full P2P-origin or distributed-network claim
would require separately verified instrumentation/topology and new
locked experiments. No DoS conclusion is supported by this valid-only
section.

The private per-case sessions contain locked plans, write-once raw
manifests and QC/aggregate/decision artifacts for every executed case.
The public audits link each case to its plan and manifest hashes without
publishing private keys, runtime homes or unsanitized logs. Historical
raw and accepted protocol semantics were not altered.

## Verification at handoff

| Command | Exit | Result |
| --- | ---: | --- |
| `make verify` | 0 | Go build/test/vet, formatting, module integrity and version passed at result commit `b63046c` before this report. |
| `python3 -m unittest discover -s research/tools -p 'test_file02*.py'` | 0 | 45 File 02 Python tests passed before the final R16 CSV integrity test was added; the full suite is rerun below for the final branch. |
| `make test-s05-telemetry` | 0 | Real four-validator C/CP/P/H telemetry regression PASS, 60.242 s. |
| `make test-s06-pilot` | 0 | Real isolated open-loop workload regression PASS, 80.996 s, including legacy burst and fixed legitimate-plus-invalid schedules. |
| `python3 research/tools/file02_r14.py verify research/artifacts/file02/r14-20261113T073256Z-6a4d335c` | 0 | 16 networks; frozen manifest `5378c732f2e368ac9bb7dd0b30b8efdbe0bf1617a4ed2e7367e79da38cdc71e0`. |
| `python3 research/tools/file02_r15.py verify research/artifacts/file02/r15-20261113T074602Z-751a6e1b` | 0 | 36 networks; frozen manifest `e94f4e413a57a3689016f507b54a5a722b95e03f6b14fbea690ac35bf0d54800`. |
| `python3 research/tools/file02_r16.py verify research/artifacts/file02/r16-20261113T122218Z-974de173` | 0 | 48 networks; frozen manifest `df7c8b5d34510f85faf577c50ff929f8e67a17e89d6acdc884734d265ff9e90b`; final strict CSV comparison passed after validator hardening. |

All executed R01–R16 cases have their own audit with plan/raw hashes
and verification result. The checklist boxes reflect case evidence, not
an automatic Gate decision. No protected branch was modified, no PR was
merged, and no raw evidence was committed.
