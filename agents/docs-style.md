# House style for human-facing documentation

**Read before creating or editing any document a reader outside this project will
see**: `README.md`, `FKLUA-GAPS.md`, `bench/README.md`, `bench/baselines/*.md`,
`test/interactive/README.md`, and any future `docs/`. It does not apply to
`CLAUDE.md`, `agents/*.md`, `.claude/`, prompt files, code comments, or generated
files; those are working notes and keep their own voice.

The distinction matters because the two audiences want opposite things. Working notes
are for the people building the project and are allowed to be dense, dated,
opinionated and full of internal shorthand. Human-facing documents are for a competent
developer who has never seen the repository, does not have the author's machine, and
cannot ask a question. Every rule below follows from that.

## Audience and voice

- Write for a public GitHub reader. Plain, direct, technical prose. Second person
  ("you") for instructions is fine.
- Describe what exists, how to use it, and what was measured. Do not narrate how the
  work was done, who did it, or how many attempts it took.
- No internal-monologue asides ("and that is the honest headline", "worth reading
  twice", "the lesson here is", "which is the whole point"). Say the thing once, well.
- Keep the precision the working notes have about numbers and conditions. A measured
  figure keeps its environment (engine version, hardware, date measured) when it backs a
  claim the reader needs.

## What must not appear

- **Local absolute paths** (`/Users/...`, `~/Library/...` presented as the reader's).
  Describe generically ("a local FkLua checkout", "your Factorio user directory") or
  link to GitHub. Sibling-checkout links like `../FkLua` become GitHub URLs (the one exception is the
  build requirement itself, which really is a sibling checkout at `../FkLua` and may be
  stated as such):
  `https://github.com/Techrocket9/<Repo>` for a repo,
  `https://github.com/Techrocket9/<Repo>/blob/master/<path>` for a file.
- **Change-history narrative** ("until 2026-08-07 it did X", "since round B2",
  "fixed 2026-08-01", strikethrough of superseded text). State the current behaviour.
- **Internal milestone and round codes** (M0-M12, "round A/B1a", "stage C", "P6") used as
  if the reader knows them. A ledger may keep its finding IDs as an index if it explains
  the scheme once at the top.
- **Test-function roll calls** ("*Enforced by TestFooBar and TestBaz*"). Developer docs
  may say something is covered by a test; READMEs do not name test functions.
- **Instructions aimed at a future maintainer or agent** ("Read before touching X",
  "do not re-try this", "a later change must not forget"). Those belong in `CLAUDE.md`
  or `agents/`.
- **Process commentary** (which gate went red, "confirmed to fail before the fix",
  "user-directed", "approved with amendments", who was orchestrating what).
- **Attribution to AI assistants, agents, an orchestrator, "the user" or "the
  operator"** as actors in the work.
- **Anything a public reader must not see**: secrets, private hostnames, account or
  instance identifiers, and content whose licence forbids public distribution.

## Formatting rules

- **No em-dashes (`—`), anywhere.** Restructure with a period, comma, colon or
  parentheses.
- **No en-dashes (`–`) as separators or in ranges.** Ranges use a plain hyphen:
  `2-3 ms`, `81-95%`, `2018-2026`. Negative numbers use ASCII `-`, not U+2212.
- The middle dot (`·`) at most once per line, never as a general separator; prefer
  columns, commas or line breaks.
- No numbered eyebrow labels (`01 / Overview`, `Step 1:`, `Stage 2:`, `Phase 03`).
  Headings name the topic. Numbered lists for genuinely sequential steps are fine.
- No filler verbs or marketing adjectives (elevate, seamless, unleash, next-gen,
  revolutionize, blazing, powerful).
- No strikethrough (`~~`).
- Unicode that is fine: `×` for ratios, `≥ ≤ ≈`, superscripts such as `2³²`, `µs`,
  `→` inside tables and diagrams. Prefer ASCII where it reads as well.
- Every document starts with a `# Title` and one or two sentences saying what it is,
  and ends with a trailing newline.

## Structure by document type

- **Top-level README**: what it is (two or three paragraphs), status, requirements,
  quickstart, what it can and cannot do, repository layout and where the docs are,
  licence. Concise and scannable. Deep detail links out.
- **Sub-README** (a benchmark harness, a test rig, a lab directory): what it is, how to
  run it, how to read its output. No history.
- **Findings or gaps ledger**: explain the ID scheme once; one labelled entry per
  finding with what was expected, what happened, current status and how it was
  resolved. Reproductions only where short and still meaningful.
- **Milestone or investigation report**: lead with what was built or found and the
  measured result; keep design decisions and numbers; drop the gate-by-gate narrative.
- **Research note**: the question, the candidates and criteria, the method and
  environment, the measurements, the decision, the open questions.

## Licence statements

BetterBeltBalancer is released under the MIT License (`LICENSE` at the repository root),
and so is FkLua, which compiles it. Say so plainly, link the file, and do not imply
anything else anywhere.

## Cross-references

Link generously between the related repositories (FkLua, BetterBeltBalancer,
fklua-ports-samples, Vibetorio) using the GitHub URL scheme above. Within a repo use relative
links, and confirm the target exists before committing. A public document may link to
`agents/*.md` as "maintainer design notes" where a reader would want the depth, but it
must stand on its own without them.

## The check before committing

Run these over every human-facing file you touched and fix every hit:

```sh
LC_ALL=en_US.UTF-8 grep -n '[—–−]' FILE          # em dash, en dash, unicode minus
grep -n '/Users/' FILE                          # local paths
grep -niE 'claude|opus|orchestrat|the user|the operator|this session' FILE
grep -n '~~' FILE                               # strikethrough
awk '{n=gsub(/·/,"·"); if(n>1) print FILENAME":"FNR}' FILE   # more than one middle dot
```

and check every relative link target exists. Documentation drift is a gate failure
here as everywhere else in this repo: a human-facing document is updated in the same
commit as the change it describes, and it is updated in this style.
