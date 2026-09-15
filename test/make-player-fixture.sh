#!/usr/bin/env bash
#
# Derive test/fixtures-player/player-<engine>.zip from a save a graphical client
# made. `make player-fixture SAVE=<path>` is the entry point; nothing else runs
# this, and no suite depends on it at test time.
#
# WHY IT IS A SCRIPT AND NOT A MAKE RECIPE: two Factorio invocations, each a
# BACKGROUND server that has to be polled for a line and then killed. A make
# recipe cannot hold that shape without becoming this file inline.
#
# THE SOURCE SAVE IS READ AND NEVER WRITTEN. It is copied into the work
# directory first and every later step touches the copy; the user's saves
# directory and mods directory are read-only to this script, and every Factorio
# it starts writes into a private `-c` config's write-data.
#
# --- the two passes ---------------------------------------------------------
#
# PASS 1 runs with the DLC enabled and the source save's own world in front of
# it. `bbb-fixplayer` empties that world -- every surface but nauvis, every
# entity but a character, the player's inventories, every chunk past the origin
# -- and writes the result with `game.server_save`.
#
# PASS 2 runs the same guest over pass 1's output with BASE ALONE. That is the
# step that makes the file committable: Factorio drops the prototypes and the
# `storage` of every mod the save names and the mod list does not enable, which
# is where the size goes (measured on the save this fixture was cut from:
# script.dat 5,868,208 B -> 1,281 B). A missing mod at load is a warning in
# headless, which is what the `mig` suite's own mod-set changes already rest on.
#
# --- what the product is, and what it is not --------------------------------
#
# A SERVER LOAD DISCONNECTS EVERY PLAYER AND DESTROYS THEIR CHARACTER, so the
# fixture holds a player that is `connected = false` with no character at all.
# That is not a defect of this route, it is the route: `--benchmark` is the only
# headless mode that keeps a client save's player connected and it never writes
# a save (`game.auto_save` returns true there and produces no file). The `curs`
# suite drives its gestures through the GOD controller, which a disconnected
# player can hold and which produces the same event trace as a character --
# guest/go/obs/curs's header carries that measurement.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FACTORIO="${FACTORIO_BIN:-$HOME/Library/Application Support/Steam/steamapps/common/Factorio/factorio.app/Contents/MacOS/factorio}"
SAVE="${SAVE:-${1:-}}"
OUT_DIR="$ROOT/test/fixtures-player"
# The observer package `make observers` writes, and the save name the guest
# hands `game.server_save`. Both are constants of guest/go/obs/fixplayer.
OBS_NAME="bbb-fixplayer"
SAVE_NAME="bbb-player-fixture"

if [ -z "$SAVE" ]; then
  echo "usage: make player-fixture SAVE=<path to a save a Factorio client made>" >&2
  exit 1
fi
[ -f "$SAVE" ] || { echo "no save at $SAVE" >&2; exit 1; }
[ -x "$FACTORIO" ] || { echo "no Factorio binary at $FACTORIO" >&2; exit 1; }

# The ENGINE the fixture is recorded on, read off the binary rather than
# written down -- the same source test/run.sh reads it from, and the same reason:
# which Factorio is installed is a Steam betas checkbox and it moves.
VER="$("$FACTORIO" --version | head -1 | sed -n 's/^Version: \([0-9.]*\) .*/\1/p')"
[ -n "$VER" ] || { echo "could not read a version out of $FACTORIO --version" >&2; exit 1; }
OUT="$OUT_DIR/player-$VER.zip"

PKG=""
for d in "$ROOT/dist/obs/${OBS_NAME}_"*/; do [ -d "$d" ] && PKG="$d"; done
[ -n "$PKG" ] || {
  echo "no $OBS_NAME package under dist/obs: run \`make observers\` first." >&2
  exit 1; }

TMP="$(mktemp -d "${TMPDIR:-/tmp}/bbb-player-fixture.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT
# `saves/` is created here and not by Factorio: `game.server_save` writes into it
# directly and refuses with `filesystem error: in canonical: No such file or
# directory` where it does not exist, which a headless server that has never
# saved leaves it as.
mkdir -p "$TMP/userdir/config" "$TMP/userdir/saves"
cat > "$TMP/userdir/config/config.ini" <<INI
[path]
read-data=__PATH__system-read-data__
write-data=$TMP/userdir

[general]
locale=auto
INI

# A private, closed server. `visibility` off on both channels and a bind to
# loopback: this starts a listening socket for a few seconds and nobody outside
# this machine may reach it.
cat > "$TMP/server-settings.json" <<'JSON'
{
  "name": "bbb-player-fixture",
  "description": "test/make-player-fixture.sh",
  "tags": [],
  "max_players": 1,
  "visibility": { "public": false, "lan": false },
  "username": "",
  "password": "",
  "token": "",
  "game_password": "",
  "require_user_verification": false,
  "max_upload_in_kilobytes_per_second": 0,
  "max_upload_slots": 1,
  "minimum_latency_in_ticks": 0,
  "ignore_player_limit_for_returning_players": false,
  "allow_commands": "false",
  "autosave_interval": 0,
  "autosave_slots": 1,
  "afk_autokick_interval": 0,
  "auto_pause": false,
  "only_admins_can_pause_the_game": true,
  "autosave_only_on_server": true,
  "non_blocking_saving": false
}
JSON

