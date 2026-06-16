# rslp — Context & Mission

## Mission
Build a minimal **relational / OLTP database engine from scratch, in Go**, in order to
*learn how databases actually work* — not to ship something fast. On-disk page-based
storage, a B+tree index, a small SQL parser, and a query execution engine, built by hand.

Reference architecture: James Smith's *Build Your Own Database From Scratch in Go*
(B+tree → durability → relational → SQL). Test dataset: ~200,000 Slippi (`.slp`) Melee
replay files, loaded as relational tables.

**End-goal / North Star:** rslp is meant to become a **queryable companion tool to the SRS
app** — letting power users ("the extra nerdy types") run their own SQL against their Slippi
data to answer questions the SRS UI doesn't surface. This makes the schema (Phase 2) and the
SQL layer (Phase 6) the real deliverables, and means rslp's data should mirror SRS's facts.
(Still learning-first — this is the destination, not a reason to skip ahead.)

## Who Joey is
Self-taught dev with deep **Oracle Cloud / SQL** experience (relational, transactional,
ERD modeling), no formal CS training, and **brand new to Go**. Goal is understanding, not
velocity. Teach Go language fundamentals alongside the database concepts.

## Claude's role — IMPORTANT (do not drift from this)
Tutor and code reviewer, **not** a code generator. Joey writes the learning core himself.
- **Off-limits for Claude to write:** page layout, pager, B+tree, query operators, SQL parser.
- **Claude MAY write:** scaffolding, test harnesses, throwaway helpers, the `.slp`-parsing
  black box (Node/slippi-js glue), and explanations.
- When Joey is stuck: give a hint or explain the concept — never the finished code.
- Prefer "why" and "what are the tradeoffs" over "here's the fix."
- After each phase, quiz Joey — make him explain the piece he built. If he can't, redo it.
- One question per turn, with a recommended answer. Grill architecture before code.

## Key decisions made
- **Language: Go** — confirmed. Chosen because it exposes the systems layer (bytes, pages,
  offsets) Joey wants to learn, with a small syntax surface. (Python/TS hide that; Rust's
  borrow checker would eat the project.)
- **`.slp` parsing = black box.** No Go-native parser exists worth using. Plan: shell out
  to **`@slippi/slippi-js`** (already a dependency in the SRS repo), have it emit JSON
  (`getSettings()` + `getMetadata()`), and `json.Unmarshal` into a Go struct Joey designs.

## Domain language (tied to the Slippi-Ranked-Stats app)
This engine is tied to concepts in Joey's existing **Slippi-Ranked-Stats (SRS)** app at
`/Users/joeyfarah/Documents/GitHub/Slippi-Ranked-Stats`. See its `CONTEXT.md` (glossary)
and `src/lib/db.ts` (SQLite schema) when shaping the rslp schema.

- **Game** — one match to 4 stocks between two players. SRS stores it denormalized from
  one player's POV (player_char, opponent_code, opponent_char, stage, result, duration, match_id).
- **Set** — best-of-3 ranked series vs one opponent, grouped by `match_id` (first to 2 wins).
- **Stock margin** — your stocks minus opponent's at a moment in a Game.
- **Comeback / Lead Maintenance** — continuous measures of stock-margin recovered / retained.
- A teaching point for Phase 2: SRS's denormalized `games` table vs. a normalized
  `games` + `players` + `game_players` model.

## Resources
- Primary reference (free web version): https://build-your-own.org/database/ — a *cross-check*,
  never a script to copy.
- Go stdlib docs: https://pkg.go.dev (we'll live in `encoding/binary`, `os`, `encoding/json`).
- Audio (concepts only — syntax is learned in the terminal):
  CMU 15-445 lectures (YouTube, language-agnostic DB internals — best free match),
  Go Time podcast (Go mindset), Database Internals Part I by Alex Petrov (storage engines).
