#!/usr/bin/env bash
# Headless verification. Thirteen suites, all real Factorio runs, not models:
#
#   M1  do balancer parts merge and split correctly?
#   M2  does the compiled hidden network actually balance?
#   M3  does it survive every way the world can change under it?
#   upg   does a MOD UPGRADE re-derive the registry and adopt what is standing?
#   plat  the SPACE AGE suite: a balancer on a SPACE PLATFORM surface, and
#         BELT STACKING -- does a recompile hand a stacked network back STACKED?
#         (belt stacking is a Space Age feature at the prototype level, so no
#         base-only suite can build a stacked belt at all). Its `smix` band is
#         STACKED SUSHI: the only rig anywhere whose hidden lines carry several
#         item KINDS and several items per POSITION at once, which is the pair
#         of conditions compile.go's kindAt needs to be reached at all
#   mar   what does one NET-ZERO world operation cost the guest heap, forever?
#         (the `-gc=leaking` marathon slope -- see CLAUDE.md, "The marathon save")
#   edge  the mid-operation edges: churn, merges, splits, forces and same-tick
#         edits, each with item conservation across it
#   mix   MORE THAN ONE KIND of item through one balancer: two pure belts, a
#         sushi belt, and 48 distinct kinds at once -- past the carry pool's
#         bound, which used to DESTROY the overflow. Every count is per item NAME
#   qual  A PART AT UNCOMMON QUALITY IS A PART. `find_entity` resolves a bare
#         name as NORMAL QUALITY ONLY, so every lookup that used it worked on
#         every other suite's save and silently failed on a quality-rolled
#         part. Four rigs of uncommon parts drive the four fixed sites --
#         restyle's picture write, the fast-replace reap in both directions,
#         and the over-limit refusal's delivery -- plus the question nothing
#         had asked: does an uncommon balancer balance (base + the quality mod)
#   mig   ADOPTING A BELT BALANCER 2 OR 3 SAVE. The only suite whose two phases
#         run under DIFFERENT MOD SETS: a mod is installed or uninstalled between
#         `--create` and `--benchmark`. Seven legs and two name probes, covering
#         both axes -- WHICH MOD owns `balancer-part` (all four incumbent names,
#         plus a stranger), and WHICH TRANSITION of legacy.go's state machine the
#         load makes (swapped in one edit, removed a session later, arriving
#         through build events and then a plain reload, an incumbent INSTALLED
#         after this mod, a stranger left alone, and a stranger UNINSTALLED).
#         The CONVERSION is the same on both engines and the OUTCOME is not:
#         Belt Balancer's own idiom is two belts on every part, so on 2.1 an
#         incumbent balancer converts and is then refused and only the `sok`
#         band -- laid two columns wide, which one of their users could
#         genuinely have -- comes out of it running, while on 2.0 the
#         grandfather pass keeps the lot and every rig delivers
#   mig21 A FACTORIO 2.0 MULTI-EDGE SAVE, OPENED. The only suite with no
#         `--create` phase: its worlds were built by a 2.0.77 binary and a 2.1
#         one cannot rebuild them at any price, so the saves are committed under
#         test/fixtures-2.0/ and each one IS phase one. WHICH ENGINE OPENS IT IS
#         THE WHOLE QUESTION and the two answers are opposites. On 2.1 the engine
#         gets there first -- it silently deletes all but one belt-connectable
#         per tile at load -- and what is asserted is what the mod does with the
#         wreck: the remnants torn down, everything they held recovered and
#         spilled, every balancer refused, each force told once with a ping per
#         balancer, and the grandfather write never attempted where the settings
#         key does not exist. On 2.0 nothing is pruned, every cluster is ADOPTED
#         whole, nothing comes down and nothing spills, and the mod writes its
#         own setting ON to keep the save working and says so
#   sedge FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART. Four single-edge
#         shapes with their port counts asserted before their rates, and the
#         three ways an edit can ask a part for a second belt -- built, rotated
#         and merged -- each refused in front of its teardown with the standing
#         network left running. agents/single-edge.md is the design.
#   flip  `bbb-multi-edge-parts`, DRIVEN -- and the one suite that cannot run on
#         Factorio 2.1, where the setting is not defined at all. Four
#         transitions: refused at the false default, ON so the refused clusters
#         compile and a multi-edge balancer can be built, OFF with those
#         balancers standing and full -- which the mod VETOES, writing the
#         setting straight back on without touching the world -- and OFF again
#         once nothing is left to veto, which sticks. On 2.1 it prints a SKIP
#         rather than passing
#   iact  THE INTERACTIVE CHECKLIST'S OWN WORLD. test/interactive/ stages the
#         five player-gesture rigs and the five mod-portal demo scenes, all of
#         them single-edge; nothing headless can make the gestures, but every
#         rig has to LAND and every one has to be legal. One `--create`, and it
#         fails on a placement the engine refused, a shape that is not the one
#         the geometry intended, or any refusal at all -- the gestures create
#         the refusals and the staging must not
#
# THIRTEEN OF THE FOURTEEN RUN ON EITHER ENGINE, and three of them ANSWER
# DIFFERENTLY on each. The estate was rebuilt for the one-belt-per-part rule in
# four tranches and `mig` was the last of them, because it is the only suite
# whose ANSWER the rule changed rather than whose geometry -- see the SUITES line
# below.
#
# WHICH FACTORIO IS RUNNING IS AN INPUT TO THREE SUITES, not an accident of the
# machine. This mod ships on two engine arms out of one tree, and:
#
#   `mig21` INVERTS. On 2.1 the engine prunes a 2.0 save's stacked interfaces at
#   load and every balancer is refused, torn down and spilled; on 2.0 nothing is
#   pruned, every one of them is ADOPTED whole, and the mod writes its own
#   setting on to keep the save working.
#   `mig` INVERTS for the same reason: an incumbent's balancers convert and are
#   refused on 2.1, and are grandfathered and RUN on 2.0.
#   `flip` exists on 2.0 only, and says so out loud on 2.1 rather than passing.
#
# The two assertion scripts take `--engine` from the series read off the binary
# below; there is no default, because a script that guessed would assert the
# wrong half and be green for the wrong reason on one of them.
#
#   make test          # builds the mod first
#   test/run.sh        # against whatever dist/ already holds
#   test/run.sh m2     # one suite only
#
# Each suite stages its own throwaway mod directory, `--create`s a save whose
# on_init has already built the patterns, `--benchmark`s it, and hands the logs
# to an assertion script.
#
# THE SAVE/RELOAD BETWEEN --create AND --benchmark IS LOAD-BEARING, not
# incidental: everything the benchmark phase sees exists only if the guest heap
# survived it, so every suite is also the --persist round-trip test.
#
# Factorio locks its user directory, so a second instance -- another agent, an
# open game -- makes a run fail on the .lock rather than on anything real. Every
# run here points write-data at a private directory via -c, which is the trick
# spike S1 arrived at.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="${BBB_TEST_TMP:-$ROOT/test/tmp}"
FACTORIO="${FACTORIO_BIN:-$HOME/Library/Application Support/Steam/steamapps/common/Factorio/factorio.app/Contents/MacOS/factorio}"

MOD_NAME="$(sed -n 's/^name = "\(.*\)"$/\1/p' "$ROOT/fklua.toml")"
MOD_VERSION="$(sed -n 's/^version = "\(.*\)"$/\1/p' "$ROOT/fklua.toml")"
MOD_DIR="$ROOT/dist/${MOD_NAME}_${MOD_VERSION}"

