#!/usr/bin/env python3
"""The DATA-STAGE EQUIVALENCE GATE: Factorio's own prototype table, hashed.

This mod's settings and data stages used to be ten hand-written Lua files under
mod-data/ and are a compiled Go guest now (guest/go/data). That port had to be
BEHAVIOR-PRESERVING to the byte, and the only instrument that can say so is the
engine's own dump of what the stages produced:

    factorio -c <private config> --mod-directory <staged mods> --dump-data

which writes script-output/data-raw-dump.json and mod-settings-dump.json and
STOPS BEFORE control.lua. That last property is why this exists as its own gate
rather than as a leg of run.sh: it is a pure data-stage instrument, it costs one
Factorio run of about three seconds per arm, and it answers a question no suite
in test/run.sh can even ask -- every one of them measures the RUNTIME.

  ./test/check-datastage.py            verify against the recorded goldens
  ./test/check-datastage.py --capture  record them (do this BEFORE a port)
  ./test/check-datastage.py --diff     on a mismatch, write both normalised
                                       dumps and print the jq -S diff

WHY THE WHOLE DUMP IS HASHED AND NOT ONLY THIS MOD'S PROTOTYPES. A data stage
can reach anything: `prototypes/technology.lua` READ base's `logistics` unit,
and the Lua it was ported from held that unit table BY REFERENCE -- so a later
edit to this mod's own copy would have silently edited base's technology. The
port kills that hazard by construction (fkdata.Get hands the guest a copy), and
hashing the whole dump is what would have CAUGHT it. A subset hash is blind to
the entire class of defect where a data stage damages somebody else's
prototypes.

STAMPING IS A NO-OP ON THE ARM THAT SHIPS, AND IT IS KEPT FOR THE ONE THAT DOES
NOT. Trunk targets 2.1 and the binary here is 2.1, so the staged manifests are
already right and nothing is rewritten -- the primary path is unstamped, and
stamp_engine says so by only writing a file it actually had to change. What it
is for is the `release/2.0` recut, where a 2.1-pinned tree has to be dumped by a
2.0 binary to capture the other flavour's golden.

That is legitimate HERE and is refused in run.sh, and the asymmetry is the
point: run.sh gates the packaged mod because the control guest's bindings are
pinned to one API and the ABI marshals event payloads BY NAME, so a mismatch
loads and then reads a mandatory field as nil. NONE OF THAT APPLIES to a
--dump-data run, which never reaches control.lua at all. What runs is the
settings and data stages, whose only engine dependency is the one this mod reads
explicitly -- `mods["base"]`, in guest/go/data/engine.go -- and that is a fact
about the BINARY rather than about the manifest.

TWO GOLDEN ARMS, AND THE SECOND ONE IS THE HALF THAT COULD ROT SILENTLY.

  base        this mod alone. The legacy stub IS defined (nobody else owns
              `balancer-part`), so data-final-fixes emits three prototypes.
  incumbent   this mod plus the `mig` suite's Belt Balancer stand-in, staged
              under an incumbent's own name. It owns `balancer-part`, so the
              stub branch takes its OTHER arm and emits nothing at all -- which
              is the "leave it alone while it is installed" half of the
              migration, and a one-armed gate would never have looked at it.
              Because the hash covers the WHOLE dump, this arm is also a
              byte-level statement about the stand-in's own prototypes, which
              is what carried them through their port to Go (`staged_mod`).

A GOLDEN IS PER ENGINE AND PER MOD SET, and says so in the file: the dump
contains every prototype every mod defined, base's own bundled data included, so
a machine with different DLC produces a different hash for a mod that is
perfectly fine. A golden line whose engine does not match the binary is a SKIP
with a message, never a failure.

...AND FOURTEEN VARIANT ARMS (ONE OF THEM THE VANILLA CONTROL) AND A SPEED ARM,
WHICH ARE NOT HASHED.

0.3.1 made the recipe's cost, the research's cost and the hidden network's belt
speed depend on things a golden cannot hold still. A hash is the right
instrument for "nothing moved"; it is the wrong one for "this moved to exactly
that", because a hash that changed tells you nothing about WHAT changed and has
to be re-captured by whoever moved it -- which is the one thing a golden must
never make easy. So:

  the DEFAULT settings are hashed, as before. Every recorded number in this
  repository was measured on them and the hash is what says they did not drift.

  every NON-DEFAULT value of both settings gets its own arm, one variable at a
  time, and a TARGETED assertion: the recipe's exact ingredient list, or the
  technology's exact unit and prerequisite. Nothing else in the dump is looked
  at, because everything else is the golden's business.

  the RECIPE CUSTOMIZER gets three more (0.3.3), because its states are not
  values of the dropdown alone: `custom` with the text untouched, `custom` with
  a recipe written into the text, and an edited text under a preset. The last
  two also assert a line of the LIBRARY'S OWN LOG, which is the only thing in
  this file that is not read out of a dump -- what the library does with a text
  under a preset is by design invisible in the prototypes, so the sentence is
  the whole of the evidence. See RECIPE_CUSTOM_ARMS.

  the RESEARCH CUSTOMIZER gets three more (0.3.3): two assert the PAIR a
  written cost decides, the unit and the prerequisite, because a cost the
  player wrote has no source technology and the tree position comes from the
  plan's own ladder rather than from the unit it copied; the third is a pack
  text edited under a tier, which is `recipe-ignored` for the research and
  asserts the tier's own unit beside the library's line. See TECH_CUSTOM_ARMS
  and TECH_IGNORED_ARMS.

  the SPEED derivation gets an arm with a mod in it that defines a faster belt,
  because no mod set this machine can otherwise install has one -- vanilla tops
  out at turbo, 0.125, which is HALF this mod's floor, so on every other arm the
  correct behaviour and a derivation that does nothing at all are the same dump.

...AND THE LEGACY STUB'S PLACING GRAPH, ON BOTH GOLDEN ARMS, WHICH IS ALSO NOT
HASHED AND FOR THE SAME REASON. `items_to_place_this` is not a field any data
stage writes -- the engine derives it, from every item whose `place_result` is
the entity plus the item its own `placeable_by` names -- so a hash holds it still
without ever saying what shape it is holding. The shape that matters is that
exactly ONE item places `bbb-balancer-part`: a second one closes a cycle between
that name and the stub's, and a mod walking item -> entity -> items_to_place_this
hangs on it. See check_legacy_stub.

A VARIANT ARM DRIVES THE REAL SETTING, through a `mod-settings.dat` this script
writes into the staged mods directory. That file is Factorio's own binary
property tree -- an eight-byte version, a bool, and a three-key dictionary of
`{value = ...}` wrappers -- and `write_mod_settings` below is a writer for it,
verified by round-tripping the engine's own file before it was used. There is no
Lua anywhere in this and there must not be: a settings stage cannot be asked a
question from outside except through this file.

Its anti-vacuity is structural rather than added. If the .dat were ignored, or
malformed enough to be skipped, every variant arm would read back the DEFAULT
recipe -- and the assertion is an equality against the variant's own ingredient
list, so it fails and names it. There is no way for these arms to pass while
measuring nothing.

THE 2.0 FLAVOUR IS DEFERRED AND THE COMMAND IS BELOW. Both of this mod's
version-gated branches key on the RUNNING ENGINE (`mods["base"]` is 2.0.x or it
is not), so a 2.1 binary can only ever produce the 2.1 flavour: no
`not_colliding_with_itself` on the linked belt, no `bbb-can-stack` marker, no
`bbb-multi-edge-parts` setting. The 2.0 flavour -- all three PRESENT -- is
unreachable here and is captured wherever a 2.0 binary is, which is the
`release/2.0` recut. See DEFERRED_OTHER_FLAVOUR below.

The branch itself does not wait for that. `guest/go/data/engine.go` is ordinary
Go and `go test ./data/` proves every arm of it -- 2.0.x true, 2.1.x false, and
false-safe for anything it cannot read -- which is the same argument
guest/go/edgemode makes for the runtime half: a fold whose interesting states
live on an engine this machine cannot run belongs somewhere `make check` can
reach it. What the deferred dump adds over that is the PROTOTYPES the true arm
emits, not the decision to emit them.
"""

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
GOLDENS = ROOT / "test" / "datastage-goldens.json"

