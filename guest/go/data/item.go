package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

//go:noinline
func item() {
	fkdata.Extend(obj(
		f("type", str("item")),
		f("name", str("bbb-balancer-part")),
		f("icon", str(partIcon)),
		f("icon_size", num(64)),
		f("subgroup", str("belt")),
		// Next to the splitters, which is where a player looks for this.
		f("order", str("c[splitter]-y[bbb-balancer]")),
		f("place_result", str("bbb-balancer-part")),
		f("stack_size", num(50)),
	))
}
