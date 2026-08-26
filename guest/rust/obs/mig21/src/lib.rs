//! Reports what a Factorio 2.0 balancer save looks like when it is opened on
//! Factorio 2.1, before and after this mod's migration runs on it.
//!
//! A COMPILED RUST OBSERVER. It was `test/mods/bbb-mig21-observer/control.lua`,
//! then `guest/go/obs/mig21`, and it is this -- sample for sample and log line
//! for log line, both times. See agents/estate-port.md, phases 2 and 8.
//!
//! # Why this one is in Rust
//!
//! FkLua maintains Go/Rust bindings at member-id parity -- 4857 bound and the
//! same 2 deferred in each -- and guards it with mirror tests that run both
//! against a host stub. What a stub cannot reach is a real engine, real
//! fixtures, and a transcript somebody is holding to the byte. This observer is
//! that gate: the Go version's 51 tagged lines over the m2 and edge fixtures are
//! the goldens, and this one has to reproduce them exactly.
//!
//! `mig21` is the suite that makes that a fair test rather than a lucky one. It
//! has NO STORAGE, builds NO WORLD, and asserts NOTHING -- it reads and it logs.
//! So a diff between the two transcripts is a difference in the bindings, the
//! marshalling or the arithmetic, and cannot be a difference in a rig.
//!
//! # The one thing it has to get right: WHEN "before" is
//!
//! The migration does not wait for a tick. The mod's guest heap is declined --
//! it is a different build -- so `fk_migrate` fires from
//! `on_configuration_changed`, BEFORE THE FIRST TICK, and by tick 0 the
//! condemned remnants are already down and their contents already on the
//! ground. A sample taken from `on_tick` is a sample taken afterwards, and the
//! only "before" any script can reach is this mod's own
//! `on_configuration_changed`.
//!
//! WHETHER THAT ONE RUNS FIRST IS FACTORIO'S CHOICE. Handlers run in mod load
//! order, and `bbb-mig21-observer` sorts before `better-belt-balancer`, which is
//! why this mod deliberately declares NO DEPENDENCY on it -- a dependency would
//! put it after. THAT IS A PACKAGING FACT AND NOT A FILE: the Makefile passes
//! `--dependency "base >= 2.1.0"` and nothing else. Measured, and it does not
//! have to be trusted: if the order ever flipped, the seeding below would find
//! nothing to seed and report zero, and the assertion script fails on a zero. It
//! cannot pass vacuously.
//!
//! IT IS ALREADY POST-PRUNING WHATEVER HAPPENS. The ENGINE deletes all but one
//! belt-connectable per tile at LOAD, before any script of any mod runs, with no
//! log line at all -- measured, m2's 77 part tiles came back with 77 interfaces
//! of the ~140 the save was built with. Nothing here can see the world as the
//! 2.0 binary left it, and the assertions are written knowing that.
//!
//! # Why it seeds the networks, which is the part that looks like cheating
//!
//! The fixtures are `--create` saves. A `--create` never reaches a tick, so the
//! rigs were built and the save was written with every belt in them EMPTY -- and
//! a migration that recovers nothing, spills nothing and conserves nothing
//! trivially would satisfy every count in this suite while proving none of them.
//!
//! So the one moment before the migration is also where the items go in: one
//! item into every transport line of every entity this mod's compiler places, on
//! every surface. That is not a stand-in for a running balancer's contents, it
//! is better than one for this purpose -- it is a KNOWN NUMBER, so "what the
//! teardown recovered" can be asserted as an equality against it rather than as
//! a floor.
//!
//! # The two exports that differ from the Go original, and neither is a choice
//!
//! `_initialize` rather than a package initialiser. TinyGo runs Go's `init()`
//! from `_initialize`, which `control.lua` calls on EVERY LOAD; Rust has no
//! pre-main initialiser in a cdylib reactor, so the guest exports the hook
//! itself. The subscription must not go in `fk_on_init`, which fires only when a
//! save is CREATED -- and this suite has no `--create` phase at all, so a
//! subscription made there would never be made.
//!
//! `#[no_mangle] pub extern "C"` rather than `//go:wasmexport`, which is the
//! same statement in the other language's spelling.

#![no_std]

extern crate alloc;

