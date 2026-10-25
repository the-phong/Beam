# D-026: Explicit B0 Initialization and Complete-Data-Loss Detection

**Status: OWNER APPROVED for Phase 03D implementation, 2026-10-25; Controller source review remains required.** This document began as the design proposal and is retained as the decision rationale. The Owner subsequently authorized implementation of its explicit B0 initialization lifecycle. D-020–D-025 remain accepted; the D-024 key, record bytes and synced single-record write remain unchanged. No commit, push, merge or Phase 03E is authorized.

## Problem and current behavior

Today `init-b0` creates the CometBFT home and writes D-025 JSON/Base64 genesis transport and Pluto metadata. It does **not** create the B0 snapshot. `start` opens a missing B0 store if no Comet data path exists; CometBFT `InitChain` then creates the height-zero snapshot. If B0 is missing while Comet `state.db`, `blockstore.db` or `cs.wal` exists, `start` refuses an implicit reset. An incomplete B0 directory also fails closed. A complete loss of both application and Comet data under retained configuration is indistinguishable from a never-started initialized home at this gate. A process kill during the first `Info` callback was observed to leave Comet data with no B0 snapshot; production `start` refused that home. The case is safe against implicit reset, but requires operator disposition.

## Proposed contract

1. `init-b0` validates the canonical Genesis V2 binary and D-025 transport, creates the fresh home, and durably creates the canonical B0 height-zero snapshot using the existing D-024 `Create` operation. It reports success only after that synced record, genesis and metadata have been written and verified. The implementation must order and sync directory/configuration changes sufficiently to define what a returned success means; `pebble.Sync` alone does not prove directory-entry durability under power loss.
2. `start` requires the B0 directory and a complete, integrity-valid snapshot. It rejects a missing or incomplete snapshot even if all Comet data is also absent. It verifies the configured chain, genesis digest and height-zero account contents when A=0; for A>0 it keeps existing store and Comet handshake validation. It never creates or repairs B0 implicitly.
3. CometBFT `InitChain` retains D-025 canonical JSON/Base64 decoding and exact genesis comparison. It may return the persisted height-zero AppHash only when the snapshot is the identical Genesis V2 state. It does not create state. A conflicting request or a snapshot at A>0 fails closed.
4. Repeated `start` and repeated identical `InitChain` calls are idempotent while A=0. `init-b0` remains fresh-home-only; it does not silently retry an incomplete home. A returned initialization error leaves the home for inspection and explicit operator recovery.
5. The D-024 physical record, AppHash V2, signed transaction bytes, transaction validity and consensus rules do not change. No marker is added to the Pebble record.

## Crash and recovery windows

| Window | Proposed outcome |
|---|---|
| Before home creation or before B0 directory creation | `init-b0` returns or is interrupted; `start` refuses missing B0. Operator can remove or archive the incomplete disposable home and initialize a new home from the verified genesis. |
| After B0 directory creation, before complete record | `start` refuses the incomplete store. No automatic deletion or retry in place. |
| During record write/sync, or after sync before `init-b0` response | Reopen and validate the record. A complete matching H0 record is distinguishable from an incomplete one, but `init-b0` does not claim success without finishing and verifying the whole home. Explicit operator inspection chooses recovery; no blind reinitialization. Physical power-loss behavior still needs hardware evidence. |
| After B0 H0, before Comet/config metadata is complete | `start` refuses missing or conflicting metadata. The initialized B0 record is preserved for inspection. The creation order must avoid reporting `init-b0` success before all required files are durable. |
| After successful `init-b0`, before or during first Comet handshake | `start` finds H0; `Info` returns H0 and `InitChain` validates the identical genesis. A lost `InitChain` response can be retried without a second create. |
| After committed height advances, B0 missing or conflicting | `start` refuses. Existing B/C/A and AppHash replay checks remain authoritative at matching heights. |
| Both application and Comet data absent, configuration retained | `start` refuses because B0 H0 is required. This closes the current ambiguity without inferring history from an absent Comet store. |
| Entire home, including configuration, absent | `start` refuses a missing home. An explicit `init-b0` on a fresh path is a new operator action; local software cannot prove whether an erased former home once existed. |

