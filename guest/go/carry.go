package main

// What happens to the items a teardown drains out of a hidden network.
//
// THE POLICY CHANGED ON 2026-08-02, and this file is the change. v1 read every
// transport line in the slot and spilled the total beside the cluster, on every
// teardown, because a teardown could not tell WHY it was happening. Conservation
// held -- nothing was ever lost, and the `edge` suite measured 0 items lost over
// 200 teardowns of a network deliberately kept full -- but the PLACEMENT was
// wrong, and a player found it before a suite did: adding one output belt to a
// running balancer emptied its hidden network onto the ground.
//
// A RECOMPILE IS NOT A REMOVAL. The cluster is still standing, the network that
// goes up a moment later is empty and has room, and the items were in transit
// through a machine that still exists. So a drain now CARRIES: the items come
// out into a pool, the flush tears down and builds, and each network built in
// that flush takes the pool of the network(s) it succeeds. Only a real removal
// -- a cluster dissolved, a surface deleted, a network forgotten with nothing
// left to rebuild -- still spills, which is the vanilla-like behaviour for a
// machine that has been mined.
//
// THE FOUR DECISIONS, all of them deterministic, because a lockstep game gives
// no credit for "roughly the same place":
//
//  1. WHICH network gets a pool. The pool remembers the surface, the force and
//     the visible bounding box of the cluster it came out of; a cluster built in
//     the same flush claims it when all three agree (boxes overlapping is the
//     test). That is one predicate covering all three shapes: a plain recompile
//     matches itself, a merge gives the surviving cluster both halves' pools,
//     and a split gives one pool to both successors. A cluster whose own network
//     was still standing when the flush began is excluded -- it opens a pool of
//     its own inside compile() and draws only from that.
//
//  2. HOW a pool divides when more than one cluster claims it (the split). Even
//     shares per item kind, the earlier claimant taking the odd item:
//     ceil(remaining / claimants-left), which is exact for one claimant and
//     splits 24 as 12/12 and 25 as 13/12. Filling the first successor and
//     leaving the second empty would be the pathological answer.
//
//     A KIND IS (name, quality, STACK SIZE), so a split divides each stack
//     group on its own and both successors get the same MIX of stack sizes the
//     old network had. Twenty-four iron plates in six 4-stacks split as 12/12,
//     which is three whole 4-stacks each; an odd share re-stacks as whole
//     stacks plus at most one partial one at the tail. Dividing the POSITIONS
//     instead would have been the other choice and it is worse: it cannot give
//     a 3-position pool to two claimants evenly, and it makes the split rule
//     depend on a quantity (positions) that the reinsertion is free to change.
//
//  3. WHERE in the new network the items go. In PLAN ORDER -- the order
//     plan.Build emits its ops, which runs input side to output side -- filling
//     each transport line before moving to the next, and INTERIOR lines
//     (splitters, lane splitters, straight belts) before any linked belt. Two
//     reasons, and the second is the one that matters: the interior is where the
//     items were, and plan order puts them at the HEAD of the butterfly, so
//     every stage that is left rebalances them on the way out. Filling the edge
//     lines first would push items straight onto a player's belts in whatever
//     order they came off the old network -- observable, and observably
//     unbalanced.
//
//  4. WHAT DOES NOT FIT. The fallback is the old path: spill_item_stack beside
//     the cluster. It is reached when the new network is materially smaller than
//     the old one (outputs removed), and -- since 2026-08-04 -- when the drain
//     found MORE KINDS than a pool may hold. Those are the only two reasons a
//     recompile can still put anything on the floor.
//
// AND THE SECOND OF THOSE WAS AN ITEM SINK UNTIL 2026-08-04. A pool holds
// maxCarryKinds groups and the thirty-third was logged and dropped -- after
// drain() had read it off a line that sweep() destroys, so those items ceased to
// exist. Every headless suite was green while it did, because all seven of them
// run iron plates: it takes thirty-three distinct (name, quality, stack) groups
// through one balancer to reach, which is a sushi belt or a mall bus and not an
// exotic base at all. The bound stays -- it is a bound on package-level memory
// that is never given back -- and what it bounds is now the CARRY rather than the
// conservation. See tally.
//
// `insert_at`, NOT `insert_at_back`, AND THAT IS THE DIFFERENCE BETWEEN 32 ITEMS
// AND ALL OF THEM. Measured, on the `edge` suite's dead-ended 2->2: the back of
// a line is ONE position, so `insert_at_back` succeeds once per line per tick and
// then refuses until the belt has moved -- 32 items for a network with 32
// transport lines, and the other 40 of a 72-item drain fell straight through to
// the spill. This is the same property `agents/design.md` records as an
// incumbent sin ("script insert_at_back cannot produce compressed belts"), met
// from the other side. `insert_at(position)` places at a named point on the line
// -- `line_length` says how far that goes -- so walking a line at the belt's own
// 0.25-tile item pitch fills it compressed, which is how the drain found it.
//
// THE STACK SIZE IS RECOVERED SINCE 2026-08-02, BEHIND A GATE THAT COSTS
// VANILLA NOTHING. `get_contents` returns totals per (name, quality) and says
// nothing about how those items were STACKED, so the first cut of this file
// handed a Space Age player's 4-stacks back as singles: conserved, but a
// quarter of the density and a real throughput dip on a saturated stacked belt
// until compression recovered. The drain reads `get_detailed_contents` now --
// one LuaItemStack handle per belt POSITION -- and the pool's kind key is
// (name, quality, stack size) rather than (name, quality).
//
// THE GATE IS `LuaForce.belt_stack_size_bonus`, READ ONCE PER FORCE PER
// TRANSACTION, and it is what makes this free for everyone who is not playing
// Space Age. Measured, headlessly, before any of it was written:
//
//   * the loader prototype field that creates stacks (`max_belt_stack_size`)
//     is REFUSED AT LOAD without the `space_travel` feature flag -- "Belt
//     stacking is disabled and can not be used" -- so base Factorio cannot
//     stack at all;
//   * with the flag AND a stacking loader, a force whose bonus is 0 still
//     receives singles. The bonus is what turns gameplay stacking on, and it
//     only ever goes up;
//   * `insert_at`'s `belt_stack_size` is NOT gated by the bonus -- a script may
//     put a stack of 255 on a bonus-0 belt. So the gate is a statement about
//     what the GAME can produce, not about what is physically possible, and a
//     third-party mod that scripts stacks onto a bonus-0 force's belts is the
//     one case that still comes back unstacked. That is the same shape as the
//     failure envelope in CLAUDE.md: conservation always holds, fidelity is
//     best-effort, and the next audit re-reads the world.
//
// Below the gate the drain and the reinsertion are byte-for-byte what they were
// -- one `get_contents` per non-empty line, `belt_stack_size` absent on every
// `insert_at` -- so no base-only measurement in this repo moves.
//
// WHAT insert_at DOES WITH A STACK, measured rather than assumed: it is ATOMIC.
// `insert_at(p, {name, count = n}, n)` places one stack of n at p and returns
// true, or places nothing and returns false; a count larger than the stack size
// is silently truncated to it, and a stack occupies exactly ONE of the line's
// 0.25-tile slots however many items are in it. So the reinsertion walk is
// unchanged -- same pitch, same order, same lazy line probing -- and a stacked
// network costs FEWER host calls to refill than the same items unstacked.
//
// ---------------------------------------------------------------------------
// THE BENEFICIARY: WHO GETS THE ITEMS A REMOVAL SPILLS
// ---------------------------------------------------------------------------
//
// Mining a vanilla splitter puts what it was holding in the MINER'S POCKET, and
// only what will not fit lands on the ground. A balancer is a machine like any
// other, so mining the last part of one should do the same -- and until now it
// did not: a dissolve spilled the whole hidden network beside the cluster, on a
// surface the player then had to walk over picking items up one at a time.
//
// So a pool carries an optional BENEFICIARY: a `player_index`, a scalar, which
// is the only shape this guest is allowed to keep (nothing here may hold a
// LuaEntity or a LuaPlayer past the dispatch it arrived in, and the flush that
// settles a pool is a TICK LATER than the event that filled it). At settle time
// the index is resolved with `game.get_player` and the items are offered to that
// player; the remainder spills exactly as it always did.
//
// WHICH REMOVALS GET ONE, AND WHY THE LIST IS SHORT:
//
//   - `on_player_mined_entity` on ANY part, whether the removal dissolves the
//     cluster or merely SHRINKS it. See below: restricting it to the dissolve
//     was this feature's first cut and made it almost inert in play.
//   - `on_player_mined_entity` on a BELT-CONNECTABLE AT A CLUSTER'S EDGE, which
//     is the second field report and the second thing this list was too narrow
//     for. See "a mine beside a machine is a mine of that machine" below.
//   - a ROBOT deconstruction does NOT, and that is a decision rather than an
//     omission -- for a belt at the edge exactly as for a part. Vanilla sends a
//     robot's haul to a logistic storage chest, which needs the robot -- an
//     entity reference the deferred flush cannot hold -- or a network-and-chest
//     search this mod has no business doing. Spilling is what a player who
//     deconstructs a balancer already gets from every other path here, and it is
//     recoverable. Revisit only with the robot's own inventory in hand, which
//     means doing the work inside the event.
//   - `on_entity_died`, `script_raised_destroy`, a surface deleted or cleared,
//     a network forgotten: no player did those, so there is nobody to credit.
//
// A MINE BESIDE A MACHINE IS A MINE OF THAT MACHINE, which is the 2026-08-02
// second correction. The list above used to end "a belt mined beside a cluster
// still spills its overflow, and that is the one case left", justified by the
// `edge` suite's `shrk` leg measuring exactly such a reinsertion fitting with
// room to spare. THE MEASUREMENT WAS TRUE AND THE GENERALISATION WAS NOT: `shrk`
// takes a 4x4 from four outputs to three, and P = next_pow2(max(N, M)) is 4
// either way, so the butterfly it rebuilds is the same size and of course
// everything fits. Cross a power-of-two boundary the other way -- five outputs
// back to four, which is exactly "place an output belt on a running balancer and
// then mine it again" -- and P goes 8 to 4, the machine halves, and the
// reinsertion overflows by decision 4 above. A player reported the spill; the
// suite could not see it because the only shrink it drove did not shrink
// anything.
//
// So the neighbour path records a claim too, and the policy sentence is finally
// what it always said: a removal a PLAYER caused offers what no network could
// take to that player before the floor. The claim is keyed by a TILE OF THE
// CLUSTER -- the registry key the gate just looked up, not the mined belt's own
// tile, which is one outside the box by construction and would answer to nobody
// (carry/beside_test.go). The gate still makes NO HOST CALL: the tile, the force
// and the player are all in hand, and the note is skipped outright unless a
// player_index arrived, which is every event but one.
//
// WHY A SHRINK NEEDS ONE, which is the 2026-08-02 correction. The first cut
// reasoned that a shrink has a successor and the successor takes the pool back
// inside the network, so there is nothing to credit. That is true only when the
// successor HAS ROOM. Taking a balancer apart by hand is a run of shrinks ending
// in one dissolve, and every shrink makes the machine smaller -- fewer ports,
// fewer stages, less line -- so each one hands back less than it drained and the
// difference goes to the floor by decision 4. Measured, on a saturated four-part
// 4x4 mined one part per tick: 8, 152 and 46 items spilled across the three
// shrinks, and 26 items left in the machine for the dissolve to pocket. The
// feature credited the miner with the dregs and gave the floor the machine, and
// in the map editor -- `mining_speed = 6`, `instant_deconstruction = true` --
// that is fast enough to look like the pocket doing nothing at all.
//
// Nothing about the precedence moves: a claimed pool is still never pocketed,
// the survivor still claims the shrink's pool, and takeCarry still reinserts
// everything that fits. The beneficiary is consulted by settleCarry over the
// remainder alone, so a shrink that fits entirely -- the ordinary one -- reaches
// none of this and costs nothing at all.
//
// PRECEDENCE, and it is one line: A CLAIMED POOL IS NEVER POCKETED. Decisions 1
// to 3 run first and unchanged -- if any cluster built in this flush succeeds
// the network geometrically, the items go into it. The beneficiary is consulted
// only by settleCarry, over what nobody claimed, and it sits BETWEEN the claim
// and the floor. That also settles the imprecision the header records above: a
// cluster that claims a pool geometrically but is then not built leaves its
// share to settle, and settle now offers it to the beneficiary before the
// ground -- which is a better answer than the one that was there, by the same
// argument.
//
// HOW A CLAIM FINDS ITS POOL, and why it is a TILE rather than a root. The
// obvious key is the cluster root the event queued, and it is wrong: a decon
// planner mining a four-part balancer in one tick removes three parts that
// merely SHRINK it -- each re-rooting the survivors at the smallest surviving id
// -- and then a fourth that dissolves, so the root at the moment of the dissolve
// is not the root the netInfo is filed under. What does not move is the GROUND:
// the mined part's tile is inside the visible bounding box of the network being
// torn down, whichever root owns it. So a claim is (surface, FORCE, tile,
// player) and openPool takes the first claim that lands on a tile of the network
// it was given. First in event order, which is deterministic, so two players
// mining into one dissolve resolve the same way on every client.
//
// AND THE FORCE IS PART OF THAT KEY SINCE 2026-08-02, which is a fix and not a
// refinement. It was not, and `matches` -- the successor test, over the same
// pool, three hundred lines away -- always compared it. Clusters are per force
// (CLAUDE.md, "semantics fixed at M3"), so two forces' parts touching are two
// balancers whose boxes are ADJACENT BY CONSTRUCTION and, around an L or a
// diagonal, overlapping; two of them coming down in one tick with a player
// mining one of them could credit that player with the other force's items.
// Conservation was never at risk -- the pool is settled either way -- but the
// wrong pocket is the whole feature being wrong. The force is `pforce[id]` at
// the mine event, registry state already in hand, so the claim costs no host
// call and no more memory than the two coordinates beside it.
//
// The fix is not "add a comparison": both tests are `carry.Region.Overlaps` now,
// a claim being the degenerate one-tile Region it always was, and the predicate
// lives in `guest/go/carry` -- pure Go, no fkapi, so `go test ./carry/` proves
// it under an ordinary toolchain the way `plan` and `skin` are proved. That is
// the only part of this feature any machine in this repo can check, and it is
// the part that was wrong.
//
// AND A FORCE IN THE KEY IS A FORCE THAT CAN BE DESTROYED, which is the rest of
// the same correction. `game.merge_forces` destroys one of the two, and
// `onForcesMerged` tears the affected networks down BEFORE it remaps the
// registry, so in that one tick both the drained pool and any claim over it name
// a force that is on its way out. `remapCarryForce` followed the merge into the
// pools and not into the claims: a player who mined a source-force part in the
// merge tick was left holding a claim the survivor's remapped pool could no
// longer match, and their items went to the floor. Both follow it now, through
// one `carry.Region.FollowMerge`, and the claim store moved to
// `guest/go/carry` with the predicate so that `make check` covers the merge as
// well as the boundary -- see carry/claims.go. Like the force check itself this
// failed CLOSED, which is exactly why seven headless suites are silent about it.
//
// WHY NOT `event.buffer`, which is how vanilla does it. `on_player_mined_entity`
// carries a LuaInventory the engine then empties into the player, and it is the
// right answer for the entity being mined -- which is the PART, a
// simple-entity-with-force holding nothing. The items are in a hidden network on
// another surface, they are not read until the flush a tick later, and the
// buffer is valid only inside its own dispatch. Draining the network inside the
// event to reach the buffer would put back the one-tick removal window
// `fk.Defer()` deleted (compile.go), for a cosmetic difference in where the
// items appear.
//
// WHAT IS VERIFIED AND WHAT IS NOT. Only the TRIGGER is interactive now. The
// wall is real and unchanged -- a `--create` has no player, `game.get_player(1)`
// is nil, and `on_player_mined_entity` is not one of the eleven events
// `script.raise_event` will raise -- so nothing headless can make this guest
// resolve a LuaPlayer. What used to be behind that wall with it was the INSERT
// ARITHMETIC, and it is not any more:
//
//   - `insert` is a member of LuaControl, and a chest and a character are both
//     LuaControls. probe.go asks one of them exactly what pocketPool asks a
//     player, through the same `insertOne`, from inside the same deferred flush,
//     and the `edge` suite asserts the counts. See probe.go's header.
//   - the CLAIM PREDICATE -- which network a claim belongs to -- is pure guest
//     logic over five scalars and a box, so it moved to `guest/go/carry` and
//     `make check` proves it: the two-force overlapping-box case that this
//     file got wrong is a unit test that fails against the code it replaced.
//     THE CLAIM STORE ITSELF followed it there for the same reason, and with it
//     the other thing a force index has to survive: a merge that destroys the
//     force it names. Both are `go test ./carry/`, and both fail against the
//     shipped code they replaced.
//   - the SHRINK OVERFLOW -- the quantity this whole beneficiary exists to
//     redirect -- is measured directly by the `edge` suite's `hand` leg, which
//     mines a saturated four-part balancer one part per tick and counts what
//     reaches the ground at each step. A player gets exactly that, and the
//     suite fails if it stops being a real quantity.
//
// What the suites still pin as before: a dissolve reached by any path with no
// player records no beneficiary and spills exactly as it did, and no `pocketed`
// line appears anywhere in seven suites -- which is the assertion that this path
// cannot fire for a robot, a death, a script destroy or a surface deletion.

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/carry"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/plan"
)