mod harness;
mod line;

use alloc::string::String;
use alloc::vec::Vec;

use fkapi::{
    read_on_tick, ChunkPosition, EntitySearchFilters, LuaEntity, LuaForce, LuaItemStack, LuaSurface,
    LuaTransportLine, Object, Value, EVENT_ON_TICK, GAME,
};

use harness::{fatal_call, Xy};
use line::Line;

const HIDDEN_SURFACE: &str = "bbb-hidden";
const SEED_ITEM: &str = "iron-plate";
const PART_NAME: &str = "bbb-balancer-part";

/// Everything the mod's compiler places, wherever it is: the hidden network
/// proper and the edge interfaces standing on the visible part tiles. Both are
/// drained by a teardown, so both are seeded.
const OURS: [&str; 4] = [
    "bbb-linked-belt",
    "bbb-belt",
    "bbb-splitter",
    "bbb-lane-splitter",
];

/// One `Line` per observer, package level, with the tag baked in -- the Go
/// original's `var out = harness.Line{Tag: "[MIG21] "}`.
///
/// A `static mut` behind `addr_of_mut!` because this is a single-threaded wasm
/// reactor with no concurrency to protect against, and because a 1 KiB buffer
/// per log line on the stack is what the alternative costs. `out()` is the one
/// place that unsafety lives.
static mut OUT: Line = Line::new("[MIG21] ");

#[allow(clippy::mut_from_ref)]
fn out() -> &'static mut Line {
    unsafe { &mut *core::ptr::addr_of_mut!(OUT) }
}

/// The `name = OURS` list every `find_entities_filtered` here passes: a name
/// filter may be a LIST, and the engine applies it in C++.
fn ours_filter() -> Value {
    let mut vs = Vec::with_capacity(OURS.len());
    for n in OURS.iter() {
        vs.push(Value::Str((*n).into()));
    }
    Value::Array(vs)
}

/// `pairs(game.surfaces)`, which yields the engine's own order -- index order in
/// practice, and the order every sample line in this suite is recorded in.
///
/// The pair's KEY is the surface NAME and not its index, whatever the type
/// declares; this observer never reads the key, and reads `LuaSurface::name`
/// off the handle instead, which is what the Go original did too.
fn surfaces() -> Vec<(Value, Object)> {
    match GAME.surfaces() {
        Ok(all) => all,
        Err(_) => {
            fatal_call("reading game.surfaces");
            Vec::new()
        }
    }
}

fn find_ours(s: &LuaSurface) -> Vec<Object> {
    let f = ours_filter();
    match s.find_entities_filtered(EntitySearchFilters {
        name: Some(f),
        ..Default::default()
    }) {
        Ok(found) => found,
        Err(_) => {
            fatal_call("finding this mod's entities");
            Vec::new()
        }
    }
}

// ---------------------------------------------------------------------------
// the seed
// ---------------------------------------------------------------------------

/// Puts one item on every transport line of one entity.
///
/// `insert_at_back` REPORTS WHETHER IT LANDED. An empty line always takes one,
/// so on these saves this is one per line and the count is exact; a line that
/// somehow already held something at the back is skipped and not counted, which
/// keeps the number honest rather than optimistic.
fn seed_entity(o: Object) -> i64 {
    let e = LuaEntity(o);
    let count = match e.get_max_transport_line_index() {
        Ok(c) => c,
        Err(_) => return 0,
    };
    let stack = Value::Map(alloc::vec![
        (Value::Str("name".into()), Value::Str(SEED_ITEM.into())),
        (Value::Str("count".into()), Value::Number(1.0)),
    ]);
    let mut n: i64 = 0;
    for i in 1..=count {
        let line = match e.get_transport_line(i) {
            Ok(l) => l,
            Err(_) => continue,
        };
        if let Ok(true) = LuaTransportLine(line).insert_at_back(&stack, None) {
            n += 1;
        }
    }
    n
}

