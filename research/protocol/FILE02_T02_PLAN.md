# T02 size and codec audit plan

Status: locked before T02 audit, 2026-11-13. Reuse the immutable T01 valid
transaction raw only as input; do not modify it or run another network.
Measure all nine on-wire transaction lengths per profile and derive each
field's byte contribution from the accepted research-v2 codec. Check magic,
version, profile tag, chain length, total length and CP's canonical zero
padding. Verify CP equals H in wire length for every shared chain length.
Run the existing Go same-intent profile equivalence, parse/encode and
chain-boundary tests, plus public corpus verification.

The accepted genesis-only key policy means on-wire public-key bytes are
exactly zero for C/CP/P/H. An on-wire-key variant is unsupported and is not
silently simulated or added to the accepted protocol; document its absence
and the separate sensitivity-study requirement. Do not infer performance
from transaction length. T02 passes only if every T01 valid wire obeys the
formula, CP/H are exactly size-matched, Go codec/correctness tests pass, and
the public corpus verifies.
