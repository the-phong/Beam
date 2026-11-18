#!/usr/bin/env python3
"""Regenerate publication figures solely from QC-passing X evidence."""

import argparse
from io import BytesIO
import json
from pathlib import Path
import sys

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

import file02_common as common
import file02_x10 as x10
import file02_x13 as x13
import file02_x16_resources as x16
import file02_x11_historical as x11

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / "research/evidence/file02/mechanism/figures"
X10 = ROOT / "research/artifacts/file02/x10-20261115T051140Z-3d69d6eb"
X13 = ROOT / "research/artifacts/file02/x13-20261115T135447Z-a960b885"
RESOURCE = ROOT / "research/evidence/file02/mechanism/X16_RESOURCE_DECOMPOSITION.json"
CLASS = ROOT / "research/evidence/file02/mechanism/X11_CLASS_CONTRASTS.json"
COLORS = {"L": "#2a6f97", "B": "#2a9d8f", "I": "#e76f51"}


def png(fig):
    fig.tight_layout()
    stream = BytesIO()
    fig.savefig(stream, format="png", dpi=180, metadata={"Software": "Beam X17 matplotlib"})
    plt.close(fig)
    return stream.getvalue()


def make_figures():
    p10, p13 = common.read_json(X10 / "plan.json"), common.read_json(X13 / "plan.json")
    x10.validate_plan(X10, p10)
    x13.validate_plan(X13, p13)
    rows10, qc10, agg10 = x10.derive(X10, p10)
    rows13, qc13, agg13 = x13.derive(X13, p13)
    resource, classes = common.read_json(RESOURCE), common.read_json(CLASS)
    if (qc10["status"] != "PASS" or qc13["status"] != "PASS" or
            resource != x16.derive() or
            classes != x11.derive() or
            resource["raw_manifest_sha256"] != qc10["raw_manifest_sha256"] or
            agg13["knee_status"] != "NOT_OBSERVED_WITHIN_SAFE_RANGE"):
        raise ValueError("X17 input QC/provenance failed")
    output = {}
    fig, ax = plt.subplots(figsize=(8.2, 4.6))
    for i, arm in enumerate("LBI"):
        vals = [r["fixed_mean_upper_ms"] for r in rows10 if r["condition"] == arm]
        ci = x10.paired_ci(vals)
        mean = sum(vals)/len(vals)
        ax.errorbar(i, mean, yerr=[[mean-ci[0]], [ci[1]-mean]], fmt="o", capsize=5,
                    color=COLORS[arm], label=arm)
    ax.set_xticks(range(3), ["L: 6 valid/s", "B: +14 valid/s", "I: +14 bad-ML/s"])
    ax.set_ylabel("Fixed H first-all-four observation upper bound (ms)")
    ax.set_title("X10: matched fixed legitimate service; 24 networks per arm")
    ax.text(.02, .03, "I−B +60.03 ms [39.51, 81.54]; all fixed H timely by 3 s",
            transform=ax.transAxes, fontsize=9)
    ax.grid(axis="y", alpha=.25)
    output["x10_service.png"] = png(fig)

    fig, ax = plt.subplots(figsize=(7.4, 4.5))
    rates = [20, 23, 25]
    means = [agg13["estimates"][str(rate)]["mean_upper_ms"] for rate in rates]
    lows = [agg13["estimates"][str(rate)]["ci95_mean_upper_ms"][0] for rate in rates]
    highs = [agg13["estimates"][str(rate)]["ci95_mean_upper_ms"][1] for rate in rates]
    ax.errorbar(rates, means, yerr=[[m-l for m,l in zip(means,lows)],
                                      [h-m for m,h in zip(means,highs)]],
                marker="o", capsize=5, color=COLORS["I"])
    ax.set_xticks(rates)
    ax.set_xlabel("Total offered H-sized transactions/s (six fixed valid/s)")
    ax.set_ylabel("Fixed H observation upper bound, mean ms; 95% CI")
    ax.set_title("X13: invalid-background safe-range sensitivity")
    ax.text(.02, .03, "12 matched blocks; knee NOT OBSERVED within safe range",
            transform=ax.transAxes, fontsize=9)
    ax.grid(alpha=.25)
    output["x13_rate_response.png"] = png(fig)

    fig, axes = plt.subplots(1, 2, figsize=(10.5, 4.3))
    for arm in "LBI":
        curve = resource["validator0_sample_index_curves"][arm]
        axes[0].plot([x["sample_index"] for x in curve],
                     [x["mean_mempool_bytes"]/1024 for x in curve],
                     color=COLORS[arm], label=arm)
        axes[1].plot([x["sample_index"] for x in curve],
                     [x["mean_comet_cpu_cumulative_s"] for x in curve],
                     color=COLORS[arm], label=arm)
    axes[0].set_ylabel("Validator-0 accepted mempool, mean KiB")
    axes[1].set_ylabel("Validator-0 Comet CPU delta, mean s")
    for ax in axes:
        ax.set_xlabel("Observer sample index (nominal ~1 s)")
        ax.grid(alpha=.25)
    axes[0].legend(title="Arm")
    fig.suptitle("X10: descriptive resource curves; 24 networks per arm")
    output["x10_resource_curves.png"] = png(fig)

    fig, ax = plt.subplots(figsize=(8.0, 4.6))
    cases = classes["cases"]
    labels = [c["case_id"] for c in cases]
    walls = [c["invalid_mean_checktx_wall_ns"]/1000 for c in cases]
    ax.bar(labels, walls, color=["#577590", "#43aa8b", "#f9c74f", "#f8961e", "#f3722c"])
    ax.set_ylabel("Invalid CheckTx application wall mean (µs/tx)")
    ax.set_title("Historical invalid classes; different networks")
    ax.text(.02, .94, "Stage CPU unavailable; bars are not paired CPU contrasts",
            transform=ax.transAxes, fontsize=9, va="top")
    ax.grid(axis="y", alpha=.25)
    output["x11_invalid_wall.png"] = png(fig)
    return output, {"schema": "pluto.file02.x17_figures.v1", "matplotlib": matplotlib.__version__,
        "x10_raw_manifest_sha256": qc10["raw_manifest_sha256"],
        "x13_raw_manifest_sha256": qc13["raw_manifest_sha256"],
        "x16_resource_sha256": common.digest(RESOURCE), "x11_class_sha256": common.digest(CLASS),
        "figures": {}}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("write", "check"))
    args = parser.parse_args()
    try:
        images, manifest = make_figures()
        if args.command == "write":
            OUT.mkdir()
        for name, data in images.items():
            path = OUT / name
            if args.command == "write":
                with path.open("xb") as stream:
                    stream.write(data)
            elif path.read_bytes() != data:
                raise ValueError(f"X17 figure bytes drift: {name}")
            manifest["figures"][name] = common.digest(path)
        if args.command == "write":
            common.write_once(OUT / "manifest.json", json.dumps(manifest, indent=2, sort_keys=True)+"\n")
        elif common.read_json(OUT / "manifest.json") != manifest:
            raise ValueError("X17 figure manifest drift")
        print(f"X17_FIGURES_{args.command.upper()}=OK n={len(images)}")
        return 0
    except (OSError, ValueError, KeyError, TypeError) as exc:
        print(f"X17_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
