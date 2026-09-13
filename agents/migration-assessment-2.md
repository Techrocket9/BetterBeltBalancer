# What an update from a published release costs a player, measured again after the fix cycle

The SECOND adversarial assessment, 2026-09-13, of how a player's stored preferences and saved games from the seven PUBLIC releases of this mod migrate onto head (0.3.3, unpublished), which declares its settings, recipe and technology through [FkRecipes](https://github.com/Techrocket9/FkRecipes) at `21f5d89` on [FkLua](https://github.com/Techrocket9/FkLua) at `b88965d`. Heads: BBB `29bad4f`, both siblings clean and read-only throughout. Nothing was fixed; every claim carries the command that produced it, and a claim that could not be measured is marked NOT REACHABLE rather than asserted.

The FIRST assessment is [`agents/migration-assessment.md`](agents/migration-assessment.md), committed `d8dc79e`, twelve findings: three BLOCKED, four MISLED, four AWKWARD and one CLEAN headline. A fix cycle answered it in FkRecipes (`3a9de88`, `696387f`, `2e5f779`, `0077f3c`, `1f8e363`, `642fec4`, `f671ea5`, `21f5d89`) and in this mod (`3fd5875`, `ff647c2`, `760d618`, `4a1b233`, `894220b`, `29bad4f`). **All twelve are regraded here by number on the same rubric and the same seven attack surfaces.** The first assessment stays as the dated record it is and is not rewritten; where it is now wrong, this file says so.

**Every measurement is on `Version: 2.0.77 (build 84539, mac-arm64, steam)`**, re-asked at the start of the run and again by each measuring arm. Head is pinned 2.1 and this is the only binary here, so every row says which package it used, in one of exactly three shapes: a **native 2.0 release**, **the 2.1 package restamped for 2.0** (honest for the settings and data stages, never for the control stage), or **the true 2.0 recut** (fklua.toml at factorio_version 2.0, base >= 2.0.0, api 2.0.77, bindings and lock regenerated, both `--check` forms exit 0), built for this assessment. The recut is what almost every row uses. One authentic artifact exists on this machine, published 0.2.2, whose sha1 `79f2803de6f191d2e503b1de49dbdb0d22e99ee4` matches the portal row exactly; every other release is a rebuild and is labelled as one. Rebuilt 0.2.2 against the authentic zip is byte-identical in every file the mod itself supplies (info.json, settings.lua, data.lua, data-final-fixes.lua, control.lua, changelog, locale, all four images) and differs only in fklua's own generated output and two source maps today's packager writes.

**The client was reached, and typing was reached with it.** The first assessment could observe the client but could not type: secure input was active for its whole session. This time every keyboard action landed, so the field's own behaviour is measured rather than owed. Eight private client sessions ran, each under its own `config.ini` with `write-data` pointed at a scratch directory. The real Factorio user directory was never named on a command line, and its `factorio-current.log` and `mods/mod-settings.dat` were byte-unchanged before and after every one of them and of every headless run beside them.

## The grades, at a glance

| # | first assessment | now | what moved |
|---|---|---|---|
| 3 | BLOCKED | **BLOCKED** | unchanged in the engine; the changelog now discloses it |
| 4 | MISLED | **MISLED** | the rule is now on screen, the state still is not |
| 5 | MISLED | **AWKWARD** | the two vocabularies no longer share a line |
| 8 | AWKWARD | **AWKWARD** | unchanged, and the fallback gives it a new way to fire |
| 9 | AWKWARD | **AWKWARD** | the 2000 is on screen now; the two numeric ranges are not |
| 10 | AWKWARD | **AWKWARD** | one policy, code points named; the NaN clause splits out as 16 |
| 1 | BLOCKED | **CLEAN** | the ladder merges and the engine takes it |
| 2 | BLOCKED | **CLEAN** | no text a player types refuses the load; residual splits out as 14 |
| 6 | MISLED | **CLEAN** | the promise is scoped and both sides measured |
| 7 | AWKWARD | **CLEAN** | the prerequisite no longer moves; see 15 for the disclosure |
| 11 | MISLED | **CLEAN** | the branch is honest and the Makefile now says what is true |
| 12 | CLEAN | **CLEAN** | all nine still migrate, and the new library line does not fire here |

New this round: **13 BLOCKED**, **14 BLOCKED**, **16 BLOCKED**, **15 MISLED**, **18 MISLED**, **17 AWKWARD**.

## The twelve, regraded, most severe first

### 3. BLOCKED, unchanged. One launch of a published release still destroys a Custom choice, silently

Re-measured on the AUTHENTIC 0.2.2 artifact rather than on a rebuild. Head (recut) with `bbb-recipe-cost = custom`, recipe text `2 iron-plate, 1 splitter`, `bbb-tech-cost = custom`, packs `1 automation-science-pack, 1 logistic-science-pack`, count 250, seconds 45: exit 0, the mod honours all six, and the file comes back byte-identical (`b4e0378a...`). That file handed to authentic 0.2.2: **exit 0, no `fkrecipes:` line, no Error, no Warning**, and `fklua modsettings read` afterwards shows both dropdowns reset to `vanilla` and `logistics` with the reset persisted (`04ad24ed...`), while the four settings 0.2.2 does not declare survive verbatim. Back on head, the only trace is four lines:

```
fkrecipes: better-belt-balancer-recipe-ingredients is edited, but bbb-recipe-cost is not on custom, so the text is ignored
fkrecipes: better-belt-balancer-tech-count is edited, but bbb-tech-cost is not on custom, so the number is ignored
fkrecipes: better-belt-balancer-tech-seconds is edited, but bbb-tech-cost is not on custom, so the number is ignored
fkrecipes: better-belt-balancer-tech-packs is edited, but bbb-tech-cost is not on custom, so the text is ignored
```

**Two things the first assessment did not say, both measured here.** The destruction is confined to the releases that DECLARE the dropdowns: rebuilt 0.2.1 destroys the choice exactly as 0.2.2 does, while 0.1.0 and 0.2.0 leave the file **byte-identical** (`b4e0378a...` before and after), because the engine only resets a value whose prototype exists and whose `allowed_values` excludes it. So the exposure is a rollback to 0.2.1, 0.2.2, 0.3.1 or 0.3.2, not to any older release. And the same mechanism reaches multiplayer: two user directories sharing one file, head then rel-0.2.2 then head, leave the modern machine on the vanilla default with its texts stranded, and because the stored VALUES now differ from an untravelled machine's, the startup-settings CRC differs too and the two would be told their settings mismatch.

