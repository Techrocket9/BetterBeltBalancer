#!/usr/bin/env bash
# BLOB PROVENANCE FOR THE `release/2.0` ARM. Git plumbing and nothing else: no
# engine, no network, no build, and the working TREE is never read -- every
# question is asked of two refs, so a feature branch or a detached worktree
# answers exactly as `master` does. Run by `make check`.
#
# WHAT THE TWO ARMS ARE. Trunk targets Factorio 2.1; `release/2.0` carries the
# last 2.0 release, because `factorio_version` is one value per release and the
# portal serves the right release per game version. The branch is cut FROM
# TRUNK at release time and the difference is a four-file stamp; the procedure
# is agents/single-edge.md, "Packaging: one tree, two releases". BETWEEN
# RELEASES THE BRANCH IS BEHIND TRUNK AND THAT IS NORMAL: it carries the last
# 2.0 release rather than head, so `master` being N commits ahead is the
# expected state and this script does not gate it.
#
# SO "THE TWO ARMS CARRY IDENTICAL SOURCE" IS NOT THE PROPERTY TO GATE. It is
# true for the few hours around a release and false the moment trunk lands a
# commit, and a gate that is red between releases is a gate that gets ignored.
# THE PROPERTY THAT HOLDS CONTINUOUSLY is provenance: nothing a 2.0 player
# receives was WRITTEN ON THE BRANCH. Every path the branch changed since its
# cut point must carry a blob that some commit reachable from `master` also
# carries, byte for byte. A recut satisfies that (trunk's tree, copied
# wholesale) and so does a trunk fix backported onto the branch file for file,
# which is what a 2.0 hotfix arm legitimately is. What does NOT satisfy it is a
# line typed on the branch, or a cherry-pick from somewhere that is not trunk,
# which is the drift this exists for.
#
# WHAT THAT PROVES AND WHAT IT DOES NOT, because the ok line below is a claim
# about LINES and must not be read as one about TREES. The unit here is the
# FILE: each changed path carries some trunk blob byte for byte, so no line on
# the branch is a line trunk never had. It does NOT say the matched blobs ever
# coexisted on trunk, that the commit each came from was not later REVERTED
# (`rev-list` walks everything reachable and a reverted commit stays
# reachable), or that the branch holds trunk's LATEST value for a path rather
# than an older one. None of the three is reachable on today's branch, where
# all eight land on the single commit cb89891, and each would still be a
# backport somebody made rather than a line typed here. The invariant also
# rests on the house rule that nothing is ever MERGED into trunk: a merged
# branch's own commits become reachable from `master`, and every file it wrote
# would then match itself.
#
# THE EXCLUSIONS ARE NAMED HERE AND EACH CARRIES ITS REASON. There is no silent
# skip list; a path that is not one of these six fails by name.
#
#   THE STAMP, four files, and a recut is DEFINED as moving exactly these:
#     fklua.toml               the series, the base dependency, the api pin and
#                              the version -- the four keys that make this the
#                              2.0 arm at all
#     fklua.lock               that pin's lock
#     guest/go/fkapi/fkapi.go  the bindings generated FROM that pin
#     mod-data/changelog.txt   its top section has to be the manifest's version,
#                              because check-changelog says so
#   By construction none of the four can match a trunk blob: they hold the 2.0
#   values trunk does not have. Excluding them is not a hole, because each is
#   gated on the branch's own `make check` and `make mod` by something that
#   reads its CONTENT rather than its history -- `fklua gen-bindings --check`
#   and `fklua lock --check` for the bindings and the lock, check-changelog.py
#   for the changelog, and the manifest is three of the keys those read.
#
#   THE RECUT'S OWN WORKING NOTES, two files, REPORTED RATHER THAN GATED:
#     CLAUDE.md, README.md
#   Neither is source: `make mod` packages `mod-data/` and the built wasm, and
#   neither file reaches the package. They are kept out of the FAILURE and
#   named on every run instead, because a BACKPORT NEED NOT make them match:
#   the paragraph it carries is trunk's and it lands in an older surrounding
#   document, so the blob CAN differ while not one sentence is new, and failing
#   on that would be a permanently red gate on a legitimate hotfix arm. It is a
#   "need not" and not a "cannot", and the counterexample is two lines down.
#   FKLUA-GAPS.md is a document too, it is deliberately NOT excluded with them,
#   and the branch's copy IS byte-identical to trunk's: that is the measurement
#   saying a document can survive a backport untouched rather than an
#   assumption that it will. The list names files rather than a directory on
#   purpose: an exception that names files is auditable, and one that names a
#   directory grows without anybody deciding to. AND ONE THING THE NEXT RECUT
#   WILL MEET: test/datastage-goldens.json is not excluded either, so a 2.0
#   golden RE-CAPTURED ON THE BRANCH fails by name. That is the right answer
#   rather than a gap, because the row is trunk's 2.0-flavour emission taken
#   from this tree wherever a 2.0 binary is, but it is a surprise better met
#   here than in a red gate.
#
# MEASURED 2026-09-09, `master` at 760d618 and `release/2.0` at 4b9f597: 14
# paths differ from the cut point cf5a78e (trunk 0.3.2) and the split is 8/4/2.
# The eight are byte-identical to their blobs at cb89891, which is TRUNK'S OWN
# freeze-fix commit, so the branch is trunk 0.3.2's source with a trunk fix
# backported file for file. The four are the stamp and the two are the notes.
# The branch is honest; what was wrong was the sentence about it.
set -uo pipefail

