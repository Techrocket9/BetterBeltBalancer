package tune

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
// It is checked in BOTH DIRECTIONS on purpose. A missing entry is a value the
// player cannot read; an ORPHAN entry is a value that used to exist, which
// means an option was renamed and one of the two places was not.

func localePath(t *testing.T) string {
	t.Helper()
	// guest/go/tune -> the repository root.
	p, err := filepath.Abs(filepath.Join("..", "..", "..",
		"mod-data", "locale", "en", "better-belt-balancer.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// localeSections parses a Factorio .cfg into section -> key -> value.
//
// The grammar is INI without quoting: `[section]` opens one, `key=value` fills
// it, `#` and `;` at the start of a line are comments, and a value may contain
// anything including `=`. Written out rather than reached for because it is
// eight lines and the alternative is a dependency in a package that has none.
func localeSections(t *testing.T) map[string]map[string]string {
	t.Helper()
	fh, err := os.Open(localePath(t))
	if err != nil {
		t.Fatalf("the locale file is the thing under test and it is not there: %v", err)
	}
	defer fh.Close()

	out := map[string]map[string]string{}
	section := ""
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			if out[section] == nil {
				out[section] = map[string]string{}
			}
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			t.Errorf("locale line is neither a section nor a key: %q", line)
			continue
		}
		if out[section] == nil {
			out[section] = map[string]string{}
		}
		if _, dup := out[section][k]; dup {
			t.Errorf("[%s] %s is defined twice; Factorio keeps the last one", section, k)
		}
		out[section][k] = v
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEveryOptionHasItsLocaleEntry(t *testing.T) {
	sec := localeSections(t)
	values := sec["string-mod-setting"]
	if values == nil {
		t.Fatal("the locale file has no [string-mod-setting] section, " +
			"so every dropdown entry renders as `Unknown key: ...`")
	}

	want := map[string]bool{}
	for _, opt := range RecipeOptions() {
		want[SettingRecipeCost+"-"+opt] = true
	}
	for _, opt := range TechOptions() {
		want[SettingTechCost+"-"+opt] = true
	}

	for key := range want {
		if strings.TrimSpace(values[key]) == "" {
			t.Errorf("no [string-mod-setting] %s: that value renders in the "+
				"settings menu as `Unknown key: \"string-mod-setting.%s\"`", key, key)
		}
	}
	for key := range values {
		if want[key] {
			continue
		}
		// Only this package's own settings are policed. Another setting's
		// values are not this test's business.
		if strings.HasPrefix(key, SettingRecipeCost+"-") ||
			strings.HasPrefix(key, SettingTechCost+"-") {
			t.Errorf("[string-mod-setting] %s names a value no longer allowed: "+
				"an option was renamed in one place and not the other", key)
		}
	}
}

func TestBothSettingsAreNamedAndDescribed(t *testing.T) {
	sec := localeSections(t)
	for _, name := range []string{SettingRecipeCost, SettingTechCost} {
		for _, s := range []string{"mod-setting-name", "mod-setting-description"} {
			if strings.TrimSpace(sec[s][name]) == "" {
				t.Errorf("no [%s] %s: the settings menu shows the raw key", s, name)
			}
		}
	}
	// The 2.0-only bool is in the same file and is not this package's, but a
	// locale file that lost it would be the same defect -- and it is one line
	// to keep watching.
	if strings.TrimSpace(sec["mod-setting-name"]["bbb-multi-edge-parts"]) == "" {
		t.Error("no [mod-setting-name] bbb-multi-edge-parts")
	}
}

func TestNoAllowedValueCollidesWithAnother(t *testing.T) {
	// `<setting>-<value>` is a FLAT namespace, so two settings whose names are
	// prefixes of each other could produce one key for two values. They are not
	// today; this is what says so when a third setting arrives.
	seen := map[string]string{}
	for _, pair := range []struct {
		setting string
		options []string
	}{
		{SettingRecipeCost, RecipeOptions()},
		{SettingTechCost, TechOptions()},
	} {
		for _, opt := range pair.options {
			key := pair.setting + "-" + opt
			if other, dup := seen[key]; dup {
				t.Errorf("%s/%s and %s produce the same locale key %q",
					pair.setting, opt, other, key)
			}
			seen[key] = pair.setting + "/" + opt
		}
	}
}