What moved since the first assessment is the changelog: 0.3.3 carries an Info row naming the four releases that do this and saying the typed text surviving is what makes it look as though nothing was lost. That is the right disclosure in the wrong place for the player who never reads it, and nothing on the settings screen says a word. The grade is the outcome, and the outcome is unchanged.

### 4. MISLED, unchanged. The settings screen still presents ignored values as if they were live

`bbb-recipe-cost = cheap`, recipe text `2 iron-plate, 1 splitter`, `bbb-tech-cost = logistics-2`, packs `3 automation-science-pack`, count 250, seconds 45. Headless: exit 0 and four ignore lines, and zero other `fkrecipes:` lines. On the client, photographed: the four Custom rows render `2 iron-plate, 1 splitter`, `3 automation-science-pack`, `250` and `45` in the ordinary style, not greyed, not struck, no icon and no note, exactly as they render when they are in force.

**Nothing on the screen can say otherwise, and that is now measured rather than argued.** The engine's own `mod-settings-dump.json` is byte-identical whatever the player stored: sha256 `7f2682f8519f64fb97dc2806154ccde39c7b5799821a0eef7fa80c17a91487ed` for the all-Custom run, for this ignored-values run and for the all-defaults run alike. The prototype is a static declaration and carries no state.

What moved is real and is not enough. Each of the four Custom fields now OPENS by naming the precondition, for example `What a balancer part is made of while the recipe setting above is set to Custom.` and `How many units of science the balancer research costs, while the research setting above is set to Custom.` The screen states the RULE. It cannot state the STATE, and a player looking at a field holding their own text has no way to tell which side of the rule they are on without reading the log.

### 5. AWKWARD, down from MISLED. The two vocabularies no longer share a line, and a smaller trap is left

The old shape printed each preset twice on one run of text separated by `": "`. The new shape gives each preset's internal list its own indented line under the label, and **the client confirms it renders**. Thirteen logical lines, measured in the engine's dump and photographed on the client; the first line is 557 characters and the twelve under it run 32 to 66, both numbers confirmed exactly as the fix round stated them. The 557 is this mod's own locale entry and nothing the library writes.

**The open claim was the WRAP, and the answer is that it wraps and nothing clips.** The 557-character line becomes ten visual lines; the tooltip is 24 visual lines in all and the whole of it is on screen, readable, inside the window. So the line count was never the risk and neither was the width.

**What the wrap does cost is the indent, and that is new.** Two of the twelve lines are long enough to wrap, and both continuations start at the LEFT MARGIN rather than under their own line:

```
Express belts: 4 steel plates, 2 gears, 2 express transport
belts
  type: 4 steel-plate, 2 iron-gear-wheel, 2
express-transport-belt
```

So `express-transport-belt` sits at label indentation and reads as a line of its own, and a player who copies what looks like a whole `type:` line gets `4 steel-plate, 2 iron-gear-wheel, 2` with the last ingredient missing.

The failure mode FkRecipes records as still open is real and was reproduced: retyping the whole line, prefix included, gives

```
fkrecipes: ERROR: better-belt-balancer-recipe-ingredients, entry 1 ("type: 4 iron-plate"): ":" has no place here; names use the letters a to z, digits, - and _, and an amount is plain digits, as in "2 iron-plate". The mod loaded with its own default instead; fix the text under Settings > Mod settings > Startup, then restart.
```

and copying the label half gives the same message class. Copying only the display names, which is what finding 5 was about, now gives the useful one:

```
fkrecipes: ERROR: better-belt-balancer-recipe-ingredients, entry 1 ("4 iron plates"): no item or fluid is named "iron plates"; did you mean iron-plate. The mod loaded with its own default instead; fix the text under Settings > Mod settings > Startup, then restart.
```

All three exit 0 and fall back. The closed dropdown still truncates: it shows `Custom: the ingredients written in t`, 36 characters of a 52-character label, where the first assessment saw 38 of a different one, because the cut is by pixel width and not by character count. Five of the seven options in the open list are truncated with an ellipsis, and hovering one shows its full label in a tooltip, which the first assessment did not record.

### 8. AWKWARD, unchanged, and the fallback gave it a new way to fire

Re-measured on head alone, so the version is not a variable. The claim holds exactly and is now sharper.

| change | assembler input before | after | chest after | on the ground |
|---|---|---|---|---|
| vanilla to splitter-express | 20 iron-gear-wheel, 20 transport-belt, 40 iron-plate | **EMPTY** | all of it, plus 5 finished parts | 0 |
| vanilla to a typed `2 iron-plate, 1 splitter` | the same 80 items | 40 iron-plate | unchanged | 0 |
| cheap to vanilla, which only ADDS an ingredient | 20 iron-plate, 10 transport-belt | unchanged | unchanged | 0 |

Eighty items gone in the worst case, not on the ground, not in the output slot, not anywhere, with zero `fkrecipes:` lines, zero Error and zero Warning. The refinements: the destruction is keyed on "not an ingredient of the NEW list" rather than on "the recipe changed", and it is per stack rather than per inventory. `crafting_progress`, `products_finished` and the output slot survive every time.

**Two new costs, and the second is this cycle's own.** First, a chest is safe from a recipe change but NOT from the item prototype disappearing: with a pack that removes the item `transport-belt`, the chest's 20 transport-belt are destroyed along with the assembler's, while the four belt ENTITIES stand there uncraftable. Second, and this is what the fallback created:

```
[save built under bbb-recipe-cost=custom, text "2 iron-plate, 1 splitter"]
[loaded with the text edited to "2 iron-plat"]
fkrecipes: ERROR: better-belt-balancer-recipe-ingredients, entry 1 ("2 iron-plat"): no item or fluid is named iron-plat. The mod loaded with its own default instead; fix the text under Settings > Mod settings > Startup, then restart.
prototype recipe bbb-balancer-part = 2 iron-gear-wheel + 2 transport-belt + 4 iron-plate
  assembler input = 20 iron-plate          (10 splitter destroyed)
  chest           = 10 splitter, 20 iron-plate, 5 bbb-balancer-part
  items-on-ground = 0
```

A typo in a settings field is now a recipe change nobody asked for, and it silently destroys every stack the two lists do not share. Fixing the typo destroys another set on the way back: the save made under the fallback recipe loses its 20 iron-gear-wheel and 20 transport-belt when the custom list returns. Under the old refusal neither loss could happen, because the load stopped. **The changelog discloses that changing the recipe empties an assembler; it does not disclose that a typo can change the recipe.**

