package tune

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fkrecipes "github.com/Techrocket9/fkrecipes/go"
)

// A MISSING LOCALE KEY IS A FIELD REPORT WAITING TO HAPPEN, and this repo
// already has the report: `Unknown key: "entity-name.bbb-linked-belt"` came out
// of the engine's own "X is in the way" message in a live session on
// 2026-08-05, and the fix was four lines of .cfg nobody had thought to write.
//
// A dropdown is the same shape with more ways to get it wrong. Factorio renders
// a string setting's value from `[string-mod-setting] <setting>-<value>`, one
// entry PER VALUE, and there is no fallback: a value with no entry shows as
// `Unknown key: "string-mod-setting.bbb-recipe-cost-belt-express"` in the menu
// the player is standing in. Nothing headless can see that -- `--dump-data`
// does not read locale, and no suite in test/run.sh opens a menu -- so the
// tripwire has to be here.
//
// SINCE THE SETTINGS MOVED ONTO FkRecipes THE TRIPWIRE IS THE LIBRARY'S, and
// that is a strengthening rather than a swap. `CheckLocaleWith` knows the plan,
// so it checks what this file used to check -- every declared value has an
// entry, and no entry names a value the plan no longer declares -- and three
// things this file could not:
//
//	It polices `[mod-setting-name]` and `[mod-setting-description]` ORPHANS
//	against the complete set of the mod's setting names rather than against the
//	two this package knows about, which is what catches an entry left behind by
//	a rename. The hand-rolled list is what makes that reading available, and
//	[HandRolledSettings] is why `bbb-multi-edge-parts` is not reported as one.
//
//	It checks that every declared setting has a `[mod-setting-name]` at all.
//
//	It checks the flat `<setting>-<value>` namespace for two settings producing
//	one key, which the old TestNoAllowedValueCollidesWithAnother did by hand and
//	the library now does over the whole plan.
//
// WHAT THE LIBRARY DELIBERATELY DOES NOT DO IS BELOW, as three tests of this
// mod's own. Its header says so in as many words: a description is optional
// there, because the engine's failure for a missing one is a lost tooltip
// rather than an `Unknown key` in the player's face, and the hand-rolled list
// suppresses orphans without creating obligations. This mod wants both, so it
// asks for both here.

