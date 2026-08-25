#!/usr/bin/env python3
"""Every sprite path the data stage names must exist in the packaged mod.

Headless Factorio never opens sprite files -- the graphical client is the
first thing that resolves them -- so a stale `filename =` passes every suite
and every benchmark and then kills the mod at the loading screen. This is the
static half of that check: walk every .lua in the packaged mod, pull out every
__better-belt-balancer__/... path, and require the file to exist in the
package. Paths into other mods (__base__, __core__) are the engine's problem
and are not checked here.

WHAT IT READS IS A COMPILED GUEST NOW, and that moved the regex. Every sprite
path this mod names used to sit in a hand-written Lua table between quotes;
since the data-stage purge they are string constants in the data guest's packed
data blob, laid END TO END with no delimiter of any kind. A GREEDY match then
runs straight off the end of one path, through whatever strings follow it, and
terminates on a LATER `.png` -- which is not a stale reference, it is one
enormous fabricated one, and it fails the build with a message nobody can act
on. Measured, the first time this ran against a packaged data guest.

So the path body is NON-GREEDY: it stops at the first extension after the mod
prefix, which in a blob of concatenated constants is the only correct answer.
Do not "tidy" the `+?` back to a `+`.

Usage: check-sprites.py <packaged-mod-dir>
"""
import re
import sys
from pathlib import Path

MOD_REF = re.compile(r'__better-belt-balancer__/([A-Za-z0-9_/.-]+?\.(?:png|ogg))')

def main() -> int:
    if len(sys.argv) < 2:
        print("usage: check-sprites.py <packaged-mod-dir>   (normally run by "
              "`make mod`)", file=sys.stderr)
        return 2
    root = Path(sys.argv[1])
    if not root.is_dir():
        print(f"check-sprites: not a directory: {root}", file=sys.stderr)
        return 2
    missing = []
    seen = 0
    for lua in sorted(root.rglob("*.lua")):
        text = lua.read_text(encoding="utf-8", errors="replace")
        for m in MOD_REF.finditer(text):
            seen += 1
            rel = m.group(1)
            if not (root / rel).is_file():
                missing.append(f"{lua.relative_to(root)}: __better-belt-balancer__/{rel}")
    if seen == 0:
        print("check-sprites: found ZERO mod-file references -- the regex or the "
              "package layout moved; an empty check reads exactly like a pass",
              file=sys.stderr)
        return 2
    if missing:
        print(f"check-sprites: {len(missing)} referenced file(s) missing from the "
              f"package (headless never notices; the graphical client refuses to load):",
              file=sys.stderr)
        for line in missing:
            print(f"  {line}", file=sys.stderr)
        return 1
    print(f"check-sprites: {seen} references, all present")
    return 0

if __name__ == "__main__":
    sys.exit(main())
