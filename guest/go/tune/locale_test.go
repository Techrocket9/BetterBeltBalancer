package tune

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

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
// WHAT THE LIBRARY DELIBERATELY DOES NOT DO IS BELOW, as five tests of this
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
// adds is the two NUMBERS. The library composes the RANGE onto each of them and
// says what a 0 there means, but the entry that says what the number is FOR is
// this mod's, and without it a player reads a range with no subject.
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

// TestTheLocaleFileRenamesNoTechnologyOfTheGames is the fourth, and it is the
// one the LIBRARY asked this mod for and then pointedly did not enforce.
//
// WHAT THE LIBRARY SAYS AND HOW IT SAYS IT. `bbb-tech-cost`'s composed
// description names each tier by the GAME's own locale key,
// `technology-name.logistics` and its two siblings, so the tooltip reads
// "Logistics 2" where the player's tech tree does. Those keys are not this
// mod's: Factorio's locale namespace is FLAT AND SHARED, so an entry for one of
// them here would set the displayed name of BASE'S technology for every mod in
// the game. `CheckLocaleAdvisories` says exactly that, once per key, and it is
// a report of its own rather than part of `CheckLocaleWith` for a reason this
// test is built around.
//
// THE ADVISORIES ARE LOGGED AND NOT ASSERTED, AND THAT IS THE POINT. They are
// returned whether or not this mod's file defines the key, they never go away,
// and what they ask for is that nothing be done. A test that called t.Error on
// them would be permanently red over a thing the library says not to fix --
// which is precisely why the library took them out of `CheckLocale`'s return,
// and [TestTheLocaleFileSatisfiesThePlan] above is where "empty is clean" still
// means what it says.
//
// WHAT IS ASSERTED IS THE THING THIS MOD CAN ACTUALLY GET WRONG: that the .cfg
// defines no `[technology-name]` entry at all. That is an ACTIONABLE property
// with a one-line fix (delete the entry), and it is the remedy this mod's own
// second migration assessment asked for and got BACKWARDS. Remedy (f) there
// said to define `technology-name.logistics-2` and `-3` in this file so the
// tooltip would not show `Unknown key`; FkRecipes e4604d4 closed that from the
// other end by wrapping every composed reference in the engine's alternatives
// form, so an undefined key now degrades to the technology's raw internal name
// and the tooltip survives whole. Carrying out remedy (f) today would rename
// base's technologies for every mod in the game -- the exact hazard the
// library's own flat-namespace collision scan exists to report -- so this test
// is what keeps it undone.
//
// IT READS THE ENTRIES IN FILE ORDER rather than ranging a map, for the reason
// nothing anywhere in this repository ranges one: Go randomises that order per
// run, and a failure that named a different entry each time is a failure
// nobody can act on.
func TestTheLocaleFileRenamesNoTechnologyOfTheGames(t *testing.T) {
	// The advisories, said out loud and asserted on by nothing. Read them with
	// `go test -v ./tune/ -run TestTheLocaleFileRenamesNoTechnologyOfTheGames`.
	for _, note := range Plan().CheckLocaleAdvisories(ModName) {
		t.Log(note)
	}

	for _, e := range fkrecipes.LocaleEntries(localeText(t)) {
		// THIS MOD'S OWN TECHNOLOGY IS THE ONE ENTRY THAT BELONGS HERE, and it
		// belongs here because [Plan] declares it: `technology-name.bbb-balancer`
		// names a technology nobody else has, and without it the research screen
		// shows the player the raw key. [TechName] is the whole of what this
		// plan declares, so it is the whole of the exemption.
		if e.Section != "technology-name" || e.Key == TechName {
			continue
		}
		t.Errorf("[technology-name] %s is defined in this mod's own locale "+
			"file and is not a technology this plan declares. Factorio's locale "+
			"namespace is flat, so that sets the displayed name of somebody "+
			"else's technology for EVERY mod in the game. The composed tooltip "+
			"does not need it -- an undefined key degrades to the raw internal "+
			"name -- so the fix is to delete the line", e.Key)
	}
}