TRUNK=master
ARM=release/2.0

# Excluded from the FAILURE, each for the reason above.
STAMP=(fklua.toml fklua.lock guest/go/fkapi/fkapi.go mod-data/changelog.txt)
NOTES=(CLAUDE.md README.md)

say() { printf 'release-arm: %s\n' "$1"; }

# GIT ITSELF IS PROBED FIRST, AND ITS ABSENCE IS A FAILURE RATHER THAN A SKIP.
# Without this, `git` not on PATH and `git` refusing a directory are the same
# nonzero status, and a broken environment prints the export's line and its
# diagnosis. A missing binary is not an export. Red-proven against
# `PATH=/tmp/nogit:/bin`.
if ! command -v git >/dev/null 2>&1; then
  say "FAIL -- \`git\` is not on PATH, so neither arm can be read."
  say "  This is a broken environment; it is not an export and it is not drift."
  exit 1
fi

# A SKIP IS EXIT 0 WITH A NAMED LINE, never a silent pass and never a failure.
# `git clone --single-branch --branch master` has no `release/2.0`, and an
# exported tarball has no repository at all; failing a clone that never asked
# for the branch would be worse than the drift this gate is about. Same
# convention as check-datastage.py's golden whose engine is not the binary.
if ! git rev-parse --git-dir >/dev/null 2>&1; then
  say "SKIP -- not a git repository, so the two arms cannot be compared."
  say "  This is an export rather than a checkout, and there is nothing to gate."
  exit 0
fi

# A SHALLOW CLONE IS A SKIP AT EXIT 0 AND IS PROBED BEFORE HISTORY IS ASKED
# ANYTHING, because every answer below would be taken over a truncated history.
# `git clone --depth 1` grafts the history short, so the cut point is not in it
# and `git merge-base` answers nothing; even where it did answer, the per-path
# `rev-list` walk could not see trunk's older blobs and a legitimate backport
# would be reported as a line written on the branch. Red-proven against a real
# `git clone --depth 1 --no-single-branch`, which before this probe printed the
# ok banner at exit 0 over two `fatal:` lines on stderr.
if [ "$(git rev-parse --is-shallow-repository 2>/dev/null)" = true ]; then
  say "SKIP -- this is a shallow clone, so the cut point is not in this history."
  say "  Every answer here would be taken over truncated history, which would report"
  say "  a legitimate backport as drift."
  say "  \`git fetch --unshallow\` to gate it."
  exit 0
