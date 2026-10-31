#!/usr/bin/env python3
"""Independent B0 V2 vector oracle; public scalar-1 fixture only.

Uses Python byte serialization/hashlib and OpenSSL via cryptography for ECDSA.
It does not import or call Pluto's Go state or transaction packages.
"""

import hashlib
import json
import platform
import struct
from pathlib import Path

import cryptography
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import ec, utils
from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat

ROOT = Path(__file__).resolve().parent
V = json.loads((ROOT / "state_vectors.json").read_text())
H = lambda raw: hashlib.sha256(raw).digest()
U16 = lambda n: struct.pack(">H", n)
U32 = lambda n: struct.pack(">I", n)
U64 = lambda n: struct.pack(">Q", n)
DOMAIN = b"PLUTO/B0/STATE/V2\0"
ADDR_DOMAIN = b"PLUTO/B0/ADDR/V1\0"
SIGN_DOMAIN = b"PLUTO/B0/SIGN/V1\0"
PUB_A = bytes.fromhex("0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798")
PUB_B = ec.derive_private_key(2, ec.SECP256K1()).public_key().public_bytes(
    encoding=Encoding.X962,
    format=PublicFormat.CompressedPoint,
)
A = H(ADDR_DOMAIN + PUB_A)[:20]
B = H(ADDR_DOMAIN + PUB_B)[:20]


def state(chain, height, supply, accounts):
    assert 1 <= len(chain) <= 50 and 1 <= len(accounts) <= 65536
    assert accounts == sorted(accounts, key=lambda x: x[0])
    assert sum(balance for _, balance, _ in accounts) == supply
    raw = U32(2) + U16(len(chain)) + chain + U64(height) + U64(supply) + U32(len(accounts))
    for address, balance, nonce in accounts:
        assert len(address) == 20
        raw += address + U64(balance) + U64(nonce)
    return raw, H(DOMAIN + raw)


def check_snapshot(label, chain, height, supply, accounts):
    snap = V[label]
    raw, digest = state(chain.encode("ascii"), height, supply, accounts)
    assert snap["chain_id"] == chain and snap["height"] == height
    assert snap["total_supply"] == supply
    assert snap["logical_hex"] == raw.hex()
    assert snap["apphash_hex"] == digest.hex()
    assert snap["accounts"] == [dict(address_hex=a.hex(), balance=b, nonce=n) for a, b, n in accounts]
    print(f"{label}: {digest.hex()} PASS")


def check_genesis(label, chain, allocations):
    raw = b"PLUTOG2!" + U32(2) + U16(len(chain)) + chain.encode() + U32(len(allocations))
    for address, balance in allocations:
        raw += address + U64(balance)
    assert V[label + "_wire_hex"] == raw.hex()


def verify_wire(hexwire, expected_sender, receiver, amount, nonce):
    raw = bytes.fromhex(hexwire)
    assert raw[:8] == b"PLUTOTX1" and raw[8:10] == U16(1)
    n = int.from_bytes(raw[10:12], "big")
    assert n == 11 and len(raw) == 145 + n and raw[12:12+n] == b"pluto-b0-v1"
    pub = raw[12+n:45+n]
    assert pub == PUB_A and H(ADDR_DOMAIN + pub)[:20] == expected_sender
    assert raw[45+n:65+n] == receiver
    assert int.from_bytes(raw[65+n:73+n], "big") == amount
    assert int.from_bytes(raw[73+n:81+n], "big") == nonce
    unsigned = raw[:81+n]
    digest = H(SIGN_DOMAIN + U32(len(unsigned)) + unsigned)
    r = int.from_bytes(raw[81+n:113+n], "big")
    s = int.from_bytes(raw[113+n:145+n], "big")
    order = int("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
    assert 1 <= r < order and 1 <= s <= order // 2
    key = ec.EllipticCurvePublicKey.from_encoded_point(ec.SECP256K1(), pub)
    key.verify(utils.encode_dss_signature(r, s), digest, ec.ECDSA(utils.Prehashed(hashes.SHA256())))


assert ec.derive_private_key(1, ec.SECP256K1()).public_key().public_numbers().x == int.from_bytes(PUB_A[1:], "big")
check_genesis("genesis", "pluto-b0-v1", [(A, 100)])
check_genesis("min_chain_genesis", "c", [(A, 100)])
check_genesis("max_chain_genesis", "c" * 50, [(A, 100)])
check_snapshot("genesis", "pluto-b0-v1", 0, 100, [(A, 100, 0)])
check_snapshot("min_chain_genesis", "c", 0, 100, [(A, 100, 0)])
check_snapshot("max_chain_genesis", "c" * 50, 0, 100, [(A, 100, 0)])
verify_wire(V["transfer_wire_hex"], A, B, 25, 0)
check_snapshot("post_transfer", "pluto-b0-v1", 1, 100, sorted([(A, 75, 1), (B, 25, 0)]))
verify_wire(V["self_wire_hex"], A, A, 25, 1)
check_snapshot("post_self_transfer", "pluto-b0-v1", 2, 100, sorted([(A, 75, 2), (B, 25, 0)]))
check_snapshot("post_empty_block", "pluto-b0-v1", 3, 100, sorted([(A, 75, 2), (B, 25, 0)]))
neg = {case["id"]: case for case in V["negative_cases"]}
assert neg["replay"]["wire_hex"] == V["transfer_wire_hex"]
assert neg["replay"]["expected_code"] == 6
assert neg["replay"]["unchanged_apphash_hex"] == V["post_transfer"]["apphash_hex"]
verify_wire(neg["insufficient_balance"]["wire_hex"], A, B, 101, 0)
assert neg["insufficient_balance"]["expected_code"] == 7
assert neg["insufficient_balance"]["unchanged_apphash_hex"] == V["genesis"]["apphash_hex"]
print(f"oracle PASS; Python {platform.python_version()}; cryptography {cryptography.__version__}")
