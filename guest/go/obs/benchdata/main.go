// bbb-bench-setup's SETTINGS STAGE, and it is nothing else.
//
// The bench harness's setup mod defines no prototype at all -- every piece of
// every rig is a stock belt, loader, chest or the balancer mod's own part -- so
// this module exports `fk_settings` and no `fk_data`. `fklua mod` writes a stage
// file per hook a module exports and for no others, so the package carries a
// settings.lua and no data.lua.
//
// WHAT IT IS FOR is the configuration channel, and the whole of the argument is
// in `protos`' own header: `bench/run.sh` used to REWRITE a `config.lua` inside
// the staged copy of the mod, which a Go guest cannot read. These eight settings
// are that file, and `tools/mod-settings.py` is the writer.
//
// EVERY ONE OF THEM IS STARTUP. Nothing here shapes a prototype -- this stage
// does not read a single one of them -- so the choice is about WHO reads them
// and when: a startup setting is read out of mod-settings.dat by the `--create`
// process and by the `--benchmark` process alike, with no state carried in the
// save between the two. See `protos`.
//
// THE DEFAULTS ARE bench/run.sh's OWN DEFAULTS, key for key, so a mod-settings
// file that failed to be written would build the same rig the old config.lua's
// committed defaults built rather than an arbitrary one. That is deliberately
// NOT the anti-vacuity guard -- the guard is that `run.sh` writes every key on
// every cell and the create log echoes all eight back on the BENCH-SETUP line.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

func obj(kvs ...fkdata.KV) fkdata.V { return obsdata.Obj(kvs...) }
func f(k string, v fkdata.V) fkdata.KV {
	return obsdata.F(k, v)
}
func str(s string) fkdata.V     { return obsdata.Str(s) }
func num(n float64) fkdata.V    { return obsdata.Num(n) }
func strs(s ...string) fkdata.V { return obsdata.Strs(s...) }

// choice is one dropdown: the allowed values, and the FIRST of them as the
// default. Factorio refuses a mod whose `default_value` is not in
// `allowed_values`, by name, at load -- so the two coming from one slice removes
// the only way to get that wrong. It is the shipped guest's `stringSetting` with
// nothing added.
//
//go:noinline
func choice(name string, values []string, order string) fkdata.V {
	return obj(
		f("type", str("string-setting")),
		f("setting_type", str("startup")),
		f("name", str(name)),
		f("default_value", str(values[0])),
		f("allowed_values", strs(values...)),
		f("order", str(order)),
	)
}

// text is a FREE-TEXT string setting: no allowed_values, because the two keys
// that use it name prototypes the harness cannot enumerate -- an item name is
// whatever `--item` says, and a part name is whichever balancer mod is under
// test (`balancer-part` for both incumbents, `bbb-balancer-part` for ours).
//
// `allow_blank` is deliberately absent, which means false: an empty item or part
// name is a misconfigured cell and Factorio refusing it in the menu is the
// earliest place that can be said.
//
//go:noinline
func text(name, def, order string) fkdata.V {
	return obj(
		f("type", str("string-setting")),
		f("setting_type", str("startup")),
		f("name", str(name)),
		f("default_value", str(def)),
		f("order", str(order)),
	)
}

// count is an int setting with a floor. The ceilings are deliberately generous
// rather than tuned: `n` is the number of rigs (or, in the mega scenarios, of
// BLOCKS) and the largest cell this harness has ever run is 500.
//
//go:noinline
func count(name string, def, lo, hi float64, order string) fkdata.V {
	return obj(
		f("type", str("int-setting")),
		f("setting_type", str("startup")),
		f("name", str(name)),
		f("default_value", num(def)),
		f("minimum_value", num(lo)),
		f("maximum_value", num(hi)),
		f("order", str(order)),
	)
}

//go:noinline
func flag(name string, def bool, order string) fkdata.V {
	return obj(
		f("type", str("bool-setting")),
		f("setting_type", str("startup")),
		f("name", str(name)),
		f("default_value", fkdata.Bool(def)),
		f("order", str(order)),
	)
}

//go:wasmexport fk_settings
func settings() {
	fkdata.Extend(
		choice(protos.BenchScenario, protos.BenchScenarios, "a"),
		count(protos.BenchN, 1, 1, 100000, "b"),
		count(protos.BenchK, 4, 1, 64, "c"),
		choice(protos.BenchTier, protos.BenchTiers, "d"),
		text(protos.BenchItem, "iron-ore", "e"),
		text(protos.BenchPartName, "balancer-part", "f"),
		// 0 disables metering, which is `--meter 0` and is why the floor is 0
		// rather than 1.
		count(protos.BenchMeter, 600, 0, 1000000, "g"),
		flag(protos.BenchHitch, false, "h"),
	)
}

func main() {}
