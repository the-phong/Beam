# X14 — source and topology boundary

The accepted X extension uses **one physical host**, four isolated Comet
validator processes with four dedicated application PIDs, one load sender,
and loopback RPC into validator 0. The other validators can receive and
commit transactions through Comet gossip; that is a downstream relay path,
not a direct P2P-origin treatment arm. X10/X13 plans and raw per-run network
configuration bind this topology and endpoint. The S06 safety caps remain
25 tx/s, 65,536 B/s, 16 outstanding and 60 s/run.

| Requested scope or measurement | Current status | Evidence / condition for a future study |
| --- | --- | --- |
| RPC-origin validator 0, four validators on one host | AVAILABLE, used by X10/X13 | Locked X10/X13 plans, fresh-network oracle, raw manifests and verifier PASS. |
| Separate application process and process CPU per validator | AVAILABLE in new X sessions | X05 QA and X10/X13 app PID/resource series; not per-stage CPU. |
| Direct P2P-origin injection into any validator | UNSUPPORTED | R12 documents no controlled injector, paired origin oracle or source timestamp. Requires separate injector and QC plan. |
| Four direct sources / many peers with per-peer fairness | UNSUPPORTED | A12 documents no multi-source injector or per-peer telemetry. Four validator peers alone are not four sources. |
| Independent physical-host replication | UNAVAILABLE | Q06 is PARTIAL; VMware guests share one physical workstation. Requires another physical host, same locked workload and independent QA. |
| Exact cross-host latency | UNAVAILABLE | No valid clock uncertainty bound over a full cross-host run. Requires clock calibration and per-event uncertainty. |
| Distributed-network DoS or P2P-origin impact | NOT_ESTABLISHED | Requires a separate authorized topology/source study and measured matched legitimate-service degradation; no X run supplies this. |

The historical [R12](../R12_RPC_P2P_ORIGIN_UNSUPPORTED.md),
[A12](../A12_MANY_PEER_UNSUPPORTED.md) and
[Q06](../quality/Q06_REPLICATION_DISPOSITION.md) dispositions remain
unchanged. No X result is generalized to independent hardware, arbitrary
sources or internet conditions. D01–D13 and ML remain closed.