### 9. AWKWARD, partly lifted. The length limit is now where a player looks; the two numeric ranges are not

The boundary is unmoved and exact: 1999 and 2000 pass the length gate, 2001 trips it, and 2001 now FALLS BACK rather than refusing.

```
fkrecipes: ERROR: better-belt-balancer-recipe-ingredients is longer than 2000 characters; that is not an ingredient list. The mod loaded with its own default instead; fix the text under Settings > Mod settings > Startup, then restart.
```

**The limit is on the screen now**, and the client confirms the whole four-line tooltip renders unclipped:

```
What a balancer part is made of while the recipe setting above is set to Custom. Write the amount, then the item's internal name, and separate ingredients with commas: 2 iron-plate, 1 splitter. Internal names are the ones the game uses in its own data and in rich text, not the names shown on screen. The word default means this mod's own recipe, and the tooltip on the setting above lists every preset written this way.
default: 4 iron-plate, 2 iron-gear-wheel, 2 transport-belt
Write internal names, as the default line above does, in at most 2000 characters.
A text this mod cannot use is set aside and that default applies instead; the reason is in the log, or in the load error if the load stops anyway.
```

The prototype still declares no maximum-length key of any kind. Read out of the engine's dump, the two text settings carry exactly `auto_trim, default_value, localised_description, name, order, setting_type, type`, which is the first assessment's set confirmed; the two dropdowns carry `allowed_values` instead of `auto_trim`, and the two NUMBERS carry `minimum_value` and `maximum_value` and **no `localised_description` at all**, so nothing composed reaches them. `better-belt-balancer-tech-count` is 1 to 1000000 and `-tech-seconds` is 1 to 3600, and neither range appears in words anywhere a player reads. That half of the finding stands.

One thing the first assessment got wrong and this one corrects: `[item=iron-plate]` is now ACCEPTED, not refused. It emits `1 iron-plate`, exit 0, zero error lines, and the field's own tooltip says internal names are the ones the game uses "in its own data and in rich text", which is now true of the field as well as of the game.

### 10. AWKWARD. Three policies became one, the NUL cliff is real, and the NaN clause is re-graded as finding 16

**The invisible characters are one policy now and it is the right one.** Nothing is deleted, and every invisible character is written as its code point inside the quoted text, so the player can find what they pasted:

```
... entry 1 ("2 iron-U+200Bplate"):  an invisible character (U+200B) has no place here; retype the entry rather than pasting it.
... entry 1 ("2 iron-U+FEFFplate"):  an invisible character (U+FEFF) has no place here; retype the entry rather than pasting it.
... entry 1 ("2 iron-U+00ADplate"):  an invisible character (U+00AD) has no place here; retype the entry rather than pasting it.
... entry 1 ("2 iron-U+E0001plate"): an invisible character (U+E0001) has no place here; retype the entry rather than pasting it.
```

All four exit 0, all four fall back to `4 iron-plate, 2 iron-gear-wheel, 2 transport-belt`, all four log exactly one `fkrecipes: ERROR: ` line. The "quotes text the player cannot see" half is closed.

**The reserved words fold.** `"  default  "`, `DEFAULT` and `Default` all load and all emit the mod's own recipe, with no line at all. The first assessment's `DEFAULT` refusing for ever is gone.

**The NUL cliff is real and it sits exactly where the fix round put it**, reproduced twice here independently. A sweep at 21, 22, 23 and 24 stored bytes:

| stored bytes | error lines | what the engine stored afterwards | file moved |
|---|---|---|---|
| 21 | 0, emits `2 iron-plate` | `2 iron-plate` (12 bytes) | **yes** |
| 22 | 0, emits `2 iron-plate` | `2 iron-plate` (12 bytes) | **yes** |
| 23 | 1, `("2 iron-plateU+0000xxxxxxxxxx"): an invisible character (U+0000) has no place here` | intact, 23 bytes | no |
| 24 | 1, same shape | intact, 24 bytes | no |

Above the cliff the byte is escaped and the sentence is whole, which closes the "cuts the message in half" half. **Below it the ENGINE cuts the stored value before the guest sees it and a completed load persists the cut, with no warning line anywhere.** `"2 iron\0 plat"` became `"2 iron"`, and the one error line the player gets quotes `("2 iron")`, which is not what they typed. That is the engine's, it is silent, it is permanent, and it is the one row in the whole refusal corpus where the player's text does not survive for them to fix.

`readDropdown`'s unescaped `holds "..."`, which FkRecipes records as the one message producer outside the language, is **NOT REACHABLE through the game on 2.0.77**: an unoffered stored dropdown value is reset by the engine to the default before any stage runs, silently and persistently, so the guest never sees one.

The NaN clause of this finding turned out to be a different mechanism from the one the first assessment described, so it is carried below as finding 16 rather than folded in here.

### 1. CLEAN, up from BLOCKED. The ladder merges and the engine takes the result

The vanilla ladder landing twice on `iron-plate` no longer reaches the engine as two entries.

```
fkrecipes: bbb-balancer-part: iron-plate is in the list twice after the fallbacks, so the amounts are added: 4 plus 2 is 6
```
emitted: `[{"name":"iron-plate","amount":6},{"name":"iron-gear-wheel","amount":2}]`

and for `cheap`, `... 2 plus 1 is 3` giving `3 iron-plate`. Exit 0, a dump written, no Error and no Warning. `Duplicate item ingredients are not allowed (iron-plate exists 2 or more times)` does not occur. The merged entry lands at the position of the first occurrence, so declaration order survives. All nine stored release values load with the remover present, and the merge line appears exactly for the two presets whose ladder holds a `transport-belt` rung.

**The shipped fixture no longer isolates this mod on this install, and that is worth recording.** `test/check-datastage.py`'s `bbbt-remover` deletes the item and every recipe naming it, and on a stock Space Age install that kills space-age before this mod can be judged: `Failed to load mod "space-age": __space-age__/base-data-updates.lua:241: attempt to index field 'transport-belt' (a nil value)`, at 0.58s, after this mod's data stage has already completed at 0.39s with the merge line written. The numbers above were taken two ways, with the three expansion mods disabled, and with a synthetic remover that deletes the item at the data stage and sweeps the dangling recipes at data-final-fixes so space-age still finds its own. Both give the same answer.

### 2. CLEAN, up from BLOCKED, for a value the player types. The residual is finding 14

