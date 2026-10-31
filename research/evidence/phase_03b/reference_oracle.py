"""Independent vector checker using Python hashlib and OpenSSL cryptography.

Run from repository root: python3 research/evidence/phase_03b/reference_oracle.py
Only public, nonproduction fixture scalars 1 and 2 are used.
"""
import hashlib
import json
from pathlib import Path

from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat
from cryptography.hazmat.primitives.asymmetric import ec, utils


ROOT = Path(__file__).resolve().parent
N = int("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
H = lambda b: hashlib.sha256(b).digest()


def main():
    vectors = json.loads((ROOT / "golden_vectors.json").read_text())
    a = ec.derive_private_key(1, ec.SECP256K1()).public_key()
    b = ec.derive_private_key(2, ec.SECP256K1()).public_key()
    a_key = a.public_bytes(Encoding.X962, PublicFormat.CompressedPoint)
    b_key = b.public_bytes(Encoding.X962, PublicFormat.CompressedPoint)
    a_addr = H(b"PLUTO/B0/ADDR/V1\0" + a_key)[:20]
    b_addr = H(b"PLUTO/B0/ADDR/V1\0" + b_key)[:20]
    for v in vectors:
        chain = v["chain_id"].encode("ascii")
        unsigned = (b"PLUTOTX1" + (1).to_bytes(2, "big") + len(chain).to_bytes(2, "big") + chain
                    + a_key + b_addr + v["amount"].to_bytes(8, "big") + v["nonce"].to_bytes(8, "big"))
        preimage = b"PLUTO/B0/SIGN/V1\0" + len(unsigned).to_bytes(4, "big") + unsigned
        digest = H(preimage)
        r = int(v["r_hex"], 16)
        s = int(v["s_hex"], 16)
        assert 1 <= r < N and 1 <= s <= N // 2
        a.verify(utils.encode_dss_signature(r, s), digest, ec.ECDSA(utils.Prehashed(hashes.SHA256())))
        wire = unsigned + bytes.fromhex(v["r_hex"] + v["s_hex"])
        txid = H(b"PLUTO/B0/TXID/V1\0" + len(wire).to_bytes(4, "big") + wire)
        checks = {
            "compressed_pubkey_hex": a_key.hex(), "sender_address_hex": a_addr.hex(),
            "receiver_hex": b_addr.hex(), "unsigned_payload_hex": unsigned.hex(),
            "signing_preimage_hex": preimage.hex(), "signing_digest_hex": digest.hex(),
            "wire_hex": wire.hex(), "pluto_txid_hex": txid.hex(),
            "comet_hash_hex": H(wire).hex(),
        }
        for field, want in checks.items():
            assert v[field] == want, (v["id"], field, v[field], want)
        assert v["txid_eligible"] is True
        assert v["expected_stateless_code"] == (5 if v["amount"] == 0 else 0)
        print(f'{v["id"]}: PASS length={len(wire)} code={v["expected_stateless_code"]}')
    print(f"independent vectors verified: {len(vectors)}")


if __name__ == "__main__":
    main()