[ -x "$FACTORIO" ] || { echo "factorio not found at: $FACTORIO (set FACTORIO_BIN)" >&2; exit 1; }
[ -d "$MOD_DIR" ]  || { echo "no built mod at $MOD_DIR; run \`make mod\` first" >&2; exit 1; }

# --- which Factorio is running, and what that does to a staged mod -----------
#
# THIS MOD SHIPS ON TWO ENGINE ARMS AND THE TEST ESTATE IS ONE TREE. Trunk
# targets 2.1 and `release/2.0` carries the same suites against a 2.0 binary
# (agents/single-edge.md, "Packaging: one tree, two releases"), so every test
# mod, observer and stand-in here has to load on whichever Factorio is on the
# machine -- and a mod whose info.json names the other series is REFUSED AT THE
# LOADER, before an entity is placed: `Incompatible Factorio version (current:
# 2.1, required: 2.0)`. That is a whole suite failing on a token.
#
# So the series is read off the binary and STAMPED into every staged copy. Two
# fields carry it and both have to move together:
#
#   factorio_version   set to the running series, always. A staged mod is a
#                      throwaway copy in $TMP; nothing here edits test/mods.
#   base >= X.Y.Z      CLAMPED DOWN when it names a series newer than this
#                      engine, and otherwise left exactly alone. Clamping only
#                      downward is what makes this a no-op on the newer engine
#                      -- `base >= 2.0.0` is satisfied by 2.1 and must keep the
#                      digits it was written with.
#
# The precedent is `mig_standin`, which has rewritten a stand-in's info.json
# since the migration suite existed; this is the same rewrite over one more
# field and over every staged mod rather than one.
#
# THE PACKAGED MOD IS GATED AND NOT STAMPED, and that asymmetry is the point.
# A test mod is engine-agnostic Lua and a stamp is free; the mod under test is
# a GUEST COMPILED AGAINST A PINNED API, and the ABI marshals event payloads BY
# NAME -- so a field 2.1 added to a subscribed event is written as mandatory and
# arrives nil on a 2.0 engine. Stamping the package would let exactly that
# mismatch load and run, silently, which is the one direction that must never
# be papered over. The mod's own factorio_version comes from fklua.toml, which
# is the release identity, so a disagreement with the binary is a real defect
# and is reported as one: build the arm that matches, or run the binary the arm
# was built for.
ENGINE_SERIES="$("$FACTORIO" --version 2>/dev/null |
  sed -n '1s/^Version: \([0-9][0-9]*\.[0-9][0-9]*\)\..*$/\1/p')"
[ -n "$ENGINE_SERIES" ] || {
  echo "could not read a Major.Minor series out of \`$FACTORIO --version\`" >&2
  "$FACTORIO" --version 2>&1 | head -3 >&2; exit 1; }
ENGINE_MAJOR="${ENGINE_SERIES%%.*}"
ENGINE_MINOR="${ENGINE_SERIES##*.}"
echo "==> Factorio $ENGINE_SERIES; staged mods are stamped for it"

# stamp_engine <staged-mod-dir>...
#
# Mechanical, and deliberately so: nothing here is conditional on WHICH series
# is running, so the 2.1 behaviour it replaces is the behaviour it produces (all
# thirteen staged manifests already say 2.1 and none of them requires a base
# newer than 2.1, so on 2.1 both rewrites are no-ops to the byte).
stamp_engine() {
  local d
  for d in "$@"; do
    [ -f "$d/info.json" ] || { echo "no info.json in $d" >&2; exit 1; }
    ENGINE_SERIES="$ENGINE_SERIES" ENGINE_MAJOR="$ENGINE_MAJOR" \
    ENGINE_MINOR="$ENGINE_MINOR" perl -pi -e '
      s/"factorio_version":\s*"[^"]*"/"factorio_version": "$ENV{ENGINE_SERIES}"/;
      # `>` survives as a literal in a hand-written manifest and as > in one
      # a JSON writer produced, so both forms are matched.
      s{(base\s*(?:>=|\\u003e=)\s*)(\d+)\.(\d+)\.(\d+)}{
        my ($lead, $maj, $min, $pat) = ($1, $2, $3, $4);
        ($maj > $ENV{ENGINE_MAJOR} ||
         ($maj == $ENV{ENGINE_MAJOR} && $min > $ENV{ENGINE_MINOR}))
          ? "$lead$ENV{ENGINE_MAJOR}.$ENV{ENGINE_MINOR}.0"
          : "$lead$maj.$min.$pat";
      }ge;
    ' "$d/info.json"
    # The rewrite is what a whole suite rests on, and a perl that silently
    # matched nothing would move the failure to the loader with no explanation.
    grep -q "\"factorio_version\": \"$ENGINE_SERIES\"" "$d/info.json" || {
      echo "$d/info.json was not stamped for Factorio $ENGINE_SERIES" >&2
      cat "$d/info.json" >&2; exit 1; }
  done
}

