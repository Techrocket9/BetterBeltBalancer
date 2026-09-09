# What an update from a published release costs a player, measured

An adversarial assessment, 2026-09-08, of how a player's stored preferences and saved games
from the seven PUBLIC releases of this mod migrate onto head (0.3.3, unpublished), which
declares its settings through [FkRecipes](https://github.com/Techrocket9/FkRecipes) at 9754f49
on [FkLua](https://github.com/Techrocket9/FkLua) at b88965d. Nothing was fixed; every claim
carries the command that produced it, and a claim that could not be measured is marked NOT
REACHABLE rather than asserted. Heads at the time: BBB 6f1013f, both siblings clean and
read-only throughout.

**Every measurement is on `Version: 2.0.77 (build 84539, mac-arm64, steam)`**, re-asked at the
start of the run and at the start of each measuring arm. Because head is pinned 2.1 and this
is the only binary here, every row says which package it used, in one of exactly three shapes:
a **native 2.0 release**, **the 2.1 package restamped for 2.0** (honest for the settings and
data stages, never for the control stage), or **the true 2.0 recut** (fklua.toml at version
0.2.3, factorio_version 2.0, base >= 2.0.0, api 2.0.77, bindings and lock regenerated, both
`--check` forms exit 0), built for this assessment. Only one authentic artifact exists on this
machine: published 0.2.2, whose sha1 `79f2803de6f191d2e503b1de49dbdb0d22e99ee4` matches the
portal row exactly. Every other release is a rebuild and is labelled as one.

**The client half, NOT REACHABLE in every previous round, was reached this time**, under two
private user directories with their own `config.ini`, and the real Factorio user directory was
never named on a command line and was byte-unchanged before and after. See "The client".

## The seven public releases, established rather than assumed

`curl -sS https://mods.factorio.com/api/mods/better-belt-balancer/full`, 2026-09-08:

| version | released | pin | settings it declares, out of the engine's own dump |
|---|---|---|---|
| 0.1.0 | 2026-08-23 | 2.0 | none |
| 0.2.0 | 2026-08-25 | 2.0 | `bbb-multi-edge-parts` only (runtime-global bool) |
| 0.3.0 | 2026-08-25 | 2.1 | none (the bool is not emitted on 2.1) |
| 0.2.1 | 2026-08-27 | 2.0 | the two dropdowns, plus the bool |
| 0.3.1 | 2026-08-27 | 2.1 | the two dropdowns |
| 0.2.2 | 2026-09-01 | 2.0 | the two dropdowns, plus the bool |
| 0.3.2 | 2026-09-01 | 2.1 | the two dropdowns |

This settles a question the earlier notes left open: **0.3.0 WAS published.** There is no
published 0.2.3 and no published 0.3.3, so every player alive is on one of these seven, and
**no public release ever shipped a customizer setting under any name**, which is why the whole
migration surface is the two dropdowns and their nine values.

## Findings, most severe first

### 1. BLOCKED. A pack that removes `transport-belt` breaks the DEFAULT configuration, and there is no way out through the settings

The vanilla ladder is `4 iron-plate, 2 iron-gear-wheel, 2 transport-belt` and its last rung is
`iron-plate`. Remove the item `transport-belt` and the ladder produces `iron-plate` a second
time, which the engine rejects outright:

```
Error Util.cpp:81: Error while running setup for recipe prototype "bbb-balancer-part" (recipe): Duplicate item ingredients are not allowed (iron-plate exists 2 or more times).
Modifications: Better Belt Balancer
```

Exit 1, no dump. **There is no `fkrecipes:` line, no setting is named, and the missing item is
not named.** It hits a player who has never opened the Startup tab, since `vanilla` is the
default. The mechanism is one constant: `guest/go/tune/tune.go:168` sets
`FallbackName = "iron-plate"`, and every ladder ends on it, so a preset whose list already names
`iron-plate` produces it twice the moment any other rung falls through. `vanilla` and `cheap`
both do. `guest/go/tune/recipe.go:184`'s own comment says "What the rungs buy is a pack that
removed one of them, where today's mod fails the load outright"; measured, the rungs buy nothing
of the kind, because the fallback lands on a name the list already carries.

Recovery is worse than for a typed text. The error dialog's `Reset mod settings` returns the
setting to `vanilla`, which is the value that fails, so the player loops: enable, crash,
disable, enable. A disabled mod's settings do not appear on the Mod Settings screen, so they
cannot pre-set a different preset before re-enabling. **The only exits are removing this mod or
removing the other one**, and nothing on screen suggests which setting would have saved them.

The trigger was produced with a synthetic remover mod rather than found in a shipping pack, and
that is worth saying plainly; the defect it exposes is in the ladder, not in the remover.

### 2. BLOCKED. A refused text cannot be corrected from the settings screen, and the only escape discards every preference

Install `bbb-recipe-cost = custom` and `better-belt-balancer-recipe-ingredients = 2 iron-plat`.
Headless the load refuses; **three consecutive runs with nothing touched between them produced
logs identical but for the wall-clock second in the header, and left `mod-settings.dat` at the
same sha256 `f3b249191d5b4f62dc4c36c0bbc60fbdcc9894f58ab814b7e1df4e3dc8cd2457` all three
times**. The engine rewrites the
file on every successful load and on no failed one, so nothing in a failed run edits the file
that caused it: the refusal is permanent until something outside the game changes the bytes.

In the client the player is shown `Error loading mods`, and the message is good:

```
Failed to load mods: fklua: at the data stage, fkrecipes: better-belt-balancer-recipe-ingredients, entry 1 ("2 iron-plat"): no item or fluid is named iron-plat
```

followed by nine lines of Lua traceback naming `fk_data_module.lua:92990` and friends, which
reads as a broken mod rather than as a typo. The five buttons are `Disable listed mods`,
`Disable all mods`, `Manage mods`, `Restart`, `Exit`, with a `Reset mod settings` checkbox
beside the mod name. **Walking `Manage mods` shows the Mods screen, which offers enable and
disable and has NO Mod settings button, and its `Back` returns to the same error dialog rather
than to the main menu.** The Mod Settings screen is not reachable, so the setting that caused
the refusal cannot be corrected from inside the game.

The one escape, measured before and after:

| | sha256 | startup |
|---|---|---|
| before | `17c48b5e...` | `bbb-recipe-cost: custom`, text `2 iron-plat` |
| after `Reset mod settings` + `Disable listed mods` | `dbdcd1c6...` | `vanilla`, `default`, `logistics`, `default`, `20`, `15.0`, and `mod-list.json` with the mod `enabled: false` |

The game then reaches the main menu. The player is back in, having lost the dropdown as well as
the text, and must re-enable the mod and restart before they can choose again. Six steps, and
the preference is gone.

Headless, the three recoveries a player would try cost: **edit the text**, 1 step, loads,
nothing lost; **disable and re-enable** (the advice a stuck player is usually given), 2 steps,
does NOT work, because the engine does not drop a disabled mod's settings and the identical
refusal returns; **delete `mod-settings.dat`**, 1 step, loads, and silently resets all six
startup settings with no log line saying a preference was lost. There is a fourth that nothing
in the refusal mentions: **move the dropdown back to a preset**, 1 step, loads, and the typed
text survives verbatim.

**What a silent fallback with a loud log would have done for the same player**: the game would
have started on the default recipe, the log would have carried the same sentence, and the
player could have walked to Settings > Mod settings and fixed the typo in one step with every
other preference intact. The design chose refusal over fallback deliberately (FkRecipes
`agents/customizer-design.md`), and this is the price, measured: the choice is defensible only
while the engine gives a refused mod a route back to its own settings, and it does not.

### 3. BLOCKED. One launch of a published release destroys a Custom choice, silently

`bbb-recipe-cost = custom` is not in any published release's `allowed_values`, and the engine
resets a stored value outside the list to the default before any stage runs, **with no log
line**, and persists the reset. Measured round trip: head (Custom, text edited) to published
0.2.2 (native 2.0 release, authentic portal artifact) to head. 0.2.2 reset both dropdowns to
`vanilla` and `logistics` and wrote that back; the four settings it does not declare survived
untouched. Back on head the only trace is four library lines:

```
fkrecipes: better-belt-balancer-recipe-ingredients is edited, but bbb-recipe-cost is not on custom, so the text is ignored
```

The typed text surviving is what makes this dangerous: the field still shows the player's
recipe, so the screen looks like nothing was lost. Any Steam rollback, any modpack pinning an
older version, any second machine on the last public release does this. The mechanism is the
engine's and neither this mod nor the library can refuse it; what the design controls is the
exposure, which was created by ADDING a value that older releases cannot know.

### 4. MISLED. The settings screen presents ignored values as if they were live

Installed on head: `bbb-recipe-cost = cheap`, recipe text `2 iron-plate, 1 splitter`,
`bbb-tech-cost = logistics-2`, packs `3 automation-science-pack`, count `250`, seconds `45`.
The log carries four ignore lines. **The screen carries none.** All four fields render their
values in the ordinary style, not greyed, not struck through, with no icon and no note, exactly
as they render when they are in force. A player cannot tell that the mod is ignoring every one
of them.

Nothing on the screen can say otherwise: the row's whole prototype key set is `auto_trim,
default_value, localised_description, name, order, setting_type, type`, and FkRecipes'
`agents/customizer-design.md:9` records that the settings screen is a flat list with no
conditional visibility. The only sentence that touches the condition is positive and lives in a
hover tooltip: `What a balancer part is made of while the recipe setting above is set to
Custom.` The library's only signal is a log line no player reads.

### 5. MISLED. The dropdown speaks a vocabulary the field beside it refuses

The composed tooltip lists each preset twice, separated by a second colon:

```
Default: 4 iron plates, 2 gears, 2 transport belts: 4 iron-plate, 2 iron-gear-wheel, 2 transport-belt
```

and the changelog invites the player to "start from the one you were on". Copy the first half
and the load refuses: `2 iron plates, 1 transport belt` gives `did you mean iron-plate`, exit 1.
Copy the second half and it works. Nothing says which half is which.

The client makes it worse than the strings suggest. **The closed dropdown truncates its label at
roughly 37 characters** (`Default: 4 iron plates, 2 gears, 2 tra`), and five of the seven
options in the open list are truncated with an ellipsis, so the half that works is the half a
player never sees without hovering the info icon.

### 6. MISLED. The recipe description promises a fallback the field does not have

The first sentence a player reads above the Custom field is
`If an option names something your mods do not have, the nearest thing they do have is used
instead.` It is true of the six presets and false of the field two rows below, where a name the
game lacks refuses the load. The contrast is measured on the same list: with `express-splitter`
removed, `bbb-recipe-cost = splitter-express` loads and silently becomes
`1 fast-splitter, 2 steel-plate` with **not one `fkrecipes:` line**, while the identical list
typed into the field refuses by name. Same ingredients, opposite policy, one sentence covering
both. **No description anywhere contains the word refuse**, and `bbb-recipe-cost`'s description
is byte-identical to published 0.2.2's and never mentions Custom at all, while its twin
`bbb-tech-cost`'s was rewritten and does.

### 7. AWKWARD. Custom research keeps the cost and moves the technology

`bbb-tech-cost = custom` with all three fields untouched emits
`{count 20, [[automation-science-pack, 1]], time 15}`, byte-identical to base `logistics.unit`,
so the changelog's "picking Custom changes nothing there until you edit one" is true of the
cost. But `prerequisites` moves from `["logistics"]` to `["logistics-3"]`. The changelog does
disclose it in the following sentence; the settings screen does not, and a player on the default
tier who opens Custom to nudge one number moves the balancer research three tiers down the tree.
The recipe side has no such effect: Custom with the text `default` produces a prototype
byte-identical to the `vanilla` preset, and prints nothing at all.

### 8. AWKWARD. Changing the recipe destroys what is already in an assembler

Across 37 loads, every stack in an assembling machine's input inventory that is not an ingredient
of the NEW list is destroyed when the recipe changes. Not dropped on the ground
(`world.item-on-ground` was empty in all 37), not moved to a character, not in the output slot,
and with no log line. Worst case measured: `iron-gear-wheel 4, iron-plate 8, transport-belt 4`
to `(empty)`. A chest beside it holding the same items keeps all of them. Everything else
survives every direction: research, recipe enabled, entity counts, belt items, chest contents,
`crafting_progress`, `products_finished`, and the finished product in the output slot, including
saves with a craft genuinely in flight.

### 9. AWKWARD. The stated limits are the library's and appear nowhere a player looks

The text length boundary is **exactly 2000 characters**, measured on both a padded valid list and
a genuinely long list of 109 distinct item names: 1999 and 2000 load, 2001 refuses with
`fkrecipes: better-belt-balancer-recipe-ingredients is longer than 2000 characters; that is not
an ingredient list`. **The prototype declares no maximum-length key at all**, so the limit is
the library's and the engine will happily store 98000 characters for it. The screen states no
length limit, no numeric range in words, and never says whether `[item=iron-plate]` is accepted
(it is parsed as a rich-text tag, then refused on the item it names).

### 10. AWKWARD. Three invisible characters, three policies, and one refusal quotes text the player cannot see

`U+00A0` becomes a space and loads. `U+00AD` gets a good message that names it: `an invisible
character (U+00AD) has no place here; retype the entry rather than pasting it`. But `U+200B` and
`U+FEFF` are deleted silently anywhere in the string, so `2 iron<ZWSP>plate` refuses with
`entry 1 ("2 ironplate"): no item or fluid is named ironplate`, quoting a word that is not on the
player's screen and cannot be found by searching for it. A NUL is worse: it survives a
successful load and the engine's rewrite byte for byte, and on a refusal it cuts the message in
half, which ends at `entry 1 ("2 iron-plate` with no reason given.

Two smaller ones in the same family: `"  default  "` is accepted but `DEFAULT` refuses the load
and survives, so it repeats; and a `NaN` in `better-belt-balancer-tech-seconds` passes the
engine's range check (the int setting rejects NaN, the double setting does not), reaches the
guest and is refused there, leaving a file that fails identically for ever. Its recovery is the
same `Reset mod settings` route as finding 2, which the headless arm could not see and therefore
overstated.

### 11. MISLED, in the build rather than in the game. `release/2.0` is not head's 2.0 arm

`Makefile:423` says "trunk and the release/2.0 recut carry identical source: only this derived
value differs, and it differs because the pin does." `git merge-base master release/2.0` is
**cf5a78e, the 0.3.2 trunk commit**, and `git rev-list --count release/2.0..master` is
**16**, including every FkRecipes commit; `4b9f597:guest/go/go.mod` carries no fkrecipes
dependency at all. The engine agrees: release/2.0's package declares **three** settings where
head's recut declares **seven**, and its locale file is byte-identical to published 0.2.2's.
**Published as 0.2.3 today, a 2.0 player would receive 0.3.2's mod under a new version number**,
with the two dropdowns lacking their Custom option and none of the four fields, while a 2.1
player on 0.3.3 receives all seven. `verify/0.3.3-2.0` (3e3b5df) is not head's 2.0 arm either:
it branches off f8db95f and carries an unrelated curved-belt-exit feature line.

### 12. CLEAN, and it is the headline. The path every real player takes is intact

**All nine stored dropdown values from every settings-bearing public release migrate onto head:
exit 0, no log line at all, and prototype output identical to what the old release emits.** A
published player's file is exactly `startup{bbb-recipe-cost, bbb-tech-cost}` plus
`runtime-global{bbb-multi-edge-parts}`; head reads both dropdowns by name, honours them, and the
engine's rewrite appends the four new names at their defaults while keeping the old two and their
values. Nothing disappears from the screen, nothing moves, no label changes: each dropdown simply
grows one option at the end. The changelog's promise that "whichever option you had chosen before
is kept" is TRUE, and the tooltip's per-preset ingredient list is TRUE ingredient for ingredient
for all six presets on a stock install.

The client confirms it end to end. Published 0.2.2 with `splitter-express` and `logistics-2`
stored, a save made on it, the 0.2.2 directory then replaced by head's 0.2.3 recut with the file
untouched: **the settings survived, the save loaded straight into the game with no confirmation
and no sync prompt, and the log carried no Error, no Warning and no `fkrecipes:` line.**

Determinism holds where it matters: the same file at two different absolute paths, the same list
with different whitespace, and the same list under `locale=en` and `locale=de` all produce
identical normalised dumps. The one difference is order: `1 splitter, 2 iron-plate` against
`2 iron-plate, 1 splitter` differs at exactly four leaves of a 6800236-byte dump, all four the
swapped ingredients, so the typed order reaches the prototype unsorted.

## The release-to-head matrix

Rebuilt packages for every release, plus the true 2.0 recut of head; settings read out of the
engine's own `mod-settings-dump.json` on 2.0.77.

| release | pin | runs natively here | what it declares | its values | onto head |
|---|---|---|---|---|---|
| 0.1.0 | 2.0 | yes | nothing | | nothing to migrate |
| 0.2.0 | 2.0 | yes | the bool | `true`, `false` | survives by name; head declares the same bool on a 2.0 engine |
| 0.3.0 | 2.1 | no, `Incompatible Factorio version` | nothing on 2.1 | | nothing to migrate |
| 0.2.1 | 2.0 | yes | 2 dropdowns + bool | 6 recipe, 3 tech | all 9 survive, values and names unchanged |
| 0.3.1 | 2.1 | no | 2 dropdowns | 6 recipe, 3 tech | all 9 survive |
| 0.2.2 | 2.0 | yes (authentic) | 2 dropdowns + bool | 6 recipe, 3 tech | all 9 survive |
| 0.3.2 | 2.1 | no | 2 dropdowns | 6 recipe, 3 tech | all 9 survive |
| head 0.3.3 | 2.1 | no; the true 2.0 recut does | 2 dropdowns + 4 fields + bool | 7 recipe, 4 tech | |

The six recipe values are `vanilla`, `cheap`, `belt-fast`, `belt-express`, `splitter`,
`splitter-express`; the three tech values `logistics`, `logistics-2`, `logistics-3`. **Not one
of the nine is reset, renamed or lost**, and no release ever offered a value head has withdrawn.
The `custom` value is new in both dropdowns and is the whole of the exposure in finding 3.

The four generated names a 0.3.3 player will hold, and which a later release must not move:
`better-belt-balancer-recipe-ingredients` (order `aab`), `-tech-packs` (`bad`), `-tech-count`
(`bae`), `-tech-seconds` (`baf`).

The rebuilt packages are honest stand-ins for the settings surface and the data stage but not for
exact bytes: rebuilt 0.2.2 against the authentic artifact has **every file the mod supplies
byte-identical** (info.json, settings.lua, data.lua, data-final-fixes.lua, changelog.txt, locale,
graphics), with every difference in fklua's own emission. That equivalence is proven for 0.2.2
only and inferred for the other five.

## The save matrix

Saves made headless on published 0.2.2 (native 2.0 release, authentic portal artifact) with a
helper mod that researches the balancer, lays a 2x2 balancer with belts feeding it, stocks an
assembler mid-craft, and fills two chests; loaded on the true 2.0 recut of head with a probe mod
reading the world back. Twelve saves, a 12 of 12 control on the same package, 37 loads on head,
**all exit 0**.

| direction | settings file | outcome |
|---|---|---|
| 0.2.2 save to head | the same file | every stored dropdown honoured, world fully intact, the four new settings appended to the player's file at defaults |
| 0.2.2 save to head | no file | **the save's embedded startup record is NOT used headless**: all ten loads run `vanilla`/`logistics` whatever the save was made with, and the engine says nothing. The runtime-global bool does survive, because it is map state |
| 0.2.2 save to head | a file that differs from the save | loads, uses the file, says nothing |
| head save to 0.2.2 (downgrade) | the same file | loads, world whole, both dropdowns silently reset and the reset persisted; the four settings 0.2.2 does not declare survive untouched |
| head save to head, text then edited | edited file | the file wins; the four ignore lines are the only signal |

**Across all 37 loads: zero `Error`, zero `Warning`, zero `Migration` or `Applying` lines.**
Nothing about the version moving up or down, nothing about the save's startup record differing
from the file, nothing about a reset. The only thing in the entire stack that tells a player
anything is the library's four ignore lines. `mod_startup_settings_changed` fires true even in
the arm where the player changed nothing, because the setting SET grew.

A save's own memory of the mod, read out of `level-init.dat`: the mod list as a name plus a
3-byte version plus a 4-byte checksum (`better-belt-balancer` reads `00 02 02`), then the startup
settings as a property tree in exactly the `mod-settings.dat` node encoding. It stores the
startup section only, and the runtime-global value is not there.

## The client

Two private user directories, each with a `config.ini` whose `write-data` names it, confirmed by
`Write data path:` in each log. The real user directory was never on a command line and its
`factorio-current.log` and `mods/mod-settings.dat` were unchanged before and after. Steam
relaunches the game through itself and drops the arguments unless a `steam_appid.txt` holding
`427520` sits in the working directory; the first launch did that, was killed, and reached
nothing. Keyboard injection was blocked for the whole session (`A password field has secure input
active`), so every value was installed with `fklua modsettings write` and the client was used to
observe.

**The Startup tab renders exactly as the prototypes declare**, six rows in the order `a`, `aab`,
`b`, `bad`, `bae`, `baf`. **The composed descriptions render their line breaks**, all seven lines
of the recipe one, readable and unclipped, which closes the first two assumptions FkRecipes was
carrying. The text field's own tooltip is good: it says internal names, gives an example, explains
the word `default`, and points at the preset tooltip.

**The startup-settings sync prompt exists and no headless run shows it.** Loading a save whose
stored settings differ from the file gives:

```
Mods have been removed or mod settings have changed. Are you sure you want to continue loading this save?
[Back] [Sync mods and load] [Load]
```

It names no setting and no value. `Sync mods and load` opens `Sync mods with save`, listing each
mod as `Keep enabled` with two checkboxes ticked by default, `Sync startup settings` and `Load
save after sync`. Confirming rewrote `mod-settings.dat` back to the save's `cheap` and
`logistics-3` and loaded the save. **So the save's embedded startup record, which the headless
engine never consults, IS a recovery route in the client, and it is the default choice**: a
player whose file was lost or reset can get their preference back from any save made under it.
That is the single most useful thing the client run found, and it materially softens the "no
file" row of the save matrix for everyone who is not running a server.

## What the design must change, and where

**FkRecipes.** (a) A refused text is unrecoverable from inside the game, because the engine's
error dialog cannot reach Mod Settings; the library's refusal-over-fallback choice needs either a
fallback with a loud log for the text settings, or a refusal message that tells the player the
one thing that works, which is to move the dropdown off Custom. Nothing in the current message
mentions it, and it is one click. (b) The strip-and-refuse policy for invisible characters must
be one policy: quoting a word the player cannot see is worse than either accepting or naming the
character, and `U+00AD` already shows the good shape. (c) A NUL must not survive into a stored
value, and must not truncate the refusal message. (d) The 2000-character limit belongs in the
composed description, since the prototype cannot carry it. (e) `DEFAULT` should be accepted where
`"  default  "` already is.

**BBB.** (f) The vanilla ladder can land on a name the list already holds; the rungs need to be
checked against what is already chosen, and a ladder that cannot produce a legal list should
refuse by name rather than let the engine reject a duplicate. (g) `bbb-recipe-cost`'s description
is byte-identical to 0.2.2's and never mentions Custom, while its twin's was rewritten; it needs
the same treatment, and the reassurance about the nearest thing must be scoped to the presets.
(h) The composed preset lines should not print both vocabularies separated by a colon when only
one of them can be typed. (i) `release/2.0` carries 0.3.2's source stamped 0.2.3 and the
Makefile comment says otherwise; whichever is meant to be true, they must agree before a 2.0
release goes out.

**Neither, and worth saying.** The silent reset of an unknown dropdown value, the destruction of
an assembler's input stock on a recipe change, and the absence of any conditional visibility on
the settings screen are the engine's. What the design controls is exposure and wording.

## What could not be reached

- **Typing into the settings text field**, and therefore the field's own GUI limits, whether it
  truncates, and whether shift-clicking an item inserts a rich-text tag: secure input was active
  for the whole session and every keyboard action was refused. The values were installed by file
  instead, so everything about rendering and behaviour on load IS measured; only the act of
  typing is not.
- **A real two-party multiplayer sync.** There is no headless client. What is measured: a server
  refuses to start on a bad text (exit 1 at 0.5s, no `Hosting game` line, so the port never opens
  and a joining client sees a refused connection with nothing from the mod); a server whose file
  differs from the save's stored settings starts normally and says nothing; and both checksums a
  server publishes (`Prototype list checksum` and `Checksum of better-belt-balancer`) are
  identical across four different recipes including a preset change.
- **Anything on a real 2.1 engine.** Only 2.0.77 is here, so every head measurement is of a
  restamped or recut package, and what a 2.1 player sees, including the absence of the Map-tab
  bool, is inferred from the pin rather than observed.
- **Authentic artifacts for six of the seven releases.** Portal downloads need credentials.
- **The client's `Restart` button** on the error dialog was not pressed; the loop is measured
  headless instead, as three byte-identical refusals over an unchanged file.
- **Saves made on 0.1.0, 0.2.0 and 0.2.1.** 0.2.1 declares the same settings as 0.2.2 with a
  byte-identical locale file, so its saves are covered by equivalence rather than by measurement;
  the two earlier releases have no dropdown to migrate.

This note is a new file and CLAUDE.md's index of `agents/` was deliberately not touched, because
the assessment was scoped to exactly one file; the index row is owed.
