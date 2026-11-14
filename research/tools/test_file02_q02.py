"""Q02 block-unit, DiD-sign and missing-arm safeguards."""

import unittest

import file02_q02 as q


class DecompositionTest(unittest.TestCase):
    def test_corrected_schedule_balances_each_profile_and_retains_v1_replay(self):
        current, legacy = q.schedule(), q.schedule(legacy=True)
        self.assertEqual(current["schema"], "pluto.file02.q02.schedule.v2")
        self.assertEqual(legacy["schema"], "pluto.file02.q02.schedule.v1")
        for profile in q.PROFILES:
            first = [next(job["condition"] for job in current["jobs"] if
                          job["block"] == block and job["profile"] == profile)
                     for block in range(12)]
            self.assertEqual(first.count("benign"), 6)
            self.assertEqual(first.count("attack"), 6)
        self.assertNotEqual(current["jobs"], legacy["jobs"])

    def block(self):
        rows = []
        fractions = {"C": (1.0, .75), "CP": (1.0, .875),
                     "P": (1.0, 1.0), "H": (1.0, .5)}
        for profile in q.PROFILES:
            for condition, value in zip(("benign", "attack"), fractions[profile]):
                rows.append({"block": 0, "profile": profile, "condition": condition,
                             "genesis_sha256": "g", "seed": 9,
                             "fixed_hashes": [profile], "fixed_offsets_ns": [1],
                             "fixed_slots": [0],
                             "fixed_timely_fraction": value})
        return rows

    def test_did_sign_is_registered_formula(self):
        result = q.block_contrasts(self.block())
        self.assertAlmostEqual(result["did_H_C"], -.25)
        self.assertAlmostEqual(result["effects_benign_minus_attack"]["H"], .5)
        self.assertAlmostEqual(result["condition_contrasts"]["attack"]["H_minus_CP"], -.375)

    def test_failed_block_cannot_be_partially_used(self):
        rows = self.block()
        with self.assertRaises(ValueError):
            q.block_contrasts(rows[:-1])
        rows = self.block()
        rows[-1]["fixed_hashes"] = ["different"]
        with self.assertRaises(ValueError):
            q.block_contrasts(rows)

    def test_ci_requires_twelve_blocks(self):
        with self.assertRaises(ValueError):
            q.ci([0] * 11)


if __name__ == "__main__":
    unittest.main()
