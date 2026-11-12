#!/usr/bin/env python3
"""Capture and reverify bounded T07 clock calibration on two private VMs."""

import argparse
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
import ipaddress
import json
from pathlib import Path
import re
import select
import subprocess
import sys
import time
from uuid import uuid4

import file02_common as common
import file02_t07_host as host_clock
from file02_common03 import ARTIFACTS, ROOT, git_value, private_session, require

SCHEMA = "pluto.file02.t07_multiguest.v1"
GUEST_PROBE = ROOT / "research/tools/file02_t07_guest_probe.py"
PROBES_PER_BOUNDARY = 32
STEP_LIMIT_NS = 10_000_000


def vmx_identity(path):
    content = Path(path).read_text()
    values = {}
    for name in ("displayName", "uuid.bios", "ethernet0.generatedAddress"):
        match = re.search(rf'^{re.escape(name)} = "([^"]+)"$', content, re.MULTILINE)
        require(match is not None, f"VMX lacks {name}")
        values[name] = match.group(1)
    values["vmx_sha256"] = common.digest(path)
    return values


def probe_interval(probe):
    c0, c3, r1, r2 = (probe[key] for key in
                       ("controller_before_wall_ns", "controller_after_wall_ns",
                        "remote_before_ns", "remote_after_ns"))
    m0, m3 = probe["controller_before_mono_ns"], probe["controller_after_mono_ns"]
    require(all(isinstance(x, int) and x > 0 for x in (c0, c3, r1, r2, m0, m3)) and
            c0 <= c3 and r1 <= r2 and m0 < m3, "malformed round-trip probe")
    lower, upper = r1-c3, r2-c0
    require(lower <= upper, "inverted offset interval")
    return {"lower_ns": lower, "upper_ns": upper, "width_ns": upper-lower}


def boundary(probes):
    require(len(probes) == PROBES_PER_BOUNDARY, "missing offset probes")
    intervals = [probe_interval(row) for row in probes]
    best = min(intervals, key=lambda row: row["width_ns"])
    return {"probe_count": len(probes), "narrowest_offset_interval_ns":
            [best["lower_ns"], best["upper_ns"]],
            "narrowest_interval_width_ns": best["width_ns"],
            "max_observed_interval_width_ns": max(row["width_ns"] for row in intervals)}


def survey_audit(raw, *, guest=True):
    require(raw.get("schema") == "pluto.file02.t07_guest_survey.v1", "guest survey schema")
    for key in ("warmup_start_ns", "warmup_end_ns", "steady_end_ns", "drain_end_ns"):
        require(isinstance(raw.get(key), int) and raw[key] > 0, f"missing {key}")
    require(raw["warmup_end_ns"]-raw["warmup_start_ns"] >= 990_000_000 and
            raw["steady_end_ns"]-raw["warmup_end_ns"] >= 9_900_000_000 and
            raw["drain_end_ns"]-raw["steady_end_ns"] >= 2_990_000_000,
            "warm-up, steady or drain too short")
    result = host_clock.summarize(raw["samples"], raw.get("throttle_before"),
                                  raw.get("throttle_after"))
    require(result["clock_continuous_within_10ms"], "clock discontinuity above 10 ms")
    del result["cross_host_offset_ns"]
    del result["cross_host_uncertainty_reason"]
    inventory = raw.get("inventory")
    require(isinstance(inventory, dict) and inventory.get("kernel") and
            inventory.get("cpu_model") and inventory.get("logical_cpus") and
            inventory.get("filesystem_type") and inventory.get("network_interfaces_mtu") and
            (not guest or inventory.get("machine_id_sha256")), "incomplete host inventory")
    return {"inventory": inventory, "clock": result,
            "warmup_ns": raw["warmup_end_ns"]-raw["warmup_start_ns"],
            "steady_ns": raw["steady_end_ns"]-raw["warmup_end_ns"],
            "drain_ns": raw["drain_end_ns"]-raw["steady_end_ns"]}