**All twenty-two texts a player could type now load.** Every one exits 0, every one writes a dump, and where a line is expected the count of `fkrecipes: ERROR: ` lines is exactly one, never two: a typo, an unknown item, a fluid, four kinds of invisible character, a display name, a decimal comma, a NUL on both sides of the cliff, the recipe's own product, five reserved-word spellings, a rich-text tag, a duplicate entry, and the list at 1999, 2000 and 2001 characters. Seventeen fall back with one line; five are cases the language can now use.

The fallback lands exactly where the claim says it does, and the proof is stronger than a prototype diff: head on `bbb-recipe-cost = custom` with the text `2 iron-plat` and head on `bbb-recipe-cost = vanilla` produce the **same normalised 27,854,440-byte data dump**, sha256 `8017bbdb5cdbbe2d9506e3b24a0956bc8e79d28938ff6952ada072f33947afff`, zero differing lines. The whole data stage is identical, not just the recipe.

Recovery for all twenty-two is **one edit and nothing lost**: the game is running, the line names the setting and the screen, the Mod Settings screen is reachable because the load succeeded, and the player's text is still in the file byte for byte when they get there. The single exception is the NUL below 22 bytes, where the engine had already shortened what they typed.

Several bad values at once give one line per setting the guest can see, in walk order, never two for one setting:

```
fkrecipes: ERROR: better-belt-balancer-recipe-ingredients, entry 1 ("2 iron-plat"): no item or fluid is named iron-plat. ...
fkrecipes: ERROR: better-belt-balancer-tech-seconds holds a value that is not a finite number. The mod loaded with its own default instead; fix the number under Settings > Mod settings > Startup, then restart.
fkrecipes: ERROR: better-belt-balancer-tech-packs, entry 1 ("2 flurb-pack"): no science pack is named flurb-pack. ...
```

Note that each names its own field word, `text` or `number`. Note also that an out-of-range count or seconds produces NO line, because the engine resets a stored number outside its declared bounds before the data stage runs and the guest never sees it.

### 6. CLEAN, up from MISLED. The promise is scoped and both sides were measured

The rewritten description, out of the engine's own dump, resolved:

> What a balancer part costs to craft, or Custom to write the recipe yourself in the setting below. The default is what this mod has always used. Every option but Custom is safe in an overhaul pack: where one names something your mods do not have, the nearest thing they do have is used instead, and where that leaves one item named twice the two amounts are added, so what you craft can be a shorter list than the one shown here. A recipe you write yourself is the opposite on purpose, and it is taken as written: nothing is substituted for a name you typed.

Both halves were re-measured with a synthetic pack removing the item `express-splitter`, on base alone. **Preset side**: `bbb-recipe-cost = splitter-express` exits 0, emits `1 fast-splitter, 2 steel-plate`, and prints **no `fkrecipes:` line at all**. **Typed side**: the identical list typed into the field exits 0, falls back to the mod's own three-ingredient default, and prints exactly one line naming `express-splitter`. Same ingredients, opposite policy, and the description now says which is which and why.

No description anywhere contains the word "refuse", which is correct: nothing about a typed name refuses any more. What the dropdown's own sentence gives is the negative, "nothing is substituted"; what actually happens is one row below, on the field: "A text this mod cannot use is set aside and that default applies instead". The two rows are adjacent on screen, so the pair is honest.

### 7. CLEAN, up from AWKWARD, on the behaviour. Which assessment was right about it

**Head today**, `bbb-tech-cost = custom` with all three fields untouched, full mod set:

```
bbb-balancer   prerequisites ["logistics"]   unit {count 20, [["automation-science-pack",1]], time 15}
logistics      prerequisites ["automation-science-pack"]  unit {count 20, [["automation-science-pack",1]], time 15}
```

`bbb-tech-cost = logistics` gives byte-identical `bbb-balancer` fields. The prerequisite does not move, and with `logistics` removed by a synthetic pack it lands on `["logistics-2"]`, which is what the current description says.

**Which assessment was right: each about a different half.** The first assessment was RIGHT about the behaviour it measured. At `ca9aa36` the plan's position ladder was headed at `logistics-3`, so Custom untouched did move the research three tiers down, and this mod changed the CODE (`ff647c2`) rather than the wording. It was WRONG that the settings screen did not disclose it: at that very revision the tooltip's first paragraph read `On Custom it sits after Logistics 3, or after the highest of Logistics 2 and Logistics that your mods have, and has no prerequisite if none of the three exists.` The fix leg was right about that disclosure, and it does not claim the prerequisite never moved; its own code comment is the record that it did.

On head the finding is moot in both halves. **But see finding 15**: the tooltip that carries this disclosure cannot be read at all on a stock install, so the sentence both rounds argued about is invisible to the player it was written for.

### 11. CLEAN, up from MISLED in the build. The branch is honest and the sentence now says what is true

Re-measured today, read-only, twice by independent arms that agree:

```
git rev-parse release/2.0            4b9f597  (version 0.2.3)
git merge-base master release/2.0    cf5a78e  (trunk 0.3.2)
git rev-list --count release/2.0..master   24
git diff --name-only master release/2.0    38 paths, 28 of them hand-written after the stamp and the notes are set aside
```

The merge-base is the first assessment's, unchanged. The count is 24 today; **the assessment's 16 cannot be re-derived at any named revision** (17 at `d8dc79e`, 18 at `ca9aa36`), and since a dated record is not rewritten, the correction lives here. The engine agrees with the shape of the claim: `rel-0.2.2` dumps exactly three setting prototypes where head's recut dumps seven, and `release/2.0`'s own `tune.go` names only the two dropdown constants. **So "release/2.0 is not head's 2.0 arm" is TRUE today and was true then.** What the first assessment got wrong was reading that as the defect, which this mod corrected.

`test/check-release-arm.sh` does NOT check byte-identity and says so in its own header: it asks per-FILE blob PROVENANCE over `merge-base..ARM`, that every path the branch changed since its cut point carries a blob some commit reachable from `master` also carries, with a named six-file exclusion list. It passes today (`release-arm: ok`, 14 paths differ from the cut point, 8 carrying trunk blobs). The Makefile's sentence is now the retraction of itself rather than the claim the assessment quoted.

**One correction owed in the other direction.** A summary of the fix leg as "release/2.0's source was byte-identical to trunk, and only the Makefile sentence was wrong" is false as literally stated: the branch is not byte-identical to trunk head (28 hand-written paths), and it is not byte-identical to its own cut point either, since eight paths carry blobs from the later trunk commit `cb89891`. What is true is the per-file provenance, which is what CLAUDE.md's own paragraph claims and what the gate holds. The gate and the record are right; a summary that reaches for the word "identical" is not.

