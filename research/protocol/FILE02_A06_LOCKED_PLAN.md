# A06 — replay and duplicate-cache mechanism, locked before execution

Status: LOCKED DESIGN, 2026-11-14. Four independent fixture blocks each
run four rotated conditions on fresh four-validator single-host networks:
`valid_distinct`, `valid_repeated`, `invalid_distinct` and
`invalid_repeated`. All use validator-0 loopback RPC, H profile, one-second
warm-up, four-second steady period, three-second drain and two-second
cooldown. Every steady second offers 20 2,568-byte writes (51,360 B/s),
including the same six fixed legitimate H transactions in all four arms.
The 14 background writes/s are either distinct valid H transfers, replay of
one signed valid H transfer, distinct signed H transfers with one altered
ECDSA signature byte, or replay of one such altered transaction. Actor B can
replay/modify a captured signed transaction; no signing key is assumed for
the repeated arms. All arms in a block share genesis and fixed legitimate
bytes/hashes/offsets, but use fresh network state.

The primary mechanism contrast is validator-0 application `CheckTx NEW`
count for invalid repeated minus invalid distinct background, one paired
difference per fixture block. The 95% interval is a 20,000-resample paired
block percentile bootstrap with seed 20261113. This assesses whether replay
is absorbed before application verification; a zero difference means no
observed saving of application calls in these conditions. Secondary outputs
are RPC `duplicate_cache` responses, code/stage, valid replay exact-once
commit, repeated and distinct hash counts, CheckTx service wall time,
recheck count, and whole-process CPU/queue envelopes. A cache response is
not counted as a crypto invocation, and process CPU is not labeled
per-request CPU.

The repeated arms may return `duplicate_cache`, a CheckTx application code
or an application invalid-nonce code after the first valid replay commits.
The outcome mix is **not** fixed in advance. The registered oracle accepts
only those classified outcomes, checks every socket write, and requires
all four validators to commit each distinct valid hash exactly once and no
bad-signature hash. The repeated-valid arm must commit its one background
hash once. Failed arms are retained and invalidate their fixture block;
there is no replacement. QC also requires exact rate/byte/fixed-stream
matching, all-four BlockID/AppHash and account-state agreement, complete
recorder/observer streams with no drops, raw integrity and no safety abort.

This is a mechanism case, not a service-degradation confirmation. It does
not infer a DoS, knee or distributed/P2P-origin result. The unchanged S06
caps remain 25 tx/s, 65,536 B/s, 16 outstanding and 60 seconds/run.
