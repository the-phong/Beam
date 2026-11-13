#!/usr/bin/env python3
"""Execute and audit paired T05 application-recorder ON/OFF preflight."""

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import statistics
import subprocess
import sys
from uuid import uuid4

import file02_common as common
from file02_common03 import ARTIFACTS, ROOT, git_value, private_session, require, steady_counts
import s07_orchestrate as s07

PROFILES=("C","CP","P","H")


def samples(path):
    with path.open() as stream:
        rows=[json.loads(line) for line in stream if line.strip()]
    require(len(rows)>=2 and all(row.get("schema")=="pluto.telemetry.v1" and
            row.get("dropped_samples")==0 for row in rows),"T05 missing/dropped observer samples")
    return rows


def process_stats(streams):
    comet_cpu=0
    comet_rss=0
    for stream in streams:
        values=[r["comet_process"] for r in stream if r.get("comet_process")]
        require(len(values)>=2 and values[0]["start_ticks"]==values[-1]["start_ticks"] and
                values[-1]["cpu_ns"]>=values[0]["cpu_ns"],"T05 Comet process discontinuity")
        comet_cpu+=values[-1]["cpu_ns"]-values[0]["cpu_ns"]
        comet_rss+=max(v["rss_bytes"] for v in values)
    app=[r["application_process"] for r in streams[0] if r.get("application_process")]
    require(len(app)>=2 and app[0]["start_ticks"]==app[-1]["start_ticks"] and
            app[-1]["cpu_ns"]>=app[0]["cpu_ns"],"T05 app process discontinuity")
    return {"comet_cpu_delta_ns":comet_cpu,
            "app_process_cpu_delta_ns":app[-1]["cpu_ns"]-app[0]["cpu_ns"],
            "comet_sum_peak_rss_bytes":comet_rss,
            "app_process_peak_rss_bytes":max(v["rss_bytes"] for v in app)}


def audit_run(run,profile,mode,source,tree):
    meta,plan,result,oracle,rec=(common.read_json(run/name) for name in
        ("metadata.json","plan.json","result.json","oracle.json","t05_recorder_summary.json"))
    require(meta.get("schema")=="pluto.file02.t05_run.v1" and meta.get("scope")=="T_PREFLIGHT" and
            meta.get("case_id")=="T05" and meta.get("profile")==profile and
            meta.get("recorder_mode")==mode and meta.get("source_commit")==source and
            meta.get("source_tree")==tree,"T05 metadata")
    require(meta["run_id"]==plan["run_id"]==result["run_id"]==oracle["run_id"]==rec["run_id"] and
            result["plan_sha256"]==common.digest(run/"plan.json") and
            plan["genesis_sha256"]==result["genesis_sha256"]==common.digest(run/"genesis.bin") and
            plan["config_sha256"]==result["config_sha256"]==meta["config_sha256"],"T05 provenance")
    require(result["status"]==oracle["status"]=="OK" and oracle["valid_committed"]==9 and
            oracle["invalid_rejected"]==0 and result["actual_send_count"]==result["planned_count"]==9 and
            rec.get("schema")=="pluto.file02.t05_recorder_summary.v1" and rec["mode"]==mode and
            rec.get("height")==8 and rec.get("app_hash_at_height") and
            rec["drop_counts"]==[0]*4 and len(rec["event_counts"])==4 and
            (sum(rec["event_counts"])>0 if mode=="ON" else sum(rec["event_counts"])==0),
            "T05 oracle/recorder")
    counts=steady_counts(plan,result)
    require(counts["legitimate"]==[2]*4 and counts["adversarial"]==[0]*4,
            "T05 rate/phase")
    stream_rows=[samples(run/f"observer-{i}.jsonl") for i in range(4)]
    stats=process_stats(stream_rows)
    lat=[]
    for event in result["events"]:
        require(event["actual_code"]==0 and event.get("actual_send_utc") and event.get("completed_utc"),
                "T05 RPC event")
        elapsed=(s07.parse_time(event["completed_utc"])-s07.parse_time(event["actual_send_utc"])).total_seconds()*1000
        require(elapsed>=0,"T05 negative RPC completion latency")
        lat.append(elapsed)
    require(len(lat)==9,"T05 latency count")
    return {"run_id":plan["run_id"],"profile":profile,"mode":mode,
            "genesis_sha256":plan["genesis_sha256"],"plan_events":plan["events"],
            "plan_seed":plan["seed"],"app_hash_at_height_8":rec["app_hash_at_height"],
            "valid_committed_once":9,"recorder_events":sum(rec["event_counts"]),
            "recorder_drops":sum(rec["drop_counts"]),"rpc_latency_n":len(lat),
            "rpc_completion_p50_ms":statistics.median(lat),**stats,"qc_status":"USABLE"}


