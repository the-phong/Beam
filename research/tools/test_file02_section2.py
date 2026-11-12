"""Contract checks for missing fields, provenance, QC and pending artifacts."""
import csv
import json
from pathlib import Path
import tempfile
import unittest

import file02_common as common
import file02_section2 as section2
from test_file02_common import plan, observation, H64


class Section2ContractTest(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name)

    def test_registry_and_pending_template(self):
        self.assertEqual(section2.registry(),63)
        case=self.root/'pending'
        section2.initialize(case)
        section2.validate_pending(case)
        (case/'result_aggregate.json').write_text('{}')
        with self.assertRaisesRegex(ValueError,'unreviewed results'):
            section2.validate_pending(case)

    def prepare_bundle(self):
        case=self.root/'case';case.mkdir()
        p=plan();(case/'plan.json').write_text(json.dumps(p))
        raw=case/'raw'/p['run_id'];raw.mkdir(parents=True)
        (raw/'events.json').write_text('{}\n')
        inventory={'schema':'pluto.file02.runtime_inventory.v1','run_id':p['run_id'],
            'profile':p['profile'],'source_clean':True,
            'node_configs':[{'config_toml_sha256':H64} for _ in range(4)]}
        inventory.update({key:p[key] for key in ('source_commit','source_tree','config_sha256',
                                                 'software_versions','host_inventory','network_config')})
        (raw/'runtime_inventory.json').write_text(json.dumps(inventory))
        (raw/'manifest.json').write_text(json.dumps({'schema':'pluto.file02.raw_manifest.v1',
            'files':{name:common.digest(raw/name) for name in ('events.json','runtime_inventory.json')}}))
        obs=observation(common.digest(raw/'manifest.json'))
        (case/'observation.json').write_text(json.dumps(obs))
        qc,result=common.evaluate(p,obs,raw)
        derived=case/'derived'/p['run_id'];derived.mkdir(parents=True)
        (derived/'qc_report.json').write_text(json.dumps(qc))
        (derived/'result_per_run.json').write_text(json.dumps(result))
        with (derived/'result_per_run.csv').open('w',newline='') as handle:
            fields=['run_id','source_tree','profile','case_id','actor_capability','traffic_source',
                    'topology','regime','seed','qc_status']
            fields += [f'planned_{key}' for key in ('valid_tx_s','invalid_tx_s','total_bytes_s')]
            fields += [f'achieved_{key}' for key in ('valid_tx_s','invalid_tx_s','total_bytes_s')]
            fields += [f'{group}__{name}' for group,names in common.METRICS.items() for name in names]
            row={key:result.get(key) for key in fields}
            row.update({f'planned_{key}':value for key,value in result['planned'].items()})
            row.update({f'achieved_{key}':value for key,value in result['achieved'].items()})
            row.update({f'{group}__{name}':value for group,values in result['metrics'].items()
                        for name,value in values.items()})
            writer=csv.DictWriter(handle,fieldnames=fields)
            writer.writeheader();writer.writerow(row)
        (case/'result_aggregate.json').write_text(json.dumps(common.aggregate([result],'valid_included_count')))
        (case/'figures').mkdir()
        (case/'figures/manifest.json').write_text(json.dumps({'schema':'pluto.file02.figure_manifest.v1',
            'status':'PENDING_CASE_EXECUTION','figures':[],'generator':None}))
        (case/'decision.md').write_text('Status: PENDING_CASE_EXECUTION\nNo RQ conclusion\n')
        return case,raw,derived

    def test_bundle_provenance_qc_and_missing_artifact(self):
        case,raw,derived=self.prepare_bundle()
        self.assertEqual(section2.validate_run_bundle(case),'USABLE')
        result=json.loads((derived/'result_per_run.json').read_text())
        result['source_tree']='f'*40
        (derived/'result_per_run.json').write_text(json.dumps(result))
        with self.assertRaisesRegex(ValueError,'provenance'):
            section2.validate_run_bundle(case)
        result['source_tree']='a'*40
        (derived/'result_per_run.json').write_text(json.dumps(result))
        qc=json.loads((derived/'qc_report.json').read_text());qc['status']='INVALID_FOR_ANALYSIS'
        (derived/'qc_report.json').write_text(json.dumps(qc))
        with self.assertRaisesRegex(ValueError,'provenance'):
            section2.validate_run_bundle(case)
        (derived/'qc_report.json').unlink()
        with self.assertRaises((OSError,ValueError)):
            section2.validate_run_bundle(case)

    def test_unavailable_metric_and_fabricated_ci_rejected(self):
        case,raw,derived=self.prepare_bundle()
        obs=json.loads((case/'observation.json').read_text())
        obs['metrics']['ingress']={'queue_wait_ms':0}
        (case/'observation.json').write_text(json.dumps(obs))
        with self.assertRaisesRegex(ValueError,'unavailable metric must be null'):
            section2.validate_run_bundle(case)
        obs['metrics'].pop('ingress');(case/'observation.json').write_text(json.dumps(obs))
        aggregate=json.loads((case/'result_aggregate.json').read_text())
        aggregate['ci95']=[0,1]
        (case/'result_aggregate.json').write_text(json.dumps(aggregate))
        with self.assertRaisesRegex(ValueError,'fabricated CI'):
            section2.validate_run_bundle(case)

    def test_csv_metric_tampering_and_wrong_schema_rejected(self):
        case,raw,derived=self.prepare_bundle()
        csv_path=derived/'result_per_run.csv'
        csv_path.write_text(csv_path.read_text().replace('consensus__valid_included_count',
                                                          'consensus__wrong_field'))
        with self.assertRaisesRegex(ValueError,'CSV schema/value mismatch'):
            section2.validate_run_bundle(case)
        case,raw,derived=self.prepare_bundle_at(self.root/'second')
        qc_path=derived/'qc_report.json';qc=json.loads(qc_path.read_text())
        qc['schema']='wrong';qc_path.write_text(json.dumps(qc))
        with self.assertRaisesRegex(ValueError,'provenance'):
            section2.validate_run_bundle(case)

    def prepare_bundle_at(self,target):
        # Reuse the same synthetic material in a distinct test directory.
        prior=self.root
        self.root=target
        target.mkdir()
        try:return self.prepare_bundle()
        finally:self.root=prior


if __name__=='__main__':unittest.main()
