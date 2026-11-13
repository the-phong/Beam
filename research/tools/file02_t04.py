#!/usr/bin/env python3
"""Execute and verify the bounded formal T04 mixed-input consensus preflight."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
from uuid import uuid4

import file02_common as common
from file02_common03 import ARTIFACTS, ROOT, git_value, private_session, require, steady_counts

PROFILES=("C","CP","P","H")


def audit_run(run, profile, source, tree):
    meta, plan, result, oracle, commits, supply=(common.read_json(run/name) for name in
        ("metadata.json","plan.json","result.json","oracle.json","commit_observations.json","t04_supply_oracle.json"))
    require(meta.get("schema")=="pluto.file02.t04_run.v1" and meta.get("scope")=="T_PREFLIGHT" and
            meta.get("case_id")=="T04" and meta.get("profile")==profile and
            meta.get("source_commit")==source and meta.get("source_tree")==tree and
            meta.get("network")=="fresh_four_validator_single_host_loopback", "T04 metadata")
    require(meta["run_id"]==plan["run_id"]==result["run_id"]==oracle["run_id"]==commits["run_id"]==supply["run_id"] and
            result["plan_sha256"]==common.digest(run/"plan.json") and
            plan["genesis_sha256"]==result["genesis_sha256"]==common.digest(run/"genesis.bin") and
            plan["config_sha256"]==result["config_sha256"]==meta["config_sha256"], "T04 provenance")
    require(result["status"]==oracle["status"]=="OK" and oracle["common_height"]>=8 and
            oracle["block_id"] and oracle["app_hash"] and oracle["valid_committed"]==5 and
            oracle["invalid_rejected"]==4 and result["actual_send_count"]==result["planned_count"]==9 and
            len(plan["events"])==len(result["events"])==9 and len(commits["events"])==5,
            "T04 execution or oracle")
    require(plan["profile"]=={"C":1,"CP":2,"P":3,"H":4}[profile] and
            plan["regime"]=="fixed_legitimate_plus_adversarial" and
            supply.get("schema")=="pluto.file02.t04_supply_oracle.v1" and
            supply.get("expected_total_supply")==32 and supply.get("validator_total_supply")==[32]*4 and
            supply.get("common_height")==oracle["common_height"], "T04 profile/regime/supply")
    hashes=set();valid=set();invalid=set()
    for planned,actual in zip(plan["events"],result["events"]):
        raw=bytes.fromhex(planned["bytes_hex"])
        require(hashlib.sha256(raw).hexdigest()==planned["sha256"] and
                planned["sha256"] not in hashes and planned["id"]==actual["id"] and
                planned["expected_code"]==actual["actual_code"] and not actual.get("submission_error"),
                "T04 transaction/code")
        hashes.add(planned["sha256"])
        (valid if planned["traffic"]=="legitimate" else invalid).add(planned["sha256"])
    observed={event["sha256"] for event in commits["events"]}
    require(observed==valid and not observed.intersection(invalid) and len(valid)==5 and len(invalid)==4 and
            all(event["height"]<=oracle["common_height"] for event in commits["events"]),
            "T04 inclusion or invalid commit")
    counts=steady_counts(plan,result)
    require(counts["legitimate"]==[1]*4 and counts["adversarial"]==[1]*4,
            "T04 fixed legitimate/invalid rates")
    require(all((run/f"observer-{i}.jsonl").stat().st_size>0 for i in range(4)),
            "T04 observer stream")
    return {"run_id":plan["run_id"],"profile":profile,"source_commit":source,
            "plan_sha256":common.digest(run/"plan.json"),"genesis_sha256":plan["genesis_sha256"],
            "valid_committed_once":5,"invalid_rejected_and_not_committed":4,
            "common_height":oracle["common_height"],"same_height_block_id":oracle["block_id"],
            "same_height_app_hash":oracle["app_hash"],"validator_total_supply":supply["validator_total_supply"],
            "steady_valid_tx_s":counts["legitimate"],"steady_invalid_tx_s":counts["adversarial"],
            "qc_status":"USABLE"}


def audit(session,source,tree):
    rows=[audit_run(session/"runs"/f"t04-{i:02d}-{profile}",profile,source,tree)
          for i,profile in enumerate(PROFILES)]
    require(len({r["genesis_sha256"] for r in rows})==4 and len({r["run_id"] for r in rows})==4,
            "T04 non-independent networks")
    return {"schema":"pluto.file02.t04_summary.v1","scope":"T_PREFLIGHT",
            "source_commit":source,"source_tree":tree,"fresh_networks":4,
            "profiles":list(PROFILES),"all_oracles_pass":True,"runs":rows}


def run():
    require(not git_value("status","--porcelain"),"commit source before T04 execution")
    source,tree=git_value("rev-parse","HEAD"),git_value("rev-parse","HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True,exist_ok=True,mode=0o700)
    session=ARTIFACTS/("t04-"+datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")+"-"+uuid4().hex[:8])
    session.mkdir(mode=0o700)
    with (session/"build.log").open("wb") as log:
        build=subprocess.run(["make","build","build-comet-s05"],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT)
    require(build.returncode==0,"T04 build failed; inspect private build.log")
    env=os.environ.copy()
    env.update(FILE02_T04_ROOT=str(session),FILE02_SOURCE_COMMIT=source,FILE02_SOURCE_TREE=tree)
    with (session/"integration.log").open("wb") as log:
        completed=subprocess.run(["go","test","-tags","integration fourval","./test/integration",
            "-run","^TestFile02T04MixedConsensus$","-timeout","600s","-count=1","-v"],
            cwd=ROOT,env=env,stdout=log,stderr=subprocess.STDOUT,timeout=700)
    require(completed.returncode==0,"T04 integration failed; inspect private integration.log")
    summary=audit(session,source,tree)
    common.write_once(session/"summary.json",json.dumps(summary,indent=2,sort_keys=True)+"\n")
    manifest=common.freeze_raw(session)
    print(f"T04=PASS networks=4 session={session} manifest_sha256={manifest}")


def verify(path):
    session=private_session(path)
    manifest=common.verify_raw(session)
    prior=common.read_json(session/"summary.json")
    require(prior.get("schema")=="pluto.file02.t04_summary.v1" and
            audit(session,prior["source_commit"],prior["source_tree"])==prior,
            "T04 summary differs from raw")
    print(f"T04_VERIFY=OK networks=4 manifest_sha256={manifest}")


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
        print(f"T04_ERROR: {exc}",file=sys.stderr)
        return 2


if __name__=="__main__":sys.exit(main())
