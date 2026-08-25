#!/usr/bin/env python3
"""The changelog must parse under Factorio's own changelog.txt grammar.

Factorio parses mod changelogs strictly and a malformed one is DROPPED WHOLE:
the in-game changelog tab shows nothing, with no error anywhere a modder looks.
Headless never reads the file at all, so every suite and every benchmark passes
over a changelog the portal cannot display. This is the static half of that
check, against the published grammar
(https://lua-api.factorio.com/latest/auxiliary/changelog-format.html):

  - a version section starts with EXACTLY 99 dashes, and the very next line
    must be a `Version: X.Y.Z` line (each number 0-65535, never 0.0.0, no two
    sections sharing a version). Blank lines are skipped everywhere EXCEPT
    there;
  - at most one `Date: ` line per section, immediately allowed after Version;
  - a category line is two spaces, a name, and a trailing colon;
  - an entry is `    - text` (four spaces, dash, space) and must follow a
    category; a continuation is exactly six spaces plus text and must follow
    an entry. No duplicate lines within one version+category;
  - no tabs and no trailing whitespace anywhere.

Category names beyond the engine's recognized list are legal (they get their
own tab), so an unknown name is reported as a note rather than a failure.

The version cross-check is the half with teeth for a RELEASE: fklua.toml's
`version` is the identity of the package being built, and a bump without a
changelog section is exactly the drift this check exists to catch -- so the
manifest version must be the FIRST section of the changelog.

Usage: check-changelog.py <changelog.txt> [<expected-version>]
"""
import re
import sys
from pathlib import Path

SEPARATOR = "-" * 99
VERSION_RE = re.compile(r"^Version: (\d+)\.(\d+)\.(\d+)$")

# The engine's recognized categories, which sort ahead of ad-hoc ones in the
# GUI. Anything else is legal and gets its own tab; it is reported, not failed.
RECOGNIZED = {
    "Major Features", "Features", "Minor Features", "Graphics", "Sounds",
    "Optimizations", "Balancing", "Combat Balancing", "Circuit Network",
    "Changes", "Bugfixes", "Modding", "Scripting", "Gui", "Control",
    "Translation", "Debug", "Ease of use", "Info", "Locale", "Compatibility",
}


def main() -> int:
    if len(sys.argv) < 2:
        print("usage: check-changelog.py <changelog.txt> [<expected-version>]"
              "   (normally run by `make mod`)", file=sys.stderr)
        return 2
    path = Path(sys.argv[1])
    expected = sys.argv[2] if len(sys.argv) > 2 else None
    if not path.is_file():
        print(f"check-changelog: no such file: {path}", file=sys.stderr)
        return 2

    errors: list[str] = []
    notes: list[str] = []

    def err(n: int, msg: str) -> None:
        errors.append(f"line {n}: {msg}")

    lines = path.read_text(encoding="utf-8").splitlines()

    versions: list[str] = []          # in file order; [0] is the top section
    seen_versions: set[str] = set()
    in_section = False                # between a separator and the next one
    expect_version = False            # the line right after a separator
    have_date = False
    have_category = False
    have_entry = False                # an entry line seen since the category
    section_entries: set[str] = set() # (category, line) dedupe per section
    category = ""

    for n, raw in enumerate(lines, 1):
        if "\t" in raw:
            err(n, "tab character (the parser allows none)")
        if raw != raw.rstrip():
            err(n, "trailing whitespace (the parser allows none)")
        line = raw.rstrip("\n")

        if line.strip("-") == "" and line != "" and set(line) == {"-"}:
            if len(line) != 99:
                err(n, f"separator is {len(line)} dashes; the grammar wants "
                       f"exactly 99")
            in_section = True
            expect_version = True
            have_date = False
            have_category = False
            have_entry = False
            section_entries = set()
            category = ""
            continue

        if expect_version:
            m = VERSION_RE.match(line)
            if not m:
                err(n, "the line after a separator must be `Version: X.Y.Z` "
                       "(blank lines are not allowed there)")
                expect_version = False
                continue
            parts = tuple(int(g) for g in m.groups())
            if any(p > 65535 for p in parts):
                err(n, f"version component over 65535 in {line!r}")
            if parts == (0, 0, 0):
                err(n, "version 0.0.0 is invalid")
            v = ".".join(m.groups())
            if v in seen_versions:
                err(n, f"duplicate section for version {v}")
            seen_versions.add(v)
            versions.append(v)
            expect_version = False
            continue

        if line == "":
            continue  # skipped by the parser everywhere else

        if not in_section:
            err(n, "content before the first version separator")
            continue

        if line.startswith("Version: "):
            err(n, "a Version line may only follow a separator")
            continue

        if line.startswith("Date: "):
            if have_date:
                err(n, "second Date line in one version section")
            if have_category:
                err(n, "Date line after a category line")
            have_date = True
            continue

        if line.startswith("      ") and line[6:7] not in ("", " "):
            # continuation: exactly six spaces, then text
            if not have_entry:
                err(n, "continuation line without a preceding entry")
            if line in section_entries:
                err(n, f"duplicate line within one version section: {line!r}")
            section_entries.add(line)
            continue

        if line.startswith("    - "):
            if not have_category:
                err(n, "entry line without a preceding category line")
            if line[6:].strip() == "":
                err(n, "empty entry")
            key = f"{category}\x00{line}"
            if key in section_entries:
                err(n, f"duplicate entry within {category!r}: {line!r}")
            section_entries.add(key)
            section_entries.add(line)
            have_entry = True
            continue

        if line.startswith("  ") and not line.startswith("   "):
            name = line[2:]
            if not name.endswith(":"):
                err(n, f"category line must end with a colon: {line!r}")
            else:
                name = name[:-1]
                if name not in RECOGNIZED:
                    notes.append(f"line {n}: category {name!r} is not one the "
                                 f"engine recognizes (legal; it gets its own "
                                 f"tab)")
            category = name
            have_category = True
            have_entry = False
            continue

        err(n, f"unrecognized line shape (indentation is exact: 2 spaces for "
               f"a category, `    - ` for an entry, 6 spaces for a "
               f"continuation): {line!r}")

    if not versions:
        errors.append("no version sections at all -- an empty check reads "
                      "exactly like a pass")
    if expected and versions and versions[0] != expected:
        errors.append(f"the manifest says version {expected} and the top "
                      f"changelog section says {versions[0]} -- a release "
                      f"without its changelog section is the drift this "
                      f"check exists for")

    for note in notes:
        print(f"check-changelog: note: {note}")
    if errors:
        print(f"check-changelog: {path}: {len(errors)} problem(s) (Factorio "
              f"drops a malformed changelog whole, silently):",
              file=sys.stderr)
        for e in errors:
            print(f"  {e}", file=sys.stderr)
        return 1
    print(f"check-changelog: {len(versions)} version section(s), grammar ok"
          + (f", top section matches manifest {expected}" if expected else ""))
    return 0


if __name__ == "__main__":
    sys.exit(main())
