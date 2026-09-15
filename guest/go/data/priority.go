package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// THE KEYBIND THAT TOGGLES A BALANCER PART'S PRIORITY FLAG.
//
// A custom-input prototype is the only way a mod gets a key, and the control
// guest subscribes to it BY NAME rather than by a `defines.events` id -- there
// is none for a custom input, measured by FkLua's own fixture against the 233
// keys that table does have. So this prototype existing is what makes
// `fkapi.SubscribeNamed` succeed, and a name that does not match is a keybind
// that silently never fires.
//
// THE NAME IS WRITTEN TWICE, HERE AND IN guest/go/priority.go, and that is the
// same shape `PartName` has: the control guest may not import `guest/go/tune`
// (it pulls FkRecipes in, and the two `main` packages' exports then collide at
// link time) and this one may not import `fkapi`, so the two halves of this mod
// share constants only through a package that imports neither. For one string
// with two readers, a literal and a comment naming the other copy is smaller
// than a package.
const inputTogglePriority = "bbb-toggle-priority"

// ALT + P, and the letter is the only part of that with a choice in it.
//
// Every custom input base and its expansions define is `ALT + <letter>` -- 14 of
// them, read out of a `--dump-data` of base, quality, elevated-rails and
// space-age on 2.0.77: A, B, C, D, E, F, G, L, R, T, U and Y, plus one on TAB
// and one with no default at all. So ALT + <letter> is the family a player
// already reads as "a mod's key", and P is not one of the twelve taken.
//
// WHAT IS NOT MEASURED is the engine's own compiled-in bindings: they are not
// prototypes, so no dump lists them and nothing here can enumerate them. If ALT
// + P turns out to collide with one, Factorio says so in the controls menu and
// the player rebinds it, which is what that menu is for.
//
// `consuming = "none"` is the permissive setting: the keypress still reaches
// whatever else is bound to it. A mod that only listens wants that, and this one
// only listens.
//
//go:noinline
func priorityInput() {
	fkdata.Extend(obj(
		f("type", str("custom-input")),
		f("name", str(inputTogglePriority)),
		f("key_sequence", str("ALT + P")),
		f("consuming", str("none")),
		f("order", str("a")),
	))
}
