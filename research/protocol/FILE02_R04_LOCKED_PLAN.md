# R04 — CP/P/H cryptographic-path contrast

Status: **LOCKED BEFORE R04 MEASUREMENT**, 2026-11-13. R03 was disposed
as partial before this plan. Twelve fresh direct-ABCI process runs will
reuse the R02 four-profile harness, each with one new genesis, 240 distinct
valid nonce-zero CheckTx calls/profile after warm-up and the same logical
intents. The 16-byte `pluto-file02-r02` chain ID and canonical codec are
fixed. Order is counterbalanced with seed `20261113`; `GOMAXPROCS=1`,
55-second subprocess timeout, no network traffic. The S06 safety file is
unchanged. Each process supplies a paired CP, P and H observation.

Record per-run mean CheckTx service, stage secp and ML wall duration,
process CPU ns/tx, B/tx, allocs/tx and exact wire bytes. Report three
predeclared differences: P−CP, H−P and H−CP in mean service ns/tx, each
with a two-sided 95% paired percentile bootstrap over 12 processes,
20,000 draws, fixed seed `20261113`. Also report each profile's reached
stage set and ML verification mean, with a separate late bad-ML stage
oracle from the accepted T03 corpus. Full service, stage wall time and
process CPU are not interchangeable.

CP and H are exact 2,568-byte controls; CP has secp-only verification,
H has secp+ML. P is 2,504 bytes and has ML-only verification. The accepted
canonical P codec has no 64-byte padding option. P cannot be made exactly
byte-matched to CP or H without a protocol change, which this task forbids.
Thus P−CP changes both verifier type and wire length, while H−P adds secp
and 64 bytes. These are bounded, confounded contrasts; a crypto-specific
effect must be limited to directly observed stage spans or the exact-size
H−CP control. Do not call P−CP or H−P a pure PQC effect.

QC requires locked source/tree, host, toolchain and safety hashes;
12 separate process exits; fresh genesis per run; all four profile outputs
and 240 accepted records each; correct wire size and stage set; zero
recorder drops; complete CPU/allocation deltas and immutable raw hashes.
Failed runs stay preserved and prevent CI. The private session must contain
per-run JSON/CSV, QC, aggregate, figure-status and decision artifacts.
R04 may be PASS only for a bounded stage/cost comparison that explicitly
reports this canonical byte-match infeasibility and the resulting
confounding; it may not claim a pure P−CP or H−P causal effect.
