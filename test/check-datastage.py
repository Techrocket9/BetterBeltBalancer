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
is for is capturing the OTHER flavour's golden: this 2.1-pinned tree, dumped by
a 2.0 binary, with the staged info.json clamped down so the binary will load it.
That route is trunk's, not the `release/2.0` recut's, and it is how every 2.0
re-capture since the first has been taken.

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

...AND SIXTEEN VARIANT ARMS (ONE OF THEM THE VANILLA CONTROL), A SPEED ARM AND
A MERGE ARM, WHICH ARE NOT HASHED.

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

  the RECIPE'S TEXT FIELD gets four more, and since fix round 2 none of them is
  a value of the dropdown: the text is the SWITCH, applying whenever it does
  not say the reserved word `default`, and the dropdown decides while it does.
  The four states are the word under a preset that is NOT the default one, a
  recipe written with the dropdown untouched, a recipe written under a preset
  -- which is where the text WINS and the preset is set aside -- and the part
  named as its own ingredient, which asserts TWO lines where the three before
  it assert one or none. EVERY ONE OF THE FOUR ASSERTS THE INGREDIENT LIST,
  which the new rule made possible: in every arm here the list a player gets
  differs from the list a planner that read the wrong field would emit, so such
  a planner fails on the recipe and not only on a sentence. THE LIBRARY'S OWN
  LOG is asserted beside it, because the set-aside clause is the one part of
  this that no prototype carries. See RECIPE_TEXT_ARMS.

  the RESEARCH COST'S THREE FIELDS get four more, split by WHO SUPPLIES THE
  UNIT. One writes all three fields, so the whole cost is the player's and the
  expected unit is written out here; the other three leave at least one field
  saying "the row above decides" -- one of them all three, and the two between
  move a single field each, which is how three switches are told from one -- so
  their expected unit is the TIER'S OWN, read out of the same dump with
  whatever the player moved written over it. The PREREQUISITE is the tier's
  source technology in all four, however the cost was written, because a player
  who writes a cost has said what the research costs and nothing about where it
  sits. See TECH_COST_ARMS and TECH_TIER_ARMS.

  the SPEED derivation gets an arm with a mod in it that defines a faster belt,
  because no mod set this machine can otherwise install has one -- vanilla tops
  out at turbo, 0.125, which is HALF this mod's floor, so on every other arm the
  correct behaviour and a derivation that does nothing at all are the same dump.

  the LADDER MERGE gets an arm with a mod in it that DELETES an item, for the
  same reason turned around: no mod set this machine can install is missing
  `transport-belt`, and that is the pack where the vanilla ladder's fallback
  lands on a name the list already carries. What it proves is the half no host
  test can reach -- that the ENGINE accepts the merged list -- and it runs on
  the prototype's own default, no `mod-settings.dat` at all, because the player
  it is about never opened the Startup tab. See REMOVER_LUA and check_remover.

...AND THE SIX STARTUP SETTINGS' `order` STRINGS, ON BOTH GOLDEN ARMS, WHICH
ARE INSIDE THE HASH AND ARE ASSERTED ANYWAY. Four of the six are GENERATED names
placed by FkRecipes' `OrderAfter` since the sync pass, so their order strings are
the LIBRARY'S ARITHMETIC rather than this mod's transcription -- `a` and then two
letters, `b` and then two letters -- and a three-letter order is a thing the
engine had never been handed by this library before. The hash covers them and
says nothing about them: a `mod_settings_sha256` that moved says a hash moved,
where what a maintainer needs to read is which setting sits where. So the six
are pinned by name against a literal table, and the SORT they make is compared
against the declaration order, which is the whole of what a player sees in the
Startup tab. See check_settings_order.

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
property tree, and THE TOOLCHAIN OWNS THE WRITER: `write_mod_settings` below
builds a JSON document and hands it to `fklua modsettings write --from FILE.json
--out mod-settings.dat`, which encodes it and reads its own bytes back before it
writes them. This repository transcribed the format itself until the toolchain
answered it, and a format with one writer inside the compiler this mod is built
with does not get a second one here. There is no Lua anywhere in this and there
must not be: a settings stage cannot be asked a question from outside except
through this file.

THE TYPE OF A NUMBER IS ITS SPELLING, which is what the arm tables below have to
get right. A JSON integer is written as the property tree's signed 64-bit type
and a JSON float as its double, and that is the ENGINE'S OWN typing: the engine
rewrites mod-settings.dat after a load with the values it settled on, and an int
setting comes back from it as the signed 64-bit type. BOTH RESEARCH NUMBERS ARE
INT SETTINGS SINCE FIX ROUND 2 -- `better-belt-balancer-tech-seconds` changed
prototype type from `double-setting` to `int-setting` with it -- so a count is
written `50` and a seconds `20`, each as the type its own setting prototype
declares, and neither carries a decimal point any more.

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
unreachable here and is captured wherever a 2.0 BINARY is, from this tree
through the stamped path above. See DEFERRED_OTHER_FLAVOUR below.

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

# THE COMPILER, and this gate needs it for two different things: `modsettings
# write`, which is where every variant arm's .dat comes from, and `mod`, which
# packages the speed arm's fixture. The default is the Makefile's own, so a bare
# run of this script and `make datastage-check` look at the same binary.
#
# ABSOLUTE, AND THAT IS NOT COSMETIC. The Makefile's own FKLUA is the relative
# `../FkLua/bin/fklua` and it passes it in; `build_fixture` runs the packager
# with cwd set to a temporary directory (it must, or the fixture would be
# packaged with this mod's asset tree merged in), so a relative path there
# resolves against a directory that has no FkLua next to it. Measured the first
# time this variable was passed through: `FileNotFoundError: [Errno 2] No such
# file or directory: '../FkLua/bin/fklua'`. Resolved against THIS process's
# working directory, which is where a relative path a caller wrote means what
# the caller meant.
#
# `or` RATHER THAN A DEFAULT ARGUMENT, so an EMPTY `FKLUA=` in the environment
# defaults too (measured: `FKLUA= test/check-datastage.py` reached the probe
# with the empty string, which `os.access` read as the working directory).
FKLUA = os.path.abspath(
    os.environ.get("FKLUA") or str(ROOT.parent / "FkLua" / "bin" / "fklua"))