fi
for ref in "$TRUNK" "$ARM"; do
  if ! git rev-parse --verify --quiet "$ref^{commit}" >/dev/null; then
    say "SKIP -- no \`$ref\` in this clone, so the two arms cannot be compared."
    say "  This is a single-branch clone, not drift."
    say "  \`git fetch origin $ref:$ref\` to gate it."
    exit 0
  fi
done

# EVERY ANSWER BELOW IS GUARDED, AND THAT IS THE LOAD-BEARING PART OF THIS
# SCRIPT. AN EMPTY STRING IS EXACTLY WHAT A FAILED `git` LOOKS LIKE, and an
# unguarded one reaches the ok banner as a clean answer: with an empty $BASE
# every counter comes out 0 and the last thing printed is
# `ok ... 0 path(s) differ`, at exit 0, with the two `fatal:` lines buried
# above it on stderr -- which is the "a skipped gate reads exactly like a pass"
# failure this repository forbids by name, in the gate written to stop a false
# claim. The same empty $BASE turns `git show "$BASE:fklua.toml"` into
# `git show ":fklua.toml"`, which is git's spelling for stage 0 of the INDEX,
# so the banner would report a version read out of the WORKING TREE that the
# header above promises is never read. Both are red-proven: a shallow clone and
# two branches with no common ancestor. THE RULE IS THAT AN EMPTY ANSWER FROM
# GIT IS NEVER A CLEAN ONE HERE, and the one place where it genuinely is (a
# path trunk never had, in the loop below) says so at that line.
#
# NO COMMON ANCESTOR IS A FAILURE AND NOT A SKIP. A shallow clone is an
# environmental difference and skipped above; two branches with no shared
# history is the drift this gate exists for, at its widest.
if ! BASE=$(git merge-base "$TRUNK" "$ARM" 2>/dev/null) || [ -z "$BASE" ]; then
  say "FAIL -- \`$TRUNK\` and \`$ARM\` have no common ancestor, so \`$ARM\` was not cut"
  say "  from trunk at all and every line on it is a line trunk never had."
  say "  Recut the branch from the trunk commit the release is meant to carry. See"
  say "  agents/single-edge.md, \"Packaging: one tree, two releases\"."
  exit 1
fi

BASE_SHORT=$(git rev-parse --short "$BASE" 2>/dev/null)
BASE_VER=$(git show "$BASE:fklua.toml" 2>/dev/null | sed -n 's/^version = "\(.*\)"$/\1/p')
ARM_VER=$(git show "$ARM:fklua.toml" 2>/dev/null | sed -n 's/^version = "\(.*\)"$/\1/p')
AHEAD=$(git rev-list --count "$ARM..$TRUNK" 2>/dev/null)
# The two TREES are the anti-vacuity for the diff below: `0 path(s) differ` is
# honest only when the arm's tree IS the cut point's, and asserting that is
# what makes an empty $CHANGED unable to wear the ok banner.
ARM_TREE=$(git rev-parse --verify --quiet "$ARM^{tree}")
BASE_TREE=$(git rev-parse --verify --quiet "$BASE^{tree}")
for answer in "the cut point's short name=$BASE_SHORT" \
              "the cut point's version out of fklua.toml=$BASE_VER" \
              "the arm's version out of fklua.toml=$ARM_VER" \
              "how far $TRUNK is ahead of $ARM=$AHEAD" \
              "the arm's tree=$ARM_TREE" \
              "the cut point's tree=$BASE_TREE"; do
  [ -n "${answer#*=}" ] && continue
  say "FAIL -- git could not answer ${answer%%=*}, so nothing below it can be trusted."
  say "  A broken environment or a repository this gate cannot read, not drift."
  exit 1
done

# `--no-relative` because `diff.relative=true` in a user's config, with the
# gate run from a subdirectory, returns paths that are both cwd-scoped and
# cwd-relative: every one then misses its spec below and is misreported as a
# deletion on the branch, a FALSE FAIL. Red-proven from test/ with that config.
# `core.quotePath=false` so a non-ASCII path comes back raw instead of
# C-quoted, since a quoted path would miss its spec the same way. The
# `rev-list` in the loop already defends itself with `:(top)`.
if ! CHANGED=$(git -c core.quotePath=false diff --no-relative --name-only "$BASE" "$ARM"); then
  say "FAIL -- git could not diff the cut point against \`$ARM\`, so nothing below it"
  say "  can be trusted. A broken environment or a repository this gate cannot read,"
  say "  not drift."
  exit 1
