# X09 instrumentation equivalence QA plan

Status: locked before two new dedicated-process QA runs. Run the X05/X08
four-validator valid-H test once with application CheckTx recorder ON and
once OFF. The two processes use separate fresh networks and separately
generated private fixtures, so this is a correctness/capture check, not
a paired performance experiment. Both arms must have four separate app
PIDs, nonzero process CPU/RSS, zero recorder drops, valid CheckTx/DeliverTx,
one shared height/AppHash within their four validators, and complete
per-validator inclusion observations. ON must export CheckTx events; OFF
must export none. Freeze both private raw trees after process exit.

Also rerun the public ON/OFF correctness corpus test on the same fixture
and the bounded safety guard unit tests. Verify the frozen T05 eight-network
matched recorder-perturbation raw and its measured but low-precision CPU,
RSS and RPC differences. Check S06 config caps and generator-rate semantics
against the accepted artifact. These tests check measurement equivalence;
they do not establish negligible observer overhead at 20 tx/s or a
distributed-network result. Any unmeasured overhead remains disclosed.