/// `#line` summed over one entity's transport lines -- the number of items
/// standing on it. `LuaTransportLine::length` is the bound form of Lua's length
/// operator, which is what the Lua used.
fn line_items(o: Object) -> i64 {
    let e = LuaEntity(o);
    let count = match e.get_max_transport_line_index() {
        Ok(c) => c,
        Err(_) => return 0,
    };
    let mut n: i64 = 0;
    for i in 1..=count {
        let line = match e.get_transport_line(i) {
            Ok(l) => l,
            Err(_) => continue,
        };
        if let Ok(l) = LuaTransportLine(line).length() {
            n += l as i64;
        }
    }
    n
}

fn seed_all() {
    let (mut hidden, mut visible): (i64, i64) = (0, 0);
    for entry in surfaces().iter() {
        let s = LuaSurface(entry.1);
        let name = match s.name() {
            Ok(n) => n,
            Err(_) => continue,
        };
        let is_hidden = name.as_bytes() == HIDDEN_SURFACE.as_bytes();
        for o in find_ours(&s).iter() {
            let n = seed_entity(*o);
            if is_hidden {
                hidden += n;
            } else {
                visible += n;
            }
        }
    }
    out()
        .open("seeded hidden=")
        .i(hidden)
        .s(" visible=")
        .i(visible)
        .s(" total=")
        .i(hidden + visible)
        .end();
}

// ---------------------------------------------------------------------------
// the samples
// ---------------------------------------------------------------------------

/// WHAT THE RIGS HAVE DELIVERED, which only matters on one engine and is
/// reported on both.
///
/// On Factorio 2.1 every balancer in these fixtures is refused and its remnant
/// torn down, so this number moves only because the rigs' own bare control belts
/// and pass-through lines are still running -- it is reported and not asserted.
/// On 2.0 nothing is torn down at all: the clusters are ADOPTED whole and the
/// grandfather pass keeps them working, and "keeps working" has to mean items
/// arriving somewhere rather than merely entities standing. Every sink in these
/// worlds is an ordinary chest; the infinity chests are the SOURCES and are
/// excluded, because their contents are held at a filter level and say nothing.
fn delivered() -> i64 {
    let mut total: i64 = 0;
    for entry in surfaces().iter() {
        let s = LuaSurface(entry.1);
        let found = match s.find_entities_filtered(EntitySearchFilters {
            r#type: Some(Value::Str("container".into())),
            ..Default::default()
        }) {
            Ok(f) => f,
            Err(_) => continue,
        };
        for o in found.iter() {
            let n = harness::inventory_total(*o);
            if n > 0 {
                total += n;
            }
        }
    }
    total
}

/// Totals every `item-on-ground` stack on one surface.
///
/// NO AREA, which means the WHOLE surface. This observer builds no world and has
/// no box to sweep: the fixtures' rigs are wherever a 2.0 binary put them, and a
/// spill can land anywhere `spill_item_stack` reaches.
fn ground_items(s: &LuaSurface) -> i64 {
    let found = match s.find_entities_filtered(EntitySearchFilters {
        name: Some(Value::Str("item-on-ground".into())),
        ..Default::default()
    }) {
        Ok(f) => f,
        Err(_) => {
            fatal_call("sweeping the ground");
            return 0;
        }
    };
    let mut ground: i64 = 0;
    for o in found.iter() {
        let st = match LuaEntity(*o).stack() {
            Ok(s) => s,
            Err(_) => continue,
        };
        if let Ok(n) = LuaItemStack(st).count() {
            ground += n as i64;
        }
    }
    ground
}