### 12. CLEAN, unchanged, and it is still the headline

All nine stored dropdown values from every settings-bearing public release, installed and loaded on head's recut and on the authentic 0.2.2 that could have produced them: **eighteen runs, all exit 0, zero `fkrecipes:` lines on either side, and the `bbb-balancer-part` recipe and `bbb-balancer` technology prototypes byte-identical between the two**. The values are genuinely honoured and not merely accepted: the six recipe presets emit six different lists and the three tiers emit count 20 / 200 / 300 against prerequisites `logistics` / `logistics-2` / `logistics-3`.

**The new unconditional library line does not fire for this mod.** FkRecipes `1f8e363` added a line for a recipe whose resolved list names its own product, and this mod's fixture is exactly that shape upstream, so it was the thing to watch. Searched across 65 logs of the release arm: zero occurrences. It fires here only when the PLAYER types the product, which is the `1 bbb-balancer-part` row of the refusal corpus:

```
fkrecipes: bbb-balancer-part takes its ingredients from better-belt-balancer-recipe-ingredients: 1 bbb-balancer-part
fkrecipes: bbb-balancer-part: bbb-balancer-part is in the list and is also what this recipe makes, so nothing can craft the first one unless something else produces it
```

Exit 0, no error line, and the player can edit it back. That closes the item this mod's notes recorded as left open.

The engine's rewrite keeps the two old names and their values and appends the four new ones at their declared defaults, and it happens on a SUCCESSFUL load only, measured rather than assumed: the same file is 196 bytes and `799c474a...` after a failed load and 432 bytes and `d085493c...` after a successful one.

## The new findings

### 13. BLOCKED. A preset research tier copies a technology's unit without checking the packs are usable, and the default configuration stops the load

The preset tiers read their cost with `CostOf`, which COPIES the source technology's unit wholesale. The copy does not run the packs through the existence ladder the typed path uses. So a pack in which a science pack is present as an ITEM but is no longer a `tool` produces a unit the engine refuses, on the DEFAULT setting, for a player who never opened the Startup tab:

```
Error Util.cpp:81: Error while running setup for technology prototype "bbb-balancer" (technology): Invalid research unit (automation-science-pack). Research unit(s) can only be tool type items at the moment.
```

Exit 1, no dump, **no `fkrecipes:` line at all, no setting named and no missing thing named**. This is finding 1's exact shape moved from the ingredient ladder, which is fixed, to the technology-cost copy, which is not. FkRecipes' own source says `THE LADDER ASKS ToolExists AND NOTHING ELSE` and cites this engine message as the reason; the copied-unit arm does not take that ladder.

The trigger is a synthetic pack, and that is worth saying plainly: a fixture that demotes `automation-science-pack` from `data.raw.tool` to `data.raw.item` and a second fixture that repairs the base game's own technologies around it, so this mod is the only thing that can fail. The defect it exposes is in the copy, not in the fixture.

Recovery makes it worse rather than better. From this state the two edits that work are moving `bbb-tech-cost` to Custom AND typing a pack the game has, two edits, and nothing on screen or in the log suggests either. Moving the dropdown BACK to a preset, which is the first thing a stuck player tries, returns to this same silent refusal.

### 14. BLOCKED. The fallback's stated boundary is real, its added sentence is correct, and the screen it names cannot be reached

The library's claim is the narrow one and it holds: what falls back lands on the author's own declaration, and an author declaration that cannot produce a legal result still stops the load. Reached with the same synthetic pack, on the Custom research arm.

**Nothing fell back.** `bbb-tech-cost = custom`, packs left on the word `default`. Exit 1, no dump, zero error lines, because the accumulated log ops never reach the host on a refused load. The whole of what the player reads, in the log and in the client dialog alike:

```
Failed to load mods: fklua: at the data stage, fkrecipes: the technology bbb-balancer has no science pack the game has; research takes at least one
```

followed by nine lines of Lua traceback. **It names no setting and no screen.**

**A stored value fell back.** The same pack with `better-belt-balancer-tech-packs = "2 unobtainium-pack"`. Exit 1, no dump, zero error lines, and one added sentence:

```
Failed to load mods: fklua: at the data stage, fkrecipes: the technology bbb-balancer has no science pack the game has; research takes at least one. The stored value of better-belt-balancer-tech-packs could not be used, so the mod's own declaration applied; correcting it under Settings > Mod settings > Startup is what a player can change here.
```

With the recipe text ALSO bad, the sentence names the recipe setting, which is the one walked first. So the mechanism is correct in all three directions: absent when nothing fell back, present when something did, and naming the first in walk order.

**And the route it names is closed**, re-measured on the client this time rather than carried from the first assessment. The dialog offers `Disable listed mods`, `Disable all mods`, `Manage mods`, `Restart`, `Exit` and a `Reset mod settings` checkbox. `Manage mods` opens the Mods screen, which has Manage, Explore and Updates tabs, a mod list with enable checkboxes, and Back and Confirm, and **no Mod settings button**; `Back` returns to the same error dialog. `Restart`, which the first assessment never pressed, relaunches the game into an identical dialog over a file whose sha256 has not moved. Three headless runs against that unchanged file leave one sha256, `d807f11a0efa7ae5ab12c538f18c1bb0ca443a476f03dc62dd3f7b45ad7c61f8`, all three times.

So the sentence tells a player to go somewhere the game will not take them. The escape is unchanged from the first assessment: `Reset mod settings` plus `Disable listed mods`, six steps, every startup preference gone and the mod disabled. The cost is the one that earned finding 2 its BLOCKED, on a narrower trigger, and the added sentence does not change the cost.

### 16. BLOCKED. A NaN in the seconds crashes the engine after the mod has already handled it, and there is no way back from inside the game

The first assessment recorded this as the guest refusing a value the engine's range check let through. The mechanism is different and worse. The library now falls back correctly and says so, and then the engine aborts:

```
fkrecipes: ERROR: better-belt-balancer-tech-seconds holds a value that is not a finite number. The mod loaded with its own default instead; fix the number under Settings > Mod settings > Startup, then restart.
fkrecipes: bbb-balancer takes its research cost from better-belt-balancer-tech-packs: count 250, time 15, packs 1 automation-science-pack, 1 logistic-science-pack
Error GlobalModSettings.cpp:204: Before: <hex>
Error GlobalModSettings.cpp:215: After:  <hex>
Error GlobalModSettings.cpp:217: Saving and loading changed data.
Error CrashHandler.cpp:643: Received 6
```