# The mod under test, gated rather than stamped. See the block above.
MOD_SERIES="$(sed -n 's/.*"factorio_version": *"\([^"]*\)".*/\1/p' "$MOD_DIR/info.json")"
[ "$MOD_SERIES" = "$ENGINE_SERIES" ] || {
  echo "the built mod targets Factorio $MOD_SERIES and the binary is $ENGINE_SERIES." >&2
  echo "That is not a manifest token: the guest's bindings are pinned to one API and" >&2
  echo "the ABI marshals event payloads BY NAME, so a field the other series added is" >&2
  echo "written as mandatory and read as nil here. Build the matching arm --" >&2
  echo "fklua.toml's factorio_version and [fklua] api, and \`fklua gen-bindings\`" >&2
  echo "-- or run the binary this one was built for." >&2
  exit 1; }

# THE DEFAULT IS ALL THIRTEEN, and getting there took four tranches of estate
# work rather than a manifest token. What each suite needed is recorded in
# agents/single-edge.md's phase sections; the short form, because it is the
# reason the list reads the way it does:
#
#   `Incompatible Factorio version (current: 2.1, required: 2.0)` -- 2.1 refuses
#   a mod whose info.json says 2.0 at all, so every suite failed at the loader
#   before an entity was placed. `m1` needed nothing but that one token, because
#   it is BELT-FREE: it asserts cluster merges, splits and sprite variations and
#   never builds an edge, so the rule this port is about cannot touch it.
#
#   and then the RULE, for the nine that build belts. `m2`, `mar`, `upg`, `mix`,
#   `plat`, `qual`, `m3` and `edge` had their rigs REBUILT single-edge -- every
#   column of parts is two columns now, and the belt an edit used to lay on a
#   cluster's free face goes against an EDGELESS part instead, because under
#   this rule a working balancer has no free face. Not one assertion in any of
#   them had to be weakened.
#
#   `mig` was last and is the only one whose ANSWER moved. Its world is
#   somebody else's: Belt Balancer's idiom is a single column of parts with a
#   belt on every free face, which is two belts per part and is exactly what
#   2.1 forbids -- so re-laying its rigs would have been re-laying the thing
#   under test. They stay as the incumbent builds them, they convert, and they
#   are then refused; what was ADDED is the `sok` band, the same balancer laid
#   two columns wide, which is a shape one of their users could genuinely have
#   and which comes out of the conversion running. Some of your balancers keep
#   working, the rest get a rebuild checklist, and both halves are measured.
#
#   `mig21` is the one suite that does not BUILD its world at all: it LOADS one
#   a 2.0.77 binary built, which is the only way a 2.1 binary can ever be shown
#   a multi-edge save. Its two fixtures are the m2 and edge saves from the last
#   2.0.77 run, committed under test/fixtures-2.0/.
#
#   `iact` is not about the mod's behaviour at all; it is about the world the
#   INTERACTIVE checklist stages. It runs here because a rig that stopped
#   landing, or one this mod refuses, costs a human a session to discover and
#   costs this a single `--create` to catch.
SUITES="${*:-m1 sedge mig21 flip m2 m3 mar upg edge mix plat qual mig iact}"

# A private write-data directory, so a concurrent Factorio cannot take the lock
# out from under us.
USERDIR="$TMP/userdir"
CONFIG="$USERDIR/config/config.ini"
mkdir -p "$USERDIR/config"
cat > "$CONFIG" <<INI
[path]
read-data=__PATH__system-read-data__
write-data=$USERDIR

[general]
locale=auto
INI

# copy_testmod <test-mod-name> <destination-directory>
#
# THE ESTATE IS BEING PORTED FROM HAND-WRITTEN LUA TO COMPILED GO OBSERVERS, ONE
# PHASE AT A TIME (agents/estate-port.md), and this is the one place that has to
# know which half a suite is in.
#
# A PORTED suite's observer is a PACKAGED MOD that `make observers` writes into
# dist/obs/<name>_<version>/: a generated control.lua, fk_module.lua, info.json,
# and for `sedge` a data stage as well. An unported one is still a directory of
# hand-written Lua under test/mods/<name>/. Both are staged identically from here
# on -- copied to $work/mods/<name> and stamped for the running engine -- so
# nothing else in this file has to care, and porting a suite is a Makefile
# recipe plus a deletion rather than an edit here.
#
# The version is GLOBBED rather than known, because it is the observer's own and
# has nothing to do with this mod's. There is exactly one package per name: the
# recipe rm -rf's its output directory before writing it, so a stale copy from a
# renamed version cannot accumulate beside it.
#
# The destination is the BARE NAME, not name_version. That is what this estate
# has always staged, Factorio accepts it, and it keeps every
# `$work/mods/$testmod` path below working whichever half the suite is in --
# including the mig21 suite's control.lua overwrite, which neutralizes a ported
# observer exactly as well as a Lua one.
copy_testmod() {
  local name="$1" dest="$2" d
  for d in "$ROOT/dist/obs/${name}_"*/; do
    [ -d "$d" ] || continue
    cp -R "$d" "$dest"
    return
  done
  [ -d "$ROOT/test/mods/$name" ] || {
    echo "no observer package under dist/obs for $name, and no test/mods/$name." >&2
    echo "A ported suite needs \`make observers\`; \`make test\` does that for you." >&2
    exit 1; }
  cp -R "$ROOT/test/mods/$name" "$dest"
}

# stage <workdir> <test-mod-name> [space-age] [quality]
#
# `quality` defaults to the Space Age flag (the DLC set moves together), and
# the `qual` suite sets it alone: the quality mod is an official base-game mod
# with no dependency on Space Age, and quality-blind lookups are a base-plus-
# quality defect class, so gating their coverage behind the DLC would be wrong
# in both directions.
stage() {
  local work="$1" testmod="$2" dlc="${3:-false}" qual="${4:-${3:-false}}"
  rm -rf "$work"
  mkdir -p "$work/mods"
  cp -R "$MOD_DIR" "$work/mods/"
  copy_testmod "$testmod" "$work/mods/$testmod"
  stamp_engine "$work/mods/$testmod"
  # The Space Age DLC mods ship inside the Factorio install's data/ directory,
  # so they load no matter what --mod-directory says. Base only wherever
  # possible: a base-only run is one fewer variable, and only the `plat` suite
  # is about things Space Age owns -- platform surfaces and belt stacking.
  cat > "$work/mods/mod-list.json" <<JSON
{
  "mods": [
    { "name": "base", "enabled": true },
    { "name": "elevated-rails", "enabled": $dlc },
    { "name": "quality", "enabled": $qual },
    { "name": "space-age", "enabled": $dlc },
    { "name": "$MOD_NAME", "enabled": true },
    { "name": "$testmod", "enabled": true }
  ]
}
JSON
}

# The `iact` suite's staging. It differs from `stage` in exactly one thing: the
# mod it copies does not live under test/mods, because it is not a test mod. It
# is the rig-staging mod a HUMAN installs beside the real one to walk
# test/interactive/README.md, and `make interactive-install` copies the same
# directory into a Factorio mods folder. Staging it from where it really lives
# is the point -- a copy under test/mods would be a second world that could
# drift from the one the checklist describes.
stage_interactive() {
  local work="$1"
  rm -rf "$work"
  mkdir -p "$work/mods"
  cp -R "$MOD_DIR" "$work/mods/"
  cp -R "$ROOT/test/interactive/bbb-interactive-setup" "$work/mods/bbb-interactive-setup"
  stamp_engine "$work/mods/bbb-interactive-setup"
  cat > "$work/mods/mod-list.json" <<JSON
{
  "mods": [
    { "name": "base", "enabled": true },
    { "name": "elevated-rails", "enabled": false },
    { "name": "quality", "enabled": false },
    { "name": "space-age", "enabled": false },
    { "name": "$MOD_NAME", "enabled": true },
    { "name": "bbb-interactive-setup", "enabled": true }
  ]
}
JSON
}

# Called between the two phases when $BETWEEN is set. The `upg` suite uses it to
# make the guest look REBUILT to the mod glue: `control.lua` compares
# `storage.fk_build` with the module's build stamp, and a mismatch means the
# saved guest heap is not adopted -- which is exactly what a mod version bump
# does, and the only way to reach the rebuild-from-world path from a test.
#
# Changing the stamp rather than recompiling with different flags is not a
# shortcut: `fk_migrate` is a NOTIFICATION on a fresh heap (FKLUA-GAPS.md item
# 13, fixed upstream), so the old heap is discarded unread either way and the
# guest starts from its own data segments. The two are the same code path to the
# byte, and this is the only place a second guest BUILD is exercised at all.
#
# ASSERT ON OUR OWN LOG LINES, NEVER ON `fklua: ` ONES. A runtime log line is
# API surface -- upstream says so, and this repo is the reason it says so: the
# `fk_migrate` redesign moved WHEN "fklua: this mod was rebuilt" fires without
# changing a character of its text, and the assertion keyed on it failed while
# the bump had worked perfectly. assert-upgrade.py greps `[BBB] the mod was
# rebuilt`, which is this guest's own statement about its own state. If a
# `fklua: ` line ever has to be matched, match the OPENING CLAUSE only; the
# detail after it is free to change.
bump_build() {
  local old="$1/mods/${MOD_NAME}_${MOD_VERSION}"
  # Factorio insists on Major.Middle.Minor, so the bump is a real one.
  local upgv="${MOD_VERSION%.*}.$(( ${MOD_VERSION##*.} + 1 ))"
  local new="$1/mods/${MOD_NAME}_${upgv}"
  local f="$new/fk_module.lua"
  mv "$old" "$new"
  # The VERSION too, so Factorio treats it as a mod upgrade and runs
  # on_configuration_changed -- which is where the glue notices the guest is a
  # different build and re-publishes the fresh heap to `storage`. Without the
  # bump, on_load alone declines to adopt and the save would keep pointing at
  # the heap nobody is using.
  perl -pi -e 's/"version": "[^"]+"/"version": "'"$upgv"'"/' "$new/info.json"
  grep -q 'build = "' "$f" || { echo "no build stamp in $f" >&2; exit 1; }
  perl -pi -e 's/build = "[^"]+"/build = "m3-upgrade-test"/' "$f"
  echo "==> mod version and guest build stamp bumped: this is an upgrade"
}

# --- the mig suite's staging -------------------------------------------------
#
# EVERY OTHER SUITE RUNS ITS TWO PHASES UNDER ONE MOD SET. This one cannot: the
# thing under test is what happens when a NEIGHBOUR is uninstalled, so the mod
# list has to differ between `--create` and `--benchmark`, and in one leg this
# mod is not present for the create at all.
#
# A mod DIRECTORY that is present but not in mod-list.json is added back by
# Factorio as enabled, so "removed" here means the directory is deleted as well
# as the entry -- which is also what a player does.

# QUALITY IS ENABLED HERE AND IN NO OTHER SUITE'S BASE-ONLY LIST, and it is
# enabled in BOTH PHASES OF EVERY LEG. `legacyConvertOne` passes the old entity's
# quality through to the new one, and a game with only `normal` in it cannot tell
# a guest that carries the quality apart from one that drops the key -- so the
# fidelity rig builds one part at `uncommon`, which needs the prototype.
#
# A CONSTANT ACROSS THE TWO PHASES IS NOT A MOD-SET CHANGE, so no leg's trigger
# moves: `fk_on_configuration_changed` fires for the mod that was added or
# removed between the phases and quality is neither. It depends on `base` alone
# (its own info.json, 2.0.77), so it loads without elevated-rails or space-age
# and the suite stays base-plus-one rather than becoming a second Space Age run.
#
# mig_list <workdir> <extra-mod-name-or-empty> <bbb-enabled>
mig_list() {
  local work="$1" extra="$2" bbb="$3"
  local extra_entry=""
  [ -n "$extra" ] && extra_entry="    { \"name\": \"$extra\", \"enabled\": true },"
  cat > "$work/mods/mod-list.json" <<JSON
{
  "mods": [
    { "name": "base", "enabled": true },
    { "name": "elevated-rails", "enabled": false },
    { "name": "quality", "enabled": true },
    { "name": "space-age", "enabled": false },
$extra_entry
    { "name": "$MOD_NAME", "enabled": $bbb },
    { "name": "bbb-mig-test", "enabled": true }
  ]
}
JSON
}

# THE STAND-IN IS PARAMETERIZED BY NAME, AND THERE IS ONE COPY OF IT IN THE
# REPOSITORY.
#
# `guest/go/legacy.go` names FOUR incumbents -- belt-balancer, belt-balancer-2,
# belt-balancer-3 and belt-balancer-performance -- and until this existed only
# one of those names had ever been in front of the guest. The other three rows
# of that list were unexercised, and a typo in one of them degrades SILENTLY: an
# unrecognised mod owning `balancer-part` takes the STRANGER path, which is
# Blocked with no log line at all, so every conversion assertion in this suite
# would still pass while the guest quietly stopped recognising a real mod.
#
# What differs between the four is the NAME and nothing else -- the prototypes
# are the same prototypes -- so the stand-in is copied and its info.json
# rewritten at staging time rather than being checked in four times. Factorio
# requires a mod directory to be named for the mod it holds, so the copy is
# staged under the target name.
#
# mig_standin <workdir> <mod-name> <version>
mig_standin() {
  local work="$1" name="$2" version="$3"
  cp -R "$ROOT/test/mods/belt-balancer-2" "$work/mods/$name"
  perl -pi -e '
    s/^(\s*)"name":\s*"[^"]*"/$1"name": "'"$name"'"/;
    s/^(\s*)"version":\s*"[^"]*"/$1"version": "'"$version"'"/;
  ' "$work/mods/$name/info.json"
  # The rewrite is what the whole mechanism rests on, and a silently unrenamed
  # copy would stage belt-balancer-2 under every name and pass every leg.
  grep -q "\"name\": \"$name\"" "$work/mods/$name/info.json" || {
    echo "the stand-in's info.json was not renamed to $name" >&2; exit 1; }
  grep -q "\"version\": \"$version\"" "$work/mods/$name/info.json" || {
    echo "the stand-in's info.json was not re-versioned to $version" >&2; exit 1; }
  stamp_engine "$work/mods/$name"
}

