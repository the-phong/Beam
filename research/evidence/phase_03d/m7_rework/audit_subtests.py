"""Verify every required M7 subtest passed and its callback marker survived."""

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
ASYMMETRIC = (
    "before-proposal",
    "before-process",
    "before-finalize",
    "before-commit",
    "post-sync-commit",
)
CALLBACKS = (
    "info-initial",
    "init-pre-write",
    "init-post-sync",
    "check-new",
    "check-recheck",
    "info",
    "query",
)
EXPECTED = (
    {f"TestB0M7AsymmetricApplicationCrash/{mode}" for mode in ASYMMETRIC}
    | {f"TestB0M7OtherApplicationCallbacks/{mode}" for mode in CALLBACKS}
    | {"TestB0ApplicationProcessKilledWhileCometChildAlive"}
)

for label in ("normal", "race"):
    output = (ROOT / f"targeted_{label}_final.log").read_text()
    passed = {name for name in EXPECTED if f"--- PASS: {name} " in output}
    skipped = [line for line in output.splitlines() if "--- SKIP:" in line or "=== SKIP" in line]
    failed = [line for line in output.splitlines() if "--- FAIL:" in line or line == "FAIL"]
    markers = ROOT / "raw" / label
    for mode in ASYMMETRIC:
        marker = json.loads((markers / f"m7-{mode}-app.blocked").read_text())
        assert marker["callback"] == mode and marker["exists"]
    for mode in CALLBACKS:
        marker = json.loads((markers / f"m7-{mode}-blocked").read_text())
        assert marker["callback"] == mode
        assert marker["exists"] == (mode not in ("info-initial", "init-pre-write"))
    ack = (markers / "m7-post-commit-ack-post-commit.ack").read_text().strip()
    assert ack.startswith("check_code=0 tx_code=0 height=")
    final_pass = output.rstrip().splitlines()[-2] == "PASS"
    print(
        f"{label}: required={len(EXPECTED)} passed={len(passed)} "
        f"missing={sorted(EXPECTED - passed)} skipped={skipped} failed={failed} "
        f"callback_markers={len(ASYMMETRIC) + len(CALLBACKS)} "
        f"post_commit_ack={ack} final_pass={final_pass}"
    )
    if passed != EXPECTED or skipped or failed or not final_pass:
        raise SystemExit(1)
