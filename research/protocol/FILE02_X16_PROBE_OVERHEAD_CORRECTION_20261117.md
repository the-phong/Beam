# X16 probe overhead QA correction — prospective lock

The first overhead QA session
`x16-overhead-20261117T060846Z-1280b56c` was stopped and quarantined.
Its immutable raw manifest is
`d6f38452707aff536aac9cdad4e1e50c1e70a344ae0dadab278a9a2d15c4ec43`.
The chain ID in its fresh fixture was longer than the earlier verified H
fixture, producing **2,580** instead of the preregistered **2,568** bytes per
transaction. This made the offered bytes/s 51,600 rather than 51,360. All
traffic remained below the unchanged safety caps, but the session is invalid
for the locked overhead estimand and supplies **no estimate**. Raw is retained
privately and will not be edited or pooled.

Before a replacement session, use the same 16-character chain ID
`pluto-file02-x16` for each newly generated fixture. Fresh random identities
still produce a new signed workload and genesis in each block. A pre-send
guard must require 241 planned events and exactly 2,568 wire bytes per event;
the independent verifier must also check those bytes in every raw run.
Retain the eight paired blocks, four OFF→ON and four ON→OFF, all other QC,
CI, topology and safety terms in
`FILE02_X16_PROBE_OVERHEAD_LOCKED_PLAN.md`. No failed run is replaced within
a session. This is a new prospectively source-locked QA session, not a
retrospective repair of the quarantined raw.