DEFERRED_OTHER_FLAVOUR = """\
THE 2.0-FLAVOUR GOLDEN IS NOT CAPTURED, and it cannot be on this binary: the
branch keys on the RUNNING ENGINE, so a 2.1 binary produces the 2.1 flavour
whatever the manifest says. Wherever a Factorio 2.0 binary is -- which is the
`release/2.0` recut:

    make mod && FACTORIO_BIN=/path/to/2.0/factorio ./test/check-datastage.py --capture

and commit test/datastage-goldens.json, which is keyed by engine and holds one
line per engine by construction. What it pins that 2.1 cannot: the linked belt
WITH `not_colliding_with_itself`, the `bbb-can-stack` marker prototype, and a
mod-settings dump carrying `bbb-multi-edge-parts` -- the three things the
version branch turns ON, and the only three prototype-level differences between
the two arms this mod ships.

Until then the DECISION is covered by `go test ./data/` (engine_test.go) and
the EMISSION is not. That is a narrower gap than it reads: the true arm adds one
key to one collision mask and one prototype, both of which are ordinary Extend
and Set calls that the false arm's own dump already proves the shape of.
"""

# The DLC set, all off. A base-only dump is one fewer variable, and nothing in
# this mod's data stage reads a feature flag or a DLC prototype -- the four
# clones are of base's own belts. run.sh makes the same default for the same
# reason.
DLC = {"elevated-rails": False, "quality": False, "space-age": False}

# The stand-in's directory name has to match the mod name inside its info.json:
# Factorio requires it. `belt-balancer-2` is one of the four names
# guest/go/legacy.go recognises, and which of the four is immaterial here --
# what the arm exercises is somebody ELSE owning `balancer-part`.
INCUMBENT = "belt-balancer-2"

ARMS = {
    "base": [],
    "incumbent": [INCUMBENT],
}


def staged_mod(name: str) -> Path:
    """Where a staged mod's source directory is, which moved in phase 6.

    THE STAND-IN IS A COMPILED PACKAGE NOW, not a directory of hand-written Lua
    (agents/estate-port.md, phase 6), so it lives under dist/obs/<name>_<version>
    and has to be BUILT before this gate can stage it. `make datastage-check`
    names that ONE package as a prerequisite rather than all twelve observers,
    and running this script by hand against a tree that has never built it is
    what the error below is for.

    The version is globbed for the reason run.sh's `copy_testmod` globs it: it is
    the stand-in's own and has nothing to do with this mod's.

    WHY THIS ARM CARES AT ALL, given that what it is testing is OUR stub branch
    taking its other arm: the golden hashes the WHOLE dump, incumbent prototypes
    included. So this arm's hash is also a byte-level statement about the
    stand-in itself -- which is what made it the sharpest gate the port of the
    stand-in had, and what makes a stale dist/obs an error rather than a
    fallback to something that would silently hash differently.
    """
    hits = sorted((ROOT / "dist" / "obs").glob(f"{name}_*"))
    if hits:
        return hits[0]
    sys.exit(f"no package under dist/obs for {name}.\n"
             f"The stand-in is a compiled data module since phase 6 of the estate "
             f"port; build it with `make observers`.")

# ---------------------------------------------------------------------------
# THE VARIANT ARMS: one per non-default value of each cost setting.
#
# ONE VARIABLE AT A TIME. `bbb-recipe-cost` and `bbb-tech-cost` decide two
# different prototypes and could be driven together in half the runs; they are
# not, because an arm that moved two things is an arm whose failure does not say
# which. Seven Factorio runs at about three seconds each is the price of a
# failure that names itself.
#
# THE EXPECTED INGREDIENT LISTS ARE WRITTEN OUT HERE, not derived from
# guest/go/tune. A gate that computed the answer from the same source as the
# thing under test would agree with a defect -- it is the same reason the test
# mods in test/run.sh assert the guest's own log lines rather than recomputing
# what the guest should have said. These are what a player gets, transcribed
# from the option's own locale label.
# ---------------------------------------------------------------------------

RECIPE_SETTING = "bbb-recipe-cost"
RECIPE_TEXT_SETTING = "bbb-recipe-ingredients"
TECH_SETTING = "bbb-tech-cost"

RECIPE_VARIANTS = {
    "cheap": [("iron-plate", 2), ("transport-belt", 1)],
    "belt-fast": [("iron-plate", 4), ("iron-gear-wheel", 2),
                  ("fast-transport-belt", 2)],
    "belt-express": [("steel-plate", 4), ("iron-gear-wheel", 2),
                     ("express-transport-belt", 2)],
    "splitter": [("splitter", 1), ("iron-plate", 2)],
    "splitter-express": [("express-splitter", 1), ("steel-plate", 2)],
}

# The default, for the one assertion that has to be made about it here as well
# as by the golden: an arm that wrote NO mod-settings.dat and one that wrote the
# default must produce the same recipe, or the writer is doing something.
RECIPE_DEFAULT = [("iron-plate", 4), ("iron-gear-wheel", 2), ("transport-belt", 2)]

