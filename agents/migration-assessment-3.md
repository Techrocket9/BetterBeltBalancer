# What an update from a published release costs a player, graded for the first time against a written scope

**VERDICT, and it is the sentence the chain's stopping rule needs: SIX findings above CLEAN remain that are in
scope and owned by FkRecipes, BBB or FkLua. They are 13 (BLOCKED), 14 (BLOCKED), 19 (MISLED), 22 (MISLED),
20 (AWKWARD), 21 (AWKWARD) and 23 (AWKWARD).** Seven, counted by number; six distinct defects, because 13
and 14 now share one trigger. THE CHAIN DOES NOT STOP HERE.

The THIRD adversarial assessment, 2026-09-14, of how a player's stored preferences and saved games from the
seven PUBLIC releases of this mod migrate onto head (0.3.3, unpublished), which declares its settings, recipe
and technology through [FkRecipes](https://github.com/Techrocket9/FkRecipes) at `61ac80c` on
[FkLua](https://github.com/Techrocket9/FkLua) at `16f0986`. Heads: BBB `731f13d`, both siblings clean and
read-only throughout. Nothing was fixed; every claim carries the command that produced it, and a claim that
could not be measured is marked NOT REACHABLE rather than asserted.

**THE SCOPE IS WRITTEN AND EVERY GRADE IS MADE AGAINST IT**, which is what makes this assessment different
from the two before it: FkRecipes' [`../FkRecipes/agents/threat-model.md`](../FkRecipes/agents/threat-model.md),
committed `221ff9c`. It names the player and the three places they look (the settings screen, the in-game
tooltip of the recipe or technology, the changelog), and its governing sentence is **THE LOG IS NOT WHERE A
PLAYER LOOKS**. It names seven in-scope situations A to G that must be CLEAN or engine-owned and disclosed
where a player looks, five out-of-scope ones H to L that go to an appendix and never carry a blocking grade,
and a rubric with an OWNER and a SCOPE letter on every finding. Every finding below carries both.

The first assessment is [`agents/migration-assessment.md`](agents/migration-assessment.md), committed `d8dc79e`,
findings 1 to 12. The second is [`agents/migration-assessment-2.md`](agents/migration-assessment-2.md),
committed `1caaf08`, which regraded those and added 13 to 18. A fix cycle answered the second in FkRecipes
(`6518971`, `67b5090`, `e4604d4`, `641cb50`, `221ff9c`, then fix round 2b to `61ac80c`) and in this mod
(`ae15b4d`, `e67a5d0`, `a9f6eca`, `90c0b89`, `e77210f`, `ea291ac`, `731f13d`). **All eighteen are regraded here
by number.** The two earlier files stay as the dated records they are and are not rewritten; where they are now
wrong, this one says so.

**Every measurement is on `Version: 2.0.77 (build 84539, mac-arm64, steam)`**, re-asked at the start of the run
and again by each of the eight measuring arms, `"$FACTORIO" --version`, exit code read directly. Head is pinned
2.1 and this is the only binary here, so every row says which package it used, in one of exactly three shapes: a
**native 2.0 release**, **the 2.1 package restamped for 2.0**, or **the true 2.0 recut** (`fklua.toml` at
`factorio_version = "2.0"`, `dependencies = ["base >= 2.0.0"]`, `api = "2.0.77"`, bindings and lock regenerated,
built under `GOTOOLCHAIN=go1.26.6`). The recut is what almost every row uses. The restamp and the recut differ in
four generated files and are byte-identical in `settings.lua`, `data.lua`, `data-final-fixes.lua`, `fk_data.lua`,
`fk_data_module.lua`, `info.json` and the locale file, so the restamp is honest for the settings and data stages
in the strongest sense available. One authentic artifact exists on this machine, published 0.2.2, and it is what
every release row that touches a save uses.

**THE CLIENT COULD NOT BE REACHED THIS ROUND, and that is a loss rather than an omission.** Computer-use access
to Factorio was granted at the start and the game was launched under a private user directory; the background
window-capture path fails on Factorio's GPU-composited window (`Failed to start stream due to audio/video capture
failure`, twice, on the app's only window) and two requests for full-screen control went unanswered. So no
screenshot was taken, control was released the moment that was established, and everything only a screen can
settle is marked NOT REACHABLE with its checklist owed. What was built in its place is stronger than nothing and
weaker than a photograph: a control-stage probe that `localised_print`s every composed description through the
engine's own locale resolver, which is the same instrument FkRecipes' decision D used. It settles what the text
RESOLVES to. It settles nothing about layout, wrapping, or whether a tooltip draws at all.

Eight arms, 245 private engine runs in all, every one through a runner that refuses to name the real user
directory and hashes two files inside it before and after. **`REAL USERDIR UNTOUCHED: both hashes unchanged` on
every run**, including the one client launch.

## The grades, at a glance

| # | second assessment | now | owner | scope | what moved |
|---|---|---|---|---|---|
| 13 | BLOCKED | **BLOCKED** | FkRecipes + BBB | F | the diagnosis is transformed; the exit code is not |
| 14 | BLOCKED | **BLOCKED** | engine for the dialog, FkRecipes closed for the sentence | F | the sentence names no route now, and the route is still closed |
| 19 | new | **MISLED** | FkRecipes | B | what a refused text lands on is the dropdown, and six sentences say otherwise |
| 22 | new | **MISLED** | FkRecipes | F | the scope document's own scope F sentence is false on three clauses |
| 20 | new | **AWKWARD** | FkRecipes | F | one degradation of three writes no note, and it is the one that moves the tech tree |
| 21 | new | **AWKWARD** | FkRecipes | B, F | the fallback note replaces the mod's own description rather than joining it |
| 23 | new | **AWKWARD** | BBB for the disclosure, engine for the behaviour, FkLua for the record | C | a settings file a newer engine wrote is replaced with defaults, silently |
| 3 | BLOCKED | **CLEAN** | FkRecipes, closed | C | `custom` is withdrawn and the round trip is a byte identity |
| 4 | MISLED | **CLEAN** | FkRecipes, closed | B | the state the screen could not render no longer exists; see 19 for what replaced it |
| 5 | AWKWARD | **CLEAN** | engine, disclosed | B | the wrap is disclosed in every shape that renders a typeable list |
| 8 | AWKWARD | **CLEAN** | engine, disclosed twice | D | the changelog now discloses the typo path; 620 items measured |
| 9 | AWKWARD | **CLEAN** | FkRecipes, closed | B | both numeric ranges are on the screen in words |
| 10 | AWKWARD | **CLEAN** in scope | engine | B, K | one policy, code points named; the NUL cut is H and K |
| 16 | BLOCKED | **CLEAN** in scope | engine | K, H | both numbers are ints, so no name this mod declares can hold a NaN |
| 18 | MISLED | **CLEAN** | FkRecipes, closed | B | `none` is disclosed beside `default`, and only on the kind that takes it |
| 1 | CLEAN | **CLEAN** | | F | unchanged, re-measured at two paths |
| 2 | CLEAN | **CLEAN** | | B | 70 of 70 stored texts load |
| 6 | CLEAN | **CLEAN** | | F | unchanged |
| 7 | CLEAN | **CLEAN** | | F | unchanged |
| 11 | CLEAN | **CLEAN** | BBB | build | `test/check-release-arm.sh` exit 0, re-run |
| 12 | CLEAN | **CLEAN** | | A | all nine still migrate, prototypes byte-identical |
| 15 | MISLED | **CLEAN** on the mechanism, NOT REACHABLE on the render | FkRecipes | B | the alternatives form is measured to rescue the key; no client has hovered it |
| 17 | AWKWARD | out of scope (J), appendix | engine | J | reproduced, both load orders |
| 24 | new | **CLEAN, and it is the headline** | | A, C | the withdrawal makes a rollback a byte identity at every step |

## The findings above CLEAN, most severe first

### 13. BLOCKED, unchanged in outcome. A modpack that demotes one science pack still stops the load on the DEFAULT setting

Owner **FkRecipes and BBB**, split below. Scope **F**, which names this exact case in its own words: "a pack that
removes, renames or demotes (a science pack that is no longer a tool) an item, fluid or technology this mod's
declarations or the player's text name. The load never stops on this mod's account."

Fixture: `automation-science-pack` moved from `data.raw.tool` to `data.raw.item`, with a second fixture repairing
base's own technologies around it so the base game still loads. **Base alone with the fixture and without this
mod is exit 0**, which is what puts the row in scope rather than in case I. The trigger is synthetic and that is
worth saying plainly; the defect it exposes is in the declaration, not in the fixture.

| `bbb-tech-cost` | exit | what the player gets |
|---|---|---|
| `logistics`, THE DEFAULT | **1** | no dump, no game |
| `logistics-2` | 0 | `{count 200, time 30, packs 1 logistic-science-pack}`, prerequisite kept, one line, a tooltip note |
| `logistics-3` | 0 | `{count 300, time 15, three packs}`, prerequisite kept, one line, a tooltip note |
| typed packs `1 automation-science-pack` | **1** | no dump |
| typed packs `1 logistic-science-pack` | 0 | `{count 20, time 15, 1 logistic-science-pack}` |

The whole of what a player reads, in the log and inside the client's own `Failed to load mods` dialog alike:

```
Error Util.cpp:81: Failed to load mod "better-belt-balancer": fklua: at the data stage, fkrecipes: the technology bbb-balancer has no science pack the game has; research takes at least one, and none of automation-science-pack is a science pack here
```

**WHAT THE FIX CYCLE ACTUALLY BOUGHT, and it is a great deal.** The second assessment measured exit 1 with **no
`fkrecipes:` line at all**, an engine sentence naming neither the mod nor the property. Today the copied unit's
packs go through `ToolExists`, the unusable one is dropped by name, the two tiers that keep a pack load and
reprice and say so on the technology's own tooltip, and where nothing is left the refusal names the mod, the
technology, the rule and every name it tried. That is a transformed diagnosis and two thirds of the cases moved
from refusal to degradation.

**WHAT IT DID NOT BUY IS THE OUTCOME ON THE DEFAULT SETTING**, and the grade is the outcome. The mechanism,
traced in source: the default tier `logistics` carries a unit, so the ladder stops there; `filterCopiedPacks`
drops its only pack; `filtered.kept == 0` sends resolution to `by.Fallback`
(`../FkRecipes/go/data.go:1035`); `FallbackUnit()` names the same demoted pack
(`guest/go/tune/plan.go:566-572`); `markPackless` fires and `checkResolvedPacks`
(`../FkRecipes/go/data.go:1252`) refuses. `logistics-2` and `logistics-3` survive only because their copied units
also charge `logistic-science-pack`.

**THE SPLIT.** FkRecipes owns the refusal, and its own record holds it up as a deliberate survivor: the
enumeration at `../FkRecipes/CLAUDE.md:15` and the table at `../FkRecipes/agents/customizer-design.md:47` both
name the packless refusal by name. BBB owns the declaration that walks into it. Every pack name in this mod's
whole technology declaration is `automation-science-pack`, once, with no ladder, and
`guest/go/tune/plan.go:261-275` argues for that: "THE PACK LIST CARRIES NO `Fallbacks` LADDER, AND THAT IS A
DECISION ... no other name is likelier to be present, so a second rung would be a guess dressed as a ladder."
The one-line change that survives the measured pack is
`Packs: []fkrecipes.Pack{{Name: "automation-science-pack", Amount: 1, Fallbacks: []string{"logistic-science-pack"}}}`
at `guest/go/tune/plan.go:570`, and because `techPacks` is built from `FallbackUnit().Packs` the same edit gives
the player's own pack field a rung behind its default. The comment above it would become false in the same commit.

**WHERE THE PLAYER IS TOLD, AND WHERE THEY ARE NOT.** `README.md:70` describes this case accurately and is the
only place that does: "The one thing that can still stop the load is a research with no science pack at all ...
a research with none is free rather than cheap, so the load stops instead with a message saying so." It says
"missing from your game", and the measured pack DEMOTED rather than removed, so a player reading only that
sentence need not recognise their case in it. Nothing in `mod-data/locale/en/better-belt-balancer.cfg` says it,
nothing in the changelog says it, and the settings screen is unreachable because the game does not start.
**Whether the portal page counts as one of the three places a player looks is a question the threat model should
settle**; it lists three and the README is not among them.

Recovery is two Mod Settings edits away and nothing says so. The escape is finding 14.

### 14. BLOCKED. The refusal is still a lock-out, and the sentence that is now honest is also now silent about what to do

Owner **engine** for the dialog, **FkRecipes, closed,** for the sentence; the path into it is finding 13's, which
is FkRecipes' and BBB's. Scope **F**.

The added sentence is correct in all three directions, measured on the same fixture:

| what was stored | the added sentence names |
|---|---|
| packs `1 no-such-pack` | `better-belt-balancer-tech-packs` |
| ingredients `2 no-such-item` | `better-belt-balancer-recipe-ingredients` |
| both bad | `better-belt-balancer-recipe-ingredients`, the earlier in walk order |
| nothing fell back | no added sentence at all |

verbatim, in the amended form fix round 2 landed, with the route clause gone:

```
Error Util.cpp:81: Failed to load mod "better-belt-balancer": fklua: at the data stage, fkrecipes: the technology bbb-balancer has no science pack the game has; research takes at least one, and none of automation-science-pack is a science pack here. The stored value of better-belt-balancer-tech-packs could not be used, so the mod's own declaration applied.
```

`fallbackFact` at `../FkRecipes/go/data.go:675` and `fallback_fact` at `../FkRecipes/rust/src/data.rs:1553`
compose it, decorating four post-resolution exits. **The second assessment's complaint is answered: the sentence
no longer sends a player to a screen the game will not take them to.** What replaced the route is nothing. The
sentence now states a fact and offers no action, and neither does the refusal it decorates. On this fixture the
two actions that work are moving `bbb-tech-cost` to `logistics-2` and typing a usable pack, and no sentence
anywhere names either.

A second boundary, weaker still, was constructed and reached: a technology unit in a shape the copier cannot
decode (`logistics`' `unit.ingredients` rewritten to a bare-string list at the data stage and restored at
data-final-fixes, base alone exit 0):

```
Error Util.cpp:81: Failed to load mod "better-belt-balancer": fklua: at the data stage, fkrecipes: bbb-balancer: the unit of logistics holds a table this library cannot copy faithfully
```

It names the mod, the technology and the source, and nothing a player could change.

**On every refused load, no `fkrecipes:` line reaches the log at all**, confirmed by contrast against the loading
row on the same fixture: the accumulated log ops never reach the host, which is exactly why the added sentence
has to exist. The settings file is never rewritten by a failed load, so the refusal is permanent until something
outside the game edits the bytes. **The client dialog was NOT RE-MEASURED this round** and the second
assessment's walk stands as its measurement: five buttons, a `Reset mod settings` checkbox, `Manage mods` with no
Mod settings button whose `Back` returns to the same dialog, `Restart` reproducing itself over an unmoved file,
and one escape costing every startup preference and the mod.

### 19. MISLED, new, and it is the round's centre. A refused text falls back to the dropdown's current preset, and six sentences say it falls back to the mod's own default

Owner **FkRecipes**. Scope **B**, whose requirement is "the recipe or technology tooltip states the outcome".

Measured on all six presets, one run each, with `better-belt-balancer-recipe-ingredients = 2 iron-plat`:

| `bbb-recipe-cost` | emitted `bbb-balancer-part` ingredients |
|---|---|
| vanilla | `4 iron-plate, 2 iron-gear-wheel, 2 transport-belt` |
| cheap | `2 iron-plate, 1 transport-belt` |
| belt-fast | `4 iron-plate, 2 iron-gear-wheel, 2 fast-transport-belt` |
| belt-express | `4 steel-plate, 2 iron-gear-wheel, 2 express-transport-belt` |
| splitter | `1 splitter, 2 iron-plate` |
| splitter-express | `1 express-splitter, 2 steel-plate` |

**The fallback target is the dropdown's currently chosen preset. It is the mod's declared default only when the
dropdown happens to sit on `vanilla`.** The same on the research channel: a refused pack text under `logistics-2`
lands on two packs and under `logistics-3` on four, where the field's own tooltip states
`default: 1 automation-science-pack`.

The BEHAVIOUR is right and is deliberate: `../FkRecipes/go/data.go:890-893` says "A text that says the reserved
word, and a text this library cannot use, both leave the decision exactly where a player who typed nothing left
it", and both halves pin it (`TestARefusedTextBesideADropdownFallsBackToTheDropdown`,
`a_refused_text_beside_a_dropdown_falls_back_to_the_dropdown`). **What is wrong is every sentence written about
it.** Six, and the first two are where a player looks:

1. **The recipe's own `localised_description`**, which is scope B's "the recipe tooltip states the outcome", and
   which was byte-identical in all six runs above:
   `The stored value of better-belt-balancer-recipe-ingredients could not be used, so this mod's own choice applies instead. The reason is in the log.`
   plus the destruction sentence. What applied was the PLAYER's other choice. `fallbackNote` at
   `../FkRecipes/go/data.go:787` and `../FkRecipes/rust/src/data.rs:1821`.
2. **The settings tooltip's own line**, `textFallbackLine` at `../FkRecipes/go/customize.go:1016`:
   `A text this mod cannot use is set aside and that default applies instead`. "That default" is deictic and its
   nearest antecedent on the screen is the `default:` line three lines above it, which renders the author's
   declared list and nothing else. **The two lines contradict each other on screen**, measured through the
   engine's own resolver: with the dropdown on Cheap, the tooltip reads
   `default: 4 iron-plate, 2 iron-gear-wheel, 2 transport-belt` while the recipe the game built is
   `2 iron-plate, 1 transport-belt`.
3. **The library's log line**, `playerFallback` at `../FkRecipes/go/customize.go:1171`:
   `The mod loaded with its own default instead`. The log is not where a player looks, so this is evidence rather
   than a grade, and it is the sentence that resolves the tooltip's ambiguity in the false direction.
4. `../FkRecipes/docs/ingredient-list.md:59`: "the text is set aside, **the mod's own list applies instead**".
5. `../FkRecipes/docs/usage.md:291` and `:61`: "**your declared list is what the fallback lands on**".
6. The refusal's own `fallbackFact`, finding 14's sentence: "the mod's own declaration applied". Three phrasings
   of one event, and the event is none of them.

Two documents get it right, and both obliquely: `docs/ingredient-list.md:67` ("as though the field still said
`default`") and `agents/customizer-design.md:21` ("as if the player had typed the reserved word"). Those are
exactly right, because the word `default` beside a dropdown means the dropdown's current choice, which the
library's own composed switch line states one row above.

**NEITHER SENTENCE IS PARAMETERISED, AND THAT IS STRUCTURAL.** `playerFallback(reason, field, tail)` and
`fallbackNote(setting, destroysInputs)` have no slot for a preset name, and the composer runs at
`go/data.go:899`, BEFORE `readDropdown` at `go/data.go:912`, so `chosen` is not in scope. Naming the preset means
hoisting a read the library's own comment keeps deliberately late. The cheaper repair is the honest generic:
"so the option chosen above applies instead", which is true on all six presets and true of the reserved word too,
and which the switch line one row up already says in those words. `textFallbackLine`'s "that default" wants the
same treatment.

**This is finding 4 one case narrower, and FkRecipes' own ownership table predicted it**: "A text the library
cannot USE falls back to the dropdown's CURRENT PRESET while the field goes on showing the player's typo, so
there is a value on the screen that is not in force." The prototype note was what was supposed to close that.
The note is there, on the right prototype, one per prototype, every element under the engine's ceiling. Its
sentence is the part that is wrong.

### 22. MISLED, new. The scope document's own scope F sentence is false on three clauses

Owner **FkRecipes**. Scope **F**. This one grades the written scope itself, which is what "graded to a written
scope" has to mean when the scope and the implementation disagree.

`../FkRecipes/agents/threat-model.md:21` says: "The load never stops on this mod's account; the library emits the
nearest legal prototype, logs one ERROR line, and the recipe or technology tooltip says what was substituted."
Measured on the canonical case, a pack that removes the item `transport-belt` with `bbb-recipe-cost = vanilla`,
the default every silent player is on:

- **"The load never stops on this mod's account"** is contradicted by finding 13, on a case the same sentence
  names ("demotes (a science pack that is no longer a tool)"). FkRecipes' own `CLAUDE.md:15` calls the total
  claim the mistake: "'nothing an environment does can stop the load' is the same mistake one level up." The
  threat model is the last document still making it.
- **"logs one ERROR line"** is false. A ladder substitution logs nothing at all; an exhausted ladder logs one
  INFORMATIONAL line; a merge logs one informational line
  (`fkrecipes: bbb-balancer-part: iron-plate is in the list twice after the fallbacks, so the amounts are added: 4 plus 2 is 6`).
  `ERROR: ` is written only by `playerFallback`, which a ladder never reaches.
- **"the recipe or technology tooltip says what was substituted"** is false BY DESIGN.
  `agents/customizer-design.md:56` and `docs/usage.md:312` both say the resolve-or-drop ingredient ladder gets no
  note deliberately, and the emitted recipe carries no `localised_description` at all on the two rows where a
  substitution actually happened with an untouched field.

**The design's own justification has a hole underneath it, and it is worth a sentence of its own.** Both
documents and both halves' source comments justify the missing note by quoting a disclosure: "the dropdown's own
composed description already discloses it ('where one names something your mods do not have, the nearest thing
they do have is used instead')". **The library composes no such sentence.** Grepping FkRecipes for it finds only
comments and documents, never a literal. The live text is THIS MOD'S OWN locale entry,
`mod-data/locale/en/better-belt-balancer.cfg:203`. The library's reason for omitting a note therefore rests on a
sentence a consumer is free to write differently, or not at all.

**THE PLAYER OUTCOME OF THE LADDER ITSELF IS CLEAN AND IS GRADED SO.** The resolved `bbb-recipe-cost` tooltip, as
the engine's own locale resolver renders it, opens: "Every option is safe in an overhaul pack: where one names
something your mods do not have, the nearest thing they do have is used instead, and where that leaves one item
named twice the two amounts are added, **so what you craft can be a shorter list than the one shown**." That is
the rule, the merge and the explicit warning that the `type:` line may not be what the game gives, on the
settings screen, where a player looks, and the recipe's own ingredient rows in game show the true list. Nothing a
player reads is false. What is false is the scope document, and the remedy is one sentence in
`agents/threat-model.md:21` plus one clause each in `agents/customizer-design.md:56` and `docs/usage.md:312`
naming the consumer's obligation rather than the library's composition.

Correcting that sentence would also retire this finding and settle what finding 20 below is measured against, so
it is the cheapest item on the whole list.

### 20. AWKWARD, new. One degradation of three writes no note, and it is the one that moves the technology to the root of the tree

Owner **FkRecipes**. Scope **F**.

Fixture: a pack that RENAMES the technology `logistics` to `renamed-logistics` before this mod's data stage and
repoints every base reference at data-final-fixes. Base alone exit 0. Eight rows measured, every one exit 0, the
load never stops.

```
fkrecipes: bbb-balancer: no source for the logistics cost carries a unit, so the fallback cost applies and the technology has no prerequisite
```

and the emitted technology on all eight rows: `unit {count 20, time 15, 1 automation-science-pack}`,
**`prerequisites` absent entirely**, `localised_description` **absent entirely**. The balancer research moves to
the root of the technology tree, researchable from the first minute, and reprices from whatever the chosen tier
charged to base's Logistics floor.

The branch is `../FkRecipes/go/data.go:996-1012` and its twin `../FkRecipes/rust/src/data.rs:1008-1030`. It
appends a log line and calls no note recorder. **Its two neighbours in the same `switch` both do**:
`res.noteOn(tgt, packlessSourceNote(source))` at `go/data.go:1034` and
`res.noteOn(tgt, packDroppedNote(filtered.dropped[0]))` at `go/data.go:1042`, and those notes were measured
arriving on the technology in the demote fixture
(`This game has no automation-science-pack, so this research was priced without it. The reason is in the log.`,
107 bytes, one element).

**The omission is not argued for anywhere.** `go/data.go:803-817` scopes the environmental notes to three and
names exactly one deliberate exclusion, the ingredient ladder. `docs/usage.md:306-310` enumerates the same three
in a fenced block. `agents/customizer-design.md:56` says "AND EACH OF THE THREE SAYS SO WHERE THE PLAYER LOOKS"
and names the clamp, the dropped pack and the ladder exclusion. This branch appears in none of them. It is also
the branch with the largest player-visible consequence of the three, because it is the only one that removes a
prerequisite, and the design record's own stated criterion for a note, "presence a player cannot check anywhere",
applies to it at least as strongly as to a dropped pack. Both halves mirror each other exactly, so this is a
design gap and not a language divergence.

**What IS disclosed, and why it is not enough for CLEAN.** This mod's own `bbb-tech-cost` description states the
rule: "A tier your mods leave unpriced steps down to the next belt tier, and where that runs out the research
takes 20 units of automation science at 15 seconds each with no prerequisite." So a player who reads that
tooltip and then notices their research at the root of the tree can work out what happened. The rule is stated
and the STATE is not, which is the structure the whole fix cycle spent itself removing everywhere else. One
`res.noteOn` call closes it.

The same fixture leaves `bbb-tech-cost`'s composed tooltip reading `Logistics (default): cost of Logistics` for a
technology that no longer exists in that game, which is recorded here rather than graded separately: base's
locale key still resolves, so the line renders as a real technology name for a prototype the pack renamed.

### 21. AWKWARD, new. The fallback note REPLACES the mod's own description on the technology rather than joining it

Owner **FkRecipes** for the rule, **BBB** for where its description lives. Scope **B** and **F**.

Measured through the engine's own resolver, one clean load and one fallback load, same package, same probe:

| load | `technology.bbb-balancer` description, as the engine resolves it |
|---|---|
| clean | `Balancers of any shape, built one tile at a time.` |
| a refused pack text | `The stored value of better-belt-balancer-tech-packs could not be used, so this mod's own choice applies instead. The reason is in the log.` |

The raw fields say why: clean it is `{"technology-description.bbb-balancer"}`, on the fallback it is
`{"", "The stored value of ..."}` and the locale key is not referenced at all. **The sentence that tells a player
what the technology DOES is gone for the whole of that load, and the note stands in its place.**

FkRecipes' decision 4 states the intended rule: "Author description present:
`{"", "<description>", "\n<note>"}`; absent: `{"", "<note>"}`". The rule reads `TechSpec.Description`, a plan
field. BBB declares its description the ordinary Factorio way, as a `[technology-description]` locale entry, so
the library sees "absent" and writes the note alone. The engine then prefers the prototype field over the locale
key and the author's sentence is displaced.

The displacement is per prototype and not per load, which is the one thing that limits it: a run that refused
only the ingredient text left the technology's own sentence intact and put the note on the recipe alone. And the
recipe has nothing to lose, because this mod ships no `[recipe-description]` entry, so on the recipe the note is
purely additive. **This is a consumer-shaped gap the library cannot see from inside**: nothing in FkRecipes can
know that a locale key exists for a prototype it is composing onto. The two repairs are a sentence in
`docs/usage.md` telling an author to declare `Description` in the plan if they want it kept, or a note shape that
appends to whatever the engine would otherwise have resolved. Neither is obviously right and neither is written
down.

### 23. AWKWARD, new. A settings file a newer engine wrote is replaced with defaults, and every value the player chose goes with it

Owner **BBB** for the disclosure, **engine** for the behaviour, **FkLua** for the record. Scope **C**, which puts
this in scope by name: "a settings file stamped by a newer engine version is a file the engine's own writer
produces, and what the older engine does with it (measured: it refuses the file and rewrites defaults) is
engine-owned **and disclosed**, not out of scope."

Two files built by hand, otherwise carrying a full set of the player's six real values plus the map bool, with
the header version stamped higher. The encoder's 2.0.77 control is byte-identical to `fklua modsettings write`'s
output, and the engine quotes the stamp back, so the header was parsed as intended.

| stamp | exit | the engine's message | the mod emitted | the file afterwards |
|---|---|---|---|---|
| `2.1.30-0` | **0** | `Error GlobalModSettings.cpp:157: Failed to read mod settings: Map version 2.1.30-0 cannot be loaded because it is higher than the game version (2.0.77-0).` | pure declared defaults | **REPLACED** by the 432-byte defaults file, re-stamped 2.0.77 |
| `3.0.0-0` | **0** | the same with `3.0.0-0` | the same | the same |

The load then SUCCEEDS. A player who plays one session on 2.1 and rolls the GAME back to 2.0 loses both dropdown
choices, both typed texts, both numbers and the map setting, and their next load quietly rebuilds the balancer
part at the vanilla cost.

**It is disclosed nowhere a player looks.** The settings screen shows six rows at their defaults, indistinguishable
from a fresh install. No tooltip mentions a settings file, a game version or a rollback:
`grep -ci "reset"` over the engine's settings dump returns 0. `grep -in 'newer|downgrade|roll back|rollback|reset'`
over `mod-data/changelog.txt` returns zero lines across all six entries. The README says nothing. The log carries
one `Error` line, and the log is not where a player looks.

Under scope K's rule an engine behaviour is CLEAN when path-removed and disclosed, and **AWKWARD when undisclosed
where disclosure is possible**. Disclosure is possible and cheap: one changelog Info row, in the same shape as the
one the fix cycle wrote for the `custom` rollback and then correctly deleted when `custom` was withdrawn. The
threat model asserts this case IS disclosed; measurement says it is not, which is the second place this round found
the scope document ahead of the code.

**FkLua's share is the record.** `../FkLua/agents/engine-findings.md` carries nine entries, and this behaviour is
not among them. The seven that touch this assessment are listed in the appendix; the version-stamp replacement of a
whole settings file, measured twice here and once by the second assessment, belongs beside them.

That the engine's own writer stamps the running version is measured in the other direction: **all 56 files this
engine wrote in the scope A, C and G arm carry `"version": [2, 0, 77, 0]`, without exception**. That a 2.1 engine
writes `2.1.x` is the same mechanism and is NOT REACHABLE here.

## The fifteen regraded to CLEAN

### 24. CLEAN, new, and it is this round's headline. The withdrawal made a rollback a byte identity

Scope **A** and **C**. The second assessment's finding 3 was BLOCKED because `custom` was a dropdown value no
published release knew, and one launch of an older release reset it silently and persisted the reset. FkRecipes
`6518971` withdrew the value and made the text the switch. The claim that buys is an IDENTITY, and it is measured
as one.

A file with a non-default value in every one of the six startup settings and the map bool
(`splitter-express`, `3 steel-plate, 1 fast-transport-belt`, `logistics-3`,
`1 automation-science-pack, 1 logistic-science-pack`, 250, 45, bool true), sha256
`093fe75ba729999ee55fa4638b56d2fabc259ce0d0341f2888a9ff70e0493b11`, taken head to an older release and back,
eleven steps through rel-0.2.2, rel-0.2.1, the AUTHENTIC 0.2.2 portal artifact, rel-0.2.0 and rel-0.1.0:

**`093fe75b...` at every one of the eleven steps. Not one byte moves anywhere in the trip.** Head's two
informational `fkrecipes:` lines are identical before and after. The four names no older release declares survive
verbatim through a release that declares nothing at all. Both dropdowns are legal on every settings-bearing
release, so nothing is reset; 0.2.1 and 0.2.2 do not merely store `logistics-3`, they PRICE the research from it.

That is finding 3 closed, and it is closed by construction rather than by care: there is no longer any value in
either dropdown that an older release can fail to recognise.

### 3. CLEAN, up from BLOCKED. Owner FkRecipes, closed. Scope C

See 24. The residual FkRecipes' own ownership table names is real and is not this consumer's: withdrawing a
dropdown value IS a design of the library's that reaches the engine's silent reset, for any consumer who already
shipped that value publicly. BBB never did, and `git grep custom` over the 0.2.1, 0.2.2, 0.3.1 and 0.3.2 trees
finds no `string-mod-setting` entry for it at any tag. A file carrying `custom` is measured in the appendix and is
case H.

### 4. CLEAN, up from MISLED. Owner FkRecipes, closed. Scope B

The state the screen could not render no longer exists to be rendered: nothing is ever edited and ignored, the two
`is edited, but ... is not on custom` lines are gone with their producers, and every field that is not at its
default is in force. Measured: a preset with a non-default text beside it emits the TEXT, and logs
`fkrecipes: bbb-balancer-part takes its ingredients from better-belt-balancer-recipe-ingredients: 2 iron-plate, 1 splitter; the bbb-recipe-cost choice cheap is set aside`.
The one residual, a refused text that the screen goes on showing while the dropdown decides, is disclosed on the
prototype exactly as decision C promised, and the sentence it discloses it with is finding 19.

### 5. CLEAN, up from AWKWARD. Owner engine, disclosed. Scope B

The wrap and its left-margin continuation are the engine's. Every composed shape that renders a typeable list now
carries `A list too long for one line continues on the next; the continuation is part of the same list.`, measured
present on both text settings and on the ingredient dropdown and measured ABSENT on the cost dropdown, which
renders no list to copy. That is the disclosure scope K asks for, in the place a player looks.

The label truncation half is **gone as a hazard and NOT REACHABLE as a measurement**. This mod's nine option
labels moved from 35 to 65 characters down to 5 to 19 (`Default`, `Cheap`, `Fast belts`, `Express belts`,
`Splitter`, `Express splitter`, `Logistics (default)`, `Logistics 2`, `Logistics 3`), so the closed dropdown has
far less to cut; whether it still cuts anything is a pixel measurement and no client was reached.

### 6, 7, 1, 2, 11, 12. CLEAN, unchanged, re-measured

- **1**, the ladder that landed twice on one name. `fkrecipes: bbb-balancer-part: iron-plate is in the list twice after the fallbacks, so the amounts are added: 4 plus 2 is 6`, emitting `6 iron-plate, 2 iron-gear-wheel`, exit 0, and identical normalised dumps at two different absolute paths.
- **2**, no text a player types refuses the load. **70 of 70 stored texts and numbers exit 0 with a dump**, across 22 distinct refusal shapes: a typo, an unknown item, a fluid, a display name, a decimal comma, four kinds of invisible character plus U+00A0, a NUL on both sides of the 22-byte cliff, the recipe's own product, rich text of two kinds, a duplicate, five reserved-word spellings, the empty field, and the list at 1999, 2000 and 2001 characters. Exactly one `fkrecipes: ERROR: ` line per bad field, never two; two bad fields give two.
- **6**, the promise is scoped. The rewritten `bbb-recipe-cost` description states the preset policy and the typed policy as opposites and says which is which, and both halves were re-measured under the remover.
- **7**, the prerequisite no longer moves. `bbb-tech-cost = logistics` emits `prerequisites ["logistics"]`, and every tier emits its own.
- **11**, the build's own claim. `test/check-release-arm.sh` exit 0, read directly: `release-arm: ok -- release/2.0 0.2.3 carries no line 'master' never had.` Recorded beside it, not graded: the 2.0 arm on the portal is version 0.2.3 cut from trunk 0.3.2, so a 2.0 player does not receive the customizer at all, and loses nothing by not receiving it.
- **12**, every stored value migrates. **All nine dropdown values, on rel-0.2.1, rel-0.2.2 and the authentic 0.2.2 zip, then handed to head: 36 runs, all exit 0, zero `fkrecipes:` lines on either side, and head's emitted `bbb-balancer-part` and `bbb-balancer` byte-identical to the old release's for every one of the nine.** The self-product line still does not fire for this mod's own fixture and fires only when the player types the product.

### 8. CLEAN, up from AWKWARD. Owner engine, disclosed twice. Scope D

The destruction is unchanged, larger than any round has measured it, and now disclosed in both channels. Nine of
21 save loads destroyed something; **the worst case is 620 items**, the whole input inventory of an
assembling-machine-3 stocked from a six-item list, with `items-on-ground count=0` on a GLOBAL sweep of every
surface and every inventory-bearing entity in the world. Every clause of the changelog's Info row was tested
against a measurement:

| clause | verdict |
|---|---|
| edits the one recipe rather than adding a second | MEASURED TRUE |
| the machine loses input-slot items the new list does not use | MEASURED TRUE, 620 worst case, the losses exactly the intersection complement |
| destroyed rather than dropped on the ground | MEASURED TRUE, global sweep, nothing anywhere |
| with nothing said in the log | MEASURED TRUE for an option change; the typo path's ERROR line carries a generic sentence naming no machine, item or count |
| what was finished stays in the output slot | MEASURED TRUE, `3 bbb-balancer-part` and `products_finished=7` in all 21 loads |
| chests and belts keep everything | MEASURED TRUE in all 21 |
| a typo does the same thing | MEASURED TRUE, 70, 70 and 190 items in three runs |
| correcting the typo is a second one | MEASURED TRUE, 120 more items on the way back |

The second assessment's remedy (h) was taken and it is what moves this to CLEAN: the changelog now discloses the
NEW way to change the recipe that the fallback created. And the recipe's own tooltip carries the destruction
sentence on a fallback load, scoped to a moved ingredient list and to nothing else.

### 9. CLEAN, up from AWKWARD. Owner FkRecipes, closed. Scope B

The 2000-character boundary is unmoved and exact (1999 and 2000 load, 2001 falls back), and it is on the screen.
The two numeric ranges are on the screen now too, in words the engine resolves to
`A whole number from 0 to 1000000. While it is 0 the option chosen above decides.` and
`A whole number from 0 to 3600. While it is 0 the option chosen above decides.`, which is what the second
assessment's open half asked for. What happens to a value outside the range is still not stated, and the widget's
declared minimum and maximum mean only a hand-edited file reaches it, which is case H.

One imprecision recorded rather than graded: on the seconds row "the option chosen above" is three rows up, with
two text fields between it and the dropdown it names.

### 10. CLEAN in scope. Owner engine for the cut. Scope B and K

Every invisible character is named by code point in the message and nothing is deleted, re-measured on U+200B,
U+FEFF, U+00AD, U+E0001 and U+0000. All five reserved-word spellings fold. The NUL cliff is exactly where the
second assessment put it, re-measured at 21, 22, 23 and 24 stored bytes, and it is the engine's: below it the
guest is handed a shortened value, the load succeeds, the cut is persisted and nothing anywhere says so.
FkRecipes' ownership table is right that the remedy is unimplementable from inside a guest. Whether a NUL can
reach the field by paste at all is NOT REACHABLE without a client and is owed.

One nit, log-only and therefore not a grade: U+00A0 is inside the language's whitespace set, so
`2 iron-<U+00A0>plate` is not named by code point but quoted raw, and the message reads
`("2 iron- plate"): no item or fluid is named "iron- plate"`, which is the one remaining case where a player
cannot tell from the message what byte they pasted.

### 15. CLEAN on the mechanism, NOT REACHABLE on the render. Owner FkRecipes. Scope B

The mechanism is measured here rather than carried, in one run, on this binary:

```
ALT-BARE    technology-name.logistics-2   ->  Unknown key: "technology-name.logistics-2"
ALT-WRAPPED technology-name.logistics-2   ->  logistics-2
ALT-BARE    technology-name.logistics     ->  Logistics
ALT-WRAPPED technology-name.logistics     ->  Logistics
```

The alternatives form turns the marker into the raw name and is a no-op on a key that exists, so the whole
tooltip no longer dies for one undefined key. Neither `technology-name.logistics-2` nor `-logistics-3` is defined
by any shipped en cfg, re-grepped here, and the reason is measured too: base composes those names
(`{"", {"technology-name.logistics"}, " 2"}`) rather than keying them. So `bbb-tech-cost`'s three lines resolve to

```
Logistics (default): cost of Logistics
Logistics 2: cost of logistics-2
Logistics 3: cost of logistics-3
```

Two of three name the technology in the game's data rather than in the player's language, which is cosmetic and
is the price of not defining a key in a flat shared namespace, exactly as FkRecipes' decision D argues. **The
repair that costs nothing is in reach and is worth naming**: the technology prototype's own `localised_name` is a
data-stage field, and copying it would render `Logistics 2` where the raw key renders `logistics-2`.

**WHAT IS NOT REACHABLE, and it is the same thing the second assessment measured with a camera:** whether the row
has an info icon and whether the tooltip draws. That was finding 15's whole substance and only a client can see
it. The resolved text above is the strongest available substitute and it is not a substitute for the icon.

### 16. CLEAN in scope. Owner engine. Scope K, residual H

Both research numbers are `int-setting` now, and the engine's own range check rejects a NaN for an int:
`Error IntSetting.cpp:71: ... Value can't be NaN.`, exit 0, reset, load continues. **No name this mod declares can
hold a NaN any more**, which is scope K's path-removal condition met.

The abort is still live through a name NOBODY declares, and that is case H: the engine does not type-check an
undeclared key and keeps it verbatim, so a NaN in a double node under a foreign name passes every check, the data
stage completes (`Prototype list checksum: 3197544275` at 0.976), and the write-back self-check aborts at 2.483
with two byte-identical hex blocks and `Received 6`. Neither this mod nor the library can see or reach that key.
This is the reproduction FkLua's finding 1 should carry, because it needs no mod with a double setting at all.

### 18. CLEAN, up from MISLED. Owner FkRecipes, closed. Scope B

`none`, `NONE` and `None` all emit `"ingredients": {}`, exit 0, one informational line. The field's own resolved
tooltip now says `The word none empties the list, so the recipe costs nothing to craft.` and the packs field's
deliberately does not, because a packs list refuses the word
(`fkrecipes: ERROR: better-belt-balancer-tech-packs: research takes at least one science pack`). The changelog
says it too, including the recycling consequence. Both the kind that takes the word and the kind that refuses it
were measured.

### 17. Out of scope (J), recorded in the appendix

## Appendix: the out-of-scope measurements, H to L

Nothing here carries a blocking grade.

### H. Hand-edited and corrupted settings files

21 files built by hand, each one run. The by-hand encoder reproduced `fklua modsettings write`'s own bytes for
the head defaults before any of them, so the layout is right.

| file | engine, before any stage | the mod | exit | the file afterwards |
|---|---|---|---|---|
| NaN in a double node under either int setting | `Value can't be NaN.` | nothing | 0 | replaced with defaults |
| **NaN under a FOREIGN, undeclared name** | **silence** | nothing; the data stage completed | **1**, abort | **unchanged**, so it repeats for ever |
| +Inf, -Inf | `Value (inf) outside of range.` | nothing | 0 | defaults |
| NUL at 21 and 22 stored bytes | silence | the CUT prefix, no ERROR line | 0 | **rewritten with the cut** |
| NUL at 23 and 24 | silence | one ERROR line naming U+0000 | 0 | untouched |
| invalid UTF-8 | silence | `contains characters that are not text; retype the list` | 0 | untouched, still not UTF-8 |
| wrong type, string and int | `Value must be a string` / `a integer` | nothing | 0 | defaults |
| out-of-range int | **silence** | nothing | 0 | defaults |
| truncated at half, two bytes short, empty | `Not enough data remaining` / `Couldn't read from input file.` | nothing | 0 | replaced with defaults |
| an unknown setting NAME beside valid ones | silence | nothing | 0 | **byte-identical**, the unknown key survives |
| a dropdown value outside `allowed_values` | **silence** | nothing | 0 | defaults |

**Two things extend FkLua's own record and should reach it.** First, the NaN abort above, reachable with no mod
declaring a double at all. Second, **the 22-byte NUL cut applies to a DROPDOWN value too, and it runs BEFORE
`allowed_values` is checked**: `bbb-recipe-cost` stored as `cheap<NUL>ZZZZZ` is cut to `cheap`, passes the list,
builds the cheap recipe and persists the cut, while the same trick over the cliff is reset by range. Finding 4 in
`engine-findings.md` is written about string settings only.

`readDropdown`'s unescaped message was hunted through four hand-built files and **could not be reached**. The
engine gates the stored value against `allowed_values` before any stage runs, in every shape tried.

**`custom`, the withdrawn value.** A file carrying it in both dropdowns loads with the engine silently coercing
both to their declared defaults and persisting that, zero `fkrecipes:` lines when the texts are also at default.
**No public release can produce this file**; it existed only between the second assessment and the withdrawal, in
a tree that was never published. What a player on such an unpublished head loses is exactly the two dropdown
positions, and they lose them invisibly, because the text still wins until they set it back to `default`.

### I. A pack that breaks the base game

Re-measured on the stock install. The second assessment's one-file remover still kills Space Age before this mod
can be judged:

```
Error Util.cpp:81: Failed to load mod "space-age": __space-age__/base-data-updates.lua:241: attempt to index field 'transport-belt' (a nil value)
```

and this mod's own data stage had already finished with its merge line written when that happened.
**The second assessment's remedy (i) was taken**: `test/check-datastage.py` at head deletes at `data.lua` and
sweeps at `data-final-fixes.lua`, and run on the stock install it is a pack Space Age survives, exit 0, `swept 12
recipe(s)`, the merged ladder emitted.

### J. A mod that loads after this one

Finding 17 reproduces exactly, in both a `data.lua` and a `data-final-fixes.lua` killer. With the killer first,
this mod emits normally. With the killer second: **zero `fkrecipes:` lines, exit 0, a dump written, the recipe
absent, `bbb-balancer` present with `effects: {}`, and the item and the entity still there.** A researchable
technology that unlocks nothing and a part nothing can craft, with no signal at load of any kind. There is no
final-fixes verify in this mod today (`grep -rn "verify" guest/go/data/*.go` returns nothing), so the threat
model's sentence "the final-fixes verify gives the one signal that can exist" describes a signal that is not
implemented. Out of scope by J and recorded.

A caution on the instrument: `Prototype list checksum` is `3197544275` for both the vanilla and the cheap preset
while their recipes differ, and it moves for `none` and for the two killer orders. It is a bell, not evidence
about what moved.

### K. Engine behaviours no mod can change

The seven `../FkLua/agents/engine-findings.md` entries that touch this assessment, by that file's numbering:
1, the NaN abort; 2, the error dialog's missing route to Mod Settings; 3, an undefined locale key deleting a whole
setting tooltip; 4, the NUL cut under 22 bytes; 5, the silent reset of an out-of-list or out-of-range stored
value; 7, assembler input stacks destroyed on a recipe change; 9, invalid UTF-8 reaching the mod untouched. Plus
three by-design entries: a later mod rewriting an earlier mod's prototypes, the settings screen's flat list, and
the settings stage seeing an empty `data.raw`.

| K item | a path of ours still into it | disclosed where a player looks |
|---|---|---|
| assembler inputs destroyed | yes, unavoidably: every cost value is a recipe change | **yes, twice**: the changelog and the recipe's own tooltip on a fallback |
| silent reset of an invalid value | not through the screen; only a hand-edited or rolled-back file | **no**, and finding 23 is the case where it costs everything |
| the error dialog's missing route | yes, through finding 13 | partly: "or in the load error if the load stops anyway" |
| the flat settings list | yes, four interacting rows | **yes**, each row states its own relation to the row above |
| the NUL cut under 22 bytes | NOT REACHABLE whether the field can take a NUL | no, and below the cliff there is nothing to disclose because the mod never learns |
| the NaN abort | **no**, for every name this mod declares | no dialog exists to disclose in; path removal is the condition and it is met |

**Seven silent rewrites were collected across the 70-run corpus**, every one the engine's, every one invisible to
the mod: an empty or whitespace-only text substituted with `default` and persisted (three cases, and benign, since
clearing the field is what `default` means), an out-of-range int reset to 0 and persisted (three cases), and the
NUL cut. Only the last destroys text the player meant to keep. `auto_trim` was measured to have exactly one
observable effect on a stored value, that substitution; it does not trim before the guest sees it, proved with a
2001-character list that trims to 2000 and is still refused as 2001.

### L. A real 2.1 binary

**There is none on this machine**, established four ways (`ls /Applications`, the Steam tree, `mdfind`, and a
`find` over six roots), all returning the one 2.0.77 bundle. NOT REACHABLE, and therefore ungraded: whether a 2.1
player sees the Map-tab bool at all (the code gate is `guest/go/data/settings.go:158` over
`engine.Is2_0(fkdata.ModVersion("base"))`, a reading of the source and not a measurement); what a 2.1 engine does
with a stored `runtime-global` value for a setting no mod declares; whether the settings and data stages differ at
all; whether the 200-byte element ceiling and the 20-parameter and 20-level limits are the same numbers; and every
row of appendix H re-asked on 2.1.

## The release-to-head matrix

Every release rebuilt from its own commit plus the authentic 0.2.2 zip; settings read out of the engine's own
`mod-settings-dump.json`, prototypes out of `data-raw-dump.json`.

| release | pin | runs natively here | what it declares | its values | onto head |
|---|---|---|---|---|---|
| 0.1.0 | 2.0 | yes | nothing; **writes no `mod-settings.dat` at all** | | nothing to migrate; head writes the 432-byte defaults file |
| 0.2.0 | 2.0 | yes | the runtime-global bool only | true, false | survives by name; head appends the six startup names |
| 0.2.1 | 2.0 | yes | 2 dropdowns + the bool | 6 recipe, 3 tech | all 9 survive, prototypes identical on head |
| 0.2.2 authentic | 2.0 | yes | identical to 0.2.1 | 6 recipe, 3 tech | all 9 survive, prototypes identical on head |
| 0.3.0 | 2.1 | no | not reachable | | nothing to migrate |
| 0.3.1 | 2.1 | no | 2 dropdowns by pin | 6 recipe, 3 tech | covered by equivalence with 0.2.1 |
| 0.3.2 | 2.1 | no | 2 dropdowns by pin | 6 recipe, 3 tech | covered by equivalence with 0.2.2 |
| head 0.3.3 | 2.1 | no; the recut and the restamp do | 2 dropdowns + 2 texts + 2 ints + the bool | 6 recipe, 3 tech | |

The equivalence is stated rather than waved at: the `[mod-setting-name]`, `[mod-setting-description]` and
`[string-mod-setting]` blocks are byte-identical at the 0.2.1, 0.2.2, 0.3.1 and 0.3.2 tags, so a 0.3.x player's
file has the same shape and the same value vocabulary as the 0.2.x files that were measured. The 2.1 refusal text
is identical for all four 2.1 packages: `Incompatible Factorio version (current: 2.0, required: 2.1)`.

Head's rewrite of a published player's file appends exactly four names at their declared defaults
(`better-belt-balancer-recipe-ingredients` and `-tech-packs` at `default`, `-tech-count` and `-tech-seconds` at
`0`) and moves no stored value. **`custom` no longer exists in either dropdown, so the whole of the second
assessment's finding 3 exposure is gone.**

**THE LABELS MOVED, AND THAT IS THE ONE THING A MIGRATING PLAYER SEES CHANGE.** All nine
`[string-mod-setting]` entries were rewritten (for example `Default: 4 iron plates, 2 gears, 2 transport belts`
becomes `Default`, and `Logistics 2, alongside fast transport belts` becomes `Logistics 2`), and the two
`[mod-setting-description]` entries a 0.2.x player knew grew from 182 to 443 and from 295 to 563 characters.
Nothing a player had SELECTED changed, and the two row labels they knew are untouched. The changelog discloses it
in the player's own words, naming all nine new labels verbatim and saying the stored choice is kept, which is what
holds scope A's "the screen shows what it showed" at CLEAN. The residual cost, taken knowingly: the settings
screen no longer names a single ingredient in the words the game shows on screen.

## The save matrix

Nine saves built, 21 loads, **every one exit 0**. Research, entities, belts, chests, crafting progress and the
output slot survive every cell in both directions; the research and entity block is byte-identical across all 21.
The only thing that ever moves is an assembling machine's input inventory, and only when the recipe changed.
**Across all 21 loads: zero `Error`, zero `Migration` and zero `Applying` lines**, and the only engine commentary
of any kind is a base-game Gleba collision-mask warning present at build time too.

| direction | settings file | outcome | destroyed |
|---|---|---|---|
| 0.2.2 save to head | the same file | every stored value honoured, four names appended at defaults, zero `fkrecipes:` lines | 0 |
| 0.2.2 save to head | **no file** | loads on the declared defaults whatever the save was made with; the engine says nothing | 0 |
| 0.2.2 save to head | a file that differs | the file wins outright, the engine says nothing | 240 |
| head save to 0.2.2 | the same file | world whole, **the four settings 0.2.2 does not declare kept verbatim**, both dropdowns legal and unmoved | 70 |
| head save to head | the text edited to another valid list | loads, the recipe changes | 190 |
| head save to head | the text edited to a typo | loads, ONE ERROR line, the recipe becomes THE DROPDOWN'S PRESET | 70 to 190 |
| head save to head | the text edited back to `default` | loads, the dropdown decides | 70 |
| head save to head | a preset changed, six-item list in the machine | loads, nothing said at all | **620** |

The downgrade row is the one the withdrawal changed: a head save on authentic 0.2.2 keeps
`"better-belt-balancer-recipe-ingredients": "2 iron-plate, 1 splitter"` and `250` and `45` verbatim, and both
dropdowns keep their values, where the second assessment measured both dropdowns silently reset.

**The save's own record**, read by exact byte substring out of `level-init.dat`: the mod list as a
length-prefixed name, a 3-byte version and a 4-byte value which is the `Checksum of better-belt-balancer` the
same run logged (`00 03 03` and `1841694694` for head, `00 02 02` and `2970780231` for 0.2.2), then the startup
settings as a property tree. **A head save records six startup values; a 0.2.2 save records two**, and
`bbb-multi-edge-parts` appears only in the prototype-name table and never in the startup value tree. So the four
settings a 0.2.2 save has never heard of have no recorded value for a sync prompt to offer.

**The runtime-global bool is the one value the save wins, in both directions**, re-measured: a save built `true`
loaded under a file saying `false` runs `true` and the file is not updated from the map, and the reverse control
runs `false` under a file saying `true`. After loading an existing save the file and the running game can
disagree indefinitely with nothing anywhere saying which one is in force.

## Recovery, demonstrated

Three refused values, each with five other settings holding non-default values beside them, each recovered
headlessly with ONE edit and nothing else lost: an ingredient typo under `bbb-recipe-cost = cheap`, a pack typo,
and a 2001-character list. In every case the stored text came back out of the file byte for byte after the
failed-to-use load, the other five settings were untouched, one write corrected the field, and the re-run carried
**zero `fkrecipes: ERROR: ` lines**. The engine rewrites the file on every successful load and writes the player's
refused text back verbatim, proved by touching the file to a 2020 mtime and watching the engine move it while the
sha256 stayed put.

**That is the property the whole fallback design exists to buy, and it holds.** Its one exception is the NUL under
22 stored bytes, where the engine had already shortened what the player typed.

## Determinism and the server

Four settings files, each run at two different absolute user-directory paths: four mutually distinct normalised
dump hashes, each reproducing exactly. The same with `locale=de` against `locale=auto` and with `mod-list.json`
reversed: six more identical hashes. **Determinism after a merge** (`6f44b3b7ea35e31c...` at two paths, the merged
entry keeping the first occurrence's position) and **after a dropped science pack**
(`9c7e3ea84d94ffe3...` at two paths) both hold.

**A headless server starts and stays up on a bad text, three ways bad**: a bad ingredient list, a bad pack list,
and both. Socket opened, `Hosting game at IP ADDR:({0.0.0.0:34197})`, the degradation logged one line per bad
field, twenty seconds up, clean shutdown, exit 0, and the bad text kept verbatim in the file the engine rewrote.

`Prototype list checksum` is `3197544275` for the declared default, for a valid typed list AND for a fallen-back
list alike, while their normalised dumps are three distinct hashes. **A client on a fallen-back text and a server
on a working one hold different prototypes and the same prototype checksum.** What catches it instead is the
binary's `startup-mod-settings-crc` and the shipped
`[gui-mod-startup-settings-mismatch] Your mod startup settings do not match with those of the server you are
connecting to.`, which compares stored VALUES. **A real two-party join is NOT REACHABLE here and is not graded**,
exactly as the threat model directs.

## The client, and what stands in for it

**NOT REACHABLE this round.** The game was launched under a private user directory with a settings file installed
by hand (both texts refused, both dropdowns on a non-default choice), it reached the main menu, and no image could
be obtained: `app_screenshot` on Factorio's only window returns `Failed to start stream due to audio/video capture
failure`, and two requests for full-screen control went unanswered. Control was released immediately
(`app_release`, then `release_full_control`), the client was killed, and the real user directory's log and
`mod-settings.dat` were byte-unchanged before and after.

What was built instead resolves every composed string through the engine's own locale resolver at the control
stage. It settled four things a dump cannot:

- **the six settings' tooltips as the player will read them**, 970, 902, 733, 768, 135 and 144 bytes resolved;
- **the alternatives form**, measured against its own bare control in the same run;
- **the chunk seam is invisible**: the recipe's three-element description resolves to a 247-byte string that is
  `cmp`-identical to the concatenation of the dump's elements, with one space at the join and nothing else;
- **the note displaces the technology's own description**, which is finding 21 and which nothing else could have
  found.

**THE CHECKLIST STILL OWED TO A CLIENT**, and it is longer than the second assessment left it:

1. Does `bbb-tech-cost`'s row have an info icon and a tooltip now that the alternatives form is in? That is
   finding 15's whole substance and the only instrument for it is a screen.
2. Where does the 247-byte fallback note wrap in a real crafting tooltip, and does the chunk boundary at
   `empties an / assembling` show?
3. Where does the 138-byte technology note wrap in the tech tree, and does the mod's own missing description read
   as missing?
4. Do the nine shortened option labels still truncate in the closed dropdown?
5. Does the 970-byte `bbb-recipe-cost` tooltip render whole and unclipped at its new size?
6. The startup-settings sync prompt: does it appear for a 0.2.2 save on head, does its default choice restore the
   save's two dropdown values, and what does it do with the four settings the save has no record of?
7. Can a NUL, or any invisible character, be pasted into the text field at all?
8. The error dialog on finding 13's refusal: re-walk `Manage mods` and `Restart`.

## What the design must change, and where

**FkRecipes.** (a) Finding 19: `fallbackNote`, `playerFallback` and `textFallbackLine` name the mod's own default
where the dropdown's current preset is what applies; the honest generic is the one the switch line already uses,
"the option chosen above", and `docs/ingredient-list.md:59` and `docs/usage.md:291` and `:61` need the same
correction. (b) Finding 22: `agents/threat-model.md:21`'s scope F sentence is false on three clauses, and
correcting it is one sentence that also settles what a ladder owes; while it moves, `customizer-design.md:56`,
`docs/usage.md:312` and both halves' matching source comments should stop citing a disclosure the library does not
compose. (c) Finding 20: `go/data.go:996-1012` and its Rust twin are the one degradation branch of three with no
`noteOn` call, and it is the one that removes a prerequisite. (d) Finding 21: a note that replaces an author's
locale-file description needs either a rule that can see one or a line in `docs/usage.md` telling an author to
declare `Description` in the plan. (e) Finding 15's cosmetic residual: a technology's own `localised_name` is a
data-stage field and would render `Logistics 2` where the raw key renders `logistics-2`.

**BBB.** (f) Finding 13: `FallbackUnit()`'s single pack is what walks into the packless refusal, and one
`Fallbacks` rung on it survives the measured pack; the comment at `guest/go/tune/plan.go:261` moves in the same
commit. (g) Finding 23: the changelog needs an Info row saying that a settings file written by a newer game
version is discarded whole on a rollback of the GAME, in the same shape as the one correctly deleted when `custom`
was withdrawn. (h) Finding 13 again, on the prose side: `README.md:70` says "missing from your game" where the
measured case is a demotion, and the settings tooltip says nothing at all.

**FkLua.** (i) `agents/engine-findings.md` does not record that an unreadable or newer-stamped settings file is
REPLACED with defaults rather than preserved, which is finding 23's mechanism. (j) Entry 4 is written about string
settings and the NUL cut applies to a dropdown value too, before `allowed_values` is checked. (k) Entry 1's
reproduction needs no mod declaring a double at all, and the one measured here is simpler than the one on file.

**Neither, and worth saying.** The silent reset of an out-of-list or out-of-range stored value, the silent
substitution of an emptied text field, the destruction of an assembler's input stock on a recipe change, the
replacement of an unreadable settings file with defaults, the NUL cut, the NaN abort through an undeclared name,
the prototype checksum's blindness to a fallen-back text, and the absence of conditional visibility on the
settings screen are all the engine's. What the design controls is exposure, wording, and what it chooses to
notice.

## What could not be reached

- **Any client render.** The whole checklist above. The capture path failed and full-screen control was
  unanswered; nothing was inferred from a screen that was not seen.
- **A real two-party multiplayer join.** No headless client here. The CRC comparison and the mismatch dialog are
  named from the binary and the shipped `core.cfg` and are not measured.
- **Anything on a real 2.1 engine.** Appendix L lists what that costs, item by item.
- **Authentic artifacts for six of the seven releases.** Only 0.2.2's zip is genuine and it was used for every
  release row that touches a save.
- **`readDropdown`'s unescaped message**, hunted through four hand-built files and unreachable in every shape.
- **The 65535 merge ceiling through this mod.** The largest merge any preset can produce is `4 plus 2 is 6`, four
  orders of magnitude away, and the language refuses a duplicate in a typed list.
- **Whether a NUL or any invisible character can be entered into the settings text field**, which is what decides
  whether the 22-byte cut has a path of ours into it at all.

This note is a new file. `CLAUDE.md`'s index of `agents/` gains one row for it in the same commit.