## Alternative: current lifecycle plus independent initialization marker

Keep `InitChain` creation and write a separate marker when initialization completes. `start` could reject missing B0 when the marker exists. This requires defining marker placement, contents (chain/genesis identity), file and parent-directory fsync, and an atomic relationship with B0 creation. If the marker is written first, a crash can leave a marker with no snapshot and force fail-closed operator recovery; if written after the snapshot, a crash can leave a complete snapshot with no marker. Retrying `InitChain` then needs a rule for that ambiguous state. A marker stored in the same lost data directory disappears with B0 and does not detect complete data loss. A marker outside that directory can survive some data loss but adds an independent durability and backup domain; loss of marker and data remains undetectable. It also leaves normal first startup responsible for a durable genesis write.

**Owner decision:** explicit H0 creation in `init-b0` and a strict existing-snapshot requirement in `start`. It reuses the accepted D-024 record as the initialization fact and makes the existing explicit command the sole creation authority. The marker alternative adds a second persistent fact and ordering problem without solving total loss of all local evidence. Implementation and evidence still require independent Controller review.

## Decision amendment requested for Owner review

The approved decision supersedes **only the B0 initialization actor and order** in the accepted `B0_ABCI_VALIDITY_SPEC_V1.md` `Info`/`InitChain` paragraph: its requirement that `InitChain` *writes genesis height 0 durably* becomes a requirement that `init-b0` performs that D-024 write before successful initialization, and that `InitChain` validates the already persisted identical H0 state without writing. The associated D-025 wording that decoding precedes durable H0 initialization is clarified: canonical binary and JSON/Base64 transport are validated during `init-b0`, and the unchanged transport is validated again at `InitChain`. The architecture sequence is amended separately. Historical accepted wording is preserved with explicit supersession.

The following requirements would remain: D-025 exact JSON/Base64 transport and canonical Genesis V2 binary; `InitialHeight==1` and matching chain ID; identical H0 retry and conflicting-genesis rejection; D-024's absent-directory precondition for `Create`, single key `pluto/committed-state/v2`, `PLUTOS2!` record bytes and synced `Set`; AppHash V2; no Phase 02 V1 migration. Historical D-014 describes the separate Phase 02 V1 store and is not changed by D-026.

Compatibility must be explicit at approval. An existing B0 home with a complete, valid H0 snapshot can start and accept an identical Comet `InitChain` retry; a later valid snapshot continues through the existing Comet height/hash handshake. A pre-D-026 `init-b0` home that has configuration but **no** B0 snapshot would stop at `start`, even if it has never run; it requires a documented, operator-controlled disposition rather than silent in-place creation. An incomplete B0 directory remains fail closed, regardless of Comet data. A complete H0 record in an otherwise incomplete home also requires inspection of configuration and genesis identity before use. No automatic migration or reset is proposed, and preservation of validator identity during any operator recovery must be addressed before implementation.

## Compatibility and proposed verification

D-025 continues to define the bytes transported through Comet `genesis.json` and `InitChain`; the same canonical binary defines the H0 record and AppHash V2. The accepted D-024 physical layout remains byte-identical. Previously initialized B0 homes with a valid H0 or later snapshot should remain openable; old fresh homes with configuration but no B0 record would require explicit disposition rather than automatic genesis creation. Do not migrate or reset them silently. Verify the exact existing-home compatibility policy with the Owner before implementation.

Implementation tests must cover fresh `init-b0`, repeated start, identical/conflicting `InitChain`, each file and WAL/fsync interruption window, response loss, incomplete home, missing B0 with and without Comet data, matching-height replay, and an explicit operator recovery procedure. Process kills and simulated I/O errors test software behavior; physical power removal and device-cache loss require separate hardware evidence. Owner approval is limited to D-026 in Phase 03D and does not authorize Phase 03E.
