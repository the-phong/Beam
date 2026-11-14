import unittest

import file02_q07 as q07


class Q07Test(unittest.TestCase):
    def test_rejects_invalid_committed_or_valid_missing(self):
        q07.assert_commit_set(["valid"], ["invalid"], ["valid"])
        with self.assertRaises(ValueError):
            q07.assert_commit_set(["valid"], ["invalid"], ["valid", "invalid"])
        with self.assertRaises(ValueError):
            q07.assert_commit_set(["valid"], ["invalid"], [])
        with self.assertRaises(ValueError):
            q07.assert_commit_set(["valid"], [], ["valid", "valid"])


if __name__ == "__main__":
    unittest.main()