# ---------------------------------------------------------------------------
# THE CUSTOMIZER'S ARMS, which are the seventh value of `bbb-recipe-cost` and
# the text setting beside it.
#
# THREE ARMS FOR THREE STATES, and none of them is reachable from the six above:
# the dropdown on `custom` with the text left alone, the dropdown on `custom`
# with a recipe written into the text, and a text EDITED while the dropdown is
# still on a preset -- which is the state where nothing must happen and the log
# has to say so.
#
# TWO OF THEM ALSO ASSERT A LOG LINE, and that is the one thing in this file
# that is not a prototype. What the library does about a text under a preset is
# by design invisible in the dump -- the preset applies, exactly as it would
# with the field untouched -- so the ONLY evidence that the text was read and
# deliberately not used is the sentence, and a player who edits a field and sees
# nothing happen is what that sentence exists to prevent. The line is compared
# from `fkrecipes:` onward, never including the engine's own timestamp in front
# of it.
#
# ANTI-VACUITY, AND IT IS NOT SPREAD EVENLY ACROSS THE THREE. A .dat that was
# ignored or malformed leaves every setting at its declared default, which is
# the recipe `RECIPE_DEFAULT` names -- so the arms that prove the file was read
# at all are the two whose expectation is something else. `recipe-custom` is
# one: `3 iron-plate, 1 splitter` is a list no preset produces, and its log line
# says the text was taken. `recipe-ignored` is the other: cheap's list, where a
# text that had been applied would give one iron plate rather than two, plus its
# own line.
#
# `recipe-custom-default` CARRIES NONE OF THAT WEIGHT and is here as coverage of
# the documented path -- `custom` with the field never touched. MEASURED on
# Factorio 2.0.77 with the packaged mod: the run with NO .dat at all and the run
# with {bbb-recipe-cost: custom} produce the same ingredient list and no
# `fkrecipes:` line in either, so nothing this arm observes tells a .dat that
# was read from one that was ignored. It stays because picking `custom` and
# typing nothing is a state a player reaches and this gate should visit; what
# makes the section a measurement is the two arms above it.
#
# The text is written as a plain string by tools/mod-settings.py, which is what
# a player's own settings screen stores.
# ---------------------------------------------------------------------------

RECIPE_CUSTOM_ARMS = [
    # `custom` with the text as it ships: the reserved word `default`, which the
    # library resolves to the mod's own declared list WITH its ladders. So this
    # is the vanilla recipe, and no line at all is expected about it -- the
    # library's rule for the word is that it takes the pre-existing path and
    # says nothing.
    ("recipe-custom-default", {RECIPE_SETTING: "custom"}, RECIPE_DEFAULT, None),
    # `custom` with a recipe written into the field. `splitter` is a base item
    # no other arm of this file emits, so the list cannot be confused with a
    # preset's, and the amounts are the player's rather than any plan's.
    ("recipe-custom",
     {RECIPE_SETTING: "custom", RECIPE_TEXT_SETTING: "3 iron-plate, 1 splitter"},
     [("iron-plate", 3), ("splitter", 1)],
     "fkrecipes: bbb-balancer-part takes its ingredients from "
     "bbb-recipe-ingredients: 3 iron-plate, 1 splitter"),
    # A text edited while the dropdown says `cheap`. The preset wins, and the
    # expected list is cheap's: the text names the SAME item at a different
    # amount, so an implementation that took it would fail the list comparison
    # and not only the line.
    ("recipe-ignored",
     {RECIPE_SETTING: "cheap", RECIPE_TEXT_SETTING: "1 iron-plate"},
     RECIPE_VARIANTS["cheap"],
     "fkrecipes: bbb-recipe-ingredients is edited, but bbb-recipe-cost is not "
     "on custom, so the text is ignored"),
]

# The non-default technologies. The expected UNIT is not written out, because
# the claim is not "20 automation science" -- it is "whatever base charges for
# that technology", which is the whole reason the cost is read rather than
# pinned. So the assertion compares this mod's unit against the SOURCE
# TECHNOLOGY'S OWN unit in the same dump, which is a statement no transcription
# could make.
TECH_VARIANTS = ["logistics-2", "logistics-3"]

# ---------------------------------------------------------------------------
# THE RESEARCH CUSTOMIZER'S ARMS, which are the fourth value of `bbb-tech-cost`
# and the three fields beside it.
#
# THE EXPECTED UNIT IS WRITTEN OUT HERE, WHERE TECH_VARIANTS' IS NOT, and the
# reason is the whole difference between the two sections. A tier's cost is
# whatever base charges for that technology, so it is compared against that
# technology in the same dump; a CUSTOM cost is three numbers the player typed,
# so the only honest expectation is those three numbers. The shape is the
# engine's own short tuple form, `{count, time, ingredients={{name, amount}}}`,
# which is what base's own units are written in and what the library emits.
#
# AND THE PREREQUISITE IS ASSERTED BESIDE IT, because a written cost has no
# source technology for the tree position to move with. It comes from the plan's
# `Position` ladder -- logistics-3, then logistics-2, then logistics -- and in a
# base game that is `logistics-3`, which is also what makes the first arm below
# a measurement rather than coverage.
#
# ANTI-VACUITY, AND BOTH ARMS CARRY IT, unlike the recipe customizer's three.
# A .dat that was ignored or malformed leaves `bbb-tech-cost` at `logistics`,
# whose unit in a stock game is 20 automation science over 15 seconds -- the
# SAME THREE NUMBERS the untouched custom arm expects, because the defaults are
# base's own logistics unit on purpose. What separates them is the
# PREREQUISITE: `logistics` for a .dat that did not arrive, `logistics-3` for
# one that did. The second arm separates them a second way, on numbers no tier
# charges, and both assert the library's own log line, which no tier emits at
# all.
#
# `bbb-tech-count` IS AN INT SETTING AND tools/mod-settings.py WRITES EVERY
# NUMBER AS A PROPERTY-TREE DOUBLE (type 2). FkRecipes measured on this same
# engine that an int setting reads a type-2 double and a type-6 signed int
# alike (agents/customizer-design.md, the mod-settings.dat table: "the int
# written as type 6 and as type 2 | both read as the same number"), and the
# second arm is where that is confirmed rather than assumed: a count the engine
# had rejected or reset would read back as the default 20 and fail the unit
# comparison by the one field it moved.
# ---------------------------------------------------------------------------

TECH_PACKS_SETTING = "bbb-tech-packs"
TECH_COUNT_SETTING = "bbb-tech-count"
TECH_SECONDS_SETTING = "bbb-tech-seconds"

TECH_CUSTOM_ARMS = [
    # `custom` with all three fields as they ship: the reserved word `default`
    # in the pack list, and the two numbers at the values the settings declare.
    # That is [FallbackUnit] in guest/go/tune, which is base's own `logistics`
    # cost, so picking Custom and typing nothing changes the price of nothing.
    ("tech-custom-default", {TECH_SETTING: "custom"},
     {"count": 20, "time": 15, "ingredients": [["automation-science-pack", 1]]},
     ["logistics-3"],
     "fkrecipes: bbb-balancer takes its research cost from bbb-tech-packs: "
     "count 20, time 15, packs 1 automation-science-pack"),
    # `custom` with all three written. `logistic-science-pack` is a pack no
    # tier of this mod's charges on its own, and 50 x 20s is a cost no
    # technology in a base game has, so neither the list nor either number can
    # be confused with something that was copied.
    ("tech-custom",
     {TECH_SETTING: "custom",
      TECH_PACKS_SETTING: "1 automation-science-pack, 1 logistic-science-pack",
      TECH_COUNT_SETTING: 50, TECH_SECONDS_SETTING: 20},
     {"count": 50, "time": 20,
      "ingredients": [["automation-science-pack", 1], ["logistic-science-pack", 1]]},
     ["logistics-3"],
     "fkrecipes: bbb-balancer takes its research cost from bbb-tech-packs: "
     "count 50, time 20, packs 1 automation-science-pack, 1 logistic-science-pack"),
]