fi

matched=0
stamped=0
noted=0
total=0
bad=()
gone=()

# THE WALK IS PER DIFFERING PATH AND IS BOUNDED BY THAT PATH'S OWN HISTORY,
# never by the repository's. `git rev-list --full-history <trunk> -- <path>` is
# every commit whose blob for that path differs from a parent's, which is
# exactly the set of distinct values the path has ever held on trunk, and one
# `git cat-file --batch-check` then reads them all in a single process, WITH
# THE BRANCH'S OWN BLOB AS THE FIRST LINE OF THAT SAME BATCH -- so a path costs
# two git processes rather than four, and a path the branch DELETED comes back
# from the same read rather than from a probe of its own. --full-history rather
# than the default simplification, so the answer does not rest on this history
# staying linear.
#
# THE BOUND IS TWO GIT PROCESSES PER PATH THAT DIFFERS, over that path's own
# commits and no others, and it is linear in the number of paths a recut moved
# -- which is four when the branch is healthy. Today that is 8 checked paths
# (14 differ, 6 are excluded above) over 2 to 15 commits each: 0.23 s wall,
# `/usr/bin/time -p`, ten consecutive runs spanning 0.22 to 0.24, and 0.22 to
# 0.25 over five runs in a fresh `--no-hardlinks` clone. The guards above cost
# about 0.02 s of that, three more git spawns over the 0.21 the unguarded draft
# measured, which is the cheapest thing in this file. WHAT THAT 0.2 s IS is
# process spawn and not history: everything before the loop is 12 git processes
# and 0.09 s, the widest walk of a CHECKED path (FKLUA-GAPS.md, 15 commits) is
# under 0.01 s, and 16 bare `git rev-parse HEAD` calls on this machine are
# 0.10 s, so 28 spawns for 8 checked paths IS the number. Fully batching the
# loop into three processes would buy about 0.13 s and cost the per-path
# diagnosis that prints the offending file; against a `make check` that runs
# six Go test packages and three wasm vets, 0.2 s is not where that trade
# goes.
#
# THE GUARD BELOW IS RED-PROVEN AND THE PROOF FOUND A REAL DEFECT. An earlier
# draft took the branch's blob and trunk's candidates out of one batch by
# stripping up to the first newline, which is a no-op when there is no newline
# -- so a file written straight onto the branch, which has NO trunk history at
# all, was compared against itself and passed. A cherry-pick of three files
# from `verify/0.3.3-2.0` was reported as two; the third, a file trunk never
# had, was the one that got through.
while IFS= read -r p; do
  [ -n "$p" ] || continue
  total=$((total + 1))
  skip=
  for x in "${STAMP[@]}"; do [ "$x" = "$p" ] && skip=stamp; done
  for x in "${NOTES[@]}"; do [ "$x" = "$p" ] && skip=note; done
  if [ "$skip" = stamp ]; then stamped=$((stamped + 1)); continue; fi
  if [ "$skip" = note ]; then noted=$((noted + 1)); continue; fi

  # THE ONE PLACE IN THIS SCRIPT WHERE AN EMPTY ANSWER IS A CLEAN ONE, so it is
  # the STATUS that is checked here and not the text: a path trunk NEVER HAD
  # has no history at all, which is precisely the drift the gate exists to
  # catch. A `rev-list` that FAILED would look the same, which is why it is a
  # command substitution with its status read rather than the process
  # substitution this used to be, whose status is unreadable.
  if ! hist=$(git rev-list --full-history "$TRUNK" -- ":(top)$p"); then
    say "FAIL -- git could not walk \`$TRUNK\`'s history for $p, so it was not checked."
    say "  A broken environment or a repository this gate cannot read, not drift."
    exit 1
  fi
  specs="$ARM:$p"
  if [ -n "$hist" ]; then
    while IFS= read -r c; do specs+=$'\n'"$c:$p"; done <<<"$hist"
  fi
  if ! seen=$(printf '%s\n' "$specs" | git cat-file --batch-check='%(objectname) %(objecttype)'); then
    say "FAIL -- git could not read the objects for $p, so it was not checked."
    say "  A broken environment or a repository this gate cannot read, not drift."
    exit 1
  fi
  first=${seen%%$'\n'*}
  # THE FIRST LINE IS THE BRANCH'S OWN AND THE REST ARE TRUNK'S, AND THE REST
  # CAN BE EMPTY. A path trunk never had (a file written straight onto the
  # branch) gets no rev-list output at all, so `seen` is one line; stripping
  # "up to the first newline" would then leave the branch's own blob in place
  # of trunk's candidates and every such file would compare equal to itself.
  # That is red-proven, and this is the guard.
  case $seen in
    *$'\n'*) rest=${seen#*$'\n'} ;;
    *) rest= ;;
  esac
  # `--batch-check` answers a blob as `<sha> blob` and a path the ref does not
  # have as `<spec> missing`, so the TYPE is what tells them apart. Length
  # alone would not: `release/2.0:` plus a 20-character path plus ` missing` is
  # also 40 characters.
  case $first in
    *" blob") want=${first% blob} ;;
    *)
      # Deleted on the branch. A recut copies trunk's tree and cannot delete a
      # file the cut point has, so this is always an edit made on the branch.
      gone+=("$p")
      continue
      ;;
  esac
  case $'\n'"$rest"$'\n' in
    *$'\n'"$want blob"$'\n'*) matched=$((matched + 1)) ;;
    *) bad+=("$p") ;;
  esac