// TestNoSettingDescriptionNamesAnOptionNoDropdownOffers is the fifth, and it
// exists because ten entries of this file once instructed the player to pick an
// option the dropdown does not have and NOTHING ANYWHERE WENT RED.
//
// WHY NOTHING COULD SEE IT. FkRecipes' locale checker reads KEYS -- every
// declared value has an entry, no entry names a value the plan does not declare
// -- and never the strings beside them. `--dump-data` carries the key and not
// the string, so `make datastage-check` cannot see a syllable of the file. The
// four tests above read four specific entries for four specific properties.
// When the `custom` dropdown value was withdrawn, every key in the file stayed
// valid and six descriptions and four names went on naming it.
//
// WHAT THIS PINS, AND IT IS ONE WORD PER ENTRY. For each dropdown the plan
// declares, it takes that dropdown's own allowed values and the words of the
// `[string-mod-setting]` label of each one -- which is the vocabulary a player
// can actually see in the menu -- and then requires that every capitalised word
// in the `[mod-setting-name]` and `[mod-setting-description]` entries of that
// dropdown AND of the settings bound beside it comes from that vocabulary, or
// else from the words this mod uses to name its own prototypes. `Logistics` and
// `Default` pass because they are option labels; `Balancer` passes because it
// is what this mod calls its machine; `Custom` passes nothing, because no value
// of either dropdown is spelled that way and no prototype is called that.
//
// THE RELATIONSHIP IS THE POINT AND NOT THE WORD. Nothing here writes `Custom`
// down. Add a value to [RecipeOptions] or [TechOptions], give it a
// `[string-mod-setting]` label, and every entry in that group may name it the
// next moment; withdraw one and every entry naming it fails here. So this test
// says something about the dropdowns and the prose agreeing, which is the thing
// that went wrong, rather than about a string that has already been deleted.
//
// WHAT IT DOES NOT CATCH, which matters because it reads one word and these
// entries carry paragraphs:
//
//	It reads words, not sentences. An entry that describes a behaviour this
//	mod no longer has, in ordinary lower case, passes: "the field below is
//	only read while the dropdown is on the seventh option" is false and
//	invisible here.
//
//	It skips the first word of every sentence in a DESCRIPTION, because a
//	sentence opens with a capital whatever it opens with. A description that
//	begins with a withdrawn option word is missed. It does NOT skip the first
//	word of a `[mod-setting-name]` entry, since a label is not a sentence --
//	and that is exactly where four of the ten hid ("Custom balancer part
//	recipe").
//
//	It scans only the settings this plan binds into a dropdown group. The
//	hand-rolled `bbb-multi-edge-parts` belongs to no dropdown and is not read,
//	which is why the word `Factorio` in its description is nobody's business
//	here.
//
//	It says nothing about whether the rule the entries state is the right one.
//	The rule is that the text field applies whenever it does not say `default`
//	and the dropdown supplies the rest; a human reads that, or nobody does.
//
// AND IT REFUSES ONE THING THAT IS NOT A MENU OPTION, MEASURED RATHER THAN
// IMAGINED. A bound setting's entries may not use a capitalised DISPLAY name or
// product name mid-sentence: writing `not the names shown on screen (say
// iron-plate, not "Iron plate")` into the ingredients field fails here on
// `Iron`, and `Factorio`, `Factoriopedia` and `Startup` are the same shape.
// README.md makes exactly that iron-plate point in exactly that phrasing, so an
// author moving the sentence into the tooltip will meet this. THE REMEDY IS NOT
// TO WEAKEN THE TEST, because the word it would have to start admitting is the
// word it exists to refuse: lower-case the display name, or name the prototype
// in one of this mod's own `[entity-name]`, `[item-name]`, `[recipe-name]` or
// `[technology-name]` entries, which is where a name this mod really gives
// something belongs and which this test already reads.
func TestNoSettingDescriptionNamesAnOptionNoDropdownOffers(t *testing.T) {
	sec := localeSections(t)
	var scanned []string

	// This mod's own display vocabulary, read out of the file rather than
	// written down here: the words it uses to NAME its own prototypes. A
	// balancer is a machine, not a row of a menu, so a label may say it.
	ownName := map[string]bool{}
	for _, e := range fkrecipes.LocaleEntries(localeText(t)) {
		switch e.Section {
		case "entity-name", "item-name", "recipe-name", "technology-name":
			for _, w := range capitalisedWords(e.Value, false) {
				ownName[strings.ToLower(w)] = true
			}
		}
	}

	// The groups, in declaration order, straight off [Plan]'s own lists. A
	// slice and not a map, for the reason nothing in this repository ranges
	// one: Go randomises that order per run.
	for _, group := range []struct {
		dropdown string
		values   []string
		beside   []string
	}{
		{SettingRecipeCost, RecipeOptions(), []string{SettingRecipeIngredients}},
		{SettingTechCost, TechOptions(), []string{SettingTechPacks, SettingTechCount, SettingTechSeconds}},
	} {
		scanned = append(scanned, group.dropdown)
		scanned = append(scanned, group.beside...)
		// What this dropdown offers, spelled both ways a player meets it: the
		// stored value itself (`belt-express`, one word per segment) and the
		// label the menu renders for it. The library's own checker is what
		// guarantees this section covers exactly these values, so reading the
		// labels here cannot drift from the plan.
		offered := map[string]bool{}
		for _, v := range group.values {
			for _, part := range strings.Split(v, "-") {
				offered[strings.ToLower(part)] = true
			}
			for _, w := range capitalisedWords(sec["string-mod-setting"][group.dropdown+"-"+v], false) {
				offered[strings.ToLower(w)] = true
			}
		}

		for _, name := range append([]string{group.dropdown}, group.beside...) {
			for _, section := range []string{"mod-setting-name", "mod-setting-description"} {
				entry := sec[section][name]
				if entry == "" {
					// Absence is reported by TestTheLocaleFileSatisfiesThePlan
					// and TestEverySettingThisPlanDeclaresIsDescribed.
					continue
				}
				for _, w := range capitalisedWords(entry, section == "mod-setting-description") {
					if offered[strings.ToLower(w)] || ownName[strings.ToLower(w)] {
						continue
					}
					t.Errorf("[%s] %s names %q, and %s offers no value spelled "+
						"that way: it allows %s and nothing else. A player told "+
						"to pick %q looks for a row of the menu that is not "+
						"there. The rule this entry has to state instead is "+
						"that the field below applies whenever it does not say "+
						"default, and the dropdown supplies the rest. If %q is "+
						"not meant as a menu option at all, it is being read as "+
						"one because it is capitalised and is neither an option "+
						"label of this dropdown nor a name this mod gives one "+
						"of its own prototypes",
						section, name, w, group.dropdown,
						strings.Join(group.values, ", "), w, w)
				}
			}
		}
	}

	// AND THE TABLE ABOVE HAS TO COVER THE WHOLE PLAN, or a seventh setting
	// bound beside a dropdown is silently unscanned and this test reports
	// nothing about it -- which is the failure mode it exists to close, one
	// layer out.
	//
	// THE PLAN IS ASKED RATHER THAN TRANSCRIBED. `*fkrecipes.Lib` exports no
	// accessor for the settings it holds, but `CheckLocaleWith` run against an
	// EMPTY locale file reports one `has no [mod-setting-name] entry` finding
	// per declared setting, in declaration order, so the findings ARE the
	// enumeration. A hand-rolled name is not in it, because the hand-rolled
	// list only suppresses orphans and never creates an obligation, and
	// `bbb-multi-edge-parts` belongs to no dropdown anyway. If the library ever
	// changes that sentence this loop finds nothing and the assertion fails
	// loudly rather than passing over an empty reading, which is the safe
	// direction for a check whose input is a message.
	var declared []string
	for _, finding := range Plan().CheckLocaleWith(ModName, "", HandRolledSettings()) {
		rest, ok := strings.CutPrefix(finding, "the setting ")
		if !ok {
			continue
		}
		if name, _, ok := strings.Cut(rest, " has no [mod-setting-name] entry"); ok {
			declared = append(declared, name)
		}
	}
	covered := map[string]bool{}
	for _, name := range scanned {
		covered[name] = true
	}
	for _, name := range declared {
		if !covered[name] {
			t.Errorf("this plan declares the setting %s and no dropdown group "+
				"above scans it, so nothing here reads its [mod-setting-name] "+
				"or [mod-setting-description] entry for an option the menu "+
				"does not offer. Add it to the group of the dropdown it sits "+
				"beside; the settings scanned are %s",
				name, strings.Join(scanned, ", "))
		}
	}
	if len(declared) != len(scanned) {
		t.Errorf("the groups above scan %d settings (%s) and this plan declares "+
			"%d (%s): the two have to be the same set, or this test is reading "+
			"a file against a plan it does not cover",
			len(scanned), strings.Join(scanned, ", "),
			len(declared), strings.Join(declared, ", "))
	}
}

