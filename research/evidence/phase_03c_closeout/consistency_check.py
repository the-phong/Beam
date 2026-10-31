"""Read-only governance checks for the Phase 03C Owner acceptance closeout."""

from pathlib import Path


ROOT = Path(__file__).resolve().parents[3]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


decisions = read("research/DECISIONS.md")
status = read("research/PHASE_STATUS.md")
plan = read("research/MASTER_PLAN.md")
architecture = read("research/ARCHITECTURE.md")
manifest = read("research/config/manifest.yaml")
agents = read("AGENTS.md")
readme = read("README.md")
report = read("research/PHASE_REPORTS/PHASE_03C_REPORT.md")
risk = read("research/evidence/phase_03c/RISK_AND_ACCEPTANCE_ADDENDUM.md")

checks = {
    "D-024 Owner acceptance and experimental scope": (
        "D-024 — Isolated B0 V2 complete-snapshot persistence (Accepted by Owner on"
        in decisions
        and "2026-10-24" in decisions.split("## Phase 03B dependency decision")[0]
        and "experimental B0 baseline" in decisions.split("## Phase 03B dependency decision")[0]
    ),
    "D-024 exact layout and synchronized write": all(
        token in decisions.split("## Phase 03B dependency decision")[0]
        for token in (
            "pluto/committed-state/v2",
            'ASCII("PLUTOS2!") || canonical_logical_state_v2 || AppHashV2[32]',
            "DB.Set(key, value, pebble.Sync)",
        )
    ),
    "Current status accepted but awaiting merge": (
        "COMPLETED, CONTROLLER ACCEPTED, OWNER ACCEPTED, AWAITING" in status
        and "OWNER-CONTROLLED MERGE OF PR #6" in status
        and "This acceptance is not a merge" in status
    ),
    "Current governance agrees on no Phase 03D authorization": all(
        "Phase 03D" in document and (
            "unauthorized" in document.lower()
            or "not authorized" in document.lower()
        )
        for document in (status, plan, architecture, agents, readme, report)
    ) and "phase_03d_implementation_authorized: false" in manifest,
    "Live runtime remains transaction-free": all(
        "transaction" in document.lower()
        for document in (status, architecture, readme, agents)
    ),
    "Risk addendum retained without rewritten status": (
        "R2 — state validation resource amplification" in risk
        and "PROPOSED — AWAITING EXPLICIT OWNER ACCEPTANCE" in risk
    ),
    "All M1–M12 remain NOT RUN": all(
        f"| M{i}." in risk for i in range(1, 13)
    ) and "Every B0 V2 cross-store case is\n**NOT RUN in Phase 03C**" in risk,
    "All G1–G8 remain future gates": all(
        f"| G{i} " in risk for i in range(1, 9)
    ) and "**NOT RUN / NOT SATISFIED in Phase 03C**" in risk,
    "R2 and future benchmarks retained": (
        "K=`1,64,4096,65536`" in risk
        and "O(K)" in risk
        and "No new benchmark was implemented or executed" in risk
    ),
}

for label, passed in checks.items():
    print(f'{"PASS" if passed else "FAIL"}: {label}')

if not all(checks.values()):
    raise SystemExit(1)
