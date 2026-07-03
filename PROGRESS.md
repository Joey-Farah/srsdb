# rslp — Progress & Roadmap

> Read `CONTEXT.md` first for the mission and Claude's tutor role.
> This file tracks **where we are**. Update the "YOU ARE HERE" marker as we move.

## Roadmap (one phase at a time — do not jump ahead)
1. **Scaffold + parse a single `.slp` file into structured Go fields.**
2. Define table schema; naively ingest all replays into a flat file; read back.  ← current phase
3. Storage layer: fixed-size pages + a pager that reads/writes pages on demand.
4. B+tree index on top of the pager.
5. Execution operators: scan, filter (WHERE), aggregate (GROUP BY/COUNT/AVG); one query end-to-end.
6. SQL parser (recursive descent) → AST → query plan from the operators.
7. (Optional) Make analytical queries fast — columnar storage / zone maps (OLTP vs OLAP).

## Overall progress — ~15% (effort-weighted; excludes optional Ph7)
Phases are NOT equal size — the engine core (Ph3–6) is the bulk. Weighting reflects that.

| Phase | Weight | Status | Done |
|---|---|---|---|
| Ph1 parse `.slp` → `Game` | 8% | ✅ complete | 8% |
| Ph2 flat-file ingest + read-back | 7% | ✅ complete | 7% |
| Ph3 pages + pager | 20% | 🟡 design in progress | 0% |
| Ph4 B+tree | 25% | ⬜ not started | 0% |
| Ph5 operators | 18% | ⬜ not started | 0% |
| Ph6 SQL parser → plan | 22% | ⬜ not started | 0% |
| **Total** | **100%** | | **~15%** |

> Update the % and status as phases close. The weighting is a rough planning estimate,
> not a precise measure — the point is to see the engine core (Ph3–6, 85% of the work) is
> still ahead.

---

## 🟡 PHASE 3 — pages + pager (DESIGN GRILLING IN PROGRESS — no code yet)

The real start of the engine. Replaces the throwaway `data/games.jsonl` flat file with a
proper storage layer. **Joey writes the pager** (explicitly his — off-limits for Claude).

### Design locked so far (Joey reasoned each out, in his own words)
- **Why fixed-size pages:** variable-length records (JSON lines of differing byte length)
  have no predictable position → finding record #k forces a full scan. Fix: carve the file
  into equal **4096-byte pages**; page #k lives at byte offset **`k × 4096`** → `seek`
  straight to it, O(1), no scan. Traded variable-length *records* for fixed-length *pages*.