# stage_mig <workdir> <extra-mod-name> <standin-version-or-empty> <bbb-in-phase-one>
#
# A non-empty version means the extra mod is an INCUMBENT STAND-IN, staged under
# that name by mig_standin. An empty one means it is a real directory under
# test/mods (bbb-mig-foreign, the stranger).
stage_mig() {
  local work="$1" extra="$2" ver="$3" bbb="$4"
  rm -rf "$work"
  mkdir -p "$work/mods"
  copy_testmod bbb-mig-test "$work/mods/bbb-mig-test"
  stamp_engine "$work/mods/bbb-mig-test"
  if [ -n "$extra" ]; then
    if [ -n "$ver" ]; then
      mig_standin "$work" "$extra" "$ver"
    else
      copy_testmod "$extra" "$work/mods/$extra"
      stamp_engine "$work/mods/$extra"
    fi
  fi
  [ "$bbb" = true ] && cp -R "$MOD_DIR" "$work/mods/"
  mig_list "$work" "$extra" "$bbb"
}

# Which incumbent the leg being staged is about. The BETWEEN hooks below are the
# only readers, and they need it because "remove the incumbent" has to know
# which of the four names the stand-in went in under.
MIG_INCUMBENT=belt-balancer-2
MIG_INCUMBENT_VERSION=2.0.9

# The incumbent goes, this mod arrives, in one edit of the mod list -- which is
# what a player does when they read "use this instead".
mig_swap_in() {
  rm -rf "$1/mods/$MIG_INCUMBENT"
  cp -R "$MOD_DIR" "$1/mods/"
  mig_list "$1" "" true
  echo "==> $MIG_INCUMBENT uninstalled and $MOD_NAME installed: this is the swap"
}

# This mod was already there; the incumbent leaves a session later.
mig_drop_incumbent() {
  rm -rf "$1/mods/$MIG_INCUMBENT"
  mig_list "$1" "" true
  echo "==> $MIG_INCUMBENT uninstalled; $MOD_NAME was already installed"
}

# The stranger STAYS. This mod arrives beside it and must leave it alone.
mig_add_bbb_beside_foreign() {
  cp -R "$MOD_DIR" "$1/mods/"
  mig_list "$1" "bbb-mig-foreign" true
  echo "==> $MOD_NAME installed beside bbb-mig-foreign, which still owns balancer-part"
}

# THE STRANGER LEAVES, which is a promise `legacyCheck` makes in as many words --
# "the stranger can be uninstalled too, and on that load the stub appears and
# their balancers become ours, which is the same promise the incumbents get" --
# and which nothing tested until this hook existed.
mig_drop_foreign() {
  rm -rf "$1/mods/bbb-mig-foreign"
  mig_list "$1" "" true
  echo "==> bbb-mig-foreign uninstalled; the stub appears and its balancers become ours"
}

