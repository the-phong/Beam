"""T02 wire-layout integrity tests with synthetic public bytes."""

import unittest

from file02_t02 import inspect_wire


class T02WireTest(unittest.TestCase):
    def test_cp_h_equal_and_padding_checked(self):
        chain=b"a"
        core=b"PLUTOTX4"+(4).to_bytes(2,"big")
        body=chain+bytes(20+20+8+8)
        cp=core+bytes([2,len(chain)])+body+bytes(64+2420)
        h=core+bytes([4,len(chain)])+body+bytes(64+2420)
        self.assertEqual(sum(inspect_wire(cp,"CP",chain).values()),len(h))
        self.assertEqual(sum(inspect_wire(h,"H",chain).values()),len(cp))
        bad=bytearray(cp);bad[-1]=1
        with self.assertRaisesRegex(ValueError,"padding"):
            inspect_wire(bytes(bad),"CP",chain)

    def test_wrong_profile_or_length_rejected(self):
        chain=b"a"
        core=b"PLUTOTX4"+(4).to_bytes(2,"big")+bytes([1,1])+chain+bytes(20+20+8+8+64)
        with self.assertRaisesRegex(ValueError,"envelope"):
            inspect_wire(core,"H",chain)
        with self.assertRaisesRegex(ValueError,"length"):
            inspect_wire(core[:-1],"C",chain)


if __name__=="__main__": unittest.main()