- **Slotted page (what's inside a page):** records pack in from one end; a **slot directory**
  grows from the other end. Each slot = **(byte offset, length)** of one record. Slots are
  fixed-size → randomly addressable (same trick, one level down). Lets a record move within
  the page without anything outside caring; slot number = stable address. (Joey's own
  intuition reached for this — "mini page-like structures within each page.")
- **Pager is dumb about games:** it moves raw bytes in fixed blocks; knows nothing about
  `Game`/players/stages. Layering — the slotted-page / B+tree layers sit *on top* and decide
  record meaning. Keeps the pager reusable (index pages, free-space maps, etc.).
- **Start cache-less:** v1 pager hits disk every `ReadPage`/`WritePage`. Add the buffer-pool
  cache (page# → in-memory page) LATER. Tracer bullet first: write known bytes to page k,
  read page k back, assert equal.

### Code-org decision (done)
- New **`storage/` package** for the pager (`storage/pager.go`, `package storage`). The pager
  is a *deep module*: tiny surface (`Open`/`ReadPage`/`WritePage`), growing hidden complexity
  (file handle → cache → free list). First real package boundary in the project.
- Import from `main.go` as `import "github.com/Joey-Farah/rslp/storage"` → call
  `storage.ReadPage(...)`. **Currently commented out** (Go errors on unused imports until
  there's something to call).
- **Skip** splitting `game.go` out of `main.go` — that's throwaway Phase 2 code, not worth it.
- Go lessons covered: import path = module path + dir (absolute, never relative; no bare
  `"storage"`); import path resolves from the **local filesystem**, the `github.com/...` name
  is just a label (only matters if someone `go get`s it — repo is `srsdb` yet module is
  `…/rslp` and it builds fine); package *name* (`package storage`) ≠ filename (`pager.go`);
  exported = Capitalized.

### Interface shape — DECIDED: `Pager` struct owns the `*os.File`
Chose the struct-with-methods shape (over free funcs that pass the file each call): the file
handle is owned state, and future state (cache, page count, free list) becomes more struct
fields, not wider param lists. Idiomatic "thing that owns a resource" (mirrors `os.File`).
```go
type Pager struct { file *os.File }
func Open(path string) (*Pager, error)                // ✅ written
func (p *Pager) WritePage(k int, data []byte) error   // ✅ written
func (p *Pager) ReadPage(k int) (?, ?)                // ⬅ next (return shape TBD)
```

### ✅ `Open` DONE (Joey wrote it)
`storage/pager.go`: `os.Open(path)` → `if err != nil { return nil, err }` →
`return &Pager{file: file}, nil`. Builds clean. Go lessons landed:
- `os.Open` returns **two** values (`*os.File, error`) — must capture both (`file, err :=`).
- **Library error idiom** ≠ `main`: a constructor *returns* the error (`return nil, err`) so the
  CALLER decides — it doesn't `log.Fatal` and kill the caller's program. Errors bubble up the
  chain to whoever (eventually `main`) handles them. First slot `nil` = no valid Pager on
  failure; pointers can be `nil`.
- `&Pager{file: file}` builds the struct and takes its address → returns the `*Pager`.
- Also covered: method-receiver syntax `func (p *Pager) Name(...)` — `p` is the receiver
  (like `this`, but you name it); the method reaches the file via `p.file`, which is the whole
  payoff of bundling the handle into the struct (don't pass it each call).
- Process note: Joey (rightly) called out that sketch/placeholder syntax is bad for learning —
  give REAL compilable Go, one declaration at a time, no skipping around.

### ✅ `PageSize` const + `WritePage` DONE (Joey wrote them)
- `const PageSize = 4096`.
- `func (p *Pager) WritePage(k int, data []byte) error`: `offset := int64(k) * PageSize`
  → `_, err := p.file.WriteAt(data, offset)` → `if err != nil { return err }` → `return nil`.
  Builds + vets clean.
- Go lessons landed (lots of them — this was a slow, deep rep):
  - **Method receiver** taught properly for the first time (flagged as NEW — it's the first
    method in the whole codebase). `func (p *Pager) Name(...)` — receiver goes BEFORE the name,
    is like `this`/`self` but named explicitly; it's what makes `p.file` exist inside. Benefit:
    don't pass the struct as a param on every call; also associates the fn with the type →
    later enables interface-based swappable storage (Ph4).
  - Header anatomy: `func (RECEIVER) NAME(PARAMS) RETURN {` — Joey had all four slots scrambled;
    walked through each.
  - **Single return value → parens optional** (`) error` vs `) (*Pager, error)`) — flagged NEW.
  - `WriteAt(b []byte, off int64) (n int, err error)` — offset is **int64** (needs `int64(k)`);
    discard `n` with `_`.
  - **`return nil` = success**, not "return nothing": the `error` return is a success/failure
    REPORT, not data. Dropping the `error` return would silently swallow disk-write failures.
  - `WriteAt` needs no separate Seek — it writes at an absolute offset in one call.

### ▶▶ RESUME HERE — write `ReadPage` (the "get" half, mirror of WritePage)
Two design Qs posed to Joey (answer first, then spec the body):
1. **Return shape?** It *produces* bytes and can fail → return `([]byte, error)`.
2. **Buffer to read into?** `ReadAt` reads into a caller-supplied `[]byte`; size it to `PageSize`
   (`make([]byte, PageSize)`). Mirror of `WriteAt`: `ReadAt(b []byte, off int64) (n, err)`.
**Then:** round-trip test in `storage/pager_test.go` (write known bytes to page k, read back,
assert equal) → only then add the cache. Note the test needs a real/temp file — use
`t.TempDir()` + create the file (Open only *opens* existing; may need an OpenOrCreate later).

---

## ✅ PHASE 1 COMPLETE — parse a single `.slp` → clean `Game`

Full pipeline runs end-to-end and `go vet` is clean:
`slp -s <path>` → JSON bytes → `json.Unmarshal` → `rawReplay` → `toGame()` → `Game`.
Output verified: `{[{Marth P1} {Falco P2}] Dream Land 10110}`.

**What got built (all in `main.go`):**
- `raw*` decode structs mirroring peppi's JSON (`rawReplay/rawStart/rawPlayer/rawMetadata`).
- Clean domain structs `Game{Players []Player, Stage string, Duration int}` +
  `Player{Character string, Port string}`.
- `stageNames` + `characterNames` `map[int]string` lookup tables (Claude-provided data).
- `toGame(replay rawReplay) Game` — the anti-corruption seam: range over players,
  id→name map lookups, `append` into a `[]Player`, assemble the `Game`.

**Modeling decisions locked (Phase 2 may revisit at the storage layer):**
1. Char/Stage as **resolved `string` names** — `Game` is a post-join *result row*, NOT
   the persisted base table. Normalized `stages`/`characters` tables belong to Phase 2.
2. Separate **`Player` struct** (repeating group → its own entity).
3. **`[]Player` slice** (format allows up to 4 ports; singles = `len == 2`).
4. Unknown-id handling: **trust ids for now** (inline lookups, no comma-ok). Revisit at scale.

**Quiz passed (Joey explained back, in his own words):**
- Two struct families = peppi's model vs ours; `toGame` is the wall (decoupling/fit).
  Decode happens at `Unmarshal` (line 94); *translate* happens at `toGame` (line 103).
- `range` yields (index, element); `rp` is the element, `_` discards the index (slots
  are positional — first is always index).
- `append` returns a new slice you MUST reassign — a slice is a {ptr,len,cap} header;
  a full backing array forces a realloc to a new address. (JS `push` mutates in place; Go doesn't.)

### ▶▶ RESUME HERE — Phase 2 (SHRUNK flat-file detour — Joey writes ALL of it)
Deliberately lean "feel the pain" detour before the Phase 3 pager. **Joey writes every
line** (he wants ownership of all final-project code; glue included). Decided scope:
- **Format: JSON-lines** (`encoding/json`, one `Game` per line). File: `data/games.jsonl`.
- **Record: denormalized** whole `Game` per line (players embedded) — naive baseline.
- **Scale: small subset (~100–500 files), NOT all 200k** — feel the pain, don't grind.
- Most of Phase 2 is **throwaway** (replaced by pages in Ph3 / B+tree in Ph4); the
  surviving part is the ingestion glue (walk dir → `slp` → `Game`). Value here = gentle
  on-ramp to Go file I/O before the pager. Joey's SQL background already knows *why*
  scans hurt, so keep it short.

Three thin slices, each demoable:
1. **Write** — start with a tracer bullet: marshal the ONE existing `Game` to JSON
   (`json.Marshal`, the mirror of `Unmarshal`) and write to `data/games.jsonl`. Then scale
   to a directory walk (`os.ReadDir`) over a subset, appending one JSON line per `Game`.
2. **Read** — open the file, decode each line back into a `Game`.
3. **"Query"** — scan all decoded games, filter/count one thing (e.g. Marth games) → feel O(n).

**Slice 1 step 1 — tracer bullet DONE.** One `Game` → `json.Marshal` → `os.WriteFile`
→ `data/games.jsonl`, verified on disk (`{"Players":[...],"Stage":"Dream Land",...}`).
Learned: `Marshal` is the mirror of `Unmarshal` (struct→bytes); `data` is an INPUT to
`WriteFile`, which returns only `err`; `0644` octal → `-rw-r--r--`; no `json` tags on
`Game`/`Player` so keys come out Capitalized (fine — only our code reads it back); WriteFile
TRUNCATES+overwrites (must switch to append when scaling).

**Slice 1 — DONE (write/ingest).** Directory walk over `~/Documents/slp replays/` (708
files) → per-file pipeline (`os.ReadDir` → `filepath.Join` → `slp -s` → `Unmarshal` into a
LOCAL `rawReplay` → `toGame` → `Marshal`) → `append` each JSON to `fileWriteLines []string`
→ after loop `strings.Join(…, "\n")` + one `os.WriteFile`. `go vet` clean.
Learned/observed:
- `os.ReadDir` → `[]os.DirEntry`; `entry.Name()` is filename only → `filepath.Join` for full path.
- Accumulate-in-slice then write-once (chose this over append-mode for simplicity).
- `strings.Join` separates (no trailing `\n`) → `wc -l` shows 707 for 708 games (correct).
- **FELT THE PAIN:** 708 files = ~4.6s, dominated by 708 `slp` subprocess spawns
  (2.16s user + 1.97s sys, NOT Go code). Extrapolates to ~22 min for 200k. Real scaling wall.

**Slice 2 — DONE (read back).** `os.ReadFile` → `strings.Split(…, "\n")` → loop
`json.Unmarshal([]byte(line), &game)` → `append` into `[]Game`. Confirmed 708 games read.
Deep-dived (Joey explained back correctly): bytes↔string conversions = API choice
(`strings.Split` vs `bytes.Split`), not necessity; `Marshal`=encode struct→text (storable),
`Unmarshal`=decode text→struct (queryable); pipeline crosses the text↔struct boundary 3×
(decode slp output → transform via toGame → encode to disk → later decode from disk);
pointer needed only when a func MUTATES your var (`*Type` param defined by author, `&var` at
call) — `Unmarshal`'s param is `any` so caller only writes `&`.

**▶ Next rep — Slice 3 (the query / the payoff). CLOSES PHASE 2.**
Scan the `[]Game`, filter/count ONE thing (e.g. count Marth games): a `for range` over all
708 with an `if` + counter. The point is to FEEL the O(n) full scan — every query reads
EVERY record, no shortcuts — which is exactly the pain Phase 4's B+tree index removes.
After this: quiz on Phase 2, then commit + move to Phase 3 (pages + pager).

### Go habits worth keeping
- Run `go vet ./...` alongside `go run .` (it caught an unexported-field bug early).
- THE Go error idiom: `if err != nil { log.Fatal(err) }` — value, not exceptions;
  each fallible call gets its own guard before the next line clobbers `err`.

**Phase 1 environment notes**
- Go installed: `go 1.26.4 darwin/amd64`. Module: `github.com/Joey-Farah/rslp`.
- Joey learned (and explained back correctly): Go's package model — a *directory* is the
  unit of compilation, all `.go` files share a namespace, so two `func main()` collide.
  `go run .` (whole package) vs `go run main.go` (single file).
- Confirmed sample `.slp` files exist locally (no download needed):
  - `~/Documents/slp replays/` (real replays)
  - `Slippi-Ranked-Stats/TestingSLPFiles/` (test files)

**Current task (Joey's rep — Claude must NOT write this)**
Design the `Game` struct in `main.go`. Target shape:
```
Game { Stage: "Battlefield", Duration: 8423 /*frames*/,
       Players: [ {Char:"Fox", Port:1}, {Char:"Marth", Port:2} ] }
```
Three modeling decisions to justify:
1. One struct or two? (Is "player-in-a-game" its own entity → a `Player` struct?)
2. Two players: fixed `P1`/`P2` vs a slice `[]Player`? (format allows 4 ports)
3. Character/stage as `int` ID, `string` name, or both? (`.slp` stores numeric IDs;
   tradeoff matters across 200k rows) — recall: fields must be Capitalized (exported)
   or `encoding/json` can't fill them.

**Next steps after the struct is reviewed**
- Write a small Node black-box script (Claude may write this) using `@slippi/slippi-js`
  to print `JSON.stringify({ settings, metadata })` for one file.
- Joey writes the Go shell-out: `os/exec` to run the script, `encoding/json` to unmarshal
  into `Game`. Print it. That completes Phase 1.

---

## The plan (agreed — stick to it)
Build **bottom-up**, like the reference book. Tables/schema are the *top* of the stack, not
the start — you can't implement a table until pages + B+tree + KV exist beneath it.

| rslp phase | Build-Your-Own-DB book |
|---|---|
| Ph 1 parse `.slp` → struct | *(not in book — our on-ramp)* |
| Ph 2 naive flat-file ingest | *(not in book — deliberate "feel the pain" detour)* |
| Ph 3 pages + pager | ch. 1, 3 |
| Ph 4 B+tree | ch. 2–5 |
| Ph 5 operators (scan/filter/aggregate) | ch. 9 + execution |
| Ph 6 SQL parser → plan | ch. 13–14 |
| schema / tables design | **ch. 8 (Tables on KV)** |

Key principle: a `Game` struct is **one flattened in-memory record** (a result row), NOT the
persisted schema. Normalization (stage/character lookup tables) is a storage-layer concern
for later, not a struct concern now.

## Open threads / parked ideas
- `scratch.go` was deleted (collided with `main()`); recreate later as its own package
  (e.g. `cmd/scratch/`) when a scratchpad is wanted.
- Phase 2 schema: revisit denormalized-vs-normalized game/player modeling.
