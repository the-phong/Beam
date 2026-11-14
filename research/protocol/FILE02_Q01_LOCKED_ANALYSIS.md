# Q01 — matched valid-service audit, locked analysis

Status: locked before the Q01 reanalysis, 2026-11-14. This is a
**secondary quality audit** of immutable A01–A05/A08 and A13 raw, not a
new attack run. The six 20-tx/s H cases each have 12 matched benign/attack
pairs, six fixed legitimate H tx/s, 14 background tx/s and 51,360 B/s in
both arms. A13 has four matched pairs with the same fixed legitimate H
stream and 6/20/6 tx/s baseline/on/recovery schedule. A06 replay, A07
valid flood and A14 process fault do not estimate the invalid-background
RQ1b contrast and are excluded with those reasons.

For every pair, require source raw-manifest and derived-output integrity,
source QC PASS, equal profile H, genesis, seed, legitimate hashes and planned
send offsets, equal fixed offered rate, valid exact-once oracle and the
registered per-case topology and byte/rate schedule. The published
`paired_comparison.csv` has one row per fresh-network **pair**, not per
transaction. It records both source run IDs, seed, offered rates/bytes,
fixed eligible/timely counts, the benign-minus-attack fraction and source
manifest SHA-256. Pairs with either failed arm are excluded without
replacement and retain a reason; no missing endpoint becomes zero.

Retain each source case's preregistered paired-block percentile-bootstrap
95% CI: 20,000 resamples, seed 20261113, 12 pair units for A01–A05/A08;
A13 uses four balanced blocks and its on-phase CI. Do not pool across attack
families or treat the 72/36 transactions within a run as independent
replicates. The 3,000-ms first-all-four-observation endpoint is an upper
bound on inclusion, not an exact commit instant or production SLO. A zero
effect at that deadline does not prove an equivalence margin or absence of
other effects. This audit does not alter historical raw or accepted plans.
