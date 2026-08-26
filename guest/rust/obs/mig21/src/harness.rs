//! The slice of `guest/go/obs/harness` that `mig21` actually uses, and nothing
//! else.
//!
//! # Why this is a module and not a crate
//!
//! The Go harness is 1,342 lines because it is shared by FOURTEEN observers, and
//! its own header says what earned it: six-way duplication, measured across the
//! estate before the package was written. There is ONE Rust observer. A
//! `harness` crate here would be structure with no duplication behind it, which
//! is the thing this repository keeps declining to ship -- and the moment a
//! second Rust observer exists, lifting these three functions out is a `git mv`.
//!
//! What is here is what `mig21` calls and it is a short list, because `mig21`
//! BUILDS NO WORLD: there is no scratch surface, no paving, no rig registry, no
//! placement helper, no tile lookup and no schedule. That is the whole reason
//! the estate recon picked this suite as the parity exercise.

use alloc::vec::Vec;

use fkapi::{defines_inventory_chest, LuaControl, LuaInventory, LuaSurface, Object, Value};

use crate::line::Line;

/// The shipped mod's synchronous "drain the deferred queue and re-classify now"
/// trigger. Building one raises `script_raised_built`, which the engine
/// dispatches BEFORE `create_entity` returns -- so a marker placed here reaches
/// the mod under test inside this same dispatch, across two separate wasm
/// instances.
pub const AUDIT_MARKER: &str = "bbb-audit";

/// The force every piece in the estate is built on unless it says otherwise.
pub const PLAYER_FORCE: &str = "player";

/// The observer estate's own error level, and `test/run.sh`'s `guest_gate` fails
/// a run on it.
///
/// A SEPARATE TAG FROM ANY OBSERVER'S. `[BBB-OBS] error:` is one thing for
/// run.sh to grep across the whole estate, in either language, and it cannot be
/// confused with the mod's own `[BBB] error:`, which means something narrower.
static mut FATAL: Line = Line::new("[BBB-OBS] ");

/// Reports that the harness could not do what was asked, and names the host's
/// own reason for it.
///
/// `fk::last_error()` hands back BYTES where the Go `fk.LastError()` hands back
/// a string; the lossy conversion here is the same one `getStr` applies on the
/// Go side, so the two produce the same text for the same host error.
pub fn fatal(what: &str, detail: &str) {
    unsafe {
        let l = &mut *core::ptr::addr_of_mut!(FATAL);
        l.open("error: ").s(what).s(": ").s(detail).end();
    }
}

/// `fatal` with the host's last error as the detail, which is every caller here.
pub fn fatal_call(what: &str) {
    let e = fk::last_error();
    fatal(what, &alloc::string::String::from_utf8_lossy(&e));
}

/// A TILE. Kept as two integers because that is what the Go harness keeps and
/// because nothing here retains a `LuaObject` across a dispatch.
#[derive(Copy, Clone, PartialEq, Eq)]
pub struct Xy {
    pub x: i64,
    pub y: i64,
}

/// Places a `bbb-audit` marker, which asks the mod under test to drain its
/// deferred queue and re-classify the world, synchronously, inside this
/// dispatch.
///
/// IT DELIBERATELY DOES NOT CHECK WHAT COMES BACK. The marker DESTROYS ITSELF
/// from inside the `script_raised_built` that `raise_built = true` dispatches,
/// so by the time `create_entity` returns there is no entity left to hand over:
/// measured on the Go side, the call comes back with no object and no error at
/// all. What says the drain happened is the mod's own `[BBB] audit ...` line
/// landing in the log between this observer's two, which is what the assertion
/// script keys on.
pub fn audit(s: &LuaSurface, x: i64, y: i64) {
    // The same key order the Go harness's `place` emits -- name, position,
    // force, then the optionals it sets. A Lua table has no order, so this is
    // readability rather than protocol, but a transcription is easier to read
    // against its original when it is in the original's order.
    let args = Value::Map(alloc::vec![
        (Value::Str("name".into()), Value::Str(AUDIT_MARKER.into())),
        (
            Value::Str("position".into()),
            Value::Array(alloc::vec![
                Value::Number(x as f64 + 0.5),
                Value::Number(y as f64 + 0.5),
            ]),
        ),
        (Value::Str("force".into()), Value::Str(PLAYER_FORCE.into())),
        (Value::Str("raise_built".into()), Value::Bool(true)),
    ]);
    let _ = s.create_entity(&args);
}

/// Every item in one container's chest inventory, or -1 when it has none.
///
/// THE -1 IS LOAD-BEARING and is the Go harness's own signal: `delivered()`
/// sums only totals ABOVE zero, so a surface entity that is a container by type
/// and has no chest inventory contributes nothing rather than contributing a
/// zero that cannot be told from an empty chest.
pub fn inventory_total(o: Object) -> i64 {
    let inv = match LuaControl(o).get_inventory(defines_inventory_chest()) {
        Ok(Some(i)) => i,
        _ => return -1,
    };
    let contents: Vec<_> = match LuaInventory(inv).get_contents() {
        Ok(c) => c,
        Err(_) => return -1,
    };
    let mut total: i64 = 0;
    for item in contents.iter() {
        total += item.count as i64;
    }
    total
}