def audit(session,source,tree):
    pairs=[]
    for index,profile in enumerate(PROFILES):
        on=audit_run(session/"runs"/f"t05-{index:02d}-ON",profile,"ON",source,tree)
        off=audit_run(session/"runs"/f"t05-{index:02d}-OFF",profile,"OFF",source,tree)
        require(on["genesis_sha256"]==off["genesis_sha256"] and
                on["plan_seed"]==off["plan_seed"] and on["plan_events"]==off["plan_events"] and
                on["app_hash_at_height_8"]==off["app_hash_at_height_8"],"T05 pair mismatch")
        order=["ON","OFF"] if index%2==0 else ["OFF","ON"]
        delta={key:on[key]-off[key] for key in ("rpc_completion_p50_ms","comet_cpu_delta_ns",
                "app_process_cpu_delta_ns","comet_sum_peak_rss_bytes","app_process_peak_rss_bytes")}
        for row in (on,off):row.pop("plan_events")
        pairs.append({"profile":profile,"order":order,"on":on,"off":off,"on_minus_off":delta})
    require(len({p["on"]["genesis_sha256"] for p in pairs})==4,"T05 duplicate profile genesis")
    return {"schema":"pluto.file02.t05_summary.v1","scope":"T_PREFLIGHT",
            "source_commit":source,"source_tree":tree,"fresh_networks":8,
            "pairs":pairs,"ci95":None,"ci_status":"NOT_ESTIMATED_FOUR_PROFILE_PAIRS"}


def run():
    require(not git_value("status","--porcelain"),"commit source before T05 execution")
    source,tree=git_value("rev-parse","HEAD"),git_value("rev-parse","HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True,exist_ok=True,mode=0o700)
    session=ARTIFACTS/("t05-"+datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")+"-"+uuid4().hex[:8])
    session.mkdir(mode=0o700)
    with (session/"build.log").open("wb") as log:
        build=subprocess.run(["make","build","build-comet-s05"],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT)
    require(build.returncode==0,"T05 build failed; inspect private build.log")
    env=os.environ.copy()
    env.update(FILE02_T05_ROOT=str(session),FILE02_SOURCE_COMMIT=source,FILE02_SOURCE_TREE=tree)
    with (session/"integration.log").open("wb") as log:
        completed=subprocess.run(["go","test","-tags","integration fourval","./test/integration",
            "-run","^TestFile02T05RecorderPairs$","-timeout","1200s","-count=1","-v"],
            cwd=ROOT,env=env,stdout=log,stderr=subprocess.STDOUT,timeout=1300)
    require(completed.returncode==0,"T05 integration failed; inspect private integration.log")
    summary=audit(session,source,tree)
    common.write_once(session/"summary.json",json.dumps(summary,indent=2,sort_keys=True)+"\n")
    manifest=common.freeze_raw(session)
    print(f"T05=PASS pairs=4 networks=8 session={session} manifest_sha256={manifest}")


def verify(path):
    session=private_session(path)
    manifest=common.verify_raw(session)
    prior=common.read_json(session/"summary.json")
    require(prior.get("schema")=="pluto.file02.t05_summary.v1" and
            audit(session,prior["source_commit"],prior["source_tree"])==prior,
            "T05 summary differs from raw")
    print(f"T05_VERIFY=OK pairs=4 manifest_sha256={manifest}")


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
        print(f"T05_ERROR: {exc}",file=sys.stderr)
        return 2


if __name__=="__main__":sys.exit(main())
