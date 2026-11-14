# Q04 — actual offered-rate fidelity audit

Status: LOCKED SECONDARY ANALYSIS before derived outputs, 2026-11-14.
Audit all 248 QC-PASS RQ1b-related networks in the A01–A05/A08/A13 and
Q02 frozen sessions. Reverify each source manifest/QC. The source of actual
offered traffic is the load generator's successful **socket write** in
`result.json`, not CheckTx accepted count, eventual commit count or goodput.
Use `actual_offset_ns` from the monotonic run origin. Exclude warm-up;
partition the 12 steady seconds into `[1+s,2+s)` seconds for s=0..11.
Compute target and achieved total, fixed-legitimate and background tx/s and
encoded bytes/s from `plan.json` and actual writes for each one-second
window. `adversarial` and `benign_background` are background. Include
per-run IDs, plan/source manifest hashes, exact counts/bytes per window,
max absolute send-offset jitter and offered-versus-eventual-valid counts.

For every nonzero planned component and total, achieved one-second rate
must be within ±5% of its target; zero targets require zero actual sends.
The existing per-second >=90% rule is also checked. Any failed window
excludes that run from the rate-valid analysis and invalidates its matched
pair/block for inference; report it, never silently replace it. A pair
passes only if both arms pass. This audit does not alter already frozen raw
or redefine the registered 3-s service endpoint.
