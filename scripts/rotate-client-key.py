#!/usr/bin/env python3
"""Rotate the shared X-API-Key used by the NPS feedback clients.

Generates a fresh key plus a fresh XOR mask, rewrites the masked byte arrays in
each client's FeedbackService.ResolveApiKey, verifies that the patched source
decodes back to the key, and prints the API_KEYS line for the server's
~/nps-api/.env. The plaintext key is only ever printed to this terminal.

Usage:
    python3 scripts/rotate-client-key.py            # patch rokdsk + Idefinity
    python3 scripts/rotate-client-key.py --test F…  # patch given files, no key output
"""
import re
import secrets
import sys
from pathlib import Path

LOCALGIT = Path(__file__).resolve().parents[2]
CLIENTS = [
    LOCALGIT / "rokdsk/src/API/FeedbackService.xojo_code",
    LOCALGIT / "idefinity/src/Managers/FeedbackService.xojo_code",
]

KEY_BYTES = 24   # 48 hex chars, same as `openssl rand -hex 24`
MASK_BYTES = 16
INDENT = "\t\t  "


def block_pattern(var_name):
    # var <name>() As UInt8 = Array( _
    #   CType(&hXX, UInt8), ... , _
    #   CType(&hXX, UInt8), CType(&hXX, UInt8))
    return re.compile(
        r"^" + INDENT + r"var " + var_name + r"\(\) As UInt8 = Array\( _\n"
        r"(?:" + INDENT + r".* _\n)*"
        + INDENT + r".*\)\)\n",
        re.MULTILINE,
    )


def render(var_name, data):
    cells = [f"CType(&h{b:02X}, UInt8)" for b in data]
    rows = [", ".join(cells[i:i + 4]) for i in range(0, len(cells), 4)]
    body = "".join(INDENT + r + ", _\n" for r in rows[:-1]) + INDENT + rows[-1] + ")\n"
    return f"{INDENT}var {var_name}() As UInt8 = Array( _\n{body}"


def parse(var_name, source):
    m = block_pattern(var_name).search(source)
    return bytes(int(h, 16) for h in re.findall(r"&h([0-9A-Fa-f]{2})", m.group(0)))


def main(argv):
    test = argv[:1] == ["--test"]
    files = [Path(p) for p in argv[1:]] if test else CLIENTS

    key = secrets.token_hex(KEY_BYTES)
    mask = secrets.token_bytes(MASK_BYTES)
    masked = bytes(b ^ mask[i % MASK_BYTES] for i, b in enumerate(key.encode("ascii")))

    patched = {}
    for f in files:
        src = f.read_text()
        for name in ("masked", "mask"):
            if len(block_pattern(name).findall(src)) != 1:
                sys.exit(f"{f}: expected exactly one `var {name}()` array in ResolveApiKey")
        src = block_pattern("masked").sub(lambda _: render("masked", masked), src)
        src = block_pattern("mask").sub(lambda _: render("mask", mask), src)
        m, k = parse("masked", src), parse("mask", src)
        if bytes(b ^ k[i % len(k)] for i, b in enumerate(m)).decode("ascii") != key:
            sys.exit(f"{f}: round-trip check failed, nothing written")
        patched[f] = src

    for f, src in patched.items():
        f.write_text(src)
        print(f"patched  {f}")

    if test:
        print("test ok")
        return
    print()
    print("Add this line to ~/nps-api/.env on the server (replacing any API_KEYS line):")
    print()
    print(f"API_KEYS={key}")
    print()
    print("Then:  cd ~/nps-api && docker compose up -d")


if __name__ == "__main__":
    main(sys.argv[1:])