The two hex blocks are **byte-identical** (`cmp` exit 0). The engine's write-back self-check compares deserialised values, `NaN != NaN`, so it declares the data changed and calls `abort()`. The data stage had already completed. `seconds-inf` does not do this, because the engine resets an infinity by range; a NaN passes a range check because every comparison against it is false.

**From the client the player gets a bare operating-system dialog**: "Unexpected error occurred. If you're running the latest version of the game you can help us solve the problem by posting the contents of the log file on the Factorio forums", a log path, and `No` / `Yes` for opening the folder. No mod name, no setting, no `Reset mod settings`, no `Disable listed mods`, no route to anything. The file is never rewritten, so every relaunch does it again. **There is no in-game recovery at all**, which is strictly worse than the lock-out finding 2 described, because that one at least reached a dialog with buttons.

The one thing that limits it: **a player cannot type a NaN.** Measured on the client, `nan` typed into the seconds field leaves `45` untouched while a digit typed straight after lands, so the numeric widget filters non-numeric input. This is reachable only through a file written outside the game, which is a modpack, a tool, a shared settings file or a corrupted write. It belongs to the engine rather than to this mod or the library, and neither can defend against it, since the crash happens after both have finished.

### 15. MISLED. The research-cost setting has no tooltip at all on a stock install, and nothing says so

`bbb-tech-cost` carries a `localised_description` in the engine's own dump, a head line and three preset lines. **On the client its row has no info icon and no tooltip**, while all five other rows do. Photographed at three magnifications.

The cause is measured with a control rather than inferred. That composed description names `["technology-name.logistics-2"]` and `["technology-name.logistics-3"]`, and **no locale file shipped with this install defines either key**: base defines `logistics` only, because the two upgrade tiers take their names by another route. Adding a three-line probe mod that defines exactly those two keys, changing nothing else, **restores the info icon**; removing it takes it away again. Two runs, identical in every other respect, photographed side by side.

So the entire description is invisible: what the setting does, that the cost is copied from the named technology, that the balancer's research moves to sit beside it, and the Custom prerequisite sentence that finding 7 and the fix leg both argued about. The player has the four dropdown labels and nothing else. And nothing anywhere reports it: exit 0, no engine warning, no `fkrecipes:` line, and the settings dump says the description is present. A headless check cannot see this; only a client can.

The same trap is live for the ingredient dropdown, which composes `["string-mod-setting.<setting>-<value>"]` entries. Those keys this mod does define, so its recipe tooltip renders; a consumer that misses one would lose its whole tooltip the same silent way.

### 18. MISLED. The word `none` makes the balancer part free, and no description mentions it

`better-belt-balancer-recipe-ingredients = none`, in either case:

```
fkrecipes: bbb-balancer-part takes its ingredients from better-belt-balancer-recipe-ingredients: none
```

Exit 0, one informational line, no error line, and the emitted recipe has `ingredients: {}`. A balancer part costs nothing, and the `bbb-balancer-part-recycling` recipe disappears from the prototype list with it, which is the one stored value in this whole corpus that moves the `Prototype list checksum`.

The field's own tooltip explains `default` and says nothing about `none`. Neither does the dropdown's, neither does the changelog, and neither does any README. So the language accepts a word the player was never told about, and the outcome is a free item with no warning of any kind. A player who typed it meaning "no custom recipe" gets the opposite of what they meant, and nothing in the game corrects them.

### 17. AWKWARD. A pack that loads AFTER this mod can delete its recipe, and this mod reports success

Same two mods, same settings file, only the load order reversed.

| order | `fkrecipes:` lines | recipe emitted | technology | prototype list checksum |
|---|---|---|---|---|
| remover first | one ERROR, one merge line | `6 iron-plate, 2 iron-gear-wheel` | unlocks bbb-balancer-part | 3527174106 |
| this mod first | **none** | **the `bbb-balancer-part` recipe does not exist** | `bbb-balancer` exists with `effects: {}` | 1950488484 |

In the reversed order the engine exits 0, this mod logs success (`takes its ingredients from ...: 4 transport-belt, 2 iron-plate`), the item prototype still exists, and the player has a mod whose part can never be crafted and a research that unlocks nothing. Nothing in this mod's error channel fires, because from its point of view the text was valid when it read it. This is the engine's staging model rather than a defect here, and it is recorded because a player hitting it has no signal at all to go on.

## The release-to-head matrix

Every release rebuilt from its own commit, plus the authentic 0.2.2 zip; settings read out of the engine's own `mod-settings-dump.json`, which `--dump-data` writes beside the data dump. `data-raw-dump.json` carries no setting prototypes at all, so nothing below is read from source.

| release | commit | pin | runs natively here | what it declares, out of the engine | its values | onto head |
|---|---|---|---|---|---|---|
| 0.1.0 | `1c3858a` | 2.0 | yes | nothing; no mod-settings.dat is written | | nothing to migrate |
| 0.2.0 | `60e32ce` (recut) | 2.0 | yes | `bbb-multi-edge-parts` bool, runtime-global | | survives by name |
| 0.2.1 | `b29e91d` | 2.0 | yes | 2 dropdowns + the bool | 6 recipe, 3 tech | all 9 survive |
| 0.2.2 authentic | portal artifact | 2.0 | yes | identical to 0.2.1 | 6 recipe, 3 tech | all 9 survive |
| 0.3.0 | `649fbb7` | 2.1 | no | not reachable | | nothing to migrate |
| 0.3.1 | `0608e8e` | 2.1 | no | not reachable | 6 recipe, 3 tech by pin | all 9 survive |
| 0.3.2 | `cf5a78e` | 2.1 | no | not reachable | 6 recipe, 3 tech by pin | all 9 survive |
| head 0.3.3 | `29bad4f` | 2.1 | no; the recut and the restamp do | 2 dropdowns + 4 fields + the bool | 7 recipe, 4 tech | |

The refusal text is identical for all four 2.1-pinned packages:

```
Error Util.cpp:81: Failed to load mod "better-belt-balancer":
• better-belt-balancer
    • Incompatible Factorio version (current: 2.0, required: 2.1)
    • Dependency base >= 2.1.0 is not satisfied (active: base 2.0.77)
```

The restamp and the recut are interchangeable for the settings and data stages: on `defaults.dat` both produce the normalised dump hash `8017bbdb5cdbbe2d9506e3b24a0956bc8e79d28938ff6952ada072f33947afff`. The recut was used for every row that touches a save or a server.