# stage <mods-dir> <dlc-enabled>
#
# The package is STAMPED for the running engine, exactly as test/run.sh stamps
# every observer it stages and for the same reason: this repository ships on two
# engine arms out of one tree, and a mod whose info.json names the other series
# is refused at the LOADER before a script of it runs.
stage() {
  local dir="$1" dlc="$2"
  rm -rf "$dir"; mkdir -p "$dir"
  cp -R "$PKG" "$dir/$OBS_NAME"
  SERIES="${VER%.*}" perl -pi -e \
    's/"factorio_version":\s*"[^"]*"/"factorio_version": "$ENV{SERIES}"/' \
    "$dir/$OBS_NAME/info.json"
  grep -q "\"factorio_version\": \"${VER%.*}\"" "$dir/$OBS_NAME/info.json" || {
    echo "$dir/$OBS_NAME/info.json was not stamped for Factorio ${VER%.*}" >&2
    exit 1; }
  cat > "$dir/mod-list.json" <<JSON
{
  "mods": [
    { "name": "base", "enabled": true },
    { "name": "elevated-rails", "enabled": $dlc },
    { "name": "quality", "enabled": $dlc },
    { "name": "space-age", "enabled": $dlc },
    { "name": "$OBS_NAME", "enabled": true }
  ]
}
JSON
}

# pass <label> <mods-dir> <input-save> <output-save> <port>
#
# The server is started in the background, polled for the guest's own last line,
# and then killed. The poll is on the LINE and not on the file: `server_save`
# returns before the writer has finished, and the guest logs `fixture written`
# a long way after it.
pass() {
  local label="$1" dir="$2" in="$3" outsave="$4" port="$5"
  local log="$TMP/$label.log"
  rm -f "$TMP/userdir/saves/$SAVE_NAME.zip"
  cp "$in" "$TMP/$label-in.zip"
  echo "==> $label: stripping $(basename "$in") ($(stat -f%z "$in" 2>/dev/null || stat -c%s "$in") B)"
  "$FACTORIO" -c "$TMP/userdir/config/config.ini" --mod-directory "$dir" \
      --start-server "$TMP/$label-in.zip" --server-settings "$TMP/server-settings.json" \
      --bind 127.0.0.1 --port "$port" >"$log" 2>&1 &
  local pid=$!
  local i
  for i in $(seq 1 90); do
    sleep 2
    grep -q 'FIXPLAYER\] fixture written' "$log" && break
    kill -0 "$pid" 2>/dev/null || break
  done
  sleep 2
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true

  sed -n 's/^ *[0-9.]* Script @__[a-z-]*__\/control.lua:[0-9]*: //p' "$log" | grep '^\[FIXPLAYER\]' || true

  grep -q 'FIXPLAYER\] fixture written' "$log" || {
    echo "$label did not reach the end of its schedule; see $log" >&2
    tail -30 "$log" >&2; exit 1; }
  # A guest that could not do what the world asked says so on its own tag, and a
  # fixture built over one of those is a fixture nobody checked.
  grep -q '\[BBB-OBS\] error:' "$log" && {
    echo "$label: the fixture guest reported an error" >&2
    grep '\[BBB-OBS\] error:' "$log" >&2; exit 1; }
  [ -f "$TMP/userdir/saves/$SAVE_NAME.zip" ] || {
    echo "$label wrote no save; see $log" >&2; exit 1; }
  mv "$TMP/userdir/saves/$SAVE_NAME.zip" "$outsave"
}

stage "$TMP/mods-dlc" true
pass pass1 "$TMP/mods-dlc" "$SAVE" "$TMP/stripped.zip" 34197

stage "$TMP/mods-base" false
pass pass2 "$TMP/mods-base" "$TMP/stripped.zip" "$TMP/final.zip" 34198

# THE ANTI-VACUITY CHECK, and it is the whole reason this script has assertions
# at all: a fixture with no player in it is a fixture the `curs` suite cannot
# drive one gesture from, and every number it would then report is a zero that
# looks like a pass.
grep -qE '\[FIXPLAYER\] fixture .* players=[1-9]' "$TMP/pass2.log" || {
  echo "the fixture has no player in it -- the source save was made by a" >&2
  echo "map generator rather than by a client, or the client never spawned." >&2
  grep '\[FIXPLAYER\] fixture ' "$TMP/pass2.log" >&2; exit 1; }

mkdir -p "$OUT_DIR"
cp "$TMP/final.zip" "$OUT"
echo
echo "==> $OUT"
echo "    $(stat -f%z "$OUT" 2>/dev/null || stat -c%s "$OUT") B, recorded against base alone on Factorio $VER"
