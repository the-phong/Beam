# Phase 03D exploratory verification failures

The development loop included an early `make test-integration` run that exited **2** before the final source snapshot. Its output was observed in the agent session but was not captured to a durable log, so this record does not pretend to reproduce exact lost lines. Two test-harness problems caused the failure:

1. The recovery gate resumed the pending application `Commit` callback after killing the CometBFT child. The test was changed to retain and inspect the gated failure boundary without resuming an orphaned callback.
2. The RPC test helper used a one-second client timeout for `broadcast_tx_commit`, which can wait for block inclusion. Its test-only timeout was extended for that call.

Both changes were confined to the new Phase 03D integration tests. The source tree after these corrections was subjected to the full `run_verification.sh` suite; `verification_initial.log` and `verification_pre_real_verbose.log` preserve successive successful complete runs, and `verification.log` is the final run with an explicit verbose real-transaction test. Those later successes do not erase the exploratory failure. The missing original output is an evidence limitation.