The six recipe values are `vanilla`, `cheap`, `belt-fast`, `belt-express`, `splitter`, `splitter-express`; the three tech values `logistics`, `logistics-2`, `logistics-3`. **Not one of the nine is reset, renamed or lost**, and no release ever offered a value head has withdrawn. `custom` is new in both dropdowns and is the whole of the exposure in finding 3.

## The save matrix

Saves built and loaded through this repository's own two-phase method (`--create` builds the world in `on_init` and writes the save, `--benchmark` loads it), with a builder mod that researches the technology, lays a 2x2 balancer with belts feeding it, stocks an assembling machine from the live recipe's own ingredient list with a craft in flight, and fills a chest, and a probe mod that reads it all back at tick 0. Saves made on the authentic 0.2.2 where the row says 0.2.2; head is the true 2.0 recut throughout.

| direction | settings file | outcome |
|---|---|---|
| 0.2.2 save to head | the same file | every stored value honoured, world fully intact, four new settings appended at defaults, zero `fkrecipes:` lines |
| 0.2.2 save to head | **no file** | loads on the DECLARED DEFAULTS whatever the save was made with; a save made under `splitter-express` runs `vanilla` and the assembler's stock is destroyed. The engine says nothing |
| 0.2.2 save to head | a file that differs | the file wins outright, the engine says nothing, the stock is destroyed |
| head save to head | the text edited, still valid | loads, the recipe changes, the stacks the two lists do not share are destroyed |
| head save to head | the text edited to a typo | loads, ONE error line, the recipe falls back to the mod's own, the stacks are destroyed, the line says nothing about them |
| head save to 0.2.2 (downgrade) | the same file | loads, world whole, both dropdowns silently reset and persisted, the four settings 0.2.2 does not declare survive verbatim |
| 0.2.2 save to head plus a pack removing an item | the same file | loads, the ladder merges, and the item is destroyed in the CHEST as well as in the assembler. Both halves of this row ran with the three expansion mods disabled, because the fixture breaks Space Age; every other row is on the stock mod set |

**The runtime-global bool is the one value the save wins.** A save made with `bbb-multi-edge-parts = true` loaded under a file saying `false` runs with `true`, because the bool is map state; the rewritten file still says `false`. Startup settings are the opposite: the file always wins and the save's embedded record is never consulted headless.

`level-init.dat` carries the mod list as a length-prefixed name, a 3-byte version and a 4-byte value, then the startup settings as a property tree in exactly the `mod-settings.dat` node encoding, proved by exact substring rather than by eye: the 88-byte value of `recipe-vanilla.dat`'s `startup` key is found at offset 0xb9 of the save's file. **One correction to the first assessment**: the 4-byte value is not a checksum of the mod. It is zero for a mod the data stage never loaded a file from, measured by adding a one-comment `data.lua` to a control-only probe and watching its field move from `00 00 00 00` to a value while the others stayed put.

Across every load: zero `Migration` and zero `Applying` lines, and the only thing told about the version moving is Lua, through `on_configuration_changed` (`mod_change better-belt-balancer old=0.2.2 new=0.3.3`, `mod_startup_settings_changed=true`). The engine tells the player nothing.

**The startup-settings sync prompt is NOT re-measured this round.** The first assessment found it in the client and found that confirming it rewrites `mod-settings.dat` back to the save's stored values, which is a recovery route no headless run shows. That stands as its measurement; nothing in this cycle touches it.

## Hostile and accidental bytes

Every file run against head's recut with `--dump-data`, the engine half and this mod's half kept apart. The ones that say something new:

| file | engine, before any stage runs | this mod, after | the file afterwards |
|---|---|---|---|
| dropdown outside the list | **silent reset** to `vanilla`, no log line | the "is edited but not on custom" advisory | rewritten |
| invalid UTF-8 | **passed straight through** to the guest | **two** ERROR lines, `contains characters that are not text; retype the list` | unchanged |
| wrong type, string | `Error StringSetting.cpp:71: ... Value must be a string ...` then reset | the advisory | rewritten |
| wrong type, int | `Error IntSetting.cpp:71: ... Value must be a integer ...` then reset | normal, count 20 | rewritten |
| count -1, count 2147483648 | **silent reset** to 20, no log line | nothing | rewritten |
| seconds +Inf, seconds -5.0 | **silent reset** to 15, no log line | nothing | rewritten |
| seconds NaN | passed through, then **abort**, finding 16 | one ERROR line, correct, before the abort | **never rewritten** |
| version stamped 2.1.30 or 3.0.0 | `Failed to read mod settings: Map version 2.1.30-0 cannot be loaded because it is higher than the game version` | nothing; the mod is on its declared defaults | **REPLACED** with a fresh all-defaults file |
| truncated at half, truncated 2 bytes short, empty | `Not enough data remaining (wrong count?)` / `Couldn't read from input file. File could be corrupted.` | nothing | **REPLACED** with a fresh all-defaults file |
| an unknown setting NAME alongside valid ones | kept verbatim, no line | nothing | unchanged, and the unknown keys survive a later rewrite |
| `default,` with a trailing comma | passed through | parses as the word `default` | unchanged |

**The engine never deletes the file; it replaces it with the mod's declared defaults**, byte-identical to a freshly written `defaults.dat`. On the two version stamps that means a settings file written by a NEWER machine is destroyed rather than preserved, and the player re-enters every setting by hand with nothing having told them why.

The four whole-file failures and the two silent resets are the engine's and neither this mod nor the library can see them, let alone report them. What this mod controls is the row above the line: an invalid-UTF-8 text now reaches the guest and gets a named, recoverable error, where the first assessment had no measurement at all.

## Multiplayer and modpacks

**A server with a bad text now starts.** The first assessment measured exit 1 at 0.5s with no `Hosting game` line, so the port never opened. Today, with `better-belt-balancer-recipe-ingredients = "2 iron-plat"`:

```
fkrecipes: ERROR: better-belt-balancer-recipe-ingredients, entry 1 ("2 iron-plat"): no item or fluid is named iron-plat. The mod loaded with its own default instead; fix the text under Settings > Mod settings > Startup, then restart.
Info UDPSocket.cpp:38: Opening socket at (IP ADDR:({0.0.0.0:34197}))
Hosting game at IP ADDR:({0.0.0.0:34197})
```

It stayed up for the full run and shut down cleanly, and the bad text was not rewritten. That is the single clearest win of the cycle: an admin's typo no longer takes the server off the network.

