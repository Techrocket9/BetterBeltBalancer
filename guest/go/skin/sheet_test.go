package skin

import (
	"encoding/binary"
	"os"
	"testing"
)

// THE COMMITTED SHEET HAS TO HOLD THE CELLS THE PROTOTYPE DECLARES, and nothing
// else in this repository can say so.
//
// `variation_count` comes from [Cells] (guest/go/data/entity.go), and the engine
// reads the PNG to find those cells -- but only the GRAPHICAL client ever opens
// a sprite file. Headless Factorio does not, so a sheet one row short passes
// every suite, every benchmark and the whole dump gate, and then refuses to load
// in a real game. `test/check-sprites.py` is the other half of that class and
// cannot reach this one: it walks the packaged mod for stale PATHS, and the cell
// count is a number in a compiled guest rather than a string in a Lua file.
//
// The layout rule is the contract stated in this package's own header: 64 px
// cells, 8 per row, cell i is the i-th canonical mask ascending. Only the
// HEADER is read -- width and height out of IHDR -- because that is the whole
// question.
func TestTheCommittedSheetHoldsEveryCell(t *testing.T) {
	const path = "../../../mod-data/graphics/entity/balancer-part-variants.png"
	const cell, cols = 64, 8

	d, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the sheet this package numbers cells in: %v", err)
	}
	if len(d) < 24 || string(d[1:4]) != "PNG" || string(d[12:16]) != "IHDR" {
		t.Fatalf("%s is not a PNG with IHDR first", path)
	}
	w := int(binary.BigEndian.Uint32(d[16:20]))
	h := int(binary.BigEndian.Uint32(d[20:24]))

	if w != cols*cell {
		t.Fatalf("the sheet is %d px wide and the prototype lays it out %d cells "+
			"of %d per row, which is %d", w, cols, cell, cols*cell)
	}
	if h%cell != 0 {
		t.Fatalf("the sheet is %d px tall, which is not a whole number of %d px rows",
			h, cell)
	}
	if have := (h / cell) * cols; have < Cells {
		t.Fatalf("the sheet holds %d cells and the prototype declares %d; the "+
			"engine wraps a variation above the count, so the second half would "+
			"draw the first half's shapes", have, Cells)
	}
}