// maxCarryKinds bounds the (name, quality, stack size) groups one drained
// network can CARRY. A network moves what the belts feeding it move; eight
// distinct (name, quality) pairs in one balancer is already an unusual factory,
// and belt stacking multiplies that by the stack sizes standing on the lines at
// once -- four in Space Age, and a partial stack at a boundary is a group of its
// own. Thirty-two is that product with room.
//
// IT BOUNDS THE POOL, NOT THE CONSERVATION, AND UNTIL 2026-08-04 IT BOUNDED
// BOTH. The thirty-third group was logged and DROPPED -- and drain() had already
// read it off a transport line that sweep() destroys a moment later, so those
// items ceased to exist. "Items are never deleted" is the one promise this mod
// makes that nothing else in it is allowed to trade against, and a sushi belt or
// a mall bus through one balancer is not an exotic base. Past the bound the
// group goes to the WORLD now, at the moment tally() decides the pool cannot
// have it; see tally.
const maxCarryKinds = 32

// carryItem is one (name, quality, stack size) and how many INDIVIDUAL items of
// it are in hand. `count` is items, never positions: it is what get_contents
// counts, what a spill puts on the floor, and what the split rule divides.
// `stack` is 1 for an ordinary belt and is what `insert_at` is asked to
// reproduce; 1 makes the call byte-identical to the pre-stacking one.
type carryItem struct {
	name    string
	quality string
	count   uint32
	stack   uint8
}