**The prototype list checksum cannot tell two recipes apart.** Measured over five stored texts, `Prototype list checksum` is **3197544275 for all of them**: the declared default, two different custom lists, and a text that fell back. It moved only for `none`, which removes a prototype from the LIST. `Checksum of better-belt-balancer: 4150411655` is constant in every run and is a checksum of the mod's files. So a client on a fallen-back text and a server on a working one hold DIFFERENT prototypes and the same prototype checksum.

What catches it instead is a separate gate: the binary carries `startup-mod-settings-crc` and the install's own `core.cfg` carries `[gui-mod-startup-settings-mismatch] ... Your mod startup settings do not match with those of the server you are connecting to.` That compares stored VALUES, and the fallback does not normalise the file, so the bad text stays in it, the values differ, and the client is asked to adopt the server's and restart. **NOT REACHABLE: an actual two-party join.** There is no headless client here, so the comparison, the dialog and the outcome of accepting it are named from the binary's symbols and the shipped locale string rather than measured, and the CRC value itself appears in no log.

The modpack rows are findings 17 and 18 above, plus: a stored text naming an item another pack removed falls back with one ERROR line and the ladder merges beside it, giving a craftable part on this mod's own terms.

## Determinism

The same file and the same mod set on two user directories at two different absolute paths, sha256 over the `jq -S .` normalisation of the whole 27MB dump:

| file | hash | same at both paths |
|---|---|---|
| defaults | `8017bbdb5cdbbe2d9506e3b24a0956bc8e79d28938ff6952ada072f33947afff` | yes |
| custom-good | `3c739cc1bcf526941fccb44ea7c1dbc57def861ed4fcaceb2cbcf74ef8d72793` | yes |
| recipe-splitter-express | `410f4532e90da817daffe86cdb2f913345e8ac39a41d510bdfab5d57a71f4727` | yes |

The three are mutually distinct, so the hash is sensitive to what it should be sensitive to.

**Determinism after a merge, which is what this cycle made necessary**, on the vanilla preset with the item removed: `efb3a2ffa38c806f8d1ef3cc3829e7a8aa72b702c4f123885cada9397d8aae05` at two different paths, with the mods listed in the opposite order in `mod-list.json`, and with `locale=de` instead of `locale=auto`. Four identical hashes; the cheap preset gives two identical hashes of its own. The merge line is byte-identical in every run and the merged entry keeps the first occurrence's position, so the emitted ingredient ORDER is stable.

Two caveats stated rather than hidden. The mod-order variation proves that the engine canonicalises install order before the dump, since it rewrites `mod-list.json` into its own order during the load. And a headless dump run prints no locale line, so the `locale=de` arm shows the hash did not move without proving the engine consumed the setting; call that half-measured.

## What the design must change, and where

**FkRecipes.** (a) `CostOf`'s copied research unit must go through the same existence check the typed and ladder paths use, or the copy must be refused by name; today it hands the engine a unit it will reject, on a default configuration, with nothing said (finding 13). (b) The boundary refusal's added sentence names a screen the engine will not let the player reach, and the case where nothing fell back names no setting and no screen at all; either the sentence should say what a player can actually do from an error dialog, or the library should state plainly that this route is closed (finding 14). (c) A composed description naming a locale key the game does not define costs the consumer its ENTIRE tooltip, silently, and only a client can see it; the locale checker should treat a key a composed description references as a required key, or the composition should degrade to the keys that resolve (finding 15). (d) A NUL below 22 stored bytes is cut by the engine and the cut persists, so the library quotes text the player did not type; the library cannot prevent the cut, but it can notice that the stored value it received is shorter than the file's and say so. (e) `none` on a RECIPE arm produces a free item; if that is intended it needs to be documented in the composed line beside `default`, and if it is not it should be refused for a recipe.

**BBB.** (f) Define `technology-name.logistics-2` and `technology-name.logistics-3` in this mod's own locale file, or stop composing the cost lines from technology names, until the library closes finding 15; today the research-cost setting has no tooltip and nothing reports it. (g) The field's description explains `default` and not `none`, while the language accepts both and `none` is the one that changes the balance (finding 18). (h) The changelog discloses that changing the recipe empties an assembler and does not disclose that a typo in the Custom field is now a way to change the recipe (finding 8). (i) `test/check-datastage.py`'s remover fixture breaks Space Age before this mod can be judged on a stock install, so the gate arm it guards is only honest with the expansions disabled; the fixture should sweep at `data-final-fixes` the way the synthetic one built for this assessment does.

**Neither, and worth saying.** The silent reset of an unknown dropdown value, the silent reset of an out-of-range number, the destruction of an assembler's input stock on a recipe change, the replacement of an unreadable settings file with defaults, the NUL truncation below 22 bytes, the NaN abort and the absence of any conditional visibility on the settings screen are all the engine's. What the design controls is exposure, wording and what it chooses to notice.

## What could not be reached

- **A real two-party multiplayer join.** There is no headless client. The startup-settings CRC comparison, the mismatch dialog and the outcome of accepting it are named from the binary and the shipped locale file, not measured.
- **Anything on a real 2.1 engine.** Only 2.0.77 is here, so every head measurement is of the recut or the restamp, and what a 2.1 player sees, including the absence of the Map-tab bool, is inferred from the pin.
- **Authentic artifacts for six of the seven releases.** Portal downloads need credentials; only 0.2.2's zip is genuine, and the rebuilt 0.2.2 was run beside it as a control with an identical settings dump.
- **The 65535 merge ceiling, through this mod.** Every declared amount in the six presets is single-digit and the largest merge reachable is `4 plus 2 is 6`, so the library's item ceiling cannot be provoked from here. Reaching it would need a pack that rewrites this mod's declared amounts, which nothing outside this repository can do.
- **`readDropdown`'s unescaped message.** The engine resets an unoffered stored dropdown value before any stage runs, so the guest never sees one; the message exists and the game cannot reach it.
- **The startup-settings sync prompt**, not re-taken this round. It is the first assessment's measurement and nothing in this cycle touches it.
- **A craft driven to partial progress by a real power network.** `crafting_progress` was written by script; it survives save, load, version change and recipe change identically in every run, but an accumulated one was not tested.
- **Whether the settings widget shows the int and double minimum and maximum to the player.** The prototype carries them; the rendered widget was photographed and shows a plain number field with no range beside it, but whether the engine surfaces them some other way was not established.

This note is a new file. CLAUDE.md's index of `agents/` gains one row for it in the same commit, which is the row the first assessment left owed.
