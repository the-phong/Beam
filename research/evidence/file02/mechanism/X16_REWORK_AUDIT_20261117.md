# X16 rework audit — resource paths and probe overhead

Engineering disposition: **PASS for the bounded X16 checklist deliverable**;
independent Gate MECHANISM/SERVICE acceptance remains pending. The source
lock `47c44cc2cf9f10d6704de253d6133a61af920294` produced 48 fresh
four-validator single-host RPC-origin B/I networks. The independent checker
returned `X16_VERIFY=OK QC=PASS runs=48 pairs=24` on immutable private raw
manifest `53848abd502b0d922277852df1f59ace910238270777943bcb40335dd329be09`.
No run was excluded. Public derived aggregate and figures are indexed by
`X16_CONFIRMATORY_PUBLIC_MANIFEST.json`; raw, run homes and transaction
material remain private.

The new balanced stage-probe QA source `28fc05d` ran eight ON/OFF pairs on
fresh I networks, trace/recorder ON in both arms. Its independent checker
returned `X16_OVERHEAD_VERIFY=OK QC=PASS runs=16 pairs=8` on manifest
`9cd3e1fd7250ab0edb4b4a1b7b335fba5ff3153256de583315c01f7434ffb860`.
Each pair had identical signed H transactions and send offsets, genesis,
config and four-validator AppHash; 73 valid committed exactly once and 168
late-bad-ML were rejected per run. App/observer drops were zero, and each
run used 2,568-byte H at 20 tx/s, 51,360 B/s. The independent analyzer
checks source commit/tree, config hash, PID/start identity, linked trace,
oracle and raw manifest before reporting effects.

| Probe QA, ON−OFF | Mean | 95% paired block CI | Unit |
| --- | ---: | ---: | --- |
| Four-app CPU | +0.0489 | [+0.0134,+0.0823] | s/run |
| Four-Comet CPU | +0.0675 | [−0.0325,+0.1663] | s/run |
| Four-observer CPU | +0.0025 | [−0.0125,+0.0188] | s/run |
| Generator CPU | −0.0006 | [−0.0055,+0.0047] | s/run |
| Producer RPC median | −0.0046 | [−0.0201,+0.0105] | ms |
| Fixed-valid all-four observation upper bound | −21.68 | [−31.49,−12.24] | ms |

The probe increased application CPU in this QA. It also increased mean
structural stage **wall** time by 2.871 µs [2.679,3.056] and binding wall
time by 0.892 µs [0.606,1.167]. These are measurement perturbations and
work redistribution at the QA workload, not a universal clock-call
correction. Consequently the confirmatory stage values are labelled
**instrumented thread CPU**; unperturbed stage CPU and `CPU_crypto` remain
`null/NO_VALIDATED_PER_STAGE_PROBE_BIAS_CORRECTION`. Process CPU and stage
CPU are not summed into exact crypto shares.

An earlier QA session `x16-overhead-20261117T060846Z-1280b56c` was
stopped because its longer chain ID made H 2,580 bytes, violating the locked
51,360 B/s fixture. Its raw manifest
`d6f38452707aff536aac9cdad4e1e50c1e70a344ae0dadab278a9a2d15c4ec43`
is retained, independently verified and **excluded**. The prospective
correction was committed before the replacement session; no old raw was
edited or pooled.

The main B−I result is +0.7035 s/run app CPU [0.6466,0.7554], with
1,297,296 more validator-0 relay envelope bytes and 61,311 more peak
accepted-mempool bytes in B. Valid background enters accepted
mempool/relay/commit while invalid background stops after CheckTx; the
paired differences support a downstream-work interpretation. They do not
numerically isolate relay or commit CPU. Comet CPU CI includes zero. The
single additive transaction-wide queue scalar remains X07 `null`, and no exact bottleneck or
capacity knee was observed within the unchanged safe range. The new
fixed-valid upper-bound interval is not exact commit latency; the accepted
historical 3-second RQ1b negative endpoint remains unchanged. No DoS or
distributed-network claim follows.