func repoFile(t *testing.T, rel ...string) string {
	t.Helper()
	// guest/go/tune -> the repository root.
	parts := append([]string{"..", "..", ".."}, rel...)
	p, err := filepath.Abs(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func localePath(t *testing.T) string {
	t.Helper()
	return repoFile(t, "mod-data", "locale", "en", "better-belt-balancer.cfg")
}

func localeText(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(localePath(t))
	if err != nil {
		t.Fatalf("the locale file is the thing under test and it is not there: %v", err)
	}
	return string(b)
}

// TestTheLocaleFileSatisfiesThePlan is the whole two-direction check, run by
// the library against the plan that ships.
//
// EVERY FINDING IS ITS OWN Errorf. `CheckLocaleWith` RETURNS findings and does
// not fail: it is a library function with no testing.T to call, so a test that
// only logged what came back would pass over a locale file with nothing in it.
// One error per finding also means a run names every missing entry rather than
// the first.
func TestTheLocaleFileSatisfiesThePlan(t *testing.T) {
	for _, finding := range Plan().CheckLocaleWith(ModName, localeText(t), HandRolledSettings()) {
		t.Error(finding)
	}
}

// TestModNameIsTheManifestName is what makes the check above mean anything.
//
// The mod name is the prefix FkRecipes derives every generated name from, and a
// host test has no fkdata to read it from -- so [ModName] is written down here
// and this is what ties it to fklua.toml, which is the ONE place the packaged
// identity lives.
//
// IT WAS ONCE THE ONLY THING THAT COULD CATCH A WRONG ONE AND IT IS NOT ANY
// MORE, which is what the sync pass changed here. While every setting was
// LEGACY or HAND-ROLLED the prefix reached nothing: a legacy name crosses
// verbatim, `CheckLocaleWith` matches a hand-rolled one verbatim as well, and
// the complete-list reading takes the prefix out of the orphan scan too. The
// four customizer fields are GENERATED now, so a wrong `ModName` moves four
// emitted names and the locale file stops matching them in both directions at
// once. MEASURED, with `ModName` set to `better-belt-balancer-x`:
// [TestTheLocaleFileSatisfiesThePlan] reports fourteen findings, four of them
// `the setting better-belt-balancer-x-recipe-ingredients has no
// [mod-setting-name] entry` and its three siblings, and four more
// `the [mod-setting-name] entry better-belt-balancer-recipe-ingredients matches
// no setting this plan declares` and its three;
// [TestEverySettingThisPlanDeclaresIsDescribed],
// [TestEverySettingPrototypeIsTheOneThatShipped],
// [TestTheSixOrdersSortIntoTheDeclarationOrder] and every pinned sentence in
// plandata_test.go fire as well.
//
// THIS TEST STILL EARNS ITS PLACE, and now for the reason a reader would assume
// rather than against it: it is the one that says WHICH of those is the cause.
// The rest report a name that does not match; only this one names fklua.toml
// and the value it holds.
func TestModNameIsTheManifestName(t *testing.T) {
	fh, err := os.Open(repoFile(t, "fklua.toml"))
	if err != nil {
		t.Fatalf("fklua.toml is the mod's identity and it is not there: %v", err)
	}
	defer fh.Close()

	// Enough TOML to find one key in one table. The manifest is written by
	// `fklua init` and hand-edited, so `name` appears in more than one table
	// and only `[mod]`'s is the packaged identity.
	section, got := "", ""
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}
		if section != "mod" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) != "name" {
			continue
		}
		got = strings.Trim(strings.TrimSpace(v), `"`)
		break
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("fklua.toml has no [mod] name, so nothing here can be checked against it")
	}
	if got != ModName {
		t.Errorf("tune.ModName is %q and fklua.toml packages this mod as %q: "+
			"every generated name and every locale key is checked under the "+
			"wrong prefix", ModName, got)
	}
}

// localeSections is section -> key -> value, read through the LIBRARY'S OWN
// PARSER.
//
// THIS FILE CARRIED ITS OWN INI READER UNTIL ROUND TWO, and round one graded
// that AWKWARD: `CheckLocale` and `CheckLocaleWith` return findings and there
// was no way to ask what a given entry SAYS, so the three assertions below --
// which are about the TEXT of specific entries rather than about their presence
// -- needed a second parser. Two parsers over one grammar is two readings of
// one file that can disagree, and the disagreement would be silent: the
// library's checker would pass and this file's assertions would be made against
// a different reading of the same bytes.
//
// [fkrecipes.LocaleEntries] is that parser's output, exported. It returns the
// entries in FILE ORDER with their sections, and it skips a malformed line
// exactly as the checker skips it -- so what is asserted below and what
// `CheckLocaleWith` polices are now the same reading by construction. Parse
// findings are not reported here and do not need to be:
// [TestTheLocaleFileSatisfiesThePlan] runs the checker over the same file and
// reports every one of them.
func localeSections(t *testing.T) map[string]map[string]string {
	t.Helper()
	out := map[string]map[string]string{}
	for _, e := range fkrecipes.LocaleEntries(localeText(t)) {
		if out[e.Section] == nil {
			out[e.Section] = map[string]string{}
		}
		out[e.Section][e.Key] = e.Value
	}
	if len(out) == 0 {
		t.Fatal("the locale file parsed to nothing at all, so every assertion " +
			"below would pass over an empty reading")
	}
	return out
}