// carryPool is one torn-down network's contents, plus everything needed to
// decide who inherits them and where they go if nobody does.
type carryPool struct {
	root uint32 // the cluster the network belonged to
	// where is the identity the two decisions over this pool are made on: the
	// surface, the force and the visible box of the cluster it came out of. It
	// is a `carry.Region` and not six fields because the CLAIM over it is one
	// too -- successor and beneficiary are one predicate, and a force merge is
	// one rule applied to both. See carry.Region.
	where    carry.Region
	first, n int // the window into carryItems
	claims   uint32
	drawn    uint32
	// owned marks a pool opened by compile()'s own teardown-before-rebuild. Only
	// `root` may draw from it: the geometric test is for successors, and this
	// cluster is its own successor.
	owned bool
	// bene is the player_index to offer whatever no network claimed, or 0. Set
	// only by a player mining the last part of the cluster; see the header.
	bene uint32
	// spilled counts the items this pool was too full of KINDS to take. They are
	// already on the ground -- tally put them there in the same statement that
	// declined them -- and this counter exists only so that closePool can say so
	// in one alert line instead of one per group. A uint32 on a struct that
	// already exists is the whole storage cost of the overflow path.
	spilled uint32
}

// The pools and their items are ONE FLUSH's worth and are truncated back to
// zero by settleCarry. Package-level and reused, like every other buffer in this
// guest: an allocation per recompile is an allocation per recompile forever
// under -gc=leaking, and under the shipped collector it is garbage the pacer has
// to walk. See compile.go's buffer block.
var (
	carryPools []carryPool
	carryItems []carryItem
	carryTake  []carryItem // one claimant's share, refilled per pool

	// carryClaims is this tick's player claims, truncated to zero by settleCarry
	// exactly as the recompile queues are truncated by every flush. It is
	// high-water in the same sense they are -- one entry per part a player mined
	// in one tick, which is a decon planner's worth at the very most -- and it
	// holds no allocation of its own: the region and the player index are
	// scalars.
	//
	// The TYPE is `carry.Claims`, in the pure package beside the predicate that
	// reads it and the merge rule that rewrites it, because a claim is nothing
	// but data and `go test ./carry/` is the only machine in this repo that can
	// check any of the miner's pocket. See carry/claims.go.
	carryClaims carry.Claims

	// carryDepth is the transaction nesting. A drain carries only inside one:
	// flush() opens one around teardown-then-build, and the three reconcile
	// paths that split their own flush in half (a cloned area, the hidden
	// surface coming back, two forces merging) open one around the whole
	// sequence. Outside one -- a surface being deleted, with nothing left to
	// rebuild onto -- a drain spills immediately, exactly as it always did.
	carryDepth int

	// curPool is the pool tally() is filling, or -1. Never nested: teardownNet
	// opens one, sweeps, and closes it.
	curPool = -1

	carryKV   [3]fkapi.KeyValue
	carryInit bool

	// The stacking gate, memoised for one force for the life of one transaction.
	// Two host calls and no allocation: an entity we are already holding for its
	// force handle, and the force for its bonus. `belt_stack_size_bonus` is a
	// research result, so it cannot change inside a dispatch, and settleCarry
	// forgets it so that the tick research completes is the last one that reads
	// the old answer.
	carryStackFor   uint32
	carryStackOn    bool
	carryStackKnown bool

	// carryOverflow is the groups the pool being drained was too full of KINDS to
	// take, held for the rest of that one teardown and then put on the ground.
	//
	// ONE TEARDOWN'S WORTH AND NOT ONE FLUSH'S: closePool empties it, so it never
	// holds two networks' overflow at once, and it is empty on every tick nobody
	// exceeded the bound -- which is every tick of base single-kind play. Its
	// high-water is (distinct groups on one network's transport lines) minus
	// maxCarryKinds, so the most a save can drive it to is about the number of
	// item prototypes the game has: a few kilobytes, once, reused forever. That is
	// the high-water discipline every other buffer in this guest keeps, and it is
	// why the BOUND can stay where it is -- the pool is what a successor draws
	// from and what a split divides, and this is neither of those things.
	//
	// The belt stack size is dropped on the way in: a ground stack has no notion
	// of one, exactly as an inventory has none (pocketPool), so two groups that
	// differ only in how they were stacked merge here into one spill.
	carryOverflow []carryItem
)