def controller_continuity(probe_rows, controller_survey):
    offsets = []
    for row in probe_rows:
        offsets.extend(((row["controller_before_mono_ns"],
                         row["controller_before_wall_ns"]-row["controller_before_mono_ns"]),
                        (row["controller_after_mono_ns"],
                         row["controller_after_wall_ns"]-row["controller_after_mono_ns"])))
    for row in controller_survey["samples"]:
        mid = (row["mono_before_ns"]+row["mono_after_ns"])//2
        offsets.append((mid, row["realtime_ns"]-mid))
    offsets.sort()
    require(all(a[0] < b[0] for a, b in zip(offsets, offsets[1:])),
            "controller monotonic order failed")
    maximum = max(abs(b[1]-a[1]) for a, b in zip(offsets, offsets[1:]))
    require(maximum <= STEP_LIMIT_NS, "controller wall-clock step above 10 ms")
    return maximum


def cross_guest(a, b):
    return [b[0]-a[1], b[1]-a[0]]


def audit(session):
    meta = common.read_json(session/"metadata.json")
    require(meta.get("schema") == SCHEMA and meta.get("scope") == "T_PREFLIGHT" and
            meta.get("source_commit") and meta.get("source_tree"), "T07 metadata")
    vmx = common.read_json(session/"vmx_identity.json")
    require(vmx["u1"]["uuid.bios"] != vmx["u2"]["uuid.bios"] and
            vmx["u1"]["ethernet0.generatedAddress"] != vmx["u2"]["ethernet0.generatedAddress"],
            "guest VM identities not distinct")
    controller = common.read_json(session/"controller_survey.json")
    controller_result = survey_audit(controller, guest=False)
    guests = {}
    all_probes = []
    for label in ("u1", "u2"):
        root = session/label
        survey = common.read_json(root/"survey.json")
        before = common.read_json(root/"before_probes.json")
        after = common.read_json(root/"after_probes.json")
        info = survey_audit(survey)
        info["before"] = boundary(before)
        info["after"] = boundary(after)
        guests[label] = info
        all_probes.extend(before+after)
    require(guests["u1"]["inventory"]["machine_id_sha256"] !=
            guests["u2"]["inventory"]["machine_id_sha256"],
            "guest machine IDs not distinct")
    controller_step = controller_continuity(all_probes, controller)
    cross = {}
    for phase in ("before", "after"):
        cross[phase] = cross_guest(
            guests["u1"][phase]["narrowest_offset_interval_ns"],
            guests["u2"][phase]["narrowest_offset_interval_ns"])
    cross["envelope_ns"] = [min(cross["before"][0], cross["after"][0]),
                            max(cross["before"][1], cross["after"][1])]
    return {"schema": "pluto.file02.t07_multiguest_summary.v1", "scope": "T_PREFLIGHT",
            "source_commit": meta["source_commit"], "source_tree": meta["source_tree"],
            "placement": "two_VMware_guests_one_physical_workstation_private_NAT",
            "controller": controller_result,
            "controller_max_wall_monotonic_step_ns": controller_step,
            "guests": guests, "u2_minus_u1_offset_intervals_ns": cross,
            "physical_host_independence": "UNAVAILABLE_SHARED_HYPERVISOR",
            "transaction_network_run": "NOT_RUN_T07_CLOCK_SURVEY_ONLY"}


class GuestSession:
    def __init__(self, label, user, ip, control, session):
        self.label, self.user, self.ip, self.control = label, user, ip, control
        self.remote = f"/tmp/pluto-file02-t07-{uuid4().hex}.py"
        self.target = f"{user}@{ip}"
        options = ["-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes",
                   "-o", "ConnectTimeout=5", "-o", f"ControlPath={control}"]
        self.ssh = ["ssh", "-T", *options]
        copied = subprocess.run(["scp", *options, str(GUEST_PROBE),
                                 f"{self.target}:{self.remote}"],
                                capture_output=True, timeout=20)
        require(copied.returncode == 0, f"{label} probe copy failed")
        self.error_file = (session/f"{label}-ssh.err").open("wb")
        self.process = subprocess.Popen([*self.ssh, self.target,
                                         f"python3 -u {self.remote} serve"],
                                        stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        stderr=self.error_file, text=True, bufsize=1)

    def command(self, name, timeout):
        require(self.process.poll() is None, f"{self.label} SSH probe exited")
        m0 = time.monotonic_ns()
        c0 = time.time_ns()
        self.process.stdin.write(name+"\n")
        self.process.stdin.flush()
        ready, _, _ = select.select([self.process.stdout], [], [], timeout)
        require(ready, f"{self.label} {name} timed out")
        line = self.process.stdout.readline()
        c3 = time.time_ns()
        m3 = time.monotonic_ns()
        require(line, f"{self.label} {name} empty reply")
        result = json.loads(line)
        if name == "clock":
            return {"controller_before_wall_ns": c0, "controller_after_wall_ns": c3,
                    "controller_before_mono_ns": m0, "controller_after_mono_ns": m3,
                    **result}
        return result

    def close(self):
        try:
            if self.process.poll() is None:
                self.process.stdin.write("quit\n")
                self.process.stdin.flush()
                self.process.wait(timeout=5)
        except (OSError, subprocess.TimeoutExpired):
            self.process.kill()
            self.process.wait(timeout=5)
        self.error_file.close()
        subprocess.run([*self.ssh, self.target, "rm", "-f", self.remote],
                       capture_output=True, timeout=10, check=False)


