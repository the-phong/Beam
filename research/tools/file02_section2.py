#!/usr/bin/env python3
"""Validate File 02 section-2 registries and pending/complete artifact bundles."""
import argparse
import csv
import json
from pathlib import Path
import sys

import file02_common as common

ROOT=Path(__file__).resolve().parents[1]
FIELDS={'definition','unit','capture_point','available_at','calculation','limitation','status'}
STATUSES={'AVAILABLE_WITH_CASE','PENDING_CASE_EXECUTION','UNAVAILABLE'}
ARTIFACTS=('plan.json','raw/run_id/','result_per_run.json','result_per_run.csv',
           'qc_report.json','result_aggregate.json','figures/','decision.md')

def registry():
    factors=common.read_json(ROOT/'config/file02_factors.json')
    metrics=common.read_json(ROOT/'config/file02_metrics.json')
    common.require(factors['schema']=='pluto.file02.factors.v1' and
                   factors['profiles']==['C','CP','P','H'] and
                   factors['tested_pilot_tx_s']==[2,6,12,20,23,25] and
                   factors['load_values']['PILOT_LOW']==[2,6] and
                   factors['load_values']['NEAR_KNEE']==factors['load_values']['AT_KNEE']==
                   factors['load_values']['SLIGHTLY_OVER_KNEE']=='NOT_OBSERVED_WITHIN_SAFE_RANGE' and
                   factors['load_values']['SUBCRITICAL']=='NOT_ESTABLISHED' and
                   factors['source_levels']['P2P_ORIGIN_INJECTION']=='UNSUPPORTED' and
                   factors['source_levels']['RPC_MULTI']=='UNSUPPORTED' and
                   factors['topology_levels']['FOUR_VALIDATOR_MULTI_HOST_PRIMARY']=='UNSUPPORTED' and
                   factors['policy_levels']['REVERSE_VERIFY_ORDER']==factors['policy_levels']['EARLY_REJECT']=='UNSUPPORTED',
                   'factor registry/status')
    common.require(metrics['schema']=='pluto.file02.metric_registry.v1' and
                   set(metrics['metrics'])==set(common.METRICS),'metric groups')
    for group,names in common.METRICS.items():
        common.require(set(metrics['metrics'][group])==set(names),'metric names')
        for name in names:
            spec=metrics['metrics'][group][name]
            common.require(set(spec)==FIELDS and spec['status'] in STATUSES and
                           all(isinstance(spec[k],str) and spec[k].strip() for k in FIELDS),
                           f'incomplete metric contract: {group}.{name}')
    templates=common.read_json(ROOT/'config/file02_artifact_templates.json')
    common.require(templates.get('schema')=='pluto.file02.artifact_templates.v1' and
                   templates.get('status')=='PENDING_CASE_EXECUTION' and
                   set(templates.get('artifacts',{}))=={
                       'plan.json','raw/run_id/manifest.json','observation.json',
                       'result_per_run.json','result_per_run.csv','qc_report.json',
                       'result_aggregate.json','figures/manifest.json','decision.md'} and
                   all(item.get('schema','').startswith('pluto.file02.') and
                       item.get('status') in ('DRAFT','PENDING_CASE_EXECUTION')
                       for item in templates['artifacts'].values()),'artifact templates')
    return len([n for group in common.METRICS.values() for n in group])

def initialize(path):
    root=Path(path)
    common.require(not root.exists(),'case directory already exists')
    root.mkdir(parents=True)
    common.write_once(root/'plan.json',(ROOT/'config/file02_plan.template.json').read_text())
    common.write_once(root/'artifact_status.json',json.dumps({'schema':'pluto.file02.artifact_status.v1',
        'status':'PENDING_CASE_EXECUTION','items':{name:'PENDING_CASE_EXECUTION' for name in ARTIFACTS}},indent=2,sort_keys=True)+'\n')
    (root/'figures').mkdir()
    common.write_once(root/'figures/manifest.json',json.dumps({'schema':'pluto.file02.figure_manifest.v1',
        'status':'PENDING_CASE_EXECUTION','figures':[],'generator':None},indent=2,sort_keys=True)+'\n')
    common.write_once(root/'decision.md','# File 02 case decision\n\nStatus: PENDING_CASE_EXECUTION\n\nNo RQ conclusion or scientific figure exists.\n')
    validate_pending(root)