// stacksPossible answers "can anything this force put on a belt be stacked".
//
// FALSE OUTSIDE A CARRY TRANSACTION, unconditionally, and that is not an
// optimisation: outside one a drain is a REMOVAL and the items go to
// `spill_item_stack`, which puts ordinary inventory stacks on the ground and has
// no notion of a belt stack. Paying for detail nobody can use would be pure
// cost, so every removal path -- a surface deleted, a cluster dissolved -- reads
// exactly as many host calls as it did before this file learned about stacking.
func stacksPossible(e fkapi.LuaEntity) bool {
	if !carrying() || curPool < 0 {
		return false
	}
	f := carryPools[curPool].where.Force
	if carryStackKnown && carryStackFor == f {
		return carryStackOn
	}
	on := false
	if o, err := e.Force(); err == nil {
		if bonus, err := (fkapi.LuaForce{Object: o}).BeltStackSizeBonus(); err == nil {
			on = bonus > 0
		}
	}
	carryStackFor, carryStackOn, carryStackKnown = f, on, true
	return on
}

func carrying() bool { return carryDepth > 0 }

func beginCarry() { carryDepth++ }

// endCarry closes the transaction and settles whatever nobody claimed.
//
// EVERY PATH THAT OPENS ONE MUST CLOSE IT, because the items are in guest memory
// until it does: a pool that outlived the dispatch would be items that exist in
// no transport line, no inventory and no ground stack. The nesting counter is
// what makes that safe when a reconcile path calls flush() from inside its own
// transaction.
func endCarry() {
	carryDepth--
	if carryDepth <= 0 {
		carryDepth = 0
		settleCarry()
	}
}

// openPool starts a drain for the network `root` had.
//
// An `owned` pool is claimed at birth: it belongs to the cluster compile() is
// about to rebuild, exactly one claimant, and claimCarry never sees it (it runs
// before the builds, and this pool is opened by one). Leaving `claims` at zero
// there is the bug that made the first cut of this file spill an entire 4x4 on
// every edge edit while the churn leg -- which reaches teardown through
// flushDead, where claimCarry does the counting -- passed.
func openPool(root uint32, ni netInfo, owned bool) {
	claims := uint32(0)
	if owned {
		claims = 1
	}
	// One Region, asked both questions: it is what a successor is compared
	// against and what a claim is compared against.
	r := carry.Region{Surf: ni.surf, Force: ni.force, X0: ni.x0, Y0: ni.y0, X1: ni.x1, Y1: ni.y1}
	carryPools = append(carryPools, carryPool{
		root: root, where: r,
		first: len(carryItems), owned: owned, claims: claims,
		bene: carryClaims.BeneficiaryFor(r),
	})
	curPool = len(carryPools) - 1
	// Paired with the truncation in closePool, so that nothing a previous
	// teardown could not carry can be spilled twice beside a different cluster.
	carryOverflow = carryOverflow[:0]
}