# THE PACK TEXT EDITED UNDER A TIER, which is `recipe-ignored` for the research
# and the one state of the three fields the arms above cannot visit: all three
# moved while the dropdown still names a tier. The tier wins, so the expected
# unit is the SOURCE TECHNOLOGY'S OWN, read out of the same dump exactly as
# TECH_VARIANTS reads it, and the prerequisite is that tier; the library's line
# is the only evidence the text was read and deliberately not used.
#
# ANTI-VACUITY: the three values are ones logistics-2 does not charge -- two
# automation packs where it charges one of each, 50 units where it charges 200,
# 20 seconds where it charges 30 -- so a planner that took any one of them under
# a tier fails the unit comparison by that field, and not only the line. The
# two numbers are written into the .dat as well, and draw no line of their own:
# under a tier no number of this mod's enters the unit, so there is nothing to
# report about them, which is the claim guest/go/tune's
# TestAnEditedPackTextUnderATierIsIgnoredAndTheLogSaysSo makes on the host.
TECH_IGNORED_ARMS = [
    ("tech-ignored", "logistics-2",
     {TECH_SETTING: "logistics-2", TECH_PACKS_SETTING: "2 automation-science-pack",
      TECH_COUNT_SETTING: 50, TECH_SECONDS_SETTING: 20},
     "fkrecipes: bbb-tech-packs is edited, but bbb-tech-cost is not on custom, "
     "so the text is ignored"),
]

# ---------------------------------------------------------------------------
# THE SPEED ARM's fixture: a Factorio mod written in Go whose whole job is to
# define a belt faster than this mod's hidden network.
#
# Built here rather than committed. The alternative is half a megabyte of
# generated Lua in test/ that would have to be rebuilt by hand whenever FkLua's
# emitter moved, and this gate already requires the toolchain that builds it --
# `make datastage-check` depends on `make mod`.
#
# THE UNDERGROUND IS THE FASTER OF THE TWO (0.5 against the belt's 0.4), so the
# speed the assertion expects can only come from a scan that walks more than
# `transport-belt`. See test/fixtures/fastbelt.
# ---------------------------------------------------------------------------

FIXTURE_SRC = ROOT / "test" / "fixtures" / "fastbelt"
FIXTURE_NAME = "bbbt-fastbelt"
FIXTURE_VERSION = "0.0.1"
FIXTURE_SPEED = 0.5

# The four prototypes the compiler places, and the families they live in.
HIDDEN_BELTS = [
    ("linked-belt", "bbb-linked-belt"),
    ("transport-belt", "bbb-belt"),
    ("splitter", "bbb-splitter"),
    ("lane-splitter", "bbb-lane-splitter"),
]

# What they run at with nothing faster installed. guest/go/tune's SpeedFloor,
# transcribed for the same reason the ingredient lists are.
SPEED_FLOOR = 0.25


def engine_version(factorio: str) -> str:
    out = subprocess.run([factorio, "--version"], capture_output=True, text=True).stdout
    m = re.match(r"^Version: (\d+\.\d+\.\d+)", out)
    if not m:
        sys.exit(f"could not read a version out of `{factorio} --version`:\n{out}")
    return m.group(1)


def stamp_engine(info: Path, series: str) -> bool:
    """Point a staged manifest at the running engine. True when it had to move.

    The same two fields run.sh's stamp_engine moves, and for the same reasons:
    `factorio_version`, because a mod naming the other series is refused at the
    loader before a prototype is read; and `base >= X.Y.Z` clamped DOWN only
    when it names a series NEWER than this engine, so a dependency that is
    already satisfied keeps the digits it was written with.

    IT WRITES NOTHING WHEN NOTHING HAD TO CHANGE, which is what makes the
    shipping arm's path an unstamped one rather than a stamped one that happens
    to be idempotent. On trunk's own engine every staged manifest is already
    right and this returns False for all of them; the caller says so.
    """
    d = json.loads(text := info.read_text())
    maj, minor = (int(x) for x in series.split("."))
    moved = d.get("factorio_version") != series
    d["factorio_version"] = series
    deps = []
    for dep in d.get("dependencies", []):
        m = re.match(r"^(.*base\s*>=\s*)(\d+)\.(\d+)\.(\d+)\s*$", dep)
        if m and (int(m.group(2)), int(m.group(3))) > (maj, minor):
            dep, moved = f"{m.group(1)}{maj}.{minor}.0", True
        deps.append(dep)
    if deps:
        d["dependencies"] = deps
    if moved:
        info.write_text(json.dumps(d, indent=2) + "\n")
    del text
    return moved


def normalised_sha(path: Path, out: Path | None) -> str:
    """jq -S over the dump, then SHA-256.

    THE NORMALISATION IS NOT COSMETIC. Key order in the dump is INSERTION order,
    so two data stages that emit the same prototypes in a different `data:extend`
    order produce byte-different dumps that describe the same game -- which is
    exactly what a port from six Lua files to one Go hook does. jq -S sorts every
    object's keys at every depth, and it preserves a real field-value change
    (measured upstream: stack_size 1 -> 42 survives it).

    Not the engine's own `Prototype list checksum`, which is order-insensitive
    and would be the tempting shortcut: it is measured BLIND TO FIELD VALUES --
    it does not move when a stack size does. A gate that cannot fail on the
    defect class a port is most likely to produce is not a gate.
    """
    if not path.exists():
        sys.exit(f"no dump at {path}: the data stage did not complete")
    blob = subprocess.run(
        ["jq", "-S", "-c", ".", str(path)], capture_output=True, check=True
    ).stdout
    if out is not None:
        out.write_bytes(subprocess.run(
            ["jq", "-S", ".", str(path)], capture_output=True, check=True).stdout)
    return hashlib.sha256(blob).hexdigest()


# THE mod-settings.dat WRITER LIVES IN tools/ NOW, because it grew a second
# caller with a different question.
#
# `bench/run.sh` configures the bench harness's setup mod through startup
# settings since that mod became a compiled guest -- a Go guest cannot require
# the `config.lua` the harness used to rewrite -- so the PropertyTree writer is
# shared rather than transcribed. Its header carries the format and the
# round-trip that verified it; what is imported here is the same function this
# file used to define, with one optional argument added that this caller does
# not pass.
sys.path.insert(0, str(ROOT / "tools"))
from importlib import import_module as _import_module

write_mod_settings = _import_module("mod-settings").write_mod_settings