fn sample(tag: &str) {
    let (mut tot_parts, mut tot_iface, mut tot_stacked, mut tot_ground): (i64, i64, i64, i64) =
        (0, 0, 0, 0);
    let (mut tot_hidden, mut tot_hitems, mut tot_vitems): (i64, i64, i64) = (0, 0, 0);

    for entry in surfaces().iter() {
        let s = LuaSurface(entry.1);
        let name_raw = match s.name() {
            Ok(n) => n,
            Err(_) => continue,
        };
        let name = name_raw.to_string_lossy();
        let parts = match s.count_entities_filtered(EntitySearchFilters {
            name: Some(Value::Str(PART_NAME.into())),
            ..Default::default()
        }) {
            Ok(p) => p,
            Err(_) => {
                let mut what = String::from("counting parts on ");
                what.push_str(&name);
                fatal_call(&what);
                0
            }
        };
        let found = find_ours(&s);
        let ground = ground_items(&s);

        if name_raw.as_bytes() == HIDDEN_SURFACE.as_bytes() {
            // The hidden surface: the networks themselves, and what is standing
            // in their transport lines. That second number is where a teardown's
            // recovered items come FROM, and it is what has to reach zero for a
            // network this mod has condemned.
            let mut items: i64 = 0;
            for o in found.iter() {
                items += line_items(*o);
            }
            tot_hidden += found.len() as i64;
            tot_hitems += items;
            out()
                .open("tag=")
                .s(tag)
                .s(" surface=")
                .s(&name)
                .s(" hidden_entities=")
                .i(found.len() as i64)
                .s(" hidden_items=")
                .i(items)
                .s(" ground=")
                .i(ground)
                .end();
            continue;
        }

        // A visible surface: the parts a player can see, the edge interfaces
        // standing on their tiles, and whether any TILE carries two
        // belt-connectables -- which is the thing 2.1 forbids and the engine has
        // already dealt with by the time anything here can look.
        let (mut stacked, mut worst, mut items): (i64, i64, i64) = (0, 0, 0);
        let mut tiles = TileCounts::new();
        for o in found.iter() {
            let pos = match LuaEntity(*o).position() {
                Ok(p) => p,
                Err(_) => continue,
            };
            tiles.add(floor_int(pos.x), floor_int(pos.y));
            items += line_items(*o);
        }
        for n in tiles.n.iter() {
            if *n > 1 {
                stacked += 1;
            }
            if *n as i64 > worst {
                worst = *n as i64;
            }
        }
        tot_parts += parts as i64;
        tot_iface += found.len() as i64;
        tot_stacked += stacked;
        tot_ground += ground;
        tot_vitems += items;
        out()
            .open("tag=")
            .s(tag)
            .s(" surface=")
            .s(&name)
            .s(" parts=")
            .i(parts as i64)
            .s(" ours=")
            .i(found.len() as i64)
            .s(" stacked_tiles=")
            .i(stacked)
            .s(" worst_per_tile=")
            .i(worst)
            .s(" iface_items=")
            .i(items)
            .s(" ground=")
            .i(ground)
            .end();
    }

    // `delivered()` is called MID-CHAIN, which is the Go original's shape and is
    // sound here rather than merely copied: it makes host calls while this line
    // is half-built, but it cannot touch `OUT`. Every error path it can reach
    // goes to `fatal_call`, which writes `FATAL` -- a different static with a
    // different tag -- so there is no aliasing to reason about. The line builder
    // holds no host resource between `open` and `end`; it is bytes in a buffer.
    out()
        .open("total tag=")
        .s(tag)
        .s(" parts=")
        .i(tot_parts)
        .s(" ours=")
        .i(tot_iface)
        .s(" stacked_tiles=")
        .i(tot_stacked)
        .s(" ground=")
        .i(tot_ground)
        .s(" hidden_entities=")
        .i(tot_hidden)
        .s(" hidden_items=")
        .i(tot_hitems)
        .s(" iface_items=")
        .i(tot_vitems)
        .s(" delivered=")
        .i(delivered())
        .end();
}

/// The Lua's `by_tile` table: how many of this mod's entities stand on each tile
/// of one surface.
///
/// A VEC AND A LINEAR SCAN, not a `BTreeMap`, and the Go original's reasoning
/// carries over unchanged: the biggest fixture here has 95 part tiles, the
/// answer is a MULTISET the caller folds to two numbers, and a vector is the
/// shape whose determinism needs no argument. (Rust would have given a
/// deterministic walk from a `BTreeMap` for free where Go would not -- see
/// FkLua's "A DICTIONARY IS NEVER A GO MAP" -- so this is fidelity to the
/// original rather than necessity. Insertion order is what the fold sees, and
/// the fold is order-independent anyway.)
struct TileCounts {
    xy: Vec<Xy>,
    n: Vec<i32>,
}

impl TileCounts {
    fn new() -> Self {
        TileCounts {
            xy: Vec::new(),
            n: Vec::new(),
        }
    }

    fn add(&mut self, x: i64, y: i64) {
        for (i, p) in self.xy.iter().enumerate() {
            if p.x == x && p.y == y {
                self.n[i] += 1;
                return;
            }
        }
        self.xy.push(Xy { x, y });
        self.n.push(1);
    }
}

