# Authentication profiles

| Implementation | Purpose | Authentication | Application |
| --- | --- | --- | --- |
| Legacy compatibility | Transaction-free Comet/ABCI baseline | No accepted transactions | `internal/legacy` through `plutod init` |
| Classical B0 | Classical transfer chain | Canonical secp256k1 | `internal/classical` through `plutod init-b0` or `init-b0-4v` |
| Hybrid B1 | Hybrid transfer chain | secp256k1 and ML-DSA-44 over one intent, with genesis-bound keys | `internal/hybrid` through `plutod init-b1-4v` |
| C | Controlled classical comparison | secp256k1 | `internal/experiment/profiles` |
| CP | Size-matched classical control | secp256k1, canonical zero padding | Same controlled application |
| P | Controlled post-quantum comparison | ML-DSA-44 | Same controlled application |
| H | Controlled hybrid comparison | secp256k1 then ML-DSA-44 | Same controlled application |

C/CP/P/H share a logical transfer, account registry, nonce/balance policy and one profile-selected ABCI application path; profile requirements and canonical wire bytes vary. In the tested File 02 fixture, C/CP/P/H are **148/2,568/2,504/2,568 bytes**. CP/H equality is exact at that chain ID. P is shorter and cannot be made an exact canonical size control without changing the accepted codec. The same profile-aware intent is authenticated by each required proof; invalid input fails closed, and a CheckTx success never substitutes for independent proposal validation.

B0 and B1 are separate application protocols, with their own genesis transport, stores and AppHash commitments. Neither is an alias for a controlled profile. Exact accepted rules are in the [B0 specifications](../research/specs/B0_CLASSICAL_TRANSACTION_PROTOCOL_V1.md), [B1 authentication specification](../research/specs/B1_HYBRID_AUTHENTICATION.md) and [controlled profile equivalence contract](../research/protocol/PROFILE_EQUIVALENCE.md). Public fixture bytes are under `tests/fixtures/`, `internal/hybrid/tx/testdata/` and `internal/experiment/profiles/testdata/`.