// noteMinedByPlayer records that `player` shrank the network standing on the
// tile `k` of force `force`.
//
// TWO CALL SITES, AND THE SECOND ONE IS THE 2026-08-02 FIELD REPORT:
//
//   - removePart, for every part a player mines -- a shrink, a split or a
//     dissolve alike. `k` is the mined part's own tile.
//   - onNeighbour, for a belt-connectable a player mines AT A CLUSTER'S EDGE.
//     `k` is the tile of the PART the belt was touching, which is the registry
//     key the gate just looked up -- NOT the belt's tile, which is one outside
//     the network's box by construction and would answer to nobody.
//
// Both are inside the event, with nothing in hand but what the event carried and
// what the registry already knew, so neither makes a host call and neither can
// fail. The force is `pforce[…]` at the call site -- registry state, not a host
// read -- so carrying it costs nothing on the guest's hottest path; removePart
// reads it BEFORE the node is freed.
//
// Whether the claim is ever USED is decided a tick later by openPool and then by
// settleCarry: everything a successor network claims and has room for goes there
// first, and only what is left over is offered to the miner. That is why this is
// a note rather than a decision, and it is why recording one on a shrink is free
// in the common case.
func noteMinedByPlayer(k key, force, player uint32) {
	carryClaims.Add(carry.Tile(k.s, force, k.x, k.y), player)
}

// remapCarryForce follows a force merge into everything the open carry
// transaction is holding: the pools already drained AND the claims over them.
//
// A pool remembers the force its network was built with, and the successor test
// compares it, because two forces' parts touching are two balancers whose boxes
// are adjacent by construction and whose items must not be swapped. A merge is
// the one event that makes that recorded force wrong: `onForcesMerged` tears the
// networks down BEFORE it remaps the registry -- it has to, because a cluster
// absorbed by the merge stops being a root -- so without this the surviving
// cluster fails to recognise the absorbed half's pool and 52 items go on the
// floor. Measured, in exactly that shape.
//
// BOTH LINES, AND THE SECOND ONE IS THE 2026-08-02 CORRECTION. A claim carries a
// force too, since it became a one-tile Region, and it is written down at the
// mine event under the same about-to-be-destroyed index. A player who mines a
// source-force part in the merge tick held a claim naming a force the survivor's
// remapped pool no longer matched, so the pocket silently became the floor. It
// fails closed -- conservation was never at risk, which is why no suite can see
// it -- and it is the sibling of the loop above rather than a separate rule:
// both are `carry.Region.FollowMerge`.
func remapCarryForce(src, dst uint32) {
	for i := range carryPools {
		carryPools[i].where.FollowMerge(src, dst)
	}
	carryClaims.FollowMerge(src, dst)
	// The stacking-gate memo is keyed by force, and the two lines above just
	// rewrote forces underneath it inside a live transaction: a pool now naming
	// `dst` could hit an entry measured on the SOURCE force, or miss one keyed
	// `src`. Dropping the memo is unconditionally safe -- the next stacking
	// question re-reads the bonus, two host calls, on a path (a merge tick that
	// reaches a stacking drain) too rare to be worth keying more cleverly.
	carryStackKnown = false
}

// closePool ends it and returns how much came out.
//
// Under a carry the pool stays pending for the build half. Outside one there is
// no successor to give it to, so it goes on the floor here and the pool is
// popped: a drain outside a transaction is a real removal.
func closePool() uint32 {
	if curPool < 0 {
		return 0
	}
	p := &carryPools[curPool]
	settleOverflow(p)
	total := poolTotal(p)
	if !carrying() || total == 0 {
		if total > 0 {
			handBack(p)
		}
		carryItems = carryItems[:p.first]
		carryPools = carryPools[:curPool]
	}
	curPool = -1
	return total
}

func poolTotal(p *carryPool) uint32 {
	total := uint32(0)
	for i := p.first; i < p.first+p.n; i++ {
		total += carryItems[i].count
	}
	return total
}

// tally is what drain() feeds. It merges by (name, quality, stack size) inside
// the pool currently open, which is always the last one.
func tally(name, quality string, stack uint8, count uint32) {
	if curPool < 0 || count == 0 {
		return
	}
	if stack == 0 {
		stack = 1
	}
	p := &carryPools[curPool]
	for i := p.first; i < p.first+p.n; i++ {
		if carryItems[i].name == name && carryItems[i].quality == quality &&
			carryItems[i].stack == stack {
			carryItems[i].count += count
			return
		}
	}
	if p.n >= maxCarryKinds {
		// PAST THE BOUND THE ITEMS GO TO THE WORLD, NOT NOWHERE, AND THAT IS THE
		// 2026-08-04 FIX. This branch used to log and return, and the items were
		// gone: drain() has already read this group off a transport line and
		// sweep() destroys the entity a few statements later, so a kind the pool
		// declines to carry is a kind that stops existing. Conservation is the one
		// promise here that nothing may be traded against, and thirty-three kinds
		// through one balancer is a sushi belt or a mall bus, not a pathology.
		//
		// WHAT IS GIVEN UP IS PLACEMENT, AND THAT IS THE DOCTRINE THIS FILE
		// ALREADY RUNS ON (compile.go's detailedTally: "conservation never depends
		// on fidelity"). An overflow group is not carried into the successor
		// network and is not offered to a miner's pocket -- both of those are what
		// the POOL is for, and a buffer general enough to reach either would be
		// the pool with no bound at all, which is the one thing maxCarryKinds
		// exists to prevent. Past thirty-two groups this guest stops promising
		// WHERE a teardown's items land and goes on promising THAT they land.
		addOverflow(name, quality, count)
		p.spilled += count
		return
	}
	carryItems = append(carryItems, carryItem{name: name, quality: quality, count: count, stack: stack})
	p.n++
}

// addOverflow merges one declined group into the buffer closePool will spill.
// Same shape as addLineGroup and returnToPool, and by (name, quality) only --
// the stack size means nothing to a ground stack.
func addOverflow(name, quality string, count uint32) {
	for i := range carryOverflow {
		if carryOverflow[i].name == name && carryOverflow[i].quality == quality {
			carryOverflow[i].count += count
			return
		}
	}
	carryOverflow = append(carryOverflow, carryItem{name: name, quality: quality, count: count, stack: 1})
}

// poolCentre is where a pool spills: the middle of the visible cluster's box.
// One statement of it, because the overflow and spillPool must not be able to
// disagree about where "beside the cluster" is.
func poolCentre(p *carryPool) (float64, float64) {
	return (float64(p.where.X0) + float64(p.where.X1) + 1) / 2,
		(float64(p.where.Y0) + float64(p.where.Y1) + 1) / 2
}

