// Command bbb-lane-test-data is the `bbb-lane-test` observer's DATA STAGE: the
// estate's express loader under this suite's own name, and nothing else. See
// obs/qualdata for why every suite brings one.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

//go:wasmexport fk_data
//go:noinline
func onData() { obsdata.ExpressLoader(protos.LaneLoader) }

func main() {}