def build_fixture(series: str, out: Path) -> Path:
    """Build test/fixtures/fastbelt into a staged mod directory.

    A WHOLE FACTORIO MOD, COMPILED FROM GO, and the reason it is built rather
    than committed is in FIXTURE_SRC's own go.mod.

    IT HAS NO CONTROL STAGE, which is what this fixture always wanted and could
    not have. `fklua mod` used to take the control module as its one positional
    argument, so a data-stage-only mod could not be packaged at all
    (FKLUA-GAPS.md item 26) and this fixture carried an inert empty `main` --
    about 113 KB of generated Lua that was `require`d and never called -- to get
    round it. The control module is optional when the mod has a data one now, so
    the workaround is deleted and the package is what it says: prototypes.
    """
    fklua = os.environ.get("FKLUA", str(ROOT.parent / "FkLua" / "bin" / "fklua"))
    if not os.access(fklua, os.X_OK):
        sys.exit(f"fklua not found at {fklua} (set FKLUA); the speed arm needs it "
                 f"to build test/fixtures/fastbelt")
    if shutil.which("tinygo") is None:
        sys.exit("tinygo is not on PATH; the speed arm builds its fixture from Go")

    flags = ["-target=wasm-unknown", "-scheduler=none", "-gc=leaking", "-opt=2"]
    subprocess.run(["tinygo", "build", *flags, "-o", str(out / "data.wasm"),
                    "./datastage"], cwd=FIXTURE_SRC, check=True)
    subprocess.run(
        # No positional module and no --persist: both describe a control guest
        # and there is none. --persist, --gc and --fuel are REFUSED here rather
        # than ignored, which is why the flag went rather than being left to be
        # harmless.
        [fklua, "mod",
         "--data-module", str(out / "data.wasm"),
         "--name", FIXTURE_NAME, "--version", FIXTURE_VERSION,
         "--title", "BBB fast-belt fixture", "--author", "BetterBeltBalancer",
         "--description", "A belt faster than the hidden network, for the "
                          "data-stage gate's speed arm. Never shipped.",
         "--dependency", f"base >= {series}.0",
         "--factorio-version", series, "-o", str(out)],
        # CWD IS THE OUTPUT DIRECTORY, WHICH HAS NO fklua.toml IN IT, and that
        # is the whole point: `fklua mod` reads the manifest in its working
        # directory for every identity it was not given a flag for. Run from the
        # repository root it would package the fixture with THIS MOD's asset
        # tree merged in (`data = "mod-data"` is the default for --include), so
        # the fixture would carry this mod's graphics, locale and changelog. A
        # directory with no manifest is a fixture built from its flags alone.
        # (The manifest's `gc = "collected"` is harmless here either way: a
        # data-only package refuses the typed --gc flag and ignores the key,
        # because both describe a control guest and there is none.)
        cwd=str(out), check=True, capture_output=True, text=True)
    return out / f"{FIXTURE_NAME}_{FIXTURE_VERSION}"


def run_arm(arm: str, factorio: str, series: str, mod_dir: Path,
            keep: Path | None, extras: list[Path] | None = None,
            startup: dict | None = None, probe=None) -> dict:
    work = Path(tempfile.mkdtemp(prefix=f"bbb-datastage-{arm}-"))
    try:
        mods = work / "mods"
        mods.mkdir(parents=True)

        shutil.copytree(mod_dir, mods / mod_dir.name)
        stamped = stamp_engine(mods / mod_dir.name / "info.json", series)

        # An ARMS entry is a MOD NAME and is staged under it; an `extras` entry is
        # a built package already named the way its builder named it. The two are
        # spelled separately because the stand-in's directory under dist/obs
        # carries a version suffix and this arm's whole point is that Factorio
        # sees the incumbent under its own name.
        staged = [(staged_mod(e), e) for e in ARMS.get(arm, [])]
        staged += [(e, e.name) for e in (extras or [])]
        for extra, dest in staged:
            shutil.copytree(extra, mods / dest)
            stamped |= stamp_engine(mods / dest / "info.json", series)
        if stamped:
            print(f"  note {arm}: a staged manifest was re-stamped for {series}; "
                  f"this is the cross-series path, not the shipping one")

        mod_name = json.loads((mods / mod_dir.name / "info.json").read_text())["name"]
        extra_names = [json.loads((mods / dest / "info.json").read_text())["name"]
                       for _, dest in staged]
        entries = [{"name": "base", "enabled": True}]
        entries += [{"name": k, "enabled": v} for k, v in sorted(DLC.items())]
        entries += [{"name": mod_name, "enabled": True}]
        entries += [{"name": e, "enabled": True} for e in extra_names]
        (mods / "mod-list.json").write_text(json.dumps({"mods": entries}, indent=2))

        if startup:
            write_mod_settings(mods / "mod-settings.dat", series + ".0", startup)

        # A private write-data, so a concurrent Factorio -- another agent, an
        # open game -- cannot take the .lock out from under the run. Same trick
        # run.sh uses and the same reason.
        userdir = work / "userdir"
        (userdir / "config").mkdir(parents=True)
        config = userdir / "config" / "config.ini"
        config.write_text(
            "[path]\nread-data=__PATH__system-read-data__\n"
            f"write-data={userdir}\n\n[general]\nlocale=auto\n"
        )

        log = work / "dump.log"
        with log.open("w") as fh:
            rc = subprocess.run(
                [factorio, "-c", str(config), "--mod-directory", str(mods),
                 "--dump-data"],
                stdout=fh, stderr=subprocess.STDOUT,
            ).returncode
        text = log.read_text()
        if rc != 0:
            sys.stderr.write(text[-4000:])
            sys.exit(f"[{arm}] --dump-data exited {rc}")

        so = userdir / "script-output"
        keep_raw = keep_set = None
        if keep is not None:
            keep.mkdir(parents=True, exist_ok=True)
            keep_raw, keep_set = keep / f"{arm}-data-raw.json", keep / f"{arm}-settings.json"

        checksum = None
        m = re.search(r"Prototype list checksum:\s*(\S+)", text)
        if m:
            checksum = m.group(1)

        out = {
            "mods": [mod_name] + extra_names,
            "data_raw_sha256": normalised_sha(so / "data-raw-dump.json", keep_raw),
            "mod_settings_sha256": normalised_sha(so / "mod-settings-dump.json", keep_set),
            # WHAT THE LIBRARY SAID, for the arms whose behaviour is a sentence
            # rather than a prototype. FkRecipes routes its log lines through
            # fkdata into the engine's own log, and the engine stamps each one
            # with a time AND the Lua file and line it came from (measured on
            # 2.0.77: `   0.399 Script @__better-belt-balancer__/fk_data.lua:609:
            # fkrecipes: ...`). Both halves of that stamp move on their own --
            # the seconds every run, the line number whenever the emitted Lua
            # shifts -- so what is kept is the text from `fkrecipes:` onward,
            # which is the convention test/assert-upgrade.py already uses for
            # the guest's own `[BBB]` lines: match from the tag, never on what
            # the engine printed in front of it.
            "fkrecipes_lines": [line[line.index("fkrecipes:"):]
                                for line in text.splitlines()
                                if "fkrecipes:" in line],
            # A SMOKE TEST AND LABELLED AS ONE. It is over the prototype LIST, so
            # it is order-insensitive (convenient) and blind to field values
            # (disqualifying). Recorded because a move in it localises a failure
            # to "a prototype appeared or vanished" in one glance.
            "prototype_list_checksum": checksum,
        }
        # The probe runs while the work directory still exists, because a
        # data-raw dump is thirteen megabytes and copying it out to look at four
        # fields would cost more than the Factorio run did.
        if probe is not None:
            out["probe"] = probe(so / "data-raw-dump.json")
        return out
    finally:
        shutil.rmtree(work, ignore_errors=True)