# THE INCUMBENT ARRIVES AFTER US: the Done -> Blocked recheck that
# fk_on_configuration_changed exists for, and the only transition of the state
# machine that no leg drove. A player installs this mod, uses it, and then
# installs a Belt Balancer beside it.
mig_add_incumbent() {
  mig_standin "$1" "$MIG_INCUMBENT" "$MIG_INCUMBENT_VERSION"
  mig_list "$1" "$MIG_INCUMBENT" true
  echo "==> $MIG_INCUMBENT INSTALLED beside $MOD_NAME, which was already there"
}

# --- the mig21 suite's staging ----------------------------------------------
#
# THE ONLY SUITE WITH NO `--create` PHASE AT ALL, and it cannot have one: the
# worlds it is about were built by a Factorio 2.0.77 binary that no longer exists
# on any machine here, and a 2.1 Factorio refuses to build a multi-edge balancer
# at the prototype level. The saves are committed instead
# (test/fixtures-2.0/README.md) and THE FIXTURE IS PHASE ONE.
#
# What has to be staged around one, and every item is load-bearing:
#
#   THE FIXTURE'S OWN TEST MOD, for its DATA STAGE. Each of these worlds is full
#   of that mod's loaders and lane splitters, and Factorio deletes every entity
#   whose prototype went with a removed mod at load, before any script runs -- so
#   dropping it would delete half the rig and the migration would be measured
#   against a world nobody built.
#
#   ...WITH ITS CONTROL STAGE NEUTRALIZED. What is under test is what THIS mod
#   does to the world at load. The test mod's own schedule would drive rig edits,
#   forced recompiles and rate measurements on top of it from tick 0, against a
#   world its `on_init` never set up -- the fixture already carries its storage.
#   Measured before it was cut: the m2 mod's on_tick raises outright on the
#   fixture load. bbb-mig21-observer does the measuring instead.
#
#   `factorio_version` STAMPED FOR THE RUNNING ENGINE, because a Factorio
#   refuses a mod whose info.json names the other series before it places an
#   entity. That is `stamp_engine` now and is no longer special to this suite.
#
# The test mod's own VERSION is deliberately left alone. What declines the saved
# guest heap is THIS mod's build stamp, which moved because the guest was
# rebuilt; the assertion script REQUIRES the rebuild-from-world line rather than
# assuming it, because a leg that silently adopted the old heap would test
# nothing at all.

# stage_fixture <workdir> <fixture> <test-mod>
stage_fixture() {
  local work="$1" fixture="$2" testmod="$3"
  rm -rf "$work"
  mkdir -p "$work/mods"
  cp -R "$MOD_DIR" "$work/mods/"
  copy_testmod "$testmod" "$work/mods/$testmod"
  copy_testmod bbb-mig21-observer "$work/mods/bbb-mig21-observer"
  stamp_engine "$work/mods/$testmod" "$work/mods/bbb-mig21-observer"

  # The prototypes, and nothing else. See above.
  cat > "$work/mods/$testmod/control.lua" <<'LUA'
-- Neutralized by test/run.sh for the mig21 suite: this mod is staged for its
-- DATA STAGE only, because the fixture world is full of its prototypes and
-- Factorio would delete every one of those entities at load without it. What is
-- under test is what better-belt-balancer does to that world, and
-- bbb-mig21-observer is what measures it.
LUA

  cp "$ROOT/test/fixtures-2.0/$fixture.zip" "$work/map.zip"
  cat > "$work/mods/mod-list.json" <<JSON
{
  "mods": [
    { "name": "base", "enabled": true },
    { "name": "elevated-rails", "enabled": false },
    { "name": "quality", "enabled": true },
    { "name": "space-age", "enabled": false },
    { "name": "$MOD_NAME", "enabled": true },
    { "name": "$testmod", "enabled": true },
    { "name": "bbb-mig21-observer", "enabled": true }
  ]
}
JSON
}

# load_fixture <workdir> <ticks> -- the benchmark phase on its own.
#
# `--benchmark` never saves, so there is nothing after this and, the fixture
# being phase one, nothing before it: one Factorio run over a save this
# repository committed rather than made.
load_fixture() {
  local work="$1" ticks="$2"
  echo "==> loading the 2.0 fixture for ${ticks}t"
  if ! "$FACTORIO" -c "$CONFIG" --mod-directory "$work/mods" --benchmark "$work/map.zip" \
        --benchmark-ticks "$ticks" --benchmark-runs 1 --disable-audio \
        >"$work/run.log" 2>&1; then
    echo "the fixture load failed; see $work/run.log" >&2
    tail -40 "$work/run.log" >&2; exit 1
  fi
  if grep -qE "stack traceback|Error while running" "$work/run.log"; then
    echo "script error during the fixture load; see $work/run.log" >&2
    grep -nE "Error|traceback" "$work/run.log" | head >&2; exit 1
  fi
  guest_gate "$work/run.log"
}

# create_phase <workdir> -- the first of the two phases, on its own.
#
# Split out of run() so that a leg which only needs to see what a guest DECIDES
# at load can stop there: `--benchmark` never saves, so a phase costs a whole
# Factorio run, and the mig suite's two name probes have nothing to measure past
# the create log.
create_phase() {
  local work="$1"
  local save="$work/map.zip"
  echo "==> creating save (phase 1 runs in on_init)"
  if ! "$FACTORIO" -c "$CONFIG" --mod-directory "$work/mods" --create "$save" \
        --map-gen-seed 12345 --disable-audio >"$work/create.log" 2>&1; then
    echo "map creation failed; see $work/create.log" >&2
    tail -40 "$work/create.log" >&2; exit 1
  fi
  if grep -qiE "^ *[0-9.]+ Error|stack traceback|Unknown key" "$work/create.log"; then
    echo "errors during map creation; see $work/create.log" >&2
    grep -inE "error|traceback" "$work/create.log" | head -20 >&2; exit 1
  fi
}

