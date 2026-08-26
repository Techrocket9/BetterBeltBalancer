#!/usr/bin/env python3
"""Factorio's own mod-settings.dat, written from scratch.

THE ONLY WAY TO ASK A SETTINGS STAGE A QUESTION FROM OUTSIDE, and since the
bench harness's setup mod became a compiled guest, the only way to CONFIGURE
one either. There is no command-line flag for a mod setting and no Lua this
repository is allowed to write.

Two callers, and they want the same eleven lines for two different reasons:

  test/check-datastage.py   drives the shipped mod's own startup settings, so
                            that the data-stage gate can hash what a NON-DEFAULT
                            cost setting emits as well as what the default does.

  bench/run.sh              configures bbb-bench-setup per matrix cell. That mod
                            was configured by a `config.lua` the harness rewrote
                            inside the staged copy; a Go guest cannot require a
                            Lua file, so the eight keys are startup settings and
                            this is the writer. See agents/estate-port.md,
                            phase 7.

THE FORMAT is a `PropertyTree`, and it is stable and documented: eight bytes of
version (four little-endian uint16), one bool byte, then one node. A node is a
type byte, an "any-type" byte, and a payload -- 1 bool, 2 double, 3 string,
4 list, 5 dictionary. A string is an empty-flag byte and then a
space-optimised length (one byte, or 255 followed by a uint32). A dictionary is
a uint32 count and then that many key-string / node pairs.

VERIFIED BY ROUND-TRIPPING THE ENGINE'S OWN FILE before it was ever used to
write one: the mod-settings.dat in this machine's Factorio mods directory parses
to exactly the three-key dictionary below, consuming every byte. That is a
stronger check than agreeing with a wiki page, and Factorio's own
mod-settings-dump.json from a `--dump-data` run is what says the engine agreed
with the writer.

As a command, for the shell caller:

    tools/mod-settings.py --out PATH --factorio-version 2.1.0 < settings.json

where the JSON on stdin is `{"startup": {...}, "runtime-global": {...}}` --
any section may be omitted and all three are written. JSON carries the three
types the tree needs (bool, number, string) without the shell having to declare
them, which is what makes an int setting arrive as a double and a bool as a
bool.
"""

import argparse
import json
import struct
import sys
from pathlib import Path

SECTIONS = ("startup", "runtime-global", "runtime-per-user")


def _string(text: str) -> bytes:
    raw = text.encode()
    if not raw:
        return b"\x01"                        # the empty-string flag
    if len(raw) < 255:
        return b"\x00" + bytes([len(raw)]) + raw
    return b"\x00\xff" + struct.pack("<I", len(raw)) + raw


def _node(v) -> bytes:
    if v is None:
        return b"\x00\x00"
    # bool BEFORE the number arm: in Python a bool IS an int, and a setting
    # written as a double where the engine wants a bool is not a setting.
    if isinstance(v, bool):
        return b"\x01\x00" + (b"\x01" if v else b"\x00")
    if isinstance(v, (int, float)):
        return b"\x02\x00" + struct.pack("<d", float(v))
    if isinstance(v, str):
        return b"\x03\x00" + _string(v)
    if isinstance(v, dict):
        out = b"\x05\x00" + struct.pack("<I", len(v))
        for k, x in v.items():
            out += _string(k) + _node(x)
        return out
    raise TypeError(f"no PropertyTree encoding for {type(v).__name__}")


def write_mod_settings(path: Path, version: str, startup: dict,
                       runtime_global: dict | None = None) -> None:
    """Write one mod-settings.dat.

    `version` is the FULL `X.Y.Z` the file stamps, which callers derive from the
    engine series they are staging for.
    """
    maj, minor, patch = (int(x) for x in version.split(".")[:3])
    # Every setting is wrapped in a one-key `{value = ...}` table, which is what
    # the engine's own file holds and what `settings.startup[name].value`
    # unwraps. All three sections must be present even when empty.
    tree = {
        "startup": {k: {"value": v} for k, v in startup.items()},
        "runtime-global": {k: {"value": v}
                           for k, v in (runtime_global or {}).items()},
        "runtime-per-user": {},
    }
    path.write_bytes(struct.pack("<HHHH", maj, minor, patch, 0) + b"\x00" + _node(tree))


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--out", required=True, type=Path)
    ap.add_argument("--factorio-version", required=True,
                    help="the X.Y.Z the file is stamped with")
    args = ap.parse_args()

    doc = json.load(sys.stdin)
    unknown = set(doc) - set(SECTIONS)
    if unknown:
        sys.exit(f"mod-settings: unknown section(s) {sorted(unknown)}; "
                 f"expected any of {list(SECTIONS)}")
    write_mod_settings(args.out, args.factorio_version,
                       doc.get("startup", {}), doc.get("runtime-global", {}))
    return 0


if __name__ == "__main__":
    sys.exit(main())