/// `math.floor` on a coordinate. A tile centre is x.5, so a plain truncation
/// would be wrong for the negative half of every fixture.
///
/// THE GO ALGORITHM, TRANSCRIBED, and not `f64::floor` -- which is `std` and
/// unavailable in a `no_std` guest without pulling in `libm`. Truncate-toward-
/// zero then step down for a negative non-integer is exactly what the Go
/// original computes, and `as` on a float in range truncates toward zero in both
/// languages.
fn floor_int(v: f64) -> i64 {
    let i = v as i64;
    if v < 0.0 && (i as f64) != v {
        i - 1
    } else {
        i
    }
}

// ---------------------------------------------------------------------------
// the audit
// ---------------------------------------------------------------------------

/// Places the shipped marker, which is the only SYNCHRONOUS "re-classify the
/// world and report" trigger there is: placing one drains the mod's deferred
/// queue inside this dispatch, so the `[BBB] audit` line it writes describes the
/// world at this tick rather than at some tick after it.
///
/// SURFACE 1, TILE (0,0), and neither is this observer's choice: it builds no
/// world, so the only surface it can be sure of is the fixture's first.
fn audit(tag: &str) {
    let all = surfaces();
    if all.is_empty() {
        fatal_call("no surfaces to audit on");
        return;
    }
    harness::audit(&LuaSurface(all[0].1), 0, 0);
    out().open("audited tag=").s(tag).end();
}

// ---------------------------------------------------------------------------
// the chart tripwire
// ---------------------------------------------------------------------------

/// WHETHER THE FORCE CAN SEE THE GROUND ITS BALANCERS STAND ON, AND THE WALL
/// THAT MAKES THAT UNANSWERABLE HERE.
///
/// A `[gps=]` is a coordinate and nothing else: clicking one opens the map there
/// whether or not the force has charted it, and an uncharted point is BLACK. So
/// the mod charts what it pings, and the obvious check is `is_chunk_charted`
/// after the message.
///
/// IT ANSWERS FALSE FOR EVERYTHING ON A HEADLESS RUN. Measured and not assumed:
/// with no players, `force.chart` charts nothing, `force.chart_all` over a fully
/// generated surface charts nothing, a radar charts nothing, and NAUVIS'S OWN
/// ORIGIN CHUNK -- which every real game charts at world creation -- reads
/// uncharted too. A force with no players has no chart to write into.
///
/// SO THIS IS A TRIPWIRE, NOT A MEASUREMENT OF THE FIX: it reports zero before
/// and zero after, and test/assert-mig21.py asserts exactly that -- so the day a
/// Factorio charts headlessly the run fails and asks for the real assertion
/// instead of this one.
///
/// ONE CHUNK PER PART TILE, counted per surface. `is_chunk_charted` takes a
/// CHUNK position, so a part at tile (x, y) is asked about at (floor(x/32),
/// floor(y/32)); several parts share a chunk and the count is over DISTINCT
/// chunks, which is what makes the samples comparable.
fn chart_state(tag: &str) {
    let forces = match GAME.forces() {
        Ok(f) => f,
        Err(_) => {
            fatal_call("reading game.forces");
            return;
        }
    };
    let players = match GAME.players() {
        Ok(p) => p,
        Err(_) => {
            fatal_call("reading game.players");
            return;
        }
    };
    let nauvis = match GAME.get_surface(&Value::Str("nauvis".into())) {
        Ok(Some(n)) => n,
        _ => {
            fatal_call("no nauvis");
            return;
        }
    };

    for fe in forces.iter() {
        let force = LuaForce(fe.1);
        let index = match force.index() {
            Ok(i) => i,
            Err(_) => continue,
        };

        // Build the per-surface list first: the line is only written for a force
        // that owns a part somewhere, which is the Lua's `#per_surface > 0`.
        let mut list: Vec<(String, i64, i64)> = Vec::new();
        for entry in surfaces().iter() {
            let s = LuaSurface(entry.1);
            let name_raw = match s.name() {
                Ok(n) => n,
                Err(_) => continue,
            };
            if name_raw.as_bytes() == HIDDEN_SURFACE.as_bytes() {
                continue;
            }
            let found = match s.find_entities_filtered(EntitySearchFilters {
                name: Some(Value::Str(PART_NAME.into())),
                force: Some(Value::Obj(fe.1)),
                ..Default::default()
            }) {
                Ok(f) => f,
                Err(_) => continue,
            };
            let mut seen = TileCounts::new();
            let mut charted: i64 = 0;
            for o in found.iter() {
                let pos = match LuaEntity(*o).position() {
                    Ok(p) => p,
                    Err(_) => continue,
                };
                let (cx, cy) = (floor_int(pos.x / 32.0), floor_int(pos.y / 32.0));
                let before = seen.xy.len();
                seen.add(cx, cy);
                if seen.xy.len() == before {
                    continue; // a chunk already asked about
                }
                if let Ok(true) = force.is_chunk_charted(
                    entry.1,
                    ChunkPosition {
                        x: cx as i32,
                        y: cy as i32,
                    },
                ) {
                    charted += 1;
                }
            }
            let n = seen.xy.len() as i64;
            if n > 0 {
                list.push((name_raw.to_string_lossy().into_owned(), charted, n));
            }
        }
        if list.is_empty() {
            continue;
        }

        // THE CONTROL, in the same line: nauvis's origin chunk, generated by
        // world creation and charted by nothing, for the same force.
        // BUILT TWICE, not moved twice: `ChunkPosition` is not `Copy` -- it is a
        // generated struct and the generator derives no `Copy` on any of them --
        // where the Go original passed one value to both calls. Two literals is
        // the transcription of what Go did, and is cheaper than the `Clone` the
        // borrow checker would otherwise want.
        let nau_surface = LuaSurface(nauvis);
        let mut nau_ok = false;
        if let Ok(true) = nau_surface.is_chunk_generated(ChunkPosition { x: 0, y: 0 }) {
            if let Ok(c) = force.is_chunk_charted(nauvis, ChunkPosition { x: 0, y: 0 }) {
                nau_ok = c;
            }
        }

        let l = out();
        l.open("chart tag=").s(tag).s(" force=").u(index as u64);
        for p in list.iter() {
            l.s(" ").s(&p.0).s(":").i(p.1).s("/").i(p.2);
        }
        l.s(" nauvis_origin=")
            .b(nau_ok)
            .s(" players=")
            .i(players.len() as i64)
            .end();
    }
}