// settleOverflow puts what the pool could not carry on the ground, and says so.
//
// AT THE END OF THE TEARDOWN AND NOT INSIDE tally(), WHICH COST A MEASURED ITEM.
// The first cut spilled at the moment of the decision -- appealingly, since there
// is then nothing to remember -- and teardownNet sweeps the HIDDEN half first and
// the visible cluster box SECOND. `spill_item_stack` allows belts, the spill
// lands at the centre of that very box, and the visible sweep then re-drained
// what had just been put there into a pool it was already too full to take. The
// `mix` suite's `many` rig measured the result: 4,336 items in, 4,335 out, one
// stone-brick gone, against exact conservation with the bound raised so that
// nothing overflowed at all. Here both sweeps have finished and nothing of ours
// is left standing in the box, so this is the same moment, and the same
// arithmetic, as the spillPool every removal path has always used.
//
// ALERT, NOT ERROR, AND THE LEVEL IS THE DIFFERENCE BETWEEN A CONDITION A SUITE
// CAN ASSERT AND A RUN THAT DIES. `test/run.sh` fails ANY run in which a
// `[BBB] error:` line appears at all, because an error from this guest means one
// thing -- a compile did not produce a network (logline.go). This is not that:
// the network was built, everything the pool could carry went back inside it, and
// the rest is on the ground where a player can pick it up. `alert:` is this
// guest's existing level for "something outside the ordinary happened and the mod
// coped", and it is what lets the `mix` suite REQUIRE the line in its overflow
// window instead of the runner killing the run before an assertion is read.
//
// ONE LINE PER TEARDOWN, not one per group: the branch above is taken once per
// group per transport line, which on a sushi 4x4 is hundreds of times for the
// same handful of kinds.
func settleOverflow(p *carryPool) {
	if len(carryOverflow) == 0 {
		return
	}
	if vis, ok := surfaceByIndex(p.where.Surf); ok {
		cx, cy := poolCentre(p)
		for i := range carryOverflow {
			spill(vis, cx, cy, carryOverflow[i].name, carryOverflow[i].quality,
				carryOverflow[i].count)
		}
	} else {
		// Unreachable from anything in this repo -- a drain whose own visible
		// surface is going happens outside a carry transaction, where closePool
		// spills the pool while the surface is still valid -- and it is an error
		// rather than an alert because it is the one shape of this path that
		// really does lose items.
		logErrStart("the surface ")
		logU(p.where.Surf)
		logS(" is gone and cluster ")
		logU(p.root)
		logS("'s overflow has nowhere to go")
		logEnd()
	}
	carryOverflow = carryOverflow[:0]
	logAlertStart("cluster ")
	logU(p.root)
	logS(" carried more than ")
	logU(maxCarryKinds)
	logS(" item kinds; ")
	logU(p.spilled)
	logS(" items of the kinds past that bound went to the ground beside it")
	logEnd()
}

// ---------------------------------------------------------------------------
// Who inherits what
// ---------------------------------------------------------------------------

// matches is the successor test: same surface, same force, boxes overlapping.
//
// A cluster that split has both successors inside the old box; a cluster that
// merged contains both old boxes; a cluster that was recompiled in place is the
// same box. The force check is not decoration -- two forces' parts touching are
// two balancers, their boxes are adjacent by construction, and their networks
// carry whatever their own belts carry.
//
// It is `carry.Region.Overlaps` because the claim test is too -- `carry.Claims.
// BeneficiaryFor` asks the same predicate the other way round. The two were
// written out separately once and one of them lost the force.
func (p *carryPool) matches(surf, force uint32, x0, y0, x1, y1 int32) bool {
	return p.where.Overlaps(carry.Region{
		Surf: surf, Force: force, X0: x0, Y0: y0, X1: x1, Y1: y1,
	})
}

// clusterBox reduces a tile list to its inclusive bounding box.
func clusterBox(tiles []key) (x0, y0, x1, y1 int32) {
	x0, y0, x1, y1 = tiles[0].x, tiles[0].y, tiles[0].x, tiles[0].y
	for i := range tiles {
		if tiles[i].x < x0 {
			x0 = tiles[i].x
		}
		if tiles[i].x > x1 {
			x1 = tiles[i].x
		}
		if tiles[i].y < y0 {
			y0 = tiles[i].y
		}
		if tiles[i].y > y1 {
			y1 = tiles[i].y
		}
	}
	return
}

// claimCarry counts, for every pool this flush's teardowns left pending, how
// many of the clusters it is about to build descend from it.
//
// It has to happen BEFORE the first build, and that is the whole reason it is a
// separate pass: a split's first successor cannot know it should take half
// unless something has already counted the second. It costs one flood fill per
// queued cluster and NOT ONE HOST CALL, and it runs at all only when a teardown
// left something behind -- which is never true of the common flush, where the
// fingerprint skipped and nothing came down.
//
// A cluster whose own network is still standing is skipped: either the
// fingerprint is about to skip it, or compile() will tear it down itself and
// open an `owned` pool. Either way it is not somebody else's successor.
func claimCarry(roots []uint32) {
	if len(carryPools) == 0 {
		return
	}
	for i := range roots {
		r := roots[i]
		if _, had := nets[r]; had {
			continue
		}
		tiles := collectCluster(r)
		if len(tiles) == 0 {
			continue
		}
		x0, y0, x1, y1 := clusterBox(tiles)
		for p := range carryPools {
			pool := &carryPools[p]
			if pool.owned {
				continue
			}
			if pool.matches(tiles[0].s, pforce[r], x0, y0, x1, y1) {
				pool.claims++
			}
		}
	}
}

// takeCarry gives the network just built for `root` the items that came out of
// the network(s) it succeeds.
//
// `hadNet` is whether the cluster's own network was standing when compile()
// started, and it is what keeps this in step with claimCarry: a cluster that had
// one was not counted as anybody's successor, so it may draw only from the pool
// its own teardown opened.
func takeCarry(root, surf, force uint32, x0, y0, x1, y1 int32, hadNet bool,
	ops []plan.Op, ents []fkapi.Object) {
	if len(carryPools) == 0 {
		return
	}
	var w lineWalk
	w.ops, w.ents = ops, ents
	took, left := uint32(0), uint32(0)
	for p := range carryPools {
		pool := &carryPools[p]
		if pool.owned {
			if pool.root != root {
				continue
			}
		} else if hadNet || !pool.matches(surf, force, x0, y0, x1, y1) {
			continue
		}
		share := pool.claims - pool.drawn
		if share == 0 {
			continue
		}
		pool.drawn++

		// The share, taken out of the pool so that a later claimant divides what
		// is actually left.
		carryTake = carryTake[:0]
		for i := pool.first; i < pool.first+pool.n; i++ {
			it := &carryItems[i]
			if it.count == 0 {
				continue
			}
			n := (it.count + share - 1) / share
			it.count -= n
			carryTake = append(carryTake, carryItem{it.name, it.quality, n, it.stack})
		}
		if len(carryTake) == 0 {
			continue
		}
		for i := range carryTake {
			took += carryTake[i].count
		}
		took -= insertRemainder(&w, carryTake)
		// Whatever would not fit goes back where it came from: a later claimant
		// may have room, and if nobody does it spills at the end of the flush.
		for i := range carryTake {
			if carryTake[i].count == 0 {
				continue
			}
			left += carryTake[i].count
			returnToPool(pool, &carryTake[i])
		}
	}
	if took > 0 && verboseLog {
		logStart("cluster ")
		logU(root)
		logS(" took back ")
		logU(took)
		logS(" items into its new network")
		if left > 0 {
			logS(", ")
			logU(left)
			logS(" would not fit")
		}
		logEnd()
	}
}