def project(dump: Path, filt: str):
    """One small jq projection out of a big dump.

    `jq` rather than `json.load`, because the dump is thirteen megabytes of
    which every assertion here wants four fields: the projection is a
    hundredth of a second and the parse is most of a gigabyte of Python objects.
    """
    out = subprocess.run(["jq", "-c", filt, str(dump)],
                         capture_output=True, check=True).stdout
    return json.loads(out)


def ingredients_of(dump: Path) -> list:
    """This mod's recipe, as (name, amount) pairs in emitted order.

    ORDER IS PART OF THE ASSERTION. The plans in guest/go/tune are slices and
    the recipe is built by walking one, so a plan that emitted its ingredients in
    a different order would be a different recipe in the crafting UI while
    holding the same items -- which a set comparison would call equal.
    """
    got = project(dump, '.recipe["bbb-balancer-part"].ingredients')
    return [(i["name"], i["amount"]) for i in got]


# ---------------------------------------------------------------------------
# THE LEGACY STUB'S PLACING GRAPH, asserted on both golden arms rather than
# hashed.
#
# A hash says nothing moved and cannot say which shape it is holding still, and
# the shape here is the one that hung a player's game. `prototypes.entity[e]
# .items_to_place_this` is not a field a data stage writes: the engine builds it
# from every item whose `place_result` is `e`, PLUS the item named by `e`'s own
# `placeable_by`. So a stub item pointing at `bbb-balancer-part` while the stub
# entity's `placeable_by` points back at that same item makes the two names each
# other's placers -- and a third-party mod that walks item -> entity ->
# items_to_place_this while inserting what it finds into the table it is walking
# never terminates on a cycle. Measured on a reporter's save: Factorio spins in
# LuaEntityPrototype::luaReadItemsToPlaceThis forever, no crash and no log line,
# so nothing downstream of the data stage could ever have seen it.
#
# THE ASSERTION IS OVER THE WHOLE DUMP AND NOT OVER THIS MOD'S TWO PROTOTYPES.
# What must hold is that exactly one item in the game places `bbb-balancer-part`,
# and any mod's item is entitled to break that -- so the projection scans every
# prototype of every type, which is the same argument the golden hash makes one
# level up.
# ---------------------------------------------------------------------------

OUR_PART = "bbb-balancer-part"
STUB_NAME = "balancer-part"
STUB_MARKER = "bbb-legacy-stub"


def legacy_stub_of(dump: Path) -> dict:
    return project(dump, '{'
                   'stub_place_result: .item["%s"].place_result, '
                   'marker: (.["simple-entity"]["%s"] != null), '
                   'placers: ([.[] | objects | .[] | objects '
                   '| select(.place_result == "%s") | .name] | sort)}'
                   % (STUB_NAME, STUB_MARKER, OUR_PART))