def validate_pending(root):
    root=Path(root)
    common.require(root.is_dir() and not root.is_symlink(),'case root')
    status=common.read_json(root/'artifact_status.json')
    common.require(status.get('schema')=='pluto.file02.artifact_status.v1' and
                   status.get('status')=='PENDING_CASE_EXECUTION' and
                   status.get('items')=={name:'PENDING_CASE_EXECUTION' for name in ARTIFACTS},'artifact status')
    plan=common.read_json(root/'plan.json')
    common.require(plan.get('schema')=='pluto.file02.plan.v1' and plan.get('status')=='DRAFT','draft plan template')
    common.require((root/'plan.json').read_bytes()==(ROOT/'config/file02_plan.template.json').read_bytes(),
                   'pending plan changed without execution')
    figure=common.read_json(root/'figures/manifest.json')
    common.require(figure=={'schema':'pluto.file02.figure_manifest.v1','status':'PENDING_CASE_EXECUTION',
                           'figures':[],'generator':None},'figure manifest')
    common.require((root/'decision.md').read_text().startswith('# File 02 case decision\n\nStatus: PENDING_CASE_EXECUTION\n') and
                   set(str(p.relative_to(root)) for p in root.rglob('*') if p.is_file())==
                   {'artifact_status.json','plan.json','figures/manifest.json','decision.md'},
                   'pending case contains unreviewed results')

def validate_run_bundle(root):
    """Validate existing future-case artifacts; do not create or analyze them."""
    root=Path(root)
    plan=common.read_json(root/'plan.json');common.validate_plan(plan)
    run_id=plan['run_id'];raw=root/'raw'/run_id;derived=root/'derived'/run_id
    manifest=common.verify_raw(raw)
    observation=common.read_json(root/'observation.json')
    qc,result=common.evaluate(plan,observation,raw)
    saved_qc=common.read_json(derived/'qc_report.json')
    saved_result=common.read_json(derived/'result_per_run.json')
    common.require(saved_qc==qc and saved_result==result and
                   result['raw_manifest_sha256']==manifest,'QC/result provenance or schema')
    with (derived/'result_per_run.csv').open(newline='') as handle:
        csv_rows=list(csv.DictReader(handle))
    fields=['run_id','source_tree','profile','case_id','actor_capability','traffic_source',
            'topology','regime','seed','qc_status']
    fields += [f'planned_{key}' for key in ('valid_tx_s','invalid_tx_s','total_bytes_s')]
    fields += [f'achieved_{key}' for key in ('valid_tx_s','invalid_tx_s','total_bytes_s')]
    fields += [f'{group}__{name}' for group,names in common.METRICS.items() for name in names]
    expected={key:result.get(key) for key in fields}
    expected.update({f'planned_{key}':value for key,value in result['planned'].items()})
    expected.update({f'achieved_{key}':value for key,value in result['achieved'].items()})
    expected.update({f'{group}__{name}':value for group,values in result['metrics'].items()
                     for name,value in values.items()})
    common.require(len(csv_rows)==1 and set(csv_rows[0])==set(fields) and
                   all(csv_rows[0][key]==('' if expected[key] is None else str(expected[key]))
                       for key in fields),'CSV schema/value mismatch')
    aggregate=common.read_json(root/'result_aggregate.json')
    expected=common.aggregate([result],aggregate.get('endpoint'))
    common.require(aggregate==expected and aggregate['ci95'] is None,'aggregate provenance or fabricated CI')
    figures=common.read_json(root/'figures/manifest.json')
    common.require(figures=={'schema':'pluto.file02.figure_manifest.v1','status':'PENDING_CASE_EXECUTION',
                            'figures':[],'generator':None} and
                   set(p.name for p in (root/'figures').iterdir())=={'manifest.json'},
                   'unreviewed figure')
    decision=(root/'decision.md').read_text()
    common.require('Status: PENDING_CASE_EXECUTION' in decision and
                   'No RQ conclusion' in decision,'unreviewed RQ decision')
    return qc['status']

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    sub=parser.add_subparsers(dest='command',required=True)
    sub.add_parser('validate-registry')
    for cmd in ('init-pending','validate-pending','validate-run-bundle'):
        p=sub.add_parser(cmd);p.add_argument('directory')
    args=parser.parse_args()
    try:
        count=registry()
        if args.command=='init-pending':initialize(args.directory)
        elif args.command=='validate-pending':validate_pending(args.directory)
        elif args.command=='validate-run-bundle':
            print(f'RUN_BUNDLE_QC={validate_run_bundle(args.directory)}')
        print(f'SECTION2_CONTRACT=OK metrics={count}')
        return 0
    except (ValueError,KeyError,TypeError,OSError) as exc:
        print(f'SECTION2_ERROR: {exc}',file=sys.stderr);return 2
if __name__=='__main__':sys.exit(main())