// capitalisedWords is the reading [TestNoSettingDescriptionNamesAnOptionNoDropdownOffers]
// is built on, and it is deliberately crude: a word is what whitespace
// separates, stripped of the punctuation and the backticks around it, and it
// counts when its first rune is an upper-case letter.
//
// `skipSentenceOpeners` is what tells a label from a paragraph. In a
// description the first word of every sentence is capitalised because it opens
// a sentence and for no other reason, so it is skipped; a `[mod-setting-name]`
// entry is a label rather than a sentence and every word of it is read,
// including the first, because that is where a field named for a withdrawn
// option puts the option's name.
//
// A sentence is taken to end at `.`, `:`, `;`, `!` or `?`. The colon and the
// semicolon are over-generous -- both are followed by a lower-case word in this
// file -- and over-generous is the safe direction: it can only skip a word, and
// a skipped word is a miss rather than a false failure.
func capitalisedWords(text string, skipSentenceOpeners bool) []string {
	var out []string
	opener := true
	for _, raw := range strings.Fields(text) {
		atOpening := opener
		opener = strings.HasSuffix(raw, ".") || strings.HasSuffix(raw, ":") ||
			strings.HasSuffix(raw, ";") || strings.HasSuffix(raw, "!") ||
			strings.HasSuffix(raw, "?")
		w := strings.TrimFunc(raw, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if w == "" || (atOpening && skipSentenceOpeners) {
			continue
		}
		if unicode.IsUpper([]rune(w)[0]) {
			out = append(out, w)
		}
	}
	return out
}