def check_legacy_stub(arm: str, got: dict) -> bool:
    """Returns True on a failure, which is the shape main() already counts in."""
    bad = False
    marker, placers = got["marker"], got["placers"]
    place = got["stub_place_result"]

    # ONE ITEM PLACES OUR PART, IN EITHER ARM. This is the whole defect stated:
    # the cycle exists exactly when a second item does.
    if placers != [OUR_PART]:
        bad = True
        print(f"FAIL {arm}: `{OUR_PART}` is placed by {placers} and must be "
              f"placed by [{OUR_PART}] alone -- a second item here is a cycle "
              f"through `placeable_by` and hangs a mod that walks it")
    else:
        print(f"  ok   {arm} placers of {OUR_PART}: {placers}")

    if arm == "base":
        # NOBODY ELSE OWNS THE NAME HERE, so the stub is defined and its item
        # must place the STUB and not our part.
        if not marker:
            bad = True
            print(f"FAIL {arm}: no `{STUB_MARKER}`; the legacy stub was not "
                  f"defined on the one arm where nobody else owns the name, so "
                  f"every assertion here is vacuous")
        if place != STUB_NAME:
            bad = True
            print(f"FAIL {arm}: the `{STUB_NAME}` item places {place!r} and must "
                  f"place {STUB_NAME!r}; the guest swaps that stub for a real "
                  f"part on the next flush")
        elif marker:
            print(f"  ok   {arm} the {STUB_NAME} item places {place}")
    elif arm == "incumbent":
        # THE OTHER ARM OF THE STUB BRANCH. The marker is defined WITH the stub
        # entity and never on its own, so its absence is the branch not firing.
        # There is no equivalent signal for the stub ITEM and none is needed: a
        # second `item` of one name is a duplicate-name load failure, so this arm
        # would have died on `--dump-data exited 1` rather than passing quietly.
        if marker:
            bad = True
            print(f"FAIL {arm}: `{STUB_MARKER}` is defined while another mod "
                  f"owns `{STUB_NAME}`; this mod ate a still-installed "
                  f"neighbour's prototype")
        else:
            print(f"  ok   {arm} no legacy stub, the incumbent owns {STUB_NAME}")
    return bad


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--capture", action="store_true",
                    help="record the goldens for this engine rather than checking them")
    ap.add_argument("--diff", action="store_true",
                    help="keep the normalised dumps so a mismatch can be diffed")
    ap.add_argument("--arm", choices=sorted(ARMS), action="append",
                    help="run one golden arm (default: all of them)")
    ap.add_argument("--golden-only", action="store_true",
                    help="skip the variant and speed arms, which are not hashed")
    args = ap.parse_args()

    factorio = os.environ.get(
        "FACTORIO_BIN",
        str(Path.home() / "Library/Application Support/Steam/steamapps/common/"
                          "Factorio/factorio.app/Contents/MacOS/factorio"))
    if not os.access(factorio, os.X_OK):
        sys.exit(f"factorio not found at: {factorio} (set FACTORIO_BIN)")

    manifest = (ROOT / "fklua.toml").read_text()
    name = re.search(r'^name = "(.*)"$', manifest, re.M).group(1)
    version = re.search(r'^version = "(.*)"$', manifest, re.M).group(1)
    mod_dir = ROOT / "dist" / f"{name}_{version}"
    if not mod_dir.is_dir():
        sys.exit(f"no built mod at {mod_dir}; run `make mod` first")

    version_full = engine_version(factorio)
    series = ".".join(version_full.split(".")[:2])
    arms = args.arm or sorted(ARMS)
    keep = (ROOT / "test" / "tmp" / "datastage") if args.diff else None
    if keep is not None:
        shutil.rmtree(keep, ignore_errors=True)

    print(f"==> Factorio {version_full}; --dump-data over {len(arms)} golden arm(s)")
    got = {a: run_arm(a, factorio, series, mod_dir, keep, probe=legacy_stub_of)
           for a in arms}

    book = json.loads(GOLDENS.read_text()) if GOLDENS.exists() else {}

    if args.capture:
        # THE PROBE IS NOT A GOLDEN, AND NEITHER IS THE LOG. check_legacy_stub
        # and the customizer's arms compare them against rules written down in
        # this file, so recording either would invite the one thing a golden
        # must never make easy -- re-capturing the answer instead of reading it.
        # Only the hashes and the checksum are the engine's to record.
        book.setdefault(version_full, {}).update(
            {a: {k: v for k, v in g.items() if k not in ("probe", "fkrecipes_lines")}
             for a, g in got.items()})
        # setdefault, not assignment: an engine's note is its own provenance
        # story, often hand-corrected after the capture -- a recapture must not
        # overwrite it with a generic one.
        book[version_full].setdefault("_note", 'Captured by --capture; read the dump before trusting a fresh line. Per engine and per mod set: the dump carries every prototype every mod defined. Only the DEFAULT settings are hashed; the variant and speed arms assert values.')
        GOLDENS.write_text(json.dumps(book, indent=2, sort_keys=True) + "\n")
        for a in arms:
            print(f"  {a:<10} data-raw {got[a]['data_raw_sha256'][:16]}  "
                  f"settings {got[a]['mod_settings_sha256'][:16]}  "
                  f"list-checksum {got[a]['prototype_list_checksum']}")
        print(f"captured into {GOLDENS.relative_to(ROOT)}")
        if series != "2.0":
            print()
            print(DEFERRED_OTHER_FLAVOUR)
        return 0

    want = book.get(version_full)
    bad = False
    if want is None:
        # A MISSING GOLDEN IS A FAILURE, NOT A SKIP. The hashed arms are this
        # gate's primary instrument, and a run that measured nothing must not
        # exit 0 -- that is this repository's own "a check that skips is a
        # check that passed", met in the gate that was written to answer a
        # question no suite can ask. The variant and speed arms still run
        # below (they need no golden), so the report stays one report; the
        # exit code says the engine's golden is owed.
        bad = True
        want = {}
        print(f"FAIL: no golden for Factorio {version_full}. A dump is a "
              f"function of the engine and the mod set, so a hash from another "
              f"engine would fail for a mod that is perfectly fine. Read the "
              f"dump, then capture this engine's own:")
        print(f"      make mod && test/check-datastage.py --capture")
        print(f"      Recorded engines: {', '.join(sorted(k for k in book)) or '(none)'}")
        if series == "2.0":
            print()
            print(DEFERRED_OTHER_FLAVOUR)
    for a in arms:
        w, g = want.get(a), got[a]
        if w is None:
            print(f"SKIP {a}: no golden for this arm on {version_full}")
            continue
        if w["mods"] != g["mods"]:
            print(f"SKIP {a}: golden mod set {w['mods']} != {g['mods']}")
            continue
        for field in ("data_raw_sha256", "mod_settings_sha256"):
            if w[field] != g[field]:
                bad = True
                print(f"FAIL {a} {field}:\n  golden {w[field]}\n  got    {g[field]}")
                # A `_stale` note is a golden whose owner already knows a change
                # invalidated it and could not re-capture it -- a line for an
                # engine this machine does not have. It is a NOTE ON A FAILURE and
                # never a skip: an unrecapturable golden that stopped failing
                # would be an engine nobody is checking at all.
                if w.get("_stale"):
                    print(f"       known stale: {w['_stale']}")
            else:
                print(f"  ok   {a} {field} {g[field][:16]}")
        if w.get("prototype_list_checksum") != g.get("prototype_list_checksum"):
            # NOT a failure on its own -- it is blind to field values, so it is
            # weaker than the hash above and can only ever agree with it or be
            # less sensitive. Printed because a move localises the cause.
            print(f"  note {a}: prototype list checksum moved "
                  f"{w.get('prototype_list_checksum')} -> "
                  f"{g.get('prototype_list_checksum')} "
                  f"(a prototype appeared or vanished)")

    # THE STUB'S PLACING GRAPH, PER ARM, and it runs whether or not the golden
    # matched. A hash that moved is exactly when somebody wants to know which
    # shape the dump is in, and this arm is the one shape a hash was never going
    # to name on its own.
    for a in arms:
        bad |= check_legacy_stub(a, got[a]["probe"])

    if bad:
        if keep is not None:
            print(f"\nnormalised dumps kept under {keep.relative_to(ROOT)}; "
                  f"diff them against a golden run's")
        else:
            print("\nre-run with --diff to keep the normalised dumps")

    # THE VARIANT ARMS RUN EVEN WHEN A GOLDEN MOVED, deliberately. A default
    # that drifted moves the hash AND every arm downstream of it, and the hash
    # alone does not say which prototype -- so stopping here would throw away
    # the eight lines that name it. One report, one exit code.
    if not args.golden_only:
        bad |= check_variants(factorio, series, mod_dir)
        bad |= check_speed(factorio, series, mod_dir)

    if bad:
        return 1
    print("check-datastage: ok")
    return 0