# guest_gate <logfile>... -- the two things no run of any suite may contain.
guest_gate() {
  # A compile that failed leaves a loud line and no network. It is never
  # acceptable, in any suite.
  if grep -qE "\[BBB\] error:" "$@"; then
    echo "the guest reported a compile error:" >&2
    grep -hE "\[BBB\] error:" "$@" | head -20 >&2; exit 1
  fi
  # THE OBSERVER'S OWN ERROR LEVEL, and it is here because the port took away
  # the thing that used to do this job.
  #
  # A hand-written test mod called Lua's `error()` when a rig failed to land --
  # `error("could not place X at (x,y)")` -- which aborts on_init outright, and
  # that is the correct severity: a rig that is not there makes every number
  # after it a measurement of a different world. A GUEST CANNOT DO THAT. There
  # are no coroutines under a wasm frame, so nothing can unwind, and the honest
  # equivalent is a line the runner refuses to see.
  #
  # A SEPARATE TAG FROM THE MOD'S OWN. `[BBB] error:` above means one narrow
  # thing -- a compile produced no network -- and is the mod under test speaking.
  # This is the HARNESS speaking, about the world it was asked to build, and one
  # tag serves all fourteen observers whichever half of the port they are in.
  if grep -qE "\[BBB-OBS\] error:" "$@"; then
    echo "the test observer could not build its world:" >&2
    grep -hE "\[BBB-OBS\] error:" "$@" | head -20 >&2; exit 1
  fi
  # THE COLLECTOR'S OWN ROOT-SET LINE, and this assertion is what replaced a
  # hand-rolled check in guest/go/gc.go.
  #
  # `fkgc` logs this once, the first time EffectiveBudget() has to floor the
  # budget because this guest's globals cost more than one step -- which means
  # gcBudget is no longer covering the root re-scan it is built to cover, and
  # the pause is the collector's choice rather than ours. It is not fatal to the
  # game (that is the whole point of the floor upstream added) and it IS fatal
  # to a run here, because gc.go's constant is derived from a number that has
  # just moved and nothing else would say so.
  #
  # Only this line. The other `fkgc:` lines -- an outrun, a refused grow -- are
  # conditions about the allocation rate that a stress suite may legitimately
  # provoke, and the `mar` suite asserts `deadlines=0` over them directly.
  if grep -qE "fkgc: this guest's ROOT SET is larger" "$@"; then
    echo "the collector had to raise this guest's step budget to cover its own" >&2
    echo "globals re-scan: gcRootGranules in guest/go/gc.go is now too small." >&2
    echo "Compare 'roots=' against 'budget='/'eff=' on the [BBB] heap line." >&2
    exit 1
  fi
}

# create_only <workdir> -- one phase and no benchmark. For a leg whose whole
# question is answered by what the guest logged at load.
create_only() {
  create_phase "$1"
  guest_gate "$1/create.log"
}

# run <workdir> <ticks>
run() {
  local work="$1" ticks="$2"
  local save="$work/map.zip"
  create_phase "$work"

  [ -n "${BETWEEN:-}" ] && "$BETWEEN" "$work"

  echo "==> benchmarking ${ticks}t"
  # One run: each --benchmark-runs pass reloads the save and replays the same
  # ticks, which would duplicate every phase in the log.
  if ! "$FACTORIO" -c "$CONFIG" --mod-directory "$work/mods" --benchmark "$save" \
        --benchmark-ticks "$ticks" --benchmark-runs 1 --disable-audio \
        >"$work/run.log" 2>&1; then
    echo "benchmark failed; see $work/run.log" >&2; tail -40 "$work/run.log" >&2; exit 1
  fi
  if grep -qE "stack traceback|Error while running" "$work/run.log"; then
    echo "script error during benchmark; see $work/run.log" >&2
    grep -nE "Error|traceback" "$work/run.log" | head >&2; exit 1
  fi
  guest_gate "$work/create.log" "$work/run.log"
}