// returnToPool puts an unplaced remainder back. The kind is always already in
// the pool's window -- it came from there -- so this cannot overflow it.
func returnToPool(p *carryPool, it *carryItem) {
	for i := p.first; i < p.first+p.n; i++ {
		if carryItems[i].name == it.name && carryItems[i].quality == it.quality &&
			carryItems[i].stack == it.stack {
			carryItems[i].count += it.count
			return
		}
	}
}

// settleCarry hands back whatever no network claimed and ends the transaction.
//
// This is the REMOVAL path and it is the one that stayed: a cluster that
// dissolved, a network forgotten because its surface went, a claimant that could
// not be built.
func settleCarry() {
	for i := range carryPools {
		p := &carryPools[i]
		if poolTotal(p) > 0 {
			handBack(p)
		}
	}
	carryPools = carryPools[:0]
	carryItems = carryItems[:0]
	carryTake = carryTake[:0]
	// A claim outlives the event that made it by one tick and no longer: the
	// flush that could have used it has run.
	carryClaims.Reset()
	curPool = -1
	// The gate is a research result, and research completes between dispatches.
	carryStackKnown = false
}

// handBack is what a REMOVAL does with a pool nobody inherited: the miner's
// pocket first where there is a miner, and the world with what is left.
//
// Both halves in one function so that the two exits a pool can take cannot drift
// apart. `closePool` reaches it for a drain outside a transaction (a surface
// being deleted, where there is nothing left to rebuild onto) and `settleCarry`
// for everything else.
//
// THE `offered` LINE IS THE ONE PIECE OF EVIDENCE A PLAYER CAN READ, and that is
// why it is here rather than inside pocketPool. Everything else about this
// feature is provable without a game -- the identity by `go test ./carry/`, the
// insert arithmetic by the chest probe, the quantity by the `edge` suite's
// shrink legs -- and the TRIGGER is provable by nothing at all, because a
// headless --create has no players and on_player_mined_entity cannot be raised.
// So the line names the quantity and the player at the exact point the decision
// is taken, before either the pocket or the floor can have changed it: a user
// checking the field report interactively greps for it.
func handBack(p *carryPool) {
	if p.bene != 0 {
		if verboseLog {
			logStart("cluster ")
			logU(p.root)
			logS(" offered ")
			logU(poolTotal(p))
			logS(" items to player ")
			logU(p.bene)
			logS(" before the floor")
			logEnd()
		}
		pocketPool(p)
	}
	if poolTotal(p) > 0 {
		spillPool(p)
	}
}

// pocketPool offers the pool to the player who mined the machine.
//
// THE PLAYER IS RESOLVED FRESH, HERE, and that is the whole reason the pool
// carries an index rather than a handle: a tick has passed since the mine, and
// in that tick the player can have left the game, been removed by another mod,
// or died. `game.get_player` returning nothing is not an error -- it is the
// ordinary case on a headless server -- so it falls through to the spill the
// removal would have done anyway.
//
// ONE HOST CALL PER ITEM KIND, and plain counts. `LuaControl.insert` takes an
// ItemStackIdentification whose `count` may span many inventory stacks, so a
// 72-item pool of one kind is one call, and it returns how many it actually
// took. The BELT stack size is deliberately dropped: an inventory has no notion
// of belt stacking, and vanilla mining loses it too -- carry.go recovers stack
// density for the reinsertion path, which is the only path where it means
// anything.
func pocketPool(p *carryPool) {
	o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(float64(p.bene)))
	if err != nil || o == nil || !o.Valid() {
		if verboseLog {
			logStart("player ")
			logU(p.bene)
			logS(" is no longer here; cluster ")
			logU(p.root)
			logS("'s items go to the world instead")
			logEnd()
		}
		return
	}
	took := uint32(0)
	for i := p.first; i < p.first+p.n; i++ {
		it := &carryItems[i]
		if it.count == 0 {
			continue
		}
		n := insertOne(*o, it.name, it.quality, it.count)
		it.count -= n
		took += n
	}
	if took > 0 && verboseLog {
		logStart("cluster ")
		logU(p.root)
		logS(" was mined by player ")
		logU(p.bene)
		logS("; pocketed ")
		logU(took)
		logS(" items")
		if left := poolTotal(p); left > 0 {
			logS(", ")
			logU(left)
			logS(" would not fit")
		}
		logEnd()
	}
}

// insertOne offers `count` of one kind to a LuaControl's inventory and reports
// what it actually took.
//
// A LuaControl, not a LuaPlayer, and that is what makes this testable: `insert`
// is LuaControl's, so the same member id, the same signature and the same
// tier-2 encode serve a player's pockets and a chest's slots. probe.go asks a
// chest exactly what pocketPool asks a player, which is the only way any of this
// arithmetic can be pinned without a player in the game.
func insertOne(o fkapi.Object, name, quality string, count uint32) uint32 {
	n, err := fkapi.LuaControl{Object: o}.Insert(carryStack(name, quality, count))
	if err != nil {
		return 0
	}
	// The engine cannot take more than it was offered; the clamp is here because
	// the caller's subtraction is what conservation rests on.
	if n > count {
		n = count
	}
	return n
}

func spillPool(p *carryPool) {
	vis, ok := surfaceByIndex(p.where.Surf)
	if !ok {
		logErrStart("the surface ")
		logU(p.where.Surf)
		logS(" is gone and cluster ")
		logU(p.root)
		logS("'s items have nowhere to go")
		logEnd()
		return
	}
	cx, cy := poolCentre(p)
	total := uint32(0)
	for i := p.first; i < p.first+p.n; i++ {
		if carryItems[i].count == 0 {
			continue
		}
		total += carryItems[i].count
		spill(vis, cx, cy, carryItems[i].name, carryItems[i].quality, carryItems[i].count)
		carryItems[i].count = 0
	}
	if total > 0 && verboseLog {
		logStart("spilled ")
		logU(total)
		logS(" items beside cluster ")
		logU(p.root)
		logEnd()
	}
}

