# T02 — wire size, codec and key-placement audit

Date: 2026-11-13. The [locked audit plan](../../protocol/FILE02_T02_PLAN.md)
was committed at `14ecb99` before inspection. The
[`T02_SIZE_SUMMARY.json`](T02_SIZE_SUMMARY.json) was recomputed from the
frozen T01 raw after its SHA-256 manifest and four-network oracle were
verified. The nine accepted valid transactions in each profile have a
constant encoded size:

| Profile | n | Wire bytes min/mean/max | secp proof | ML proof | CP zero padding | on-wire key |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| C | 9 | 148/148/148 | 64 | 0 | 0 | 0 |
| CP | 9 | 2568/2568/2568 | 64 | 0 | 2420 | 0 |
| P | 9 | 2504/2504/2504 | 0 | 2420 | 0 | 0 |
| H | 9 | 2568/2568/2568 | 64 | 2420 | 0 | 0 |

Every wire also contains 12 envelope/header bytes, a 16-byte chain ID,
20-byte sender, 20-byte receiver, 8-byte amount and 8-byte nonce. The
auditor rejected wrong profile/length and nonzero CP padding in unit tests.
CP and H are **exactly byte-matched**. The Go same-intent valid-transfer,
parse/re-encode and chain-length-bound tests passed; `make corpus-s04-verify`
also exited 0. T01's independent four-validator oracle confirms the
sampled CP transfers are valid and committed, so padding is not merely
size-matched malformed data.

The accepted codec has genesis-only registered keys and **zero on-wire
public-key bytes** for every profile. The with-key variant is unsupported;
it would require a separate protocol/sensitivity decision. No transaction
length is used here as a performance or DoS finding.

**T02 PASS within the accepted genesis-only codec.** T03 is next; T04–T07
are not inferred from the T01/T02 results.