DEFERRED_OTHER_FLAVOUR = """\
THE 2.0-FLAVOUR GOLDEN IS NOT CAPTURED, and it cannot be on this binary: the
branch keys on the RUNNING ENGINE, so a 2.1 binary produces the 2.1 flavour
whatever the manifest says. Wherever a Factorio 2.0 BINARY is, from this tree,
because stamping makes the manifest loadable and the manifest is not what the
branch reads:

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
RECIPE_TEXT_SETTING = "better-belt-balancer-recipe-ingredients"
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
# THE RECIPE'S TEXT FIELD, WHICH IS THE SWITCH.
#
# FOUR ARMS FOR FOUR STATES, and none of them is a value of the dropdown any
# more. Fix round 2 withdrew `custom` from `bbb-recipe-cost` and moved the
# state into the text itself: `better-belt-balancer-recipe-ingredients` applies
# whenever it does not say the reserved word `default`, and while it does say
# it the dropdown beside it decides. So the four are the word under a preset
# that is NOT the default one, a recipe written with the dropdown untouched, a
# recipe written under a preset -- the state that used to be "nothing happens
# and the log says so" and is now the text winning -- and a recipe naming the
# balancer part as its own ingredient.
#
# THE WITHDRAWN VALUE IS WHY NONE OF THESE ARMS COULD BE LEFT ALONE. The engine
# SILENTLY RESETS a stored value a dropdown's `allowed_values` does not list,
# and persists the reset, with no line in the log. MEASURED HERE and not by the
# library: agents/migration-assessment-2.md finding 3 handed the AUTHENTIC
# 0.2.2 artifact a .dat head had written and read it back with `fklua
# modsettings read` -- exit 0, no `fkrecipes:` line, no Error, no Warning, and
# both dropdowns reset to `vanilla` and `logistics` with the reset persisted,
# while the four settings 0.2.2 does not declare survived verbatim. FkRecipes
# decision A is where that measurement was ADOPTED, and withdrawing the value
# is what it bought. An arm still
# storing `bbb-recipe-cost = custom` would therefore be driving the DEFAULT
# PRESET under a name that says otherwise: green, and measuring the one thing
# it was not about.
#
# EVERY ARM ASSERTS THE INGREDIENT LIST, WHICH IS NEW HERE. Under the old rule
# a text under a preset was by design invisible in the prototypes and the log
# line was the whole of the evidence; the text wins now, so the emitted list
# itself says which of the two fields the planner read. The LOG is asserted
# beside it because the SET-ASIDE CLAUSE is the part no prototype carries: it
# names the dropdown and the value being set aside, which is what a player goes
# looking for when the recipe is not what the row they are standing on says.
#
# THE STREAM IS COMPARED WHOLE AND IN ORDER ON ALL FOUR, where two of these
# arms used to be satisfied by a stream that merely contained their line. Every
# load here has a stream this file can state completely -- none, one or two
# lines -- so there is nothing left for the weaker rule to protect, and an extra
# line in a game that has everything is a degradation rather than noise to be
# filtered out. The line is compared from `fkrecipes:` onward, never including
# the engine's own timestamp in front of it.
#
# ANTI-VACUITY, AND IT IS ONE SENTENCE FOR ALL FOUR NOW, where the old table had
# to carry an arm that could not make the claim at all. A .dat that was ignored
# or malformed leaves every setting at its declared default, which is `vanilla`
# with the text on the reserved word, and that is the recipe `RECIPE_DEFAULT`
# names. NO ARM HERE EXPECTS THAT LIST. `recipe-default-word` expects cheap's,
# because the dropdown is the field it moved and the text is the field it left
# alone; the other three expect LISTS no preset of this mod emits. Two of them
# hold an item no preset holds at all, `bbb-balancer-part` in one and nothing a
# ladder of this mod can reach in either; the third, `recipe-text-alone`, holds
# the same two items as the `splitter` preset and is separated by the amounts
# and the order instead (3 iron-plate then 1 splitter against 1 splitter then 2
# iron-plate), which is a claim about the list and not about the item set. So
# every
# arm fails on its INGREDIENTS, not only on a sentence, if the file never
# arrived -- which is precisely what `recipe-custom-default` could not say:
# it stored `custom` and expected the default recipe, byte for byte what a run
# with no .dat at all produces, MEASURED on 2.0.77 and recorded here as its own
# weakness. The state it covered is `recipe-default-word` now, where the preset
# underneath is a different one and the list says which field decided.
#
# AND IT IS MEASURED RATHER THAN ARGUED. Every arm below was run once with
# `startup=None` -- which is the run a missing file produces, byte for byte --
# against this package on 2.0.77: the recipe came back
# `[('iron-plate', 4), ('iron-gear-wheel', 2), ('transport-belt', 2)]` with an
# EMPTY `fkrecipes:` stream, and all four arms fail on their ingredient list
# against it, `recipe-default-word` included.
#
# The text is written as a plain string, which is what a player's own settings
# screen stores.
# ---------------------------------------------------------------------------

RECIPE_TEXT_ARMS = [
    # THE RESERVED WORD, UNDER A PRESET THAT IS NOT THE DEFAULT ONE. `default`
    # is what the field ships saying and it means "the dropdown decides", so the
    # expected list is cheap's and the library says nothing at all -- the rule
    # for the word is that it takes the pre-existing path in silence. The preset
    # is `cheap` rather than `vanilla` for the reason the whole table turns on:
    # a planner that ignored the dropdown while the text said the word would
    # come out with the vanilla list, which is exactly what a .dat that never
    # arrived produces.
    ("recipe-default-word",
     {RECIPE_SETTING: "cheap", RECIPE_TEXT_SETTING: "default"},
     RECIPE_VARIANTS["cheap"], []),
    # A RECIPE WRITTEN WITH THE DROPDOWN UNTOUCHED, which is the state a player
    # who only ever opens the text field is in. THE ITEM SET IS NOT WHAT
    # SEPARATES IT: `RECIPE_VARIANTS["splitter"]` holds exactly these two items
    # and is driven three arms above as `recipe-splitter`. The AMOUNTS and the
    # ORDER are -- the preset emits `1 splitter, 2 iron-plate` where this arm
    # asks for `3 iron-plate, 1 splitter` -- and order is part of the
    # assertion here for the reason `ingredients_of` gives, so neither a preset
    # nor any plan of this mod can produce this list. THE
    # CLAUSE NAMES `vanilla`, the value the dropdown still holds: nothing moved
    # it, and the library reads it anyway in order to say what it set aside.
    ("recipe-text-alone",
     {RECIPE_TEXT_SETTING: "3 iron-plate, 1 splitter"},
     [("iron-plate", 3), ("splitter", 1)],
     ["fkrecipes: bbb-balancer-part takes its ingredients from "
      "better-belt-balancer-recipe-ingredients: 3 iron-plate, 1 splitter"
      "; the bbb-recipe-cost choice vanilla is set aside"]),
    # A RECIPE WRITTEN UNDER A PRESET, WHICH IS THE ROUND IN ONE ARM. This is
    # the state that used to emit cheap's list and log "the text is ignored";
    # the text wins now and cheap is what is set aside, and the sentence about
    # an edited field that does nothing exists nowhere any more because there
    # is no state left for it to be about. The text names the SAME item as the
    # preset at a DIFFERENT amount -- one iron plate against two and a belt --
    # so a planner that still preferred the preset fails on the list and not
    # only on the line. THE CLAUSE NAMES THE CHOICE AND NOT THE SETTING:
    # `cheap` is the row the player is standing on, which is the thing they
    # would go looking for; `bbb-recipe-cost` would name the dropdown rather
    # than the value it holds.
    ("recipe-text-over-preset",
     {RECIPE_SETTING: "cheap", RECIPE_TEXT_SETTING: "1 iron-plate"},
     [("iron-plate", 1)],
     ["fkrecipes: bbb-balancer-part takes its ingredients from "
      "better-belt-balancer-recipe-ingredients: 1 iron-plate"
      "; the bbb-recipe-cost choice cheap is set aside"]),
    # THE BALANCER PART NAMED AS ITS OWN INGREDIENT. THE LOAD STILL COMPLETES,
    # which is the half of this arm no host test can make: base 2.0.77 ships
    # `kovarex-enrichment-process`, which takes 40 `uranium-235` and gives back
    # 41, so a recipe naming its own product is a shape the engine accepts and
    # the library deliberately does not refuse. What it does is SAY SO, and
    # before FkRecipes 1f8e363 it did not: the load drew the ordinary "takes its
    # ingredients from" line and stopped there, so a player got a recipe nothing
    # can craft with no line about THAT. TWO LINES, WHOLE AND IN ORDER: the text
    # is read first and the self-product check runs on the FINAL list, so it
    # lands second. The first of them carries the set-aside clause like every
    # other line in this table, naming the `vanilla` the dropdown still holds;
    # the second does not, because it is about the list rather than about which
    # field produced it.
    ("recipe-self-product",
     {RECIPE_TEXT_SETTING: "2 bbb-balancer-part"},
     [("bbb-balancer-part", 2)],
     ["fkrecipes: bbb-balancer-part takes its ingredients from "
      "better-belt-balancer-recipe-ingredients: 2 bbb-balancer-part"
      "; the bbb-recipe-cost choice vanilla is set aside",
      "fkrecipes: bbb-balancer-part: bbb-balancer-part is in the list and is "
      "also what this recipe makes, so nothing can craft the first one unless "
      "something else produces it"]),
]

# The non-default technologies. The expected UNIT is not written out, because
# the claim is not "20 automation science" -- it is "whatever base charges for
# that technology", which is the whole reason the cost is read rather than
# pinned. So the assertion compares this mod's unit against the SOURCE
# TECHNOLOGY'S OWN unit in the same dump, which is a statement no transcription
# could make.
TECH_VARIANTS = ["logistics-2", "logistics-3"]

# ---------------------------------------------------------------------------
# THE RESEARCH COST'S THREE FIELDS, WHICH ARE THREE SWITCHES.
#
# WHAT FIX ROUND 2 MADE OF THEM. `bbb-tech-cost` lost its `custom` value and the
# three fields beside it decide one at a time: a pack text on the reserved word
# `default` and a number at 0 leave that field to the chosen tier, and anything
# else overwrites it. The two numbers ship at 0 now rather than at
# [FallbackUnit]'s 20 and 15, and `better-belt-balancer-tech-seconds` changed
# prototype type from `double-setting` to `int-setting` with them.
#
# THE TABLE IS SPLIT BY WHO SUPPLIES THE UNIT, which is the whole difference
# between the two below. TECH_COST_ARMS writes all three fields, so no field of
# the cost is the tier's and the expected unit is written out here -- three
# numbers the player typed are the only honest expectation for them.
# TECH_TIER_ARMS leaves at least one field saying "the row above decides", so
# its expectation is the TIER'S OWN UNIT read out of the same dump, with
# whatever the player moved written over it, which is TECH_VARIANTS' rule
# applied to a mixture.
#
# THE SHAPE IS THE ENGINE'S OWN SHORT TUPLE FORM, `{count, time,
# ingredients={{name, amount}}}`, which is what base's own units are written in
# and what the library emits.
#
# AND THE PREREQUISITE IS ASSERTED BESIDE THE UNIT IN EVERY ARM. It is the
# TIER'S SOURCE TECHNOLOGY however the cost was written: a player who writes a
# cost has said what the research costs and nothing about where it sits.
# `CustomCost.Position` is gone from the library with the dropdown value, so
# there is no second ladder left to place a written cost with, and the arms
# below no longer assert one.
#
# ANTI-VACUITY FOR TECH_COST_ARMS, AND ALL THREE OF ITS ASSERTIONS NOW MAKE IT,
# where the old table had only the log line. A .dat that was ignored or
# malformed leaves `bbb-tech-cost` at `logistics`, both numbers at 0 and the
# pack text on the word, which is the DEFAULT TIER DECIDING: base's own
# `logistics` unit, `logistics` as the prerequisite, and NOT ONE `fkrecipes:`
# line. The arm expects a unit no technology in a base game charges, the
# prerequisite `logistics-2`, and exactly one line. What used to be true and is
# not any more is the old note here: under `Position` a written cost was placed
# by the plan's own ladder, which started at this mod's default tier, so the
# untouched custom arm's unit AND prerequisite were what a missing file
# produces and only the line could tell. The tier supplies the placement now,
# so the prerequisite is a separator again -- which is why this arm's dropdown
# is on `logistics-2` and not left alone. MEASURED with the same `startup=None`
# run the recipe table names: the technology came back
# `{'count': 20, 'ingredients': [['automation-science-pack', 1]], 'time': 15}`
# after `logistics` with an empty stream, so this arm fails on its unit, on its
# prerequisite and on its line, each on its own.
#
# FOUR SETTINGS ARE WRITTEN AT ONCE AND THAT IS DELIBERATE, against this file's
# one-variable-at-a-time rule. The three cost fields are ONE COST, and the
# states where they can be read APART are TECH_TIER_ARMS' business rather than
# this arm's; the dropdown moves with them so the prerequisite and the clause
# name something that was READ rather than something that was never touched. No
# two of the four can be confused in the result: 50 units, 20 seconds and a
# three-of-one-plus-one-of-another pack list against a tier charging 200, 30
# and one of each, so a planner that dropped any one of them fails on that
# field by name.
#
# THE EXPECTED UNIT IS COMPLETE BECAUSE THE TIER'S IS. The library overwrites
# fields on the tier's own unit map rather than building a fresh one, so a
# `max_level` or a `count_formula` the tier carried would survive into this
# arm's prototype and a transcription that named only three fields would fail.
# MEASURED on 2.0.77, base-only `--dump-data`, `jq -c
# '.technology["logistics-2"].unit' script-output/data-raw-dump.json`:
# `{"count":200,"ingredients":[["automation-science-pack",1],
# ["logistic-science-pack",1]],"time":30}` -- exactly the three fields this arm
# writes over. If base ever gives that unit a fourth, the library keeps it and
# this arm fails naming it, which is a transcription to fix rather than a
# defect to chase.
#
# BOTH NUMBERS ARE INT SETTINGS AND BOTH ARE WRITTEN AS INTEGERS: the toolchain
# encodes a JSON integer as the property tree's signed 64-bit type (type 6),
# which is the type the engine itself writes an int setting back as. The
# seconds used to be spelled `20.0` here because the setting used to be a
# double; it is `20` now, and the READ is confirmed rather than assumed in
# every arm below -- a number the engine had rejected or reset would come back
# as the declared 0, which means "the tier decides", and fail the unit
# comparison by the one field it moved.
# ---------------------------------------------------------------------------

TECH_PACKS_SETTING = "better-belt-balancer-tech-packs"
TECH_COUNT_SETTING = "better-belt-balancer-tech-count"
TECH_SECONDS_SETTING = "better-belt-balancer-tech-seconds"

TECH_COST_ARMS = [
    # ALL THREE FIELDS WRITTEN UNDER A TIER, so the whole price is the player's
    # and the tier supplies nothing but the place in the tree. The pack list is
    # neither the tier's (one of each) nor this mod's declared default (one
    # automation science pack); 20 seconds is not the 30 the tier charges; and
    # THE PAIR is what separates the numbers, not the count. MEASURED over the
    # base-only dump on 2.0.77 with
    # `jq '[.technology[] | select(.unit.count == 50)] | length'`: NINETEEN of
    # the 196 technologies charge exactly 50, `steel-processing`, `landfill`,
    # `lubricant`, `speed-module` and fifteen others, so the count alone proves
    # nothing. `select(.unit.count == 50 and .unit.time == 20)` answers 0, and
    # so does `select(.unit.time == 20)` on its own -- no base technology is
    # priced at 20 seconds at all. So 50 units AT 20 SECONDS is a cost no
    # technology in a base game charges, and no field of the expected unit can
    # be confused with something that was copied.
    ("tech-cost-whole",
     {TECH_SETTING: "logistics-2",
      TECH_PACKS_SETTING: "3 automation-science-pack, 1 logistic-science-pack",
      TECH_COUNT_SETTING: 50, TECH_SECONDS_SETTING: 20},
     {"count": 50, "time": 20,
      "ingredients": [["automation-science-pack", 3],
                      ["logistic-science-pack", 1]]},
     ["logistics-2"],
     ["fkrecipes: bbb-balancer takes its research cost from "
      "better-belt-balancer-tech-packs: count 50, time 20, "
      "packs 3 automation-science-pack, 1 logistic-science-pack"
      "; the bbb-tech-cost choice logistics-2 supplies what the settings leave "
      "at default"]),
]

# ---------------------------------------------------------------------------
# THE ARMS WHERE THE TIER STILL SUPPLIES SOMETHING, and the expected unit is
# therefore the tier's own, read out of the same dump exactly as TECH_VARIANTS
# reads it, with the fields the player moved written over it.
#
# THREE ARMS, AND WHAT EACH ONE SEPARATES IS A DIFFERENT FIELD. Each of the
# three is a switch of its own, so a table that moved them only together could
# not tell three switches from one:
#
#   tech-tier-word     all three fields say "the row above decides" -- the word
#                      `default` and two zeros -- so the technology is priced
#                      by the tier BYTE FOR BYTE and the library says nothing.
#                      It separates a planner that treats a STORED default as
#                      an override, which would reprice every load of every
#                      game. It is the research twin of `recipe-default-word`
#                      and the state every player who never opens the Startup
#                      tab is in.
#   tech-count-alone   the COUNT moved and the other two did not, so the count
#                      is the player's and the time and the packs are the
#                      tier's. It separates a planner that takes all three
#                      fields whenever any one of them moves.
#   tech-packs-alone   the PACK TEXT moved and the two numbers did not, which
#                      is the other half of that and cannot be got from the
#                      count arm: a planner that consulted the text only when
#                      a number was non-zero passes `tech-count-alone` and
#                      fails here. It is also the one arm in this file where a
#                      typed list is written INTO a tier's unit rather than
#                      into a unit the library built, so the count and the time
#                      beside it have to come back as the tier's.
#
# THE SECONDS-ALONE STATE IS NOT REACHED HERE AND IS NOT UNGUARDED. A fourth
# engine run would separate it the way the two above do, and it is left to the
# host suites because nothing about it is an ENGINE question: this mod's own
# `TestOneFieldMovedTakesTheOtherTwoFromTheTier` in guest/go/tune's
# plandata_test.go drives all three fields one at a time, and the library's own
# `TestATimeAloneOverridesTheTier` and `TestAPackTextAloneOverridesTheTier`
# (FkRecipes go/customize_test.go, with their Rust twins) hold the rule at the
# source. What an engine adds over those is that the .dat ROUND TRIP carries
# the value, which the count arm already measures for a number.
#
# THE TIER IS `logistics-2` IN ALL THREE AND NOT THE DEFAULT ONE, and that is
# what makes the partial arms say anything. This mod's `CostChoices.Fallback`
# is [FallbackUnit], which IS base's `logistics` cost -- 20 units of 15 seconds
# in one automation science pack -- so under the default tier "the tier
# supplied the fields I did not move" and "the mod's own declared fallback
# supplied them" are the same three numbers and the arms could not tell them
# apart. Under `logistics-2` they are 200 and 30 (measured with the jq above),
# which neither the fallback nor the settings' declared defaults can produce.
#
# ANTI-VACUITY, AND IT IS NOT THE SAME ASSERTION FOR EVERY ARM. The unit and
# the prerequisite make it for ALL THREE: a .dat that was ignored or malformed
# leaves `bbb-tech-cost` at `logistics`, so the unit would be base's
# `logistics` unit where every arm here expects `logistics-2`'s (200 and 30
# against 20 and 15, two fields at once, and for `tech-packs-alone` the
# ingredients as well), and the prerequisite would be `logistics` where every
# arm expects `logistics-2`. The LOG LINE makes it for the two partial arms and
# NOT for `tech-tier-word`, which deliberately expects no line and would be
# handed none by a file that never arrived -- which is exactly why that arm's
# unit and prerequisite are read against a tier this file never defaults to,
# and why it would be a poor arm under `logistics`. What this table cannot rest
# on at all is the old table's separator: the three ignored-field sentences are
# gone with the state that produced them. An expectation that a missing file
# would also satisfy is worthless, and a missing file here produces the wrong
# tier twice over: the `startup=None` run above reads 20 units of 15 seconds in
# one automation science pack after `logistics`, where all three arms here
# expect `logistics-2`'s own 200 and 30 after `logistics-2`.
#
# THE MIXTURE IS BUILT IN THE DRIVER AND NOT TRANSCRIBED, for TECH_VARIANTS'
# reason: the claim is "whatever base charges for that technology, with the
# fields the player moved written over it", and a figure written out here would
# go stale the day base re-costs a tier. THE LOG LINES ARE TRANSCRIBED, because
# they are the library's own sentences and this file pins those word for word;
# each quotes the tier's own fields inside itself -- the count arm the tier's
# time and packs, the pack arm the tier's count and time -- so base re-costing
# `logistics-2` fails those arms on the line while the unit comparison still
# passes, and the remedy is to re-transcribe the sentence.
# ---------------------------------------------------------------------------
TECH_TIER_ARMS = [
    # THE THREE WAYS OF SAYING "THE ROW ABOVE DECIDES", ALL AT ONCE. Nothing is
    # overridden, so no field is written over the tier's unit and the library
    # has nothing to report: a line here would be the library narrating an
    # override on every load of every game. It is not the same run as
    # `tech-logistics-2` above, which leaves the three fields out of the .dat
    # altogether: this one STORES the word and the two zeros, which is what a
    # player who typed `default` back into the field leaves behind. `moved` is
    # empty, so the driver's expectation is the tier's unit unaltered.
    ("tech-tier-word", "logistics-2",
     {TECH_SETTING: "logistics-2", TECH_PACKS_SETTING: "default",
      TECH_COUNT_SETTING: 0, TECH_SECONDS_SETTING: 0},
     {}, []),
    # ONE FIELD MOVED. 50 is a count `logistics-2` does not charge and the
    # other two fields are left saying the word and the zero, so the expected
    # unit is that tier's with 50 written into `count` alone -- the time and
    # the pack list have to come back as the tier's 30 seconds and its two
    # packs. A planner that took all three fields whenever any of them moved
    # would write this mod's declared 0 seconds and its one declared pack into
    # the other two slots, and a 0 in a research unit is a load the engine
    # refuses outright, so that defect arrives here as `--dump-data exited 1`
    # rather than as a quiet pass.
    ("tech-count-alone", "logistics-2",
     {TECH_SETTING: "logistics-2", TECH_COUNT_SETTING: 50},
     {"count": 50},
     ["fkrecipes: bbb-balancer takes its research cost from "
      "better-belt-balancer-tech-packs: count 50, time 30, "
      "packs 1 automation-science-pack, 1 logistic-science-pack"
      "; the bbb-tech-cost choice logistics-2 supplies what the settings leave "
      "at default"]),
    # THE PACK TEXT ALONE, which is the count arm turned around: the two
    # numbers say the zero and the text is the one field that moved, so the
    # expected unit is `logistics-2`'s own COUNT AND TIME with the typed list
    # written over its `ingredients`. `2 chemical-science-pack` is a list no
    # tier of this mod charges (logistics-2 charges one automation and one
    # logistic, logistics-3 charges four packs of which chemical is one at
    # amount 1) and is not this mod's declared default either, so the
    # ingredients cannot be confused with anything that was copied, and the
    # line quotes the tier's 200 and 30 back beside it.
    ("tech-packs-alone", "logistics-2",
     {TECH_SETTING: "logistics-2",
      TECH_PACKS_SETTING: "2 chemical-science-pack"},
     {"ingredients": [["chemical-science-pack", 2]]},
     ["fkrecipes: bbb-balancer takes its research cost from "
      "better-belt-balancer-tech-packs: count 200, time 30, "
      "packs 2 chemical-science-pack"
      "; the bbb-tech-cost choice logistics-2 supplies what the settings leave "
      "at default"]),
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

# ---------------------------------------------------------------------------
# THE MERGE ARM'S FIXTURE, and it is a Lua mod rather than a compiled guest.
#
# WHAT IT IS FOR. Every ladder in guest/go/tune ends at `iron-plate` and the
# vanilla list names `iron-plate` at the top, so a pack that removes the item
# `transport-belt` makes the third ladder land on a name the list already
# carries. Until FkRecipes 696387f that reached the engine as two `iron-plate`
# entries and refused the load -- `Duplicate item ingredients are not allowed
# (iron-plate exists 2 or more times).`, exit 1, no dump, no `fkrecipes:` line
# and no setting named -- on the DEFAULT preset, for a player who never opened
# the Startup tab (agents/migration-assessment.md, finding 1). The library
# merges the second landing into the first now. THE HALF THAT NEEDS AN ENGINE
# is whether the engine ACCEPTS the merged list, and no host test can observe
# it: guest/go/tune's own arm asserts the plan, and this asserts the load.
#
# WHY LUA AND NOT A `build_fixture` GUEST. It has to run at the DATA stage
# BEFORE this mod, and a plain `data.lua` is the only thing that can be written
# that cheaply -- no tinygo, no `fklua mod`, no compile at all. Written out at
# run time rather than committed under test/fixtures/ because its whole content
# is one name list, and a committed directory would be sixty-seven lines of Lua
# nobody reads beside a constant somebody edits. SIXTY-SEVEN IS MEASURED, off
# `REMOVER_LUA` itself and off the engine, which prints
# `Script @__bbbt-remover__/data.lua:67` on every run of this arm.
#
# THE NAME SORTS BEFORE `better-belt-balancer`, which is how it comes to run
# first, and the arm ASSERTS that rather than trusting it: Factorio's data-stage
# ordering is measured here (`bbbt-remover` at 0.261 against this mod at 0.266
# on 2.0.77) and was not found documented. `bbbt-fastbelt` already depends on
# the same ordering.
#
# ITS `pairs` WALKS ARE NOT AN ITERATION-ORDER DEPENDENCE, which this
# repository's determinism rule would otherwise refuse. Every one of them
# DELETES a set of keys, or rebuilds a list in `ipairs` order, so what it
# leaves behind is a function of the set and not of the order it was walked
# in; nothing it computes reaches a prototype field whose value could differ
# between two clients. There is no ordered output here to be a desync.
#
# IT CLEARS `minable` AND `next_upgrade` TOGETHER, WHICH IS A MEASURED TRAP.
# Clearing the first alone refuses the load with `Error while running setup for
# entity prototype "transport-belt": Entity must be minable when next_upgrade is
# set. (was fast-transport-belt)`. The entity is KEPT rather than deleted for a
# second measured reason: deleting it breaks this mod unconditionally (`Error in
# assignID: entity with name 'transport-belt' does not exist`), which is a
# different defect and not this arm's.
# ---------------------------------------------------------------------------

REMOVER_NAME = "bbbt-remover"
REMOVER_VERSION = "0.0.1"
REMOVER_ITEM = "transport-belt"

# A NAME THE FIXTURE MUST NOT TOUCH, asked through the SAME jq path as
# REMOVER_ITEM and asserted PRESENT. "The item is gone" is satisfied by a path
# that went stale and answers null for everything, which is the opposite of
# what `check_speed`'s anti-vacuity does: that one asserts the fixture's own
# positive values first. This is the positive half here, and it costs one more
# key on a projection the arm already runs.
REMOVER_ALIVE = "iron-plate"

# What this mod's recipe comes out as once that item is gone, and the line the
# library writes on the way. TRANSCRIBED, like every other expectation in this
# file: 4 iron plates plus the two the belt ladder falls back to.
REMOVER_RECIPE = [("iron-plate", 6), ("iron-gear-wheel", 2)]
REMOVER_LINE = ("fkrecipes: bbb-balancer-part: iron-plate is in the list twice "
                "after the fallbacks, so the amounts are added: 4 plus 2 is 6")

REMOVER_LUA = '''\
-- bbbt-remover: a pack that removed an item, at the data stage and before
-- better-belt-balancer's. Written out by test/check-datastage.py; never
-- shipped. The ITEM and every recipe naming it go; the ENTITY stays, because
-- deleting it breaks this mod for a different reason.
local REMOVE = { %s }
local gone = {}
for _, n in ipairs(REMOVE) do gone[n] = true end

local ITEM_CLASSES = {"item", "tool", "module", "capsule", "gun", "ammo",
  "armor", "repair-tool", "rail-planner", "item-with-entity-data",
  "spidertron-remote", "space-platform-starter-pack"}
for _, cls in ipairs(ITEM_CLASSES) do
  local t = data.raw[cls]
  if t then for n in pairs(gone) do t[n] = nil end end
end

local dead_recipes = {}
for rname, r in pairs(data.raw.recipe or {}) do
  local hit = gone[rname] or false
  for _, ing in pairs(r.ingredients or {}) do
    if gone[ing.name] or gone[ing[1]] then hit = true end
  end
  for _, res in pairs(r.results or {}) do
    if gone[res.name] or gone[res[1]] then hit = true end
  end
  if hit then dead_recipes[rname] = true end
end
for rname in pairs(dead_recipes) do data.raw.recipe[rname] = nil end

for _, tech in pairs(data.raw.technology or {}) do
  if tech.effects then
    local keep = {}
    for _, e in ipairs(tech.effects) do
      if not (e.type == "unlock-recipe" and dead_recipes[e.recipe]) then
        keep[#keep+1] = e
      end
    end
    tech.effects = keep
  end
end

for _, cls in pairs(data.raw) do
  if type(cls) == "table" then
    for _, ent in pairs(cls) do
      if type(ent) == "table" then
        local m = ent.minable
        local hit = m and (gone[m.result] or
          (m.results and #m.results > 0 and gone[m.results[1].name]))
        -- An entity that cannot be mined may not name a next_upgrade
        -- (MEASURED: "Entity must be minable when next_upgrade is set"), so
        -- the two are cleared together or neither is.
        if hit then ent.minable = nil; ent.next_upgrade = nil end
        if ent.place_result and gone[ent.place_result] then
          ent.place_result = nil
        end
        if ent.placeable_by and ent.placeable_by.item and
           gone[ent.placeable_by.item] then
          ent.placeable_by = nil
        end
        if ent.next_upgrade and gone[ent.next_upgrade] then
          ent.next_upgrade = nil
        end
      end
    end
  end
end
log("bbbt-remover: removed " .. #REMOVE .. " item name(s)")
'''


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


# THE mod-settings.dat WRITER IS THE TOOLCHAIN'S, and what is left here is the
# document it reads.
#
# `fklua modsettings write --from FILE.json --out mod-settings.dat` is the
# writer for Factorio's property tree, and this repository used to hold one too:
# a transcription of the format under tools/, verified against a round trip of
# the engine's own file, with two callers -- this one and bench/run.sh. A second
# implementation of a format the compiler this mod is built with already writes
# is a second answer the day the two disagree, so it is deleted rather than
# kept in step -- which is what host.go, the layout check and the tinygo.wasm
# vet tag each got for the same reason.
#
# THE NAME AND THE SIGNATURE ARE AN INTERFACE and are kept: a scratch driver
# that replaces `write_mod_settings` by name, to INSTALL a .dat instead of
# writing one, is how a release-against-head comparison drives this file, and it
# binds to this name and these four arguments.
#
# NUMBERS KEEP THE TYPE PYTHON GIVES THEM. json.dump writes an `int` without a
# decimal point and a `float` with one, and the toolchain reads the first as the
# signed 64-bit type and the second as the double, so the arm tables spell each
# number as its own setting's type.
def write_mod_settings(path: Path, version: str, startup: dict,
                       runtime_global: dict | None = None) -> None:
    """Write one mod-settings.dat.

    `version` is the FULL `X.Y.Z` the file stamps, which callers derive from the
    engine series they are staging for.
    """
    maj, minor, patch = (int(x) for x in version.split(".")[:3])
    doc = {
        "version": [maj, minor, patch, 0],
        "startup": startup,
        "runtime-global": runtime_global or {},
        "runtime-per-user": {},
    }
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as fh:
        json.dump(doc, fh)
        src = fh.name
    try:
        cmd = [FKLUA, "modsettings", "write", "--from", src, "--out", str(path)]
        done = subprocess.run(cmd, capture_output=True, text=True)
        if done.returncode != 0:
            # LOUDLY, AND WITH THE COMMAND. The writer refuses rather than
            # guesses -- it names the key it cannot encode -- so its stderr is
            # the whole diagnosis and swallowing it would turn a sentence about
            # one setting into an engine run that measured the defaults.
            sys.exit(f"`{' '.join(cmd)}` exited {done.returncode}\n"
                     f"{done.stderr.strip()}")
    finally:
        os.unlink(src)


# THE WRITER IS PROBED ONCE, BEFORE THE ENGINE IS ASKED ANYTHING.
#
# A skipped gate reads exactly like a pass, and the shape that could skip here
# is a toolchain from before the writer landed: `fklua modsettings` is a command
# this repository's own build already depends on the binary for, and an older
# one answers `unknown command "modsettings"` on stderr. Both that and a binary
# that is not there refuse with the remedy, so the variant arms are never
# quietly dropped from a run that otherwise looks green.
#
# THE EXIT CODE AND THE STDERR ARE READ DIRECTLY. A shell pipeline would report
# the last command's status, and what is being read here is the first's.
def check_toolchain() -> None:
    remedy = ("      an FkLua checkout at c21ff07 or later, built with\n"
              "      `cd ../FkLua && go build -o bin/fklua ./cmd/fklua`, or FKLUA=<path>")
    need = ("every variant arm of this gate drives its setting through "
            "`fklua modsettings write`")
    # A FILE that is executable: a directory passes os.X_OK (it is searchable)
    # and then fails inside subprocess with a traceback rather than this line.
    if not (os.path.isfile(FKLUA) and os.access(FKLUA, os.X_OK)):
        sys.exit(f"NOT RUN: no executable fklua at {FKLUA}, and {need}.\n{remedy}")
    done = subprocess.run([FKLUA, "modsettings"], capture_output=True, text=True)
    if 'unknown command "modsettings"' in done.stderr:
        sys.exit(f"NOT RUN: the fklua at {FKLUA} has no `modsettings` command "
                 f"(it exited {done.returncode} saying "
                 f'`unknown command "modsettings"`), and {need}.\n{remedy}')


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
    fklua = FKLUA
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


def build_remover(series: str, out: Path) -> Path:
    """Write the bbbt-remover fixture into a staged mod directory.

    NO TOOLCHAIN AT ALL, which is the whole difference between this and
    build_fixture: no tinygo, no fklua, no compile. It is an info.json and a
    data.lua, and its content is REMOVER_ITEM.
    """
    d = out / f"{REMOVER_NAME}_{REMOVER_VERSION}"
    d.mkdir(parents=True)
    (d / "info.json").write_text(json.dumps({
        "name": REMOVER_NAME,
        "version": REMOVER_VERSION,
        "title": "BBB item-remover fixture",
        "author": "BetterBeltBalancer",
        "factorio_version": series,
        "description": "Deletes a named item at the data stage, for the merge "
                       "arm of test/check-datastage.py. Never shipped.",
        "dependencies": [f"base >= {series}.0"],
    }, indent=2) + "\n")
    # A LIST OF ONE, because the fixture takes a list. WHAT IS GENERAL HERE IS
    # THE LUA AND NOT THE NAME: this arm is measured on `transport-belt` and on
    # nothing else, and REMOVER_ITEM is not a knob. MEASURED on Factorio 2.0.77
    # (build 84539) with `iron-gear-wheel` in its place: both data stages run
    # and the merge line is written, and then the engine exits 1 on `Error in
    # assignID: recipe with name 'steam-engine' does not exist. It was removed
    # by bbbt-remover. Source: electric-network (tips-and-tricks-item).` --
    # which run_arm turns into a `sys.exit`, so the whole gate stops rather
    # than one arm printing FAIL. The fixture deletes recipes without pruning
    # the tips-and-tricks entries that name them; another item costs that
    # prune and whatever else its removal dangles, and this comment rather
    # than the constant is where that starts.
    (d / "data.lua").write_text(REMOVER_LUA % json.dumps(REMOVER_ITEM))
    return d


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
            # WHICH MODS RAN THE DATA STAGE, IN THE ORDER THE ENGINE RAN THEM,
            # for the one arm whose whole premise is that another mod went
            # FIRST. `check_remover` stages a mod that deletes an item ahead of
            # this one, and Factorio's data-stage order is not something this
            # repository found documented anywhere -- it is measured, once,
            # here, on every run of that arm. Without it a reordered engine
            # would make the arm pass by removing nothing at all, which is
            # `check_speed`'s own anti-vacuity lesson met a second time. The
            # names only: the times move every run and the versions move every
            # release.
            "load_order": re.findall(r"Loading mod (\S+) \S+ \(data\.lua\)", text),
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


# ---------------------------------------------------------------------------
# THE SIX STARTUP SETTINGS' ORDER STRINGS, on both golden arms.
#
# WRITTEN OUT HERE AND DERIVED FROM NOTHING. Two of the six are this mod's own
# strings, hand-passed to a Legacy constructor; the other four are FkRecipes'
# arithmetic, the order named by `OrderAfter` followed by the two letters the
# declaration index gives them. A table that computed the second four would be a
# gate agreeing with the library about the library, which is the one thing it
# must not be -- so they are transcribed, and the transcription is what a
# maintainer compares a moved dump against.
#
# THE LIST IS IN DECLARATION ORDER, which is the second assertion: sorted by
# (order, name), the way Factorio sorts a mod's settings for the Startup tab,
# the six have to come back in exactly this sequence. Each generated field sits
# directly under the dropdown it can override, which is what `OrderAfter` was
# called for and what a per-name comparison alone cannot say.
#
# FIX ROUND 2 MOVED NEITHER A NAME NOR AN ORDER. Withdrawing the `custom` value
# changed which of the two rows decides and not where either sits: the two
# dropdowns keep their names and their `a` and `b`, and the four generated
# settings keep theirs and their three letters, so this table is the same six
# pairs it was before the text became the switch. What DID move is inside the
# hash and not here, and the difference is the whole reason this table exists:
# a `mod_settings_sha256` that has to be re-captured against an order string
# that must not have moved at all. FIVE MOVES, read off this package's own
# `mod-settings-dump.json` on 2.0.77 (this script's `--diff` keeps it as
# `<arm>-settings.json`) rather than off the library's notes:
#
#   FOUR SETTINGS CHANGED SHAPE, not three. `bbb-recipe-cost` went from seven
#   `allowed_values` to six and `bbb-tech-cost` from four to three;
#   `better-belt-balancer-tech-count` went from default 20 and minimum 1 to
#   default 0 and minimum 0; and `better-belt-balancer-tech-seconds` made those
#   same two moves AND changed prototype type, from `double-setting` to
#   `int-setting`, which is why the dump now carries it in the `int-setting`
#   table beside the count.
#
#   THE COMPOSED DESCRIPTIONS MOVED BY DIFFERENT AMOUNTS AND TWO OF THEM HAD
#   NOTHING TO GROW. `bbb-tech-cost` gained ONE line, the switch line ("\nThe
#   setting below applies instead while it does not say default.").
#   `bbb-recipe-cost`, `better-belt-balancer-recipe-ingredients` and
#   `better-belt-balancer-tech-packs` gained TWO, a wrap line and a switch
#   line; the ingredient dropdown is the only DROPDOWN carrying a wrap line,
#   which is why its twin gained one fewer.
#   `better-belt-balancer-recipe-ingredients` also had a third line EDITED
#   rather than added, its format line gaining the `none` clause ("The word
#   none empties the list, so the recipe costs nothing to craft."), which
#   `better-belt-balancer-tech-packs` does not carry. BOTH DROPDOWNS ALSO LOST
#   a line, because the composition writes one per allowed value and the
#   withdrawn one took its own with it. And the two numbers grew nothing: they
#   carried NO description at all and carry a composed one now ("\nA whole
#   number from 0 to 1000000. While it is 0 the option chosen above decides.").
#
#   AND EVERY COMPOSED LOCALE REFERENCE CHANGED FORM, which nothing else in
#   this file names. A key a description points at goes out as the engine's
#   fallback group `{"?", {"<section>.<key>"}, "<raw name>"}` where it went out
#   as the bare `{"<section>.<key>"}` before (FkRecipes e4604d4), so a key the
#   game does not define degrades to the raw internal name instead of taking
#   the tooltip with it. That is a move in all six, the two numbers included.
# ---------------------------------------------------------------------------

OUR_SETTINGS_ORDERS = [
    ("bbb-recipe-cost", "a"),
    ("better-belt-balancer-recipe-ingredients", "aab"),
    ("bbb-tech-cost", "b"),
    ("better-belt-balancer-tech-packs", "bad"),
    ("better-belt-balancer-tech-count", "bae"),
    ("better-belt-balancer-tech-seconds", "baf"),
]


def settings_orders_of(dump: Path) -> dict:
    """This mod's six startup settings' `order` strings, by name.

    THE SETTINGS DUMP IS THE SIBLING FILE. `--dump-data` writes
    data-raw-dump.json and mod-settings-dump.json side by side, and a probe is
    handed the first, so this walks across to the second. Its shape is one
    object per setting TYPE (`string-setting`, `int-setting`, ...), each a map
    of name to prototype, so the projection flattens all four before picking.

    A NAME THE DUMP DOES NOT CARRY COMES BACK None rather than missing, so a
    setting that vanished is reported by check_settings_order as an order it
    could not find rather than passing over.
    """
    got = project(dump.parent / "mod-settings-dump.json",
                  "[.[] | to_entries[]] | map({key: .key, value: .value.order}) "
                  "| from_entries")
    return {name: got.get(name) for name, _ in OUR_SETTINGS_ORDERS}


def golden_probe(dump: Path) -> dict:
    """Both golden-arm probes in one pass over one Factorio run.

    `run_arm` takes ONE probe, and the two questions the golden arms ask are of
    two different dumps. Composing them here rather than adding a second probe
    parameter keeps the arm's cost at one engine run, which is what it has
    always been.
    """
    return {"stub": legacy_stub_of(dump), "orders": settings_orders_of(dump)}


def check_settings_order(arm: str, got: dict) -> bool:
    """Returns True on a failure, which is the shape main() already counts in."""
    bad = False
    for name, want in OUR_SETTINGS_ORDERS:
        if got.get(name) != want:
            bad = True
            print(f"FAIL {arm}: the setting `{name}` carries the order "
                  f"{got.get(name)!r} and has to carry {want!r}")

    # THE SORT, over what the ENGINE handed back rather than over the table.
    # Factorio orders a mod's settings by `order` and then by name, so this is
    # the sequence of rows in the Startup tab; a missing order sorts as the
    # empty string rather than crashing the comparison, and the loop above has
    # already named it.
    seen = [(name, got.get(name) or "") for name, _ in OUR_SETTINGS_ORDERS]
    ranked = sorted(seen, key=lambda pair: (pair[1], pair[0]))
    if ranked != seen:
        bad = True
        print(f"FAIL {arm}: the settings sort into "
              f"{[n for n, _ in ranked]}\n{'':>5}  and the declaration order is "
              f"{[n for n, _ in seen]}")

    if not bad:
        print(f"  ok   {arm} settings orders "
              f"{', '.join(o for _, o in OUR_SETTINGS_ORDERS)}, sorted as declared")
    return bad


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
                    help="skip the variant, speed and merge arms, which are not hashed")
    args = ap.parse_args()

    # BEFORE THE ENGINE IS ASKED ANYTHING, because a gate that cannot write a
    # .dat has nothing to say about a setting and must not spend a Factorio run
    # finding that out.
    check_toolchain()

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
    got = {a: run_arm(a, factorio, series, mod_dir, keep, probe=golden_probe)
           for a in arms}

    book = json.loads(GOLDENS.read_text()) if GOLDENS.exists() else {}

    if args.capture:
        # THE PROBE IS NOT A GOLDEN, AND NEITHER IS THE LOG OR THE LOAD ORDER.
        # check_legacy_stub, the customizer's arms and check_remover compare
        # them against rules written down in this file, so recording any of
        # them would invite the one thing a golden must never make easy --
        # re-capturing the answer instead of reading it. Only the hashes and
        # the checksum are the engine's to record.
        book.setdefault(version_full, {}).update(
            {a: {k: v for k, v in g.items()
                 if k not in ("probe", "fkrecipes_lines", "load_order")}
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
        # question no suite can ask. The variant, speed and merge arms still
        # run below (they need no golden), so the report stays one report; the
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

    # THE STUB'S PLACING GRAPH AND THE SIX ORDER STRINGS, PER ARM, and both run
    # whether or not the golden matched. A hash that moved is exactly when
    # somebody wants to know which shape the dump is in, and these are the two
    # shapes a hash was never going to name on its own.
    for a in arms:
        bad |= check_settings_order(a, got[a]["probe"]["orders"])
        bad |= check_legacy_stub(a, got[a]["probe"]["stub"])

    if bad:
        if keep is not None:
            print(f"\nnormalised dumps kept under {keep.relative_to(ROOT)}; "
                  f"diff them against a golden run's")
        else:
            print("\nre-run with --diff to keep the normalised dumps")

    # THE VARIANT, SPEED AND MERGE ARMS RUN EVEN WHEN A GOLDEN MOVED,
    # deliberately. A default that drifted moves the hash AND every arm
    # downstream of it, and the hash alone does not say which prototype -- so
    # stopping here would throw away the lines that name it. One report, one
    # exit code.
    if not args.golden_only:
        bad |= check_variants(factorio, series, mod_dir)
        bad |= check_speed(factorio, series, mod_dir)
        bad |= check_remover(factorio, series, mod_dir)

    if bad:
        return 1
    print("check-datastage: ok")
    return 0


def check_variants(factorio: str, series: str, mod_dir: Path) -> bool:
    """Every non-default value of both cost settings, one arm each.

    Returns True on a failure, which is the shape main() already counts in.
    """
    print(f"==> the cost settings, "
          f"{len(RECIPE_VARIANTS) + len(TECH_VARIANTS) + len(RECIPE_TEXT_ARMS) + len(TECH_COST_ARMS) + len(TECH_TIER_ARMS) + 1} "
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

    # THE TEXT FIELD, four arms, each driving the real settings through the
    # same writer. See RECIPE_TEXT_ARMS for what each state is, why every one
    # of them asserts the ingredient list, and what makes the set anti-vacuous.
    for arm, startup, want, want_lines in RECIPE_TEXT_ARMS:
        # THE WHOLE STREAM, IN ORDER, ON EVERY ARM. The old table carried two
        # rules -- one line the stream must CARRY, against the whole stream
        # compared in order -- because two of its arms could not state what
        # else the plan would say. Every arm here can: none, one or two lines,
        # written down beside the recipe. So the weaker rule is gone with the
        # state that needed it, and an extra line is a failure the way it
        # already is in `check_remover`.
        got = run_arm(arm, factorio, series, mod_dir, None,
                      startup=startup, probe=ingredients_of)
        ings, lines = got["probe"], got["fkrecipes_lines"]
        if ings != want:
            bad = True
            print(f"FAIL {arm}: the recipe is {ings}\n"
                  f"{'':>5}  and {startup} should be {want}")
            continue
        if lines != want_lines:
            bad = True
            print(f"FAIL {arm}: the library's log lines are {lines}\n"
                  f"{'':>5}  and they have to be, whole and in order, "
                  f"{want_lines!r}")
            continue
        print(f"  ok   {arm:<26} {ings}"
              + ("".join(f"\n{'':>5}  said {ln!r}" for ln in want_lines)
                 if want_lines else f"\n{'':>5}  and said nothing"))
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

    # THE WHOLE COST WRITTEN, one arm, asserting the unit, the prerequisite and
    # the whole log stream. See TECH_COST_ARMS for what the state is, why the
    # expected unit is written out where TECH_TIER_ARMS' is not, and what makes
    # it anti-vacuous.
    for arm, startup, want_unit, want_after, want_lines in TECH_COST_ARMS:
        got = run_arm(arm, factorio, series, mod_dir, None, startup=startup,
                      probe=lambda d: project(d, '.technology["bbb-balancer"]'))
        ours, lines = got["probe"], got["fkrecipes_lines"]
        # NO ASSERTION HERE IS LOAD-BEARING FOR THE OTHERS, WHICH IS WHAT THE
        # WITHDRAWAL BOUGHT. The old order put the log line first because it
        # was the only separator: `Position` placed a written cost from this
        # mod's own default tier, so the unit and the prerequisite were both
        # what a `mod-settings.dat` that never reached the engine produces. The
        # tier places it now and this arm's tier is not the default one, so
        # each of the three fails on its own against a missing file, and the
        # order below is the order a maintainer reads them in: what the player
        # is charged, where it sits, what they were told.
        if ours["unit"] != want_unit:
            bad = True
            print(f"FAIL {arm}: the research unit is {ours['unit']}\n"
                  f"{'':>5}  and {startup} should be {want_unit}")
            continue
        if ours.get("prerequisites") != want_after:
            bad = True
            print(f"FAIL {arm}: the prerequisite is {ours.get('prerequisites')}, "
                  f"not {want_after} -- a written cost is placed by the tier "
                  f"the dropdown names, and nothing else places it")
            continue
        if lines != want_lines:
            bad = True
            print(f"FAIL {arm}: the library's log lines are {lines}\n"
                  f"{'':>5}  and the whole stream, in order, has to be "
                  f"{want_lines!r}")
            continue
        u = ours["unit"]
        print(f"  ok   {arm:<26} {u['count']} x {u['time']}s in "
              f"{[p[0] for p in u['ingredients']]}, after {want_after[0]}")
        for line in lines:
            print(f"{'':>5}  said {line!r}")

    # THE ARMS THE TIER STILL SUPPLIES SOMETHING TO. See TECH_TIER_ARMS: the
    # tier's own unit, read the way the tier arms read it, with the fields the
    # player moved written over it, plus the prerequisite and the whole log
    # stream in the library's order.
    for arm, tier, startup, moved, want_lines in TECH_TIER_ARMS:
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
        # THE MIXTURE, BUILT FROM THE DUMP AND NOT FROM A TRANSCRIPTION: the
        # tier's own unit with exactly the fields this arm moved written over
        # it, which is the library's per-field rule stated as an equation. An
        # empty `moved` is the whole tier, byte for byte.
        want_unit = dict(src["unit"])
        want_unit.update(moved)
        if ours["unit"] != want_unit:
            bad = True
            print(f"FAIL {arm}: the research unit is {ours['unit']}\n"
                  f"{'':>5}  and `{tier}` charges {src['unit']}, which with "
                  f"{moved or 'nothing'} written over it is {want_unit}")
            continue
        if ours.get("prerequisites") != [tier]:
            bad = True
            print(f"FAIL {arm}: the prerequisite is {ours.get('prerequisites')}, "
                  f"not [{tier}]")
            continue
        if lines != want_lines:
            bad = True
            print(f"FAIL {arm}: the library's log lines are {lines}\n"
                  f"{'':>5}  and the whole stream, in order, has to be "
                  f"{want_lines!r}")
            continue
        u = ours["unit"]
        print(f"  ok   {arm:<26} {u['count']} x {u['time']}s in "
              f"{[p[0] for p in u['ingredients']]}, after {tier}")
        if not lines:
            print(f"{'':>5}  and said nothing")
        for line in lines:
            print(f"{'':>5}  said {line!r}")
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


def check_remover(factorio: str, series: str, mod_dir: Path) -> bool:
    """A pack that removed `transport-belt`, and the engine takes the merge.

    THE ARM THE ASSESSMENT'S FINDING 1 IS ABOUT, and the only one that can
    answer it: the host suite proves what the PLAN emits, and what refused the
    load was the ENGINE, on a recipe naming one item twice. See REMOVER_LUA's
    block above for the fixture and why it is Lua.

    NO mod-settings.dat, DELIBERATELY. The arm runs on the prototype's own
    `default_value`, `bbb-recipe-cost = vanilla`, which is the configuration of
    the player the finding is about: one who never opened the Startup tab.
    """
    print("==> the ladder merge, 1 arm with an item taken out from under it")
    work = Path(tempfile.mkdtemp(prefix="bbb-remover-"))
    try:
        fixture = build_remover(series, work)
        got = run_arm("remover", factorio, series, mod_dir, None,
                      extras=[fixture],
                      probe=lambda d: project(d, '{item: .item["%s"], '
                                                 'alive: .item["%s"], '
                                                 'ingredients: .recipe["%s"]'
                                                 '.ingredients}'
                                              % (REMOVER_ITEM, REMOVER_ALIVE,
                                                 OUR_PART)))
    finally:
        shutil.rmtree(work, ignore_errors=True)

    # ANTI-VACUITY FIRST, AND IT IS THREE QUESTIONS. Factorio's data-stage
    # ordering is not something this repository found documented, so the arm
    # measures it rather than resting on the alphabet; and a fixture that
    # loaded first and removed nothing would leave the vanilla recipe, which is
    # also what a library that stopped merging would emit on the FIRST of its
    # three ladders. None of the three is asked of the dump's recipe, so all
    # three come first.
    #
    # THE ITEM QUESTION IS ASKED IN BOTH DIRECTIONS, POSITIVE FIRST. `is not
    # None` alone would be satisfied by a jq path that stopped matching
    # anything, so a stale probe would read as a removal that never happened;
    # asking the same path for a name that must SURVIVE is what tells an absent
    # item from a dead projection.
    order, ours = got["load_order"], got["mods"][0]
    if REMOVER_NAME not in order or ours not in order or \
            order.index(REMOVER_NAME) > order.index(ours):
        print(f"FAIL remover: the data stages ran {order}, and this arm needs "
              f"{REMOVER_NAME} before {ours}; it removed nothing this mod "
              f"could have seen and proves nothing")
        return True
    if got["probe"]["alive"] is None:
        print(f"FAIL remover: `{REMOVER_ALIVE}` is not in the item table "
              f"either, and the fixture never touches it: the probe reads "
              f"nothing, so the question below cannot tell a removed item "
              f"from a stale path")
        return True
    if got["probe"]["item"] is not None:
        print(f"FAIL remover: `{REMOVER_ITEM}` is still in the item table "
              f"after the fixture ran, so nothing was taken away and this arm "
              f"proves nothing")
        return True

    # THE CLAIM, AND IT IS BOTH HALVES. The list alone would pass on a library
    # that dropped the belt ingredient instead of merging it, and the line
    # alone would pass on one that said the right thing and emitted something
    # else -- which is why RECIPE_TEXT_ARMS asserts both too.
    bad = False
    ings = [(i["name"], i["amount"]) for i in got["probe"]["ingredients"]]
    if ings != REMOVER_RECIPE:
        bad = True
        print(f"FAIL remover: with no `{REMOVER_ITEM}` in the game the recipe "
              f"is {ings}\n{'':>5}  and the fallback merged into the first "
              f"entry has to make it {REMOVER_RECIPE}")
    if got["fkrecipes_lines"] != [REMOVER_LINE]:
        bad = True
        print(f"FAIL remover: the library's log lines are "
              f"{got['fkrecipes_lines']}\n{'':>5}  and the whole stream has to "
              f"be [{REMOVER_LINE!r}]")
    if not bad:
        print(f"  ok   remover{'':<20} {ings}, and the engine loaded it")
        print(f"{'':>5}  said {REMOVER_LINE!r}")
    return bad


if __name__ == "__main__":
    sys.exit(main())