def local_survey():
    run = subprocess.run([sys.executable, str(GUEST_PROBE), "survey"],
                         capture_output=True, text=True, timeout=30)
    require(run.returncode == 0, "controller survey failed")
    return json.loads(run.stdout)


def capture(args):
    require(not git_value("status", "--porcelain"), "commit source before T07 calibration")
    require(args.u1_ip != args.u2_ip and all(ipaddress.ip_address(ip).is_private
            for ip in (args.u1_ip, args.u2_ip)), "guests need distinct private IPs")
    for path in (args.u1_socket, args.u2_socket):
        require(Path(path).is_socket(), "SSH control socket missing")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session = ARTIFACTS/("t07-multiguest-"+datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
                         +"-"+uuid4().hex[:8])
    session.mkdir(mode=0o700)
    common.write_once(session/"metadata.json", json.dumps({"schema": SCHEMA,
        "scope": "T_PREFLIGHT", "source_commit": source, "source_tree": tree,
        "guests": {"u1": {"user": "u1", "private_ip": args.u1_ip},
                   "u2": {"user": "u2", "private_ip": args.u2_ip}}}, indent=2)+"\n")
    common.write_once(session/"vmx_identity.json", json.dumps({
        "u1": vmx_identity(args.u1_vmx), "u2": vmx_identity(args.u2_vmx)}, indent=2)+"\n")
    sessions = []
    try:
        for label, ip, control in (("u1", args.u1_ip, args.u1_socket),
                                   ("u2", args.u2_ip, args.u2_socket)):
            (session/label).mkdir(mode=0o700)
            sessions.append(GuestSession(label, label, ip, control, session))
        before = {guest.label: [guest.command("clock", 5) for _ in range(PROBES_PER_BOUNDARY)]
                  for guest in sessions}
        with ThreadPoolExecutor(max_workers=3) as executor:
            futures = {guest.label: executor.submit(guest.command, "survey", 35) for guest in sessions}
            local = executor.submit(local_survey)
            surveys = {label: future.result() for label, future in futures.items()}
            controller = local.result()
        after = {guest.label: [guest.command("clock", 5) for _ in range(PROBES_PER_BOUNDARY)]
                 for guest in sessions}
        common.write_once(session/"controller_survey.json", json.dumps(controller, indent=2)+"\n")
        for label in ("u1", "u2"):
            for name, value in (("survey", surveys[label]), ("before_probes", before[label]),
                                ("after_probes", after[label])):
                common.write_once(session/label/f"{name}.json", json.dumps(value, indent=2)+"\n")
    finally:
        for guest in sessions:
            guest.close()
    summary = audit(session)
    common.write_once(session/"summary.json", json.dumps(summary, indent=2, sort_keys=True)+"\n")
    manifest = common.freeze_raw(session)
    print(f"T07_MULTIGUEST=PASS session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    require(common.read_json(session/"summary.json") == audit(session), "T07 summary differs from raw")
    print(f"T07_MULTIGUEST_VERIFY=OK manifest_sha256={manifest}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    run = sub.add_parser("capture")
    for label in ("u1", "u2"):
        run.add_argument(f"--{label}-ip", required=True)
        run.add_argument(f"--{label}-socket", required=True)
        run.add_argument(f"--{label}-vmx", required=True)
    check = sub.add_parser("verify")
    check.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "capture":
            capture(args)
        else:
            verify(args.session)
        return 0
    except (OSError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"T07_MULTIGUEST_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