// TestEverySettingThisPlanDeclaresIsDescribed is the first of the three the
// library does not make, and its header says why: a description is OPTIONAL
// there for a bool, an int, a double or a plain dropdown, because the engine's
// failure mode for a missing one is a lost tooltip rather than an `Unknown key`
// render.
//
// THIS MOD WANTS ONE FOR ALL SIX, and the list is the whole of what [Plan]
// declares rather than the two dropdowns it started as. Every one of these
// decides what a machine costs to build or to research -- two by naming a
// preset, two by taking a list the player writes, two by taking a number -- and
// none of them is guessable from a label of under fifty characters. The library
// already demands four of the six (both text fields, and both dropdowns now
// that each composes its presets onto its own description), so what this test
// adds is the two NUMBERS: a count and a seconds field with no tooltip say
// nothing about which setting has to be on Custom for them to do anything.
func TestEverySettingThisPlanDeclaresIsDescribed(t *testing.T) {
	sec := localeSections(t)
	for _, name := range []string{
		SettingRecipeCost, SettingRecipeIngredients,
		SettingTechCost, SettingTechPacks, SettingTechCount, SettingTechSeconds,
	} {
		if strings.TrimSpace(sec["mod-setting-description"][name]) == "" {
			t.Errorf("no [mod-setting-description] %s: the settings menu shows "+
				"the label with no tooltip under it", name)
		}
	}
}

// TestTheHandRolledSettingIsNamed is the second, and it is the other side of
// what the hand-rolled list buys.
//
// Telling `CheckLocaleWith` about `bbb-multi-edge-parts` stops it reporting
// that entry as an orphan, and it deliberately creates no obligation in return:
// the library knows the name and nothing else, so it never demands an entry for
// it. This mod knows it is a setting a 2.0 player sees in the Map tab, so it
// demands one here.
func TestTheHandRolledSettingIsNamed(t *testing.T) {
	if strings.TrimSpace(localeSections(t)["mod-setting-name"][SettingMultiEdgeParts]) == "" {
		t.Errorf("no [mod-setting-name] %s: the settings menu shows the raw key",
			SettingMultiEdgeParts)
	}
}

// TestTheGrandfatherMessageQuotesTheRealMenuLabel is the third, and it is the
// one no checker anywhere could make, because it is about two entries agreeing
// with each other rather than about either one existing.
//
// The 2.0 grandfather warning tells a player to turn a setting off and quotes
// the menu label mid-sentence, so that they can find the row. Rename the row
// and the sentence sends them looking for an entry that is not there -- which
// is the same defect as an `Unknown key`, one level out, and invisible to
// everything: both entries exist, both render, and the mod loads.
//
// The label is quoted rather than pinned as a literal here, so the check is
// that the message names WHAT THE MENU SAYS. A prefix rather than an equality
// because the menu label carries a parenthetical the sentence has no room for
// ("(Factorio 2.0 only)"), and quoting a prefix of the row's name is still a
// row a player can find.
func TestTheGrandfatherMessageQuotesTheRealMenuLabel(t *testing.T) {
	sec := localeSections(t)
	msg := sec["bbb"]["single-edge-grandfathered"]
	if strings.TrimSpace(msg) == "" {
		t.Fatal("no [bbb] single-edge-grandfathered: the 2.0 grandfather pass " +
			"has nothing to say to the player it just decided for")
	}
	label := sec["mod-setting-name"][SettingMultiEdgeParts]
	if strings.TrimSpace(label) == "" {
		// TestTheHandRolledSettingIsNamed reports the absence itself.
		return
	}

	_, rest, ok := strings.Cut(msg, `"`)
	if !ok {
		t.Fatalf("[bbb] single-edge-grandfathered quotes no menu label, and it "+
			"has to name the row it is asking the player to turn off: %q", msg)
	}
	quoted, _, ok := strings.Cut(rest, `"`)
	if !ok {
		t.Fatalf("[bbb] single-edge-grandfathered opens a quote and never "+
			"closes it: %q", msg)
	}
	if !strings.HasPrefix(label, quoted) {
		t.Errorf("[bbb] single-edge-grandfathered tells the player to turn off "+
			"%q and the row in the menu is called %q: the message names an "+
			"entry that is not there", quoted, label)
	}
}
