#!/usr/bin/env python3
"""Independent D-025 JSON/Base64 and Genesis/AppHash reference calculation."""

import base64
import hashlib
import json
import struct
from pathlib import Path


vectors = json.loads((Path(__file__).resolve().parents[2] / "phase_03c" / "state_vectors.json").read_text())
genesis = bytes.fromhex(vectors["genesis_wire_hex"])
assert genesis[:8] == b"PLUTOG2!"
version, chain_len = struct.unpack(">IH", genesis[8:14])
assert version == 2 and 1 <= chain_len <= 50
chain = genesis[14 : 14 + chain_len]
offset = 14 + chain_len
count = struct.unpack(">I", genesis[offset : offset + 4])[0]
offset += 4
assert len(genesis) == offset + 28 * count
accounts = []
for _ in range(count):
    address = genesis[offset : offset + 20]
    balance = struct.unpack(">Q", genesis[offset + 20 : offset + 28])[0]
    accounts.append((address, balance, 0))
    offset += 28
assert all(accounts[i - 1][0] < accounts[i][0] for i in range(1, count))
supply = sum(balance for _, balance, _ in accounts)
logical = struct.pack(">IH", version, chain_len) + chain
logical += struct.pack(">QQI", 0, supply, count)
for address, balance, nonce in accounts:
    logical += address + struct.pack(">QQ", balance, nonce)
apphash = hashlib.sha256(b"PLUTO/B0/STATE/V2\0" + logical).hexdigest()
assert logical.hex() == vectors["genesis"]["logical_hex"]
assert apphash == vectors["genesis"]["apphash_hex"]

encoded = base64.b64encode(genesis).decode("ascii")
transport = json.dumps(encoded, ensure_ascii=True).encode("ascii")
assert base64.b64decode(json.loads(transport), validate=True) == genesis
assert transport == json.dumps(encoded).encode("ascii")
assert 8 + 4 + 2 + 50 + 4 + 28 * 65536 == 1_835_076
assert 4 * ((1_835_076 + 2) // 3) == 2_446_768

print("genesis_binary_hex=" + genesis.hex())
print("genesis_binary_sha256=" + hashlib.sha256(genesis).hexdigest())
print("genesis_transport_hex=" + transport.hex())
print("genesis_transport_length=" + str(len(transport)))
print("genesis_apphash_v2=" + apphash)
print("max_binary=1835076 max_base64=2446768 max_canonical_json=2446770")
print("reference PASS")
