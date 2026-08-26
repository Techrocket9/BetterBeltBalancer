//! THE OBSERVER LINE BUILDER, IN RUST, and it is `guest/go/obs/harness/line.go`
//! field for field.
//!
//! It is transcribed rather than reimagined, and that is the whole point of this
//! crate: the gate on phase 8 is that the two observers' log lines are
//! BYTE-IDENTICAL on the same fixtures, so every place the two builders could
//! differ is a place the transcript could differ. `format!` would hide exactly
//! those places behind a format string -- and would pull `core::fmt` into a
//! guest whose entire output is a tag, some names and some decimal integers.
//!
//! THE LINE TEXT IS THE ASSERTION SURFACE. `test/assert-mig21.py` keys every
//! regex on what follows the `[MIG21]` tag, so a space, a separator or a digit's
//! format IS the contract with the assertion script.
//!
//! # What is NOT here, and why that is a measurement
//!
//! `F1` and `F4`, the Go builder's two float formatters. Neither is reachable
//! from this observer: `mig21` logs counts, names and booleans and not one
//! float, which is the property that makes it the right suite to port first --
//! Go's untyped float constants and Rust's `f64` fold at different times, so a
//! computed float in a log line is the one thing a cross-language transcript
//! gate would have to argue about instead of measure. There IS float arithmetic
//! here (a chunk coordinate is `pos.x / 32.0`); it is IEEE 754 double division
//! on both sides and it lands in an integer before it is ever printed. See
//! `floor_int`.

use fk;

/// One observer's log line under construction.
///
/// A FIXED ARRAY, so an over-long line is TRUNCATED rather than grown -- Go's
/// `copy` into a fixed array is a memcpy with no reallocation path behind it,
/// and this is that. 1 KiB for the same reason the Go one is 1 KiB.
pub struct Line {
    /// Written at the head of every line this builder opens, including the
    /// trailing space: `"[MIG21] "`. It is what the assertion script selects on
    /// and it is per observer, never per line.
    pub tag: &'static str,

    buf: [u8; 1024],
    n: usize,
    digits: [u8; 20],
}

impl Line {
    pub const fn new(tag: &'static str) -> Self {
        Line { tag, buf: [0u8; 1024], n: 0, digits: [0u8; 20] }
    }

    /// Starts a line: the tag, then `s`. Anything half-built is discarded.
    pub fn open(&mut self, s: &str) -> &mut Self {
        self.n = 0;
        let tag = self.tag;
        self.s(tag).s(s)
    }

    /// Appends a string, truncating at capacity exactly as Go's `copy` does.
    pub fn s(&mut self, s: &str) -> &mut Self {
        let b = s.as_bytes();
        let room = self.buf.len() - self.n;
        let k = if b.len() < room { b.len() } else { room };
        self.buf[self.n..self.n + k].copy_from_slice(&b[..k]);
        self.n += k;
        self
    }

    /// Appends an unsigned decimal.
    pub fn u(&mut self, mut v: u64) -> &mut Self {
        let mut i = self.digits.len();
        loop {
            i -= 1;
            self.digits[i] = b'0' + (v % 10) as u8;
            v /= 10;
            if v == 0 {
                break;
            }
        }
        // Copied out before `s`, because `digits` and `buf` are both fields of
        // self and the borrow checker will not lend one while the other is
        // borrowed mutably. Go had no such objection and needed no such copy;
        // 20 bytes on the stack is what the difference costs.
        let mut tmp = [0u8; 20];
        let len = self.digits.len() - i;
        tmp[..len].copy_from_slice(&self.digits[i..]);
        let room = self.buf.len() - self.n;
        let k = if len < room { len } else { room };
        self.buf[self.n..self.n + k].copy_from_slice(&tmp[..k]);
        self.n += k;
        self
    }

    /// Appends `"true"` or `"false"`, which is what Lua's `tostring` on a
    /// boolean writes and therefore what the assertion scripts read.
    pub fn b(&mut self, v: bool) -> &mut Self {
        if v {
            self.s("true")
        } else {
            self.s("false")
        }
    }

    /// Appends a signed decimal. Tile coordinates go negative.
    pub fn i(&mut self, v: i64) -> &mut Self {
        if v < 0 {
            self.s("-").u((-v) as u64)
        } else {
            self.u(v as u64)
        }
    }

    /// Hands the buffer to the host.
    ///
    /// Every byte that reaches this buffer arrives either from a `&'static str`
    /// literal, from a decimal digit, or from `LuaStr::to_string_lossy` -- which
    /// is `getStr`'s own behaviour and is what the Go observer's names came
    /// through too -- so the contents are valid UTF-8 by construction and the
    /// check below never fails. It is a check rather than
    /// `from_utf8_unchecked` because being right by construction and being
    /// right by assertion cost the same here.
    pub fn end(&mut self) {
        if let Ok(s) = core::str::from_utf8(&self.buf[..self.n]) {
            fk::log(s);
        }
    }
}