done <<<"$CHANGED"

# THE ANTI-VACUITY ASSERTION, because `0 path(s) differ` under an `ok` banner
# is what EVERY failure of the guards above used to look like. Zero is honest
# only when the arm's tree IS the cut point's; with the trees differing it is
# impossible, so it is an internal failure and never a pass.
if [ "$total" -eq 0 ] && [ "$ARM_TREE" != "$BASE_TREE" ]; then
  say "FAIL -- \`$ARM\`'s tree differs from the cut point $BASE_SHORT and yet no path"
  say "  came back as differing, which cannot happen. This gate checked NOTHING; do"
  say "  not read it as a pass. A broken environment or a repository it cannot read."
  exit 1
fi

if [ ${#bad[@]} -gt 0 ] || [ ${#gone[@]} -gt 0 ]; then
  say "FAIL -- \`$ARM\` $ARM_VER carries source no commit on \`$TRUNK\` ever had."
  if [ ${#bad[@]} -gt 0 ]; then
    say "  written on the branch, or cherry-picked from somewhere that is not trunk:"
    for p in "${bad[@]}"; do say "    $p"; done
  fi
  if [ ${#gone[@]} -gt 0 ]; then
    say "  deleted on the branch, and the cut point $BASE_SHORT has it:"
    for p in "${gone[@]}"; do say "    $p"; done
  fi
  say "  The 2.0 arm is trunk's source with a four-file stamp on it, so every other"
  say "  line it carries has to exist on trunk. Two ways out: land the change on"
  say "  trunk and recut the branch from the trunk commit that carries it, or take"
  say "  it off the branch. See agents/single-edge.md, \"Packaging: one tree, two"
  say "  releases\", and the Makefile's OBS_TAGS block."
  exit 1
fi

say "ok -- $ARM $ARM_VER carries no line \`$TRUNK\` never had."
say "  $total path(s) differ from the cut point $BASE_SHORT (trunk $BASE_VER); $TRUNK is $AHEAD ahead, which is normal between releases."
say "  $matched carry a blob some commit on \`$TRUNK\` carries byte for byte."
say "  $stamped are the recut's stamp: ${STAMP[*]}"
say "  $noted are the recut's own working notes, reported and not gated: ${NOTES[*]}"
exit 0
