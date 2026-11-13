#!/usr/bin/env python3
"""Run and audit the Owner-authorized bounded T06 knee search."""
import argparse
from datetime import datetime, timezone, timedelta
import hashlib
import json
import math
import os
from pathlib import Path
import statistics
import subprocess
import sys
from uuid import uuid4

import file02_common as common
from file02_common03 import git_value, private_session, require
from file02_t06_precision import parse_go_rfc3339, quantile99

ROOT=Path(__file__).resolve().parents[2]
ARTIFACTS=ROOT/'research/artifacts/file02'
PROFILES=('C','CP','P','H')
RATES=(20,23,25)

def percentile(values,p):
    ordered=sorted(values)
    return ordered[math.ceil(p*len(ordered))-1]

def row(run,source,tree,rate,profile,repetition):
    meta,plan,result,oracle,commits,resources=(common.read_json(run/name) for name in
        ('metadata.json','plan.json','result.json','oracle.json','commit_observations.json','generator_resources.json'))
    require(meta['source_commit']==source and meta['source_tree']==tree and meta['profile']==profile and
            meta['rate']==rate and meta['repetition']==repetition and meta['run_id']==plan['run_id']==result['run_id']==oracle['run_id']==commits['run_id'] and
            result['plan_sha256']==common.digest(run/'plan.json') and plan['genesis_sha256']==common.digest(run/'genesis.bin') and
            meta['config_sha256']==plan['config_sha256']==result['config_sha256'], 'run identity/hash')
    count=1+rate*12
    require(plan['target_tx_per_second']==rate and plan['steady_ns']==12_000_000_000 and
            result['status']==oracle['status']=='OK' and oracle['valid_committed']==count and oracle['invalid_rejected']==0 and
            result['planned_count']==result['actual_send_count']==len(plan['events'])==len(result['events'])==count and
            len(commits['events'])==count and oracle['common_height']>=8 and oracle['block_id'] and oracle['app_hash'], 'execution/oracle QC')
    start=parse_go_rfc3339(result['started_utc'])
    steady_end=start+timedelta(seconds=13)
    by_id={item['id']:item for item in commits['events']}
    require(len(by_id)==count,'duplicate commit observation')
    windows=[0]*12; byte_windows=[0]*12; latencies=[]; timely=0; hashes=set(); slots=set()
    for expected,actual in zip(plan['events'],result['events']):
        raw=bytes.fromhex(expected['bytes_hex'])
        sent=actual['actual_offset_ns']; target=expected['offset_ns']
        require(hashlib.sha256(raw).hexdigest()==expected['sha256'] and expected['sha256'] not in hashes and
                expected['account_slot'] not in slots and expected['id']==actual['id'] and
                expected['expected_code']==actual['actual_code']==0 and not actual.get('submission_error') and
                actual['wire_bytes']==len(raw) and target==actual['planned_offset_ns'] and
                (target<plan['warmup_ns'])==(sent<plan['warmup_ns']), 'event QC')
        hashes.add(expected['sha256']);slots.add(expected['account_slot'])
        seen=by_id[expected['id']]
        require(seen['sha256']==expected['sha256'] and 1<=seen['height']<=oracle['common_height'],'commit QC')
        sent_time=parse_go_rfc3339(actual['actual_send_utc']);seen_time=parse_go_rfc3339(seen['all_four_seen_utc'])
        require(seen_time>=sent_time,'negative latency')
        if target>=plan['warmup_ns']:
            index=(sent-plan['warmup_ns'])//plan['window_ns']
            require(0<=index<12,'steady send outside window')
            windows[index]+=1;byte_windows[index]+=len(raw)
            latencies.append((seen_time-sent_time).total_seconds()*1000)
            if seen_time<steady_end:timely+=1
    require(len(hashes)==len(slots)==count and windows==[rate]*12 and max(byte_windows)<=65536 and
            resources['wall_ns']>0 and resources['user_cpu_ns']+resources['system_cpu_ns']<.9*resources['wall_ns'], 'rate/cap/resource QC')
    queue_wait=[]; occupancy=[]; occupancy_bytes=[]; drops=0
    for i in range(4):
        samples=[json.loads(line) for line in (run/f'observer-{i}.jsonl').read_text().splitlines()]
        require(samples,'missing recorder')
        drops=max(drops,max(s.get('dropped_samples',0) for s in samples))
        require(drops==0 and all(s.get('kind')=='node_sample' for s in samples),'recorder QC')
        window_samples=[s for s in samples if start+timedelta(seconds=1)<=parse_go_rfc3339(s['observed_utc'])<=steady_end]
        require(len(window_samples)>=2,'insufficient steady recorder samples')
        occupancy.extend(s['accepted_mempool_count'] for s in window_samples if s.get('accepted_mempool_count') is not None)
        occupancy_bytes.extend(s['accepted_mempool_bytes'] for s in window_samples if s.get('accepted_mempool_bytes') is not None)
        first,last=window_samples[0],window_samples[-1]
        a,b=first.get('checktx_socket_queue_count'),last.get('checktx_socket_queue_count')
        x,y=first.get('checktx_socket_queue_seconds'),last.get('checktx_socket_queue_seconds')
        if all(isinstance(v,(int,float)) for v in (a,b,x,y)) and b>a and y>=x:queue_wait.append((y-x)/(b-a)*1000)
    return {'run_id':plan['run_id'],'profile':profile,'rate_tx_s':rate,'repetition':repetition,
            'achieved_offered_tx_s':sum(windows)/12,'achieved_offered_bytes_s':sum(byte_windows)/12,
            'timely_all_four_goodput_tx_s':timely/12,'eventual_goodput_tx_s':len(latencies)/12,
            'inclusion_upper_bound_p50_ms':percentile(latencies,.5),'inclusion_upper_bound_p95_ms':percentile(latencies,.95),
            'latency_n':len(latencies),'max_accepted_mempool_count':max(occupancy) if occupancy else None,
            'max_accepted_mempool_bytes':max(occupancy_bytes) if occupancy_bytes else None,
            'checktx_socket_queue_wait_mean_ms':statistics.mean(queue_wait) if queue_wait else None,
            'queue_source':'Comet ABCI socket cumulative count/sum; full queue depth unavailable',
            'observer_drop_count':drops,'generator_cpu_core_fraction':(resources['user_cpu_ns']+resources['system_cpu_ns'])/resources['wall_ns'],
            'steady_tx_windows':windows,'steady_byte_windows':byte_windows,'genesis_sha256':plan['genesis_sha256'],
            'qc_status':'USABLE_FOR_PILOT_ONLY'}