def check_variants(factorio: str, series: str, mod_dir: Path) -> bool:
    """Every non-default value of both cost settings, one arm each.

    Returns True on a failure, which is the shape main() already counts in.
    """
    print(f"==> the cost settings, "
          f"{len(RECIPE_VARIANTS) + len(TECH_VARIANTS) + len(RECIPE_CUSTOM_ARMS) + len(TECH_CUSTOM_ARMS) + len(TECH_IGNORED_ARMS) + 1} "
          f"variant arm(s)")
    bad = False

    # THE DEFAULT, WRITTEN EXPLICITLY, and it is not redundant with the golden
    # above. The golden arm writes NO mod-settings.dat at all, so it proves the
    # prototype's own `default_value`; this one writes `vanilla` through the same
    # writer every variant uses. A writer that produced a file the engine could
    # not read would make every variant arm silently fall back to the default and
    # this arm is the only one that could tell.
    for value, want in [("vanilla", RECIPE_DEFAULT)] + sorted(RECIPE_VARIANTS.items()):
        arm = f"recipe-{value}"
        got = run_arm(arm, factorio, series, mod_dir, None,
                      startup={RECIPE_SETTING: value},
                      probe=ingredients_of)["probe"]
        if got != want:
            bad = True
            print(f"FAIL {arm}: the recipe is {got}\n"
                  f"{'':>5}  and `{value}` should be {want}")
        else:
            print(f"  ok   {arm:<26} {got}")

    # THE CUSTOMIZER, three arms, each driving the real settings through the
    # same writer. See RECIPE_CUSTOM_ARMS for what each state is and why the
    # log line is part of two of them.
    for arm, startup, want, want_line in RECIPE_CUSTOM_ARMS:
        got = run_arm(arm, factorio, series, mod_dir, None,
                      startup=startup, probe=ingredients_of)
        ings, lines = got["probe"], got["fkrecipes_lines"]
        if ings != want:
            bad = True
            print(f"FAIL {arm}: the recipe is {ings}\n"
                  f"{'':>5}  and {startup} should be {want}")
            continue
        if want_line is not None and want_line not in lines:
            bad = True
            print(f"FAIL {arm}: the library's log lines are {lines}\n"
                  f"{'':>5}  and one of them has to be {want_line!r}")
            continue
        print(f"  ok   {arm:<26} {ings}"
              + (f"\n{'':>5}  said {want_line!r}" if want_line else ""))

    for value in TECH_VARIANTS:
        arm = f"tech-{value}"

        def probe(dump: Path, src=value):
            # THE SOURCE TECHNOLOGY'S OWN UNIT, out of the same dump. The claim
            # is not a number, it is "this costs what base charges for that
            # technology" -- so the comparison has to be against that technology
            # rather than against a figure transcribed here, which would go
            # stale the day base re-costs a tier.
            return project(dump, '{ours: .technology["bbb-balancer"], '
                                 'src: .technology["%s"]}' % src)

        p = run_arm(arm, factorio, series, mod_dir, None,
                    startup={TECH_SETTING: value}, probe=probe)["probe"]
        ours, src = p["ours"], p["src"]
        if src is None:
            bad = True
            print(f"FAIL {arm}: base has no `{value}` technology, so this arm "
                  f"proves nothing; the fallback would pass it")
            continue
        if ours["unit"] != src["unit"]:
            bad = True
            print(f"FAIL {arm}: the research unit is {ours['unit']}\n"
                  f"{'':>5}  and `{value}` charges {src['unit']}")
        elif ours["prerequisites"] != [value]:
            bad = True
            print(f"FAIL {arm}: the prerequisite is {ours['prerequisites']}, "
                  f"not [{value}] -- the unit moved and the tree position did not")
        else:
            u = ours["unit"]
            print(f"  ok   {arm:<26} {u['count']} x {u['time']}s, after {value}")

    # THE RESEARCH CUSTOMIZER, two arms, each asserting the unit AND the
    # prerequisite. See TECH_CUSTOM_ARMS for what each state is, why the
    # expected unit is written out where TECH_VARIANTS' is not, and what makes
    # each arm anti-vacuous.
    for arm, startup, want_unit, want_after, want_line in TECH_CUSTOM_ARMS:
        got = run_arm(arm, factorio, series, mod_dir, None, startup=startup,
                      probe=lambda d: project(d, '.technology["bbb-balancer"]'))
        ours, lines = got["probe"], got["fkrecipes_lines"]
        if ours["unit"] != want_unit:
            bad = True
            print(f"FAIL {arm}: the research unit is {ours['unit']}\n"
                  f"{'':>5}  and {startup} should be {want_unit}")
            continue
        if ours.get("prerequisites") != want_after:
            bad = True
            print(f"FAIL {arm}: the prerequisite is {ours.get('prerequisites')}, "
                  f"not {want_after} -- a written cost is placed by the plan's "
                  f"own ladder, and nothing else places it")
            continue
        if want_line not in lines:
            bad = True
            print(f"FAIL {arm}: the library's log lines are {lines}\n"
                  f"{'':>5}  and one of them has to be {want_line!r}")
            continue
        u = ours["unit"]
        print(f"  ok   {arm:<26} {u['count']} x {u['time']}s in "
              f"{[p[0] for p in u['ingredients']]}, after {want_after[0]}\n"
              f"{'':>5}  said {want_line!r}")

    # THE PACK TEXT EDITED UNDER A TIER. See TECH_IGNORED_ARMS: the tier's own
    # unit and prerequisite, read the way the tier arms read them, plus the line.
    for arm, tier, startup, want_line in TECH_IGNORED_ARMS:
        def tier_probe(dump: Path, src=tier):
            return project(dump, '{ours: .technology["bbb-balancer"], '
                                 'src: .technology["%s"]}' % src)

        got = run_arm(arm, factorio, series, mod_dir, None, startup=startup,
                      probe=tier_probe)
        ours, src, lines = got["probe"]["ours"], got["probe"]["src"], got["fkrecipes_lines"]
        if src is None:
            bad = True
            print(f"FAIL {arm}: base has no `{tier}` technology, so this arm "
                  f"proves nothing")
            continue
        if ours["unit"] != src["unit"]:
            bad = True
            print(f"FAIL {arm}: the research unit is {ours['unit']}\n"
                  f"{'':>5}  and `{tier}` charges {src['unit']}: a field the "
                  f"player wrote reached the unit under a tier")
            continue
        if ours.get("prerequisites") != [tier]:
            bad = True
            print(f"FAIL {arm}: the prerequisite is {ours.get('prerequisites')}, "
                  f"not [{tier}]")
            continue
        if want_line not in lines:
            bad = True
            print(f"FAIL {arm}: the library's log lines are {lines}\n"
                  f"{'':>5}  and one of them has to be {want_line!r}")
            continue
        u = ours["unit"]
        print(f"  ok   {arm:<26} {u['count']} x {u['time']}s, after {tier}\n"
              f"{'':>5}  said {want_line!r}")
    return bad


def check_speed(factorio: str, series: str, mod_dir: Path) -> bool:
    """The hidden network follows a modded belt faster than its floor.

    THE ONE ARM THAT NEEDS A MOD THIS REPOSITORY WROTE. Vanilla's fastest belt is
    turbo at 0.125, half the floor, so on every other arm in this file a correct
    derivation and one that did nothing produce the same dump -- which is why the
    default goldens are the no-change proof and this is the change proof.
    """
    print("==> the belt-speed derivation, 1 arm with a faster belt in it")
    work = Path(tempfile.mkdtemp(prefix="bbb-fastbelt-"))
    try:
        fixture = build_fixture(series, work)
        got = run_arm("speed", factorio, series, mod_dir, None, extras=[fixture],
                      probe=lambda d: project(d, "{ours: [%s], fixture: [%s]}" % (
                          ", ".join('.["%s"]["%s"].speed' % (t, n) for t, n in HIDDEN_BELTS),
                          '.["transport-belt"]["bbbt-fast-belt"].speed, '
                          '.["underground-belt"]["bbbt-fast-underground"].speed')))["probe"]
    finally:
        shutil.rmtree(work, ignore_errors=True)

    # ANTI-VACUITY FIRST. A fixture that failed to load, or one whose belts came
    # out at some other speed, would leave the four hidden prototypes at the
    # floor -- which is also what a broken derivation leaves them at. Check the
    # fixture's own belts before believing anything about ours.
    if got["fixture"] != [0.4, FIXTURE_SPEED]:
        print(f"FAIL speed: the fixture's own belts are {got['fixture']}, "
              f"not [0.4, {FIXTURE_SPEED}]; this arm proves nothing")
        return True
    if got["ours"] != [FIXTURE_SPEED] * len(HIDDEN_BELTS):
        print("FAIL speed: the hidden network runs at "
              f"{dict(zip((n for _, n in HIDDEN_BELTS), got['ours']))}\n"
              f"{'':>5}  and the fastest belt in that game is {FIXTURE_SPEED}. "
              f"{SPEED_FLOOR} means the derivation did not run or did not find it")
        return True
    print(f"  ok   speed{'':<22} all four hidden prototypes at {FIXTURE_SPEED}, "
          f"from an underground belt")
    return False


if __name__ == "__main__":
    sys.exit(main())