// ---------------------------------------------------------------------------

/// Where a Rust guest's subscriptions go, and NOT `fk_on_init`.
///
/// `script.on_init` fires once, when a save is CREATED; `_initialize` is called
/// by `control.lua` on every load. This suite has no `--create` phase at all --
/// its worlds are committed fixtures a 2.0 binary built -- so a subscription
/// made in `fk_on_init` would never be made here, and the tick handler below
/// would simply never run.
///
/// TinyGo gave the Go original this for free by running package `init()` from
/// `_initialize`. Rust has no pre-main initialiser in a cdylib reactor.
#[no_mangle]
pub extern "C" fn _initialize() {
    let _ = fkapi::subscribe(EVENT_ON_TICK);
}

/// BEFORE, as near as any script can get to it -- and the one moment the
/// networks can be given something to lose. See the module header for both.
#[no_mangle]
pub extern "C" fn fk_on_configuration_changed() {
    seed_all();
    sample("cfg");
    chart_state("cfg");
}

#[no_mangle]
pub extern "C" fn fk_on_event(id: u32, ptr: u32) {
    if id != EVENT_ON_TICK {
        return;
    }
    match read_on_tick(ptr).tick {
        1 => {
            // The migration's own flush lands on the first deferred tick, so
            // this is the first moment everything it does has happened.
            sample("t1");
            chart_state("t1");
            audit("t1");
        }
        2 => {
            // One tick later, because the audit above forces a flush of its own
            // and what matters is that the state settled rather than that the
            // audit ran.
            sample("post-audit");
        }
        300 => {
            // And a long way after nothing has been touched: a refused cluster
            // has to be a STABLE state, not one that oscillates between teardown
            // and rebuild.
            sample("final");
            chart_state("final");
            audit("final");
        }
        _ => {}
    }
}