def decide(rows):
    result={}
    for profile in PROFILES:
        baseline=statistics.median(r['inclusion_upper_bound_p95_ms'] for r in rows if r['profile']==profile and r['rate_tx_s']==20)
        qbase=[r['checktx_socket_queue_wait_mean_ms'] for r in rows if r['profile']==profile and r['rate_tx_s']==20]
        queue_baseline=statistics.median(qbase) if all(v is not None for v in qbase) else None
        qualifying={}
        for rate in (23,25):
            cells=[r for r in rows if r['profile']==profile and r['rate_tx_s']==rate]
            goodput=sum(r['timely_all_four_goodput_tx_s'] < .9*r['achieved_offered_tx_s'] for r in cells)
            latency_queue=sum(r['inclusion_upper_bound_p95_ms']>=2*baseline and queue_baseline is not None and queue_baseline>0 and
                r['checktx_socket_queue_wait_mean_ms'] is not None and r['checktx_socket_queue_wait_mean_ms']>=2*queue_baseline for r in cells)
            qualifying[rate]={'goodput_runs':goodput,'latency_and_queue_runs':latency_queue,
                              'qualifies':goodput>=2 or latency_queue>=2}
        result[profile]={'status':'KNEE_OBSERVED_BRACKET_20_TO_23_TX_S' if all(x['qualifies'] for x in qualifying.values()) else
                         'KNEE_NOT_OBSERVED_WITHIN_TESTED_RANGE','criteria':qualifying,
                         'reference_p95_ms':baseline,'reference_queue_wait_ms':queue_baseline}
    return result

def audit(session,source,tree):
    rows=[row(session/'runs'/f't06k-r{rate:02d}-n{rep}-{profile}',source,tree,rate,profile,rep)
          for rate in RATES for rep in range(3) for profile in PROFILES]
    require(len({r['run_id'] for r in rows})==len({r['genesis_sha256'] for r in rows})==36,'network reuse')
    return {'schema':'pluto.file02.t06_knee_audit.v1','scope':'PILOT_ONLY','source_commit':source,'source_tree':tree,
            'config_sha256':common.digest(ROOT/'research/config/s06_pilot.json'),'rates_tx_s':list(RATES),
            'runs':rows,'decision':decide(rows)}

def smoke():
    require(not git_value('status','--porcelain'),'commit source and preregistration before pilot')
    source,tree=git_value('rev-parse','HEAD'),git_value('rev-parse','HEAD^{tree}')
    ARTIFACTS.mkdir(parents=True,exist_ok=True,mode=0o700)
    session=ARTIFACTS/('t06-knee-'+datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')+'-'+uuid4().hex[:8])
    session.mkdir(mode=0o700)
    with (session/'build.log').open('wb') as log:
        build=subprocess.run(['make','build','build-comet-s05'],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT)
    require(build.returncode==0,'build failed')
    env=os.environ.copy();env.update(FILE02_T06_KNEE_ROOT=str(session),FILE02_SOURCE_COMMIT=source,FILE02_SOURCE_TREE=tree)
    with (session/'integration.log').open('wb') as log:
        run=subprocess.run(['go','test','-tags','integration fourval','./test/integration','-run','^TestFile02T06KneePilot$',
                            '-timeout','2400s','-count=1','-v'],cwd=ROOT,env=env,stdout=log,stderr=subprocess.STDOUT,timeout=2500)
    require(run.returncode==0,f'pilot stopped on failed run; private session={session}')
    summary=audit(session,source,tree)
    common.write_once(session/'summary.json',json.dumps(summary,indent=2,sort_keys=True)+'\n')
    manifest=common.freeze_raw(session)
    print(f'T06_KNEE=COMPLETE runs=36 decisions={[(p,summary["decision"][p]["status"]) for p in PROFILES]} session={session} manifest_sha256={manifest}')

def verify(path):
    session=private_session(path);manifest=common.verify_raw(session);prior=common.read_json(session/'summary.json')
    recalculated=audit(session,prior['source_commit'],prior['source_tree'])
    # JSON object keys are strings: the in-memory decision uses integer rate keys.
    canonical=lambda value:json.loads(json.dumps(value,sort_keys=True))
    require(canonical(recalculated)==prior,'summary differs from frozen raw')
    print(f'T06_KNEE_VERIFY=OK runs=36 manifest_sha256={manifest}')

def main():
    parser=argparse.ArgumentParser(description=__doc__);sub=parser.add_subparsers(dest='command',required=True)
    sub.add_parser('smoke');p=sub.add_parser('verify');p.add_argument('session');args=parser.parse_args()
    try:
        if args.command=='smoke':smoke()
        else:verify(args.session)
        return 0
    except (ValueError,KeyError,TypeError,OSError,subprocess.TimeoutExpired) as exc:
        print(f'T06_KNEE_ERROR: {exc}',file=sys.stderr);return 2
if __name__=='__main__':sys.exit(main())
