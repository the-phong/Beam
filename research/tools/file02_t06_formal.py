#!/usr/bin/env python3
"""Execute and reverify formal bounded T06 calibration on fresh networks."""

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import sys
from uuid import uuid4

import file02_common as common
from file02_common03 import ARTIFACTS, ROOT, git_value, private_session, require
from file02_t06 import PROFILES, RATES, validate_run

SCHEMA="pluto.file02.t06_formal.v1"


def audit(session, source, tree):
    rows=[]
    for rate in RATES:
        for profile in PROFILES:
            run=session/f"rate-{rate:02d}"/profile
            meta=common.read_json(run/"metadata.json")
            require(meta.get("case_id")=="T06", "formal T06 case identity")
            row=validate_run(run,source,tree,profile,rate,schema=SCHEMA,scope="T_PREFLIGHT")
            require(row["generator_cpu_core_fraction"]<0.25 and row["dropped_or_unsent_requests"]==0,
                    "formal T06 generator saturation or drop")
            rows.append(row)
    require(len(rows)==16 and len({r["run_id"] for r in rows})==16 and
            len({r["genesis_sha256"] for r in rows})==16,
            "formal T06 reused run or genesis")
    return {"schema":"pluto.file02.t06_formal_summary.v1","scope":"T_PREFLIGHT",
            "source_commit":source,"source_tree":tree,"fresh_networks":16,
            "testbed":"four_validators_single_host_loopback",
            "rate_sequence_tx_s":list(RATES),"generator_cpu_core_fraction_limit":0.25,
            "all_cells_achieved":True,"knee_status":"NOT_OBSERVED_WITHIN_SAFE_RANGE",
            "runs":rows}


def run():
    require(not git_value("status","--porcelain"),"commit source before formal T06")
    source,tree=git_value("rev-parse","HEAD"),git_value("rev-parse","HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True,exist_ok=True,mode=0o700)
    session=ARTIFACTS/("t06-formal-"+datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")+"-"+uuid4().hex[:8])
    session.mkdir(mode=0o700)
    with (session/"build.log").open("wb") as log:
        build=subprocess.run(["make","build","build-comet-s05"],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT)
    require(build.returncode==0,"formal T06 build failed; inspect private build.log")
    env=os.environ.copy()
    env.update(FILE02_T06_FORMAL_ROOT=str(session),FILE02_SOURCE_COMMIT=source,FILE02_SOURCE_TREE=tree)
    with (session/"integration.log").open("wb") as log:
        completed=subprocess.run(["go","test","-tags","integration fourval","./test/integration",
            "-run","^TestFile02T06FormalCalibration$","-timeout","1800s","-count=1","-v"],
            cwd=ROOT,env=env,stdout=log,stderr=subprocess.STDOUT,timeout=1900)
    require(completed.returncode==0,"formal T06 integration failed; inspect private integration.log")
    summary=audit(session,source,tree)
    common.write_once(session/"summary.json",json.dumps(summary,indent=2,sort_keys=True)+"\n")
    manifest=common.freeze_raw(session)
    print(f"T06_FORMAL=PASS networks=16 session={session} manifest_sha256={manifest}")


def verify(path):
    session=private_session(path)
    manifest=common.verify_raw(session)
    prior=common.read_json(session/"summary.json")
    require(prior.get("schema")=="pluto.file02.t06_formal_summary.v1" and
            audit(session,prior["source_commit"],prior["source_tree"])==prior,
            "formal T06 summary differs from raw")
    print(f"T06_FORMAL_VERIFY=OK networks=16 manifest_sha256={manifest}")


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    sub=parser.add_subparsers(dest="command",required=True)
    sub.add_parser("run")
    p=sub.add_parser("verify");p.add_argument("session")
    args=parser.parse_args()
    try:
        if args.command=="run":run()
        else:verify(args.session)
        return 0
    except (OSError,ValueError,KeyError,TypeError,subprocess.TimeoutExpired) as exc:
        print(f"T06_FORMAL_ERROR: {exc}",file=sys.stderr)
        return 2


if __name__=="__main__":sys.exit(main())
