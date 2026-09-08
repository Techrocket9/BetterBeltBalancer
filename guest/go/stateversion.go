//go:build !prestate

package main

// stateVersion is what fk_state_version reports (main.go), and the rungs are
// what a load compares itself against.
//
//	0  every build up to and including 0.3.2, none of which exported
//	   fk_state_version at all. A belt running across a balancer's face was
//	   nothing to any of them, so a save at this rung is UNDECIDED about curved
//	   exits and the first load that can see one decides (curveupg.go).
//	1  0.3.3, where a belt across a face became an output.
//
// A RULE CHANGE ADDS A RUNG AND A COMPARISON, which is the pattern this exists
// to establish: the next rule that changes what a standing world MEANS bumps
// this to 2 and asks `oldVersion < 2` wherever the curve pass asks
// `oldVersion < 1`. What it must not become is a bag of per-feature flags in the
// save -- there is nothing here to keep in step with anything, because the save
// carries one number and the guest carries one table.
//
// A SAME-VERSION DEV REBUILD OF THE SHIPPED BUILD SEES old == 1 AND DECIDES
// NOTHING, which is right rather than a gap: since upstream's 2026-08-07 fix
// `fk_migrate` fires for ANY rebuild, same-version included, and a save stamped
// 1 was written by a guest that already had the curve rule -- its standing
// networks ARE the curve-on reading, so there is nothing about it to keep.
const stateVersion = 1