for suite in $SUITES; do
  case "$suite" in
    m1)
      echo "=== M1: cluster merge/split ==="
      stage "$TMP/m1" bbb-m1-test
      run "$TMP/m1" "${BBB_TEST_TICKS:-100}"
      echo "==> asserting cluster transitions"
      python3 "$ROOT/test/assert-log.py" "$TMP/m1/create.log" "$TMP/m1/run.log"
      ;;
    qual)
      # EVERY PART IN THIS SUITE IS UNCOMMON QUALITY. `find_entity` resolves a
      # bare name as normal quality only, so a guest lookup that used it worked
      # on every other suite's save and silently failed on a quality-rolled
      # part -- see guest/go/findpart.go for the fix and CLAUDE.md for the four
      # sites it covers. Base plus the quality mod, nothing from Space Age.
      echo "=== qual: a part at uncommon quality is a part everywhere the guest asks ==="
      stage "$TMP/qual" bbb-qual-test false true
      run "$TMP/qual" "${BBB_QUAL_TICKS:-2160}"
      echo "==> asserting the quality paths"
      python3 "$ROOT/test/assert-qual.py" "$TMP/qual/create.log" "$TMP/qual/run.log"
      ;;
    m2)
      echo "=== M2: compiled network balance ==="
      stage "$TMP/m2" bbb-m2-test
      run "$TMP/m2" "${BBB_M2_TICKS:-3600}"
      echo "==> asserting network behaviour"
      python3 "$ROOT/test/assert-m2.py" "$TMP/m2/create.log" "$TMP/m2/run.log"
      ;;
    m3)
      # Twelve lifecycle rigs and 600 ticks of randomised churn, all built to
      # the one-belt rule. The churn AVOIDS the refusal rather than embracing
      # it -- its edits are aimed at a row's own single input, its own single
      # output and an EDGELESS part below the column -- and the suite asserts
      # that negative, because a refusal would make its compile and teardown
      # counters a function of the rule rather than of the path under test.
      echo "=== M3: lifecycle hardening ==="
      stage "$TMP/m3" bbb-m3-test
      run "$TMP/m3" "${BBB_M3_TICKS:-1560}"
      echo "==> asserting lifecycle behaviour"
      python3 "$ROOT/test/assert-m3.py" "$TMP/m3/create.log" "$TMP/m3/run.log"
      ;;
    upg)
      # The M2 rigs, created by one guest and loaded by another. The saved heap
      # is discarded, so the first event of the benchmark phase re-derives the
      # whole registry from the world -- and ADOPTS the networks already
      # standing rather than rebuilding them. Whether it adopted correctly is
      # then decided by running M2's own assertions over the result: if a single
      # network were mis-adopted, its rig would not balance.
      echo "=== M3: a mod upgrade -- the guest heap is discarded mid-save ==="
      stage "$TMP/upg" bbb-m2-test
      BETWEEN=bump_build run "$TMP/upg" "${BBB_M2_TICKS:-3600}"
      unset BETWEEN
      echo "==> asserting the rebuild, then M2's own numbers over it"
      python3 "$ROOT/test/assert-upgrade.py" "$TMP/upg/create.log" "$TMP/upg/run.log"
      python3 "$ROOT/test/assert-m2.py" "$TMP/upg/create.log" "$TMP/upg/run.log"
      ;;
    plat)
      echo "=== Space Age: a space platform surface, and belt stacking ==="
      stage "$TMP/plat" bbb-plat-test true
      run "$TMP/plat" "${BBB_PLAT_TICKS:-1560}"
      echo "==> asserting the platform rig and the stacking leg"
      python3 "$ROOT/test/assert-plat.py" "$TMP/plat/create.log" "$TMP/plat/run.log"
      ;;
    mar)
      # The `-gc=leaking` marathon question, measured rather than argued: what
      # does ONE net-zero world cycle cost the guest heap permanently, and is
      # that slope flat. The instrument is the guest's own `[BBB] heap` probe,
      # read after each of ~500 cycles.
      echo "=== marathon: the permanent-heap slope per world operation ==="
      stage "$TMP/mar" bbb-marathon-test
      run "$TMP/mar" "${BBB_MAR_TICKS:-4600}"
      echo "==> fitting the slopes"
      python3 "$ROOT/test/assert-marathon.py" "$TMP/mar/create.log" "$TMP/mar/run.log"
      ;;
    edge)
      # Correctness under churn: every edit that lands while a network is
      # MID-OPERATION, with items in flight. Conservation across each one is the
      # assertion, because a teardown that drops items is the failure this
      # architecture can have and the incumbent cannot -- and, since 2026-08-02,
      # WHERE they end up: a recompile must put them back inside the network it
      # just built and only a real removal may spill them. Every count carries a
      # ground-item total and the recompile tags must read zero.
      # Fifteen clusters over one hundred and ninety-eight parts under the
      # one-belt rule. Four rigs are a redesign rather than a re-lay, because
      # the EDIT they take is what the rule changed: `aout`, `ain` and `bmin`
      # carry a spare part with nothing against it, `lim` is sixty-four belts
      # over sixty-six parts, `brdg`'s gap tile is flanked by one belt rather
      # than two, and `frepa`'s belt line ends on the tile the part lands on.
      echo "=== edge: mid-operation churn, merges, splits and forces ==="
      stage "$TMP/edge" bbb-edge-test
      run "$TMP/edge" "${BBB_EDGE_TICKS:-5900}"
      echo "==> asserting conservation and the edge cases"
      python3 "$ROOT/test/assert-edge.py" "$TMP/edge/create.log" "$TMP/edge/run.log"
      ;;
    mix)
      # MORE THAN ONE KIND. Every other suite runs iron plates through
      # everything, which is the right default for a throughput number and left
      # the multi-kind half of guest/go/carry.go unexercised: the pool's
      # (name, quality, stack size) key, the per-kind split, insertRemainder's
      # walk over several groups, and the BOUND on how many groups one pool can
      # carry -- which until 2026-08-04 dropped what it could not hold, after
      # the drain had already read it off a line about to be destroyed.
      # Every count in this suite is per item NAME, on both surfaces, inside one
      # tick -- because a teardown that dropped one kind and reinserted the rest
      # conserves nothing and a single total can hide it.
      echo "=== mix: several item kinds through one balancer ==="
      stage "$TMP/mix" bbb-mix-test
      run "$TMP/mix" "${BBB_MIX_TICKS:-3220}"
      echo "==> asserting per-kind conservation and the mixed-load rates"
      python3 "$ROOT/test/assert-mix.py" "$TMP/mix/create.log" "$TMP/mix/run.log"
      ;;
    mig)
      # THE ONLY SUITE WHOSE TWO PHASES RUN UNDER DIFFERENT MOD SETS. Seven legs
      # and two name probes, covering both axes the feature has: WHICH MOD owns
      # `balancer-part`, and WHICH TRANSITION of legacy.go's state machine the
      # load makes.
      echo "=== mig: adopting a Belt Balancer 2 or 3 save ==="

      echo "--- leg 1: the incumbent swapped out and this mod in, in one edit ---"
      MIG_INCUMBENT=belt-balancer-2 MIG_INCUMBENT_VERSION=2.0.9
      stage_mig "$TMP/mig1" belt-balancer-2 2.0.9 false
      BETWEEN=mig_swap_in run "$TMP/mig1" "${BBB_MIG_TICKS:-3600}"
      unset BETWEEN
      python3 "$ROOT/test/assert-mig.py" --engine "$ENGINE_SERIES" --leg added \
        "$TMP/mig1/create.log" "$TMP/mig1/run.log"

      echo "--- leg 2: this mod installed first, the incumbent removed later ---"
      stage_mig "$TMP/mig2" belt-balancer-2 2.0.9 true
      BETWEEN=mig_drop_incumbent run "$TMP/mig2" "${BBB_MIG_TICKS:-3600}"
      unset BETWEEN
      python3 "$ROOT/test/assert-mig.py" --engine "$ENGINE_SERIES" --leg later \
        "$TMP/mig2/create.log" "$TMP/mig2/run.log"

      # THE LIVE SUCCESSOR, and the leg is the coexistence shape rather than the
      # swap shape on purpose. A name this guest does not recognise degrades into
      # the STRANGER path, which is Blocked with no log line -- so a leg that only
      # removed belt-balancer-3 and watched the conversion happen would pass with
      # the name misspelled. The NAMED blocked line from phase one is the only
      # observable that pins the row of `legacyIncumbents`.
      echo "--- leg 3: Belt Balancer 3 installed beside this mod, then removed ---"
      MIG_INCUMBENT=belt-balancer-3 MIG_INCUMBENT_VERSION=1.0.1
      stage_mig "$TMP/mig3" belt-balancer-3 1.0.1 true
      BETWEEN=mig_drop_incumbent run "$TMP/mig3" "${BBB_MIG_TICKS:-3600}"
      unset BETWEEN
      python3 "$ROOT/test/assert-mig.py" --engine "$ENGINE_SERIES" --leg bb3 \
        "$TMP/mig3/create.log" "$TMP/mig3/run.log"

      # No incumbent has ever been installed here, so `balancer-part` is this
      # mod's own stub from the first byte and the observer's parts arrive one at
      # a time through BUILD EVENTS -- which is the path an old blueprint's
      # ghosts take. It is also the only leg whose save is written AFTER the
      # conversion and whose second phase changes no mod at all, so it is the
      # only one that can show the once-per-save flag surviving a save and
      # costing nothing on the way back.
      echo "--- leg 4: legacy parts arriving through build events, then a plain reload ---"
      stage_mig "$TMP/mig4" "" "" true
      run "$TMP/mig4" "${BBB_MIG_TICKS:-3600}"
      python3 "$ROOT/test/assert-mig.py" --engine "$ENGINE_SERIES" --leg built \
        "$TMP/mig4/create.log" "$TMP/mig4/run.log"

      # DONE -> BLOCKED, the one transition fk_on_configuration_changed exists
      # for that nothing drove: this mod first, an incumbent installed beside it
      # afterwards. Phase one is leg 4's -- our stub, converted through the build
      # path, the save written with the phase Done -- and then belt-balancer-2
      # arrives. The teeth are in the late build: it places the INCUMBENT'S
      # `balancer-part` now, and a build-path gate reading the wrong phase would
      # swap a working mod's freshly built entity out from under it.
      echo "--- leg 5: an incumbent INSTALLED after this mod, on a converted save ---"
      MIG_INCUMBENT=belt-balancer-2 MIG_INCUMBENT_VERSION=2.0.9
      stage_mig "$TMP/mig5" "" "" true
      BETWEEN=mig_add_incumbent run "$TMP/mig5" "${BBB_MIG_TICKS:-3600}"
      unset BETWEEN
      python3 "$ROOT/test/assert-mig.py" --engine "$ENGINE_SERIES" --leg readd \
        "$TMP/mig5/create.log" "$TMP/mig5/run.log"

      echo "--- leg 6: a stranger owns balancer-part and must be left alone ---"
      stage_mig "$TMP/mig6" bbb-mig-foreign "" false
      BETWEEN=mig_add_bbb_beside_foreign run "$TMP/mig6" "${BBB_MIG_TICKS:-3600}"
      unset BETWEEN
      python3 "$ROOT/test/assert-mig.py" --engine "$ENGINE_SERIES" --leg foreign \
        "$TMP/mig6/create.log" "$TMP/mig6/run.log"

      # THE STRANGER REMOVED. `legacyCheck` promises this in as many words and
      # nothing tested it: a Blocked state re-tests the marker prototype, so the
      # load on which the stranger leaves is a load on which our stub appears and
      # their balancers become ours -- the same promise the incumbents get.
      echo "--- leg 7: the stranger UNINSTALLED, and its balancers become ours ---"
      stage_mig "$TMP/mig7" bbb-mig-foreign "" true
      BETWEEN=mig_drop_foreign run "$TMP/mig7" "${BBB_MIG_TICKS:-3600}"
      unset BETWEEN
      python3 "$ROOT/test/assert-mig.py" --engine "$ENGINE_SERIES" --leg fgone \
        "$TMP/mig7/create.log" "$TMP/mig7/run.log"

      # THE OTHER TWO ROWS OF `legacyIncumbents`, one create phase each. What a
      # full leg would add over leg 2 is nothing -- the conversion side is
      # identical whichever name blocked it -- and what it would cost is a
      # benchmark phase. The name is the only thing under test, so the blocked
      # line is the only thing asserted.
      echo "--- probes: the two incumbent names with no leg of their own ---"
      stage_mig "$TMP/migp1" belt-balancer 3.4.4 true
      create_only "$TMP/migp1"
      python3 "$ROOT/test/assert-mig.py" --engine "$ENGINE_SERIES" --leg probe \
        --incumbent belt-balancer --version 3.4.4 "$TMP/migp1/create.log"

      stage_mig "$TMP/migp2" belt-balancer-performance 1.0.5 true
      create_only "$TMP/migp2"
      python3 "$ROOT/test/assert-mig.py" --engine "$ENGINE_SERIES" --leg probe \
        --incumbent belt-balancer-performance --version 1.0.5 "$TMP/migp2/create.log"
      ;;
    sedge)
      # FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART. Every edge is an
      # interface linked belt standing on the cluster's own tile, and 2.1 closed
      # the collision-mask loophole that let two of them share one. So the port
      # is a RULE change and this is the suite for it: four single-edge shapes
      # measured against a bare express belt with their PORT COUNTS asserted
      # first, and the three ways an edit can ask for a second belt on one part
      # -- built, ROTATED (which raises no event at all, so the audit is what
      # finds it) and a MERGE, whose teardowns are AddPart's and are queued
      # before the compiler ever sees the cluster they make.
      #
      # See agents/single-edge.md for the design and guest/go/sedge.go for the
      # rule. `m3`, `edge` and `mig` are the suites still built in the
      # multi-edge idiom and they do not run on 2.1; rebuilding their rigs is
      # a later phase.
      echo "=== sedge: one belt per balancer part, and every way of breaking it ==="
      stage "$TMP/sedge" bbb-sedge-test
      run "$TMP/sedge" "${BBB_SEDGE_TICKS:-3500}"
      echo "==> asserting the rule and its refusals"
      python3 "$ROOT/test/assert-sedge.py" "$TMP/sedge/create.log" "$TMP/sedge/run.log"
      ;;
    mig21)
      # A FACTORIO 2.0 MULTI-EDGE SAVE, OPENED ON 2.1. The one suite with no
      # `--create` phase: its worlds were built by a 2.0.77 binary that is gone,
      # and a 2.1 Factorio cannot rebuild them at any price, so the saves are
      # committed under test/fixtures-2.0/ and each is phase one.
      #
      # What the load has to survive is not hypothetical. The ENGINE gets there
      # first: opening one of these under 2.1.14 silently deletes all but one
      # belt-connectable per tile, with no log line of any kind, and leaves the
      # hidden networks fully intact -- so the guest wakes into balancers whose
      # standing networks are missing most of their interfaces and whose
      # remaining ports are a lottery. The migration's job is to turn that into
      # clusters that are refused and explained, with what the hidden half was
      # still holding recovered and put on the ground beside each one.
      #
      # Two fixtures, and they are different shapes rather than two of a kind:
      # m2's 21 rigs are the ordinary geometry a player builds, and edge's `lim`
      # is 64 belts over 32 parts -- the biggest network this mod makes, and
      # tiles carrying THREE.
      echo "=== mig21: a Factorio 2.0 multi-edge save, opened on $ENGINE_SERIES ==="

      echo "--- m2: 21 rigs, 77 parts, saturated ---"
      stage_fixture "$TMP/mig21-m2" m2-2.0.77 bbb-m2-test
      load_fixture "$TMP/mig21-m2" "${BBB_MIG21_TICKS:-320}"
      python3 "$ROOT/test/assert-mig21.py" --engine "$ENGINE_SERIES" --fixture m2 "$TMP/mig21-m2/run.log"

      echo "--- edge: 15 clusters, 95 parts, including lim at 64 belts over 32 parts ---"
      stage_fixture "$TMP/mig21-edge" edge-2.0.77 bbb-edge-test
      load_fixture "$TMP/mig21-edge" "${BBB_MIG21_TICKS:-320}"
      python3 "$ROOT/test/assert-mig21.py" --engine "$ENGINE_SERIES" --fixture edge "$TMP/mig21-edge/run.log"
      ;;
    flip)
      # `bbb-multi-edge-parts`, DRIVEN. The setting exists on Factorio 2.0 only
      # -- guest/go/data/settings.go defines it on 2.0.x and never on 2.1.x -- so
      # this is the one suite that cannot run on trunk's own engine, and it says
      # so out loud rather than passing.
      #
      # A CHECK THAT SKIPS IS A CHECK THAT PASSED (CLAUDE.md's own review-gate
      # finding), so the skip is a line in the log and a line in the assertion
      # script's output, not a silent `continue`. What covers the FOLD on 2.1 is
      # `go test ./edgemode/`, which proves all eighteen of its states; what
      # nothing covers there is the WORLD, which is what this drives.
      echo "=== flip: the multi-edge setting, turned on and off under load ==="
      if [ "$ENGINE_SERIES" != "2.0" ]; then
        echo "==> SKIPPED on Factorio $ENGINE_SERIES: \`bbb-multi-edge-parts\` is"
        echo "    defined on 2.0.x only, so there is nothing here to flip. The fold"
        echo "    behind it is proved by \`go test ./edgemode/\` on every engine."
        continue
      fi
      stage "$TMP/flip" bbb-flip-test
      run "$TMP/flip" "${BBB_FLIP_TICKS:-3200}"
      echo "==> asserting the flip, the veto and the refusals"
      python3 "$ROOT/test/assert-flip.py" "$TMP/flip/create.log" "$TMP/flip/run.log"
      ;;
    iact)
      # THE INTERACTIVE CHECKLIST'S OWN WORLD, staged and never touched.
      #
      # `test/interactive/bbb-interactive-setup` is what a human enables before
      # walking test/interactive/README.md, and it is also where the mod
      # portal's demo scenes live. Nothing headless can make the gestures -- if
      # anything could, the checklist would not exist -- but everything it
      # STAGES is ordinary world-building, and a rig that no longer lands, or
      # one this mod itself refuses, is a checklist that wastes a human's
      # session before they have done anything.
      #
      # One `--create` and no benchmark: the whole question is answered by what
      # the guest logged at load. What it asserts is that every piece landed,
      # every rig compiled to the SHAPE the geometry intended, and NOTHING was
      # refused -- the gestures are what create the refusals, and a refusal
      # here means a rig is built to something this Factorio cannot compile.
      echo "=== iact: the interactive checklist's staged world ==="
      stage_interactive "$TMP/iact"
      create_only "$TMP/iact"
      echo "==> asserting the staged world"
      python3 "$ROOT/test/assert-interactive.py" "$TMP/iact/create.log"
      ;;
    *)
      echo "unknown suite: $suite (expected m1, m2, m3, upg, plat, mar, edge, mix, mig, qual, sedge, mig21, flip or iact)" >&2
      exit 1
      ;;
  esac
done