// ---------------------------------------------------------------------------
// Putting them back
// ---------------------------------------------------------------------------

// itemPitch is how far apart items sit on a compressed transport line, which is
// a property of the belt rather than of the item: four per tile per lane, in
// every tier. Walking a line in these steps is what makes the reinsertion put
// the network back the way the drain found it; a coarser step would leave gaps
// and spill the difference, and a finer one would only waste refused calls.
const itemPitch = 0.25

// lineWalk yields the positions of a fresh network's transport lines, ONE SLOT
// AT A TIME and only as far as the items reach.
//
// Lazy on purpose: a 4x4 network is ~32 entities and ~130 transport lines, and
// asking each entity how many lines it has and each line how long it is is ~160
// host calls before an item has moved. A recompile that recovered twenty items
// touches two entities and pays for two. The cost is proportional to what was
// actually in flight, which is the quantity this whole path is about.
//
// The order is pass 0 (every op that is NOT a linked belt, in plan order, every
// line ascending, every position from 0) and then pass 1 (every linked belt the
// same way). Plan order runs the network input side to output side, so pass 0
// starts at the lane splitters and works through the stages, and every item put
// there is rebalanced by the stages still in front of it. The linked belts --
// the jumpers, and the two ends of every visible edge -- come last, because
// inserting there is how an item reaches a player's belt WITHOUT passing through
// the butterfly, which is the one way reinserted items could be observable as an
// imbalance.
type lineWalk struct {
	ops  []plan.Op
	ents []fkapi.Object
	pass int    // 0 interior, 1 linked belts, 2 exhausted
	op   int    // the next op to consider
	li   uint32 // the next line index on the current entity, 1-based
	nl   uint32 // how many lines that entity has; 0 means "fetch the next one"
	ent  fkapi.LuaEntity
	cur  fkapi.LuaTransportLine
	have bool    // cur is a real line
	pos  float64 // the next position to try on it
	end  float64 // its line_length: positions run 0 up to this
}

func (w *lineWalk) next() (fkapi.LuaTransportLine, float32, bool) {
	for {
		if w.have && w.pos <= w.end {
			p := w.pos
			w.pos += itemPitch
			return w.cur, float32(p), true
		}
		if w.nl > 0 && w.li <= w.nl {
			i := w.li
			w.li++
			l, err := w.ent.GetTransportLine(i)
			if err != nil {
				continue
			}
			line := fkapi.LuaTransportLine{Object: l}
			// `line_length` is the API's own statement of how far a position may
			// go, which is why nothing here assumes a length per prototype.
			n, err := line.LineLength()
			if err != nil {
				continue
			}
			w.cur, w.end, w.pos, w.have = line, float64(n), 0, true
			continue
		}
		w.have = false
		if w.pass > 1 {
			return fkapi.LuaTransportLine{}, 0, false
		}
		if w.op >= len(w.ops) {
			w.pass++
			w.op = 0
			continue
		}
		i := w.op
		w.op++
		linked := w.ops[i].Proto == plan.ProtoLinkedBelt
		if (w.pass == 0) == linked {
			continue
		}
		e := fkapi.LuaEntity{Object: w.ents[i]}
		// The line count is ASKED FOR rather than assumed per prototype, for the
		// same reason drain() asks: a constant that was wrong by one would lose
		// capacity here and lose items there.
		n, err := e.GetMaxTransportLineIndex()
		if err != nil || n == 0 || n > maxLinesToProbe {
			continue
		}
		w.ent, w.nl, w.li = e, n, 1
	}
}

// insertRemainder puts `items` into the network `w` walks, decrementing each
// count as it goes, and returns how many are LEFT OVER.
//
// Fill in order: every slot of the current line before the next line's. A
// refusal -- `insert_at` reports one as a false return rather than an error --
// costs one call and the slot is simply skipped, which is what makes the pitch a
// tuning constant rather than a correctness one.
//
// ONE CALL PER POSITION, NOT PER ITEM. A group whose stack size is s puts
// min(remaining, s) into each slot it is offered, and `insert_at` is atomic: it
// places the whole stack or nothing at all, so `ok` is the only thing that has
// to be believed and a group of 24 in 4-stacks costs six calls rather than
// twenty-four. At s = 1 that is exactly the one-item call this loop always made.
func insertRemainder(w *lineWalk, items []carryItem) uint32 {
	line, pos, have := w.next()
	for k := range items {
		for items[k].count > 0 {
			if !have {
				n := uint32(0)
				for j := k; j < len(items); j++ {
					n += items[j].count
				}
				return n
			}
			n := uint32(items[k].stack)
			if n == 0 {
				n = 1
			}
			if n > items[k].count {
				n = items[k].count
			}
			ok, err := line.InsertAt(pos, carryStack(items[k].name, items[k].quality, n), beltStack(n))
			if err == nil && ok {
				items[k].count -= n
			}
			line, pos, have = w.next()
		}
	}
	return 0
}

// beltStack is the `belt_stack_size` argument, and it is ABSENT for an ordinary
// unstacked item rather than an explicit 1.
//
// That is deliberate: below the stacking gate every group has stack size 1, and
// passing nil here is what makes the call the engine receives byte-for-byte the
// one it received before this file knew what a belt stack was. An explicit 1
// would mean the same thing and would move a measurement for no reason.
var beltStackBuf uint8

func beltStack(n uint32) *uint8 {
	if n <= 1 {
		return nil
	}
	if n > 255 {
		n = 255
	}
	beltStackBuf = uint8(n)
	return &beltStackBuf
}

// carryStack builds the ItemStackIdentification `insert_at` takes.
//
// `count` is how many items this one call is placing -- one whole belt stack,
// or one item where nothing is stacked. The engine truncates a count larger than
// the accompanying `belt_stack_size` silently, so the two are always passed
// equal and neither is guessed.
func carryStack(name, quality string, count uint32) fkapi.Value {
	if !carryInit {
		carryKV[0].Key = fkapi.OfString("name")
		carryKV[0].Val.Tag = fkapi.TagString
		carryKV[1].Key = fkapi.OfString("count")
		carryKV[1].Val = fkapi.OfNumber(1)
		carryKV[2].Key = fkapi.OfString("quality")
		carryKV[2].Val.Tag = fkapi.TagString
		carryInit = true
	}
	carryKV[0].Val.Str = name
	carryKV[1].Val.Number = float64(count)
	n := 2
	if quality != "" {
		carryKV[2].Val.Str = quality
		n = 3
	}
	return fkapi.Value{Tag: fkapi.TagMap, Map: carryKV[:n]}
}
