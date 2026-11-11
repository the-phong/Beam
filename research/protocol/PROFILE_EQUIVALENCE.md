# S03 four-profile equivalence — implementation candidate

Source-path note (2026-11-11): the controlled-profile implementation now
lives at `internal/experiment/profiles/`; its app and testdata moved with it.
The S03 checkpoint text below retains its original source-path spelling.
See the [refactor mapping](../../docs/REFACTOR_MAPPING.md) for current paths.

Status: S03 source and direct-ABCI test candidate, 2026-11-09. Base: merged S02 `test-mainet` commit `2a3dcbceb52b996425d911dde87b691317f56efd`. Branch: `research/s03-controlled-profiles`. This report concerns the new `internal/profiles` and `internal/profiles/app` packages only. Accepted B0/B1 wire formats, accounts, stores and nodes are unchanged. Real four-validator correctness, durable restart and loaded P2P behavior are S04–S07 gates, not S03 claims.

## One logical task and one application path

Each profile transfers a positive amount from one genesis-registered 20-byte account ID to another at the exact next nonce. The account ID is a registry key independent of the profile. One canonical genesis registers a compressed secp256k1 public key and an ML-DSA-44 public key for **every** account. The public keys appear only in genesis; no profile carries them on the transaction wire. Duplicate public keys and duplicate/unsorted account IDs are rejected at genesis. The same B0 authenticated balance/nonce primitive enforces all four transfers, and the research-v2 AppHash binds the same registry and logical state under one new domain.

`researchv2app.App` selects one profile when constructed, then uses the **same** `CheckTx`, `PrepareProposal`, `ProcessProposal`, `FinalizeBlock`, `Commit` and account-query callbacks for all four profiles. `ProcessProposal` verifies independently of `CheckTx`; `FinalizeBlock` holds a pending state until `Commit`. A transaction marked for another profile fails before cryptographic verification. The S03 direct-ABCI app is in-memory, so S04 must add durable storage and node wiring before a real BFT run or restart claim.

## Canonical wire and authentication

All integers are unsigned big-endian. The exact wire is `PLUTOTX4` (8 bytes), version `4` (2), profile (1), chain-ID length (1), ASCII chain ID (1–50), sender (20), receiver (20), amount (8), nonce (8), then the profile proof. There are no optional fields, trailing bytes or alternate encodings. The common core is `68 + n` bytes for chain-ID length `n`.

| Profile | Required proof tail | Exact wire length | Authenticated condition |
|---|---|---:|---|
| C | 64-byte canonical low-S secp256k1 signature | `132 + n` | Registered secp key verifies. |
| CP | Same 64-byte secp signature, then **exactly 2,420 zero bytes** | `2,552 + n` | Registered secp key verifies; canonical padding is valid. |
| P | 2,420-byte ML-DSA-44 signature | `2,488 + n` | Registered ML key verifies. |
| H | 64-byte secp signature, then 2,420-byte ML-DSA-44 signature | `2,552 + n` | Both registered keys verify, secp first, ML second. |

For the public vector chain ID `rv2-golden-v1` (`n=13`), the observed sizes are C **145**, CP **2,565**, P **2,501**, H **2,565** bytes. CP/H equality is exact for every valid chain ID, not an average. CP padding is a single all-zero encoding; a changed byte or length is malformed. This makes CP an accepted transfer with the same wire length as H. C→CP changes accepted bytes and canonical-padding validation while keeping the secp condition; CP→H changes the proof work at equal bytes, but parser and structural-check differences remain and must be measured separately in S05 before any claim of isolated ML verification cost.

The signed digest is SHA-256 over a research-v2 domain plus the versioned canonical **logical intent** (profile, chain, sender, receiver, amount, nonce). The profile byte separates signatures across C/CP/P/H. Neither CP padding nor any proof bytes enter this digest. H's secp and ML proofs therefore authenticate the same digest; ML uses the fixed `PLUTO/RV2/MLDSA/V1` context. Secp signatures require nonzero canonical `r`, low `s` and registered-key verification. Key lookup occurs before proof verification, then state checks occur after all required proofs. A rejected input leaves the committed or candidate state unchanged. `Result.LastStage` records the last attempted validation stage; it is a deterministic stage label, not a timing metric.

## Verification and public vectors

`internal/profiles/testdata/public_vectors.json` contains one synthetic, valid signed transaction for each profile, public genesis bytes, wire size, SHA-256, signing digest, transaction ID and post-transfer AppHash. It contains no private key or signing seed. `go run ./cmd/profile-vectors` creates a fresh set of public vectors with ephemeral in-memory keys; randomness means it will not reproduce the checked-in bytes, while the checked-in file is independently reverified by `TestPublicGoldenVectors`.

`TestFourProfilesSameTransferAndExactSize` verifies canonical encode/parse, exact sizes, valid transfer, identical sender/receiver balances, nonce, total supply and AppHash. `TestProfileNegativeBoundariesAndReachedStages` checks malformed/trailing input, chain, zero amount, unknown sender, unregistered signing key, nonce, balance, bad secp/ML proof, bad CP padding and wrong profile for all applicable profiles. `TestOneABCIPathForFourProfiles` checks admission, proposal, pending finalization and commit via the same callbacks. These tests use fresh ephemeral accounts and direct ABCI calls. The S04 oracle must extend them to four real validators, cache/recheck and fault boundaries.
