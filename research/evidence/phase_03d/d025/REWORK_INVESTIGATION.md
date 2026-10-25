# R-025 exploratory test observations — 2026-10-24

These are observed outputs from bounded tests during rework. They are retained
separately from the final verification log; the initial failures were not
presented as passing evidence.

1. After initially extending `TestPinnedCometGenesisTransportAndB0Restart`
   to expect that removal of the entire disposable `data/` directory would
   start a fresh chain, the command
   `make build && go test -tags integration ./test/integration -run TestPinnedCometGenesisTransportAndB0Restart -timeout 120s -count=1 -v`
   exited **1**. The test reported `B0 node did not reach expected height`
   after approximately 40 seconds. This showed the expectation was wrong,
   but did not yet identify the startup error because child output was held
   in a temporary log.
2. After adding diagnostic child-output reporting, the command
   `go test -tags integration ./test/integration -run TestPinnedCometGenesisTransportAndB0Restart -timeout 120s -count=1 -v`
   exited **1** after approximately 17 seconds. The observed child output was:

   ```text
   open .../b0-home/data/priv_validator_state.json: no such file or directory
   CometBFT exited: exit status 1
   ```

   The test was corrected to assert this actual pinned-CometBFT outcome,
   verify that no B0 snapshot was created and disclose that the B0
   missing-store guard itself cannot distinguish total loss from first
   startup. No production logic was changed to hide the failure.
3. The corrected targeted integration command exited **0** and logged:

   ```text
   all local data lost under retained config: CometBFT refused startup because priv_validator_state.json was missing; no B0 snapshot was created
   ```

Only disposable test homes were used. These observations do not establish
physical failure recovery or a general fail-closed policy for all variants of
complete data loss.
