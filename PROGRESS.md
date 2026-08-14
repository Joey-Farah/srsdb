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
| Ph3 pages + pager | 20% | 🟡 core done (Open/Write/Read + test) — cache next | 14% |
| Ph4 B+tree | 25% | ⬜ not started | 0% |
| Ph5 operators | 18% | ⬜ not started | 0% |
| Ph6 SQL parser → plan | 22% | ⬜ not started | 0% |
| **Total** | **100%** | | **~29%** |

> Update the % and status as phases close. The weighting is a rough planning estimate,
> not a precise measure — the point is to see the engine core (Ph3–6, 85% of the work) is
> still ahead.

---

## 🟡 PHASE 3 — pages + pager (CORE DONE — Open/WritePage/ReadPage + round-trip test; cache next)

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
func (p *Pager) ReadPage(k int) ([]byte, error)       // ✅ written
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

### ✅ `ReadPage` + read-write `Open` + round-trip test DONE (Joey wrote them) — commit `6bf659b`
- `func (p *Pager) ReadPage(k int) ([]byte, error)`: `buffer := make([]byte, PageSize)` →
  `offset := int64(k) * PageSize` → `_, err := p.file.ReadAt(buffer, offset)` →
  `if err != nil { return nil, err }` → `return buffer, nil`.
- **`Open` upgraded read-only → read-write + create:** `os.Open(path)` →
  `os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)`. The round-trip test *drove* this — a
  read-only handle can't `WriteAt`, and a fresh temp file doesn't exist yet (TDD found the gap).
- **First test in the codebase:** `storage/pager_test.go`, `TestReadWriteRoundTrip` — write
  known bytes to page 3, read back, `bytes.Equal` assert. `go test ./storage/` → green.
- Go lessons landed:
  - **`make([]byte, PageSize)`** — builtin (not user-written); allocates a zeroed slice you own.
    `ReadAt` *fills a buffer you supply* — the data lands in `buffer`, NOT the return value.
  - **`ReadAt` returns `(n int, err error)`** — first value is a byte *count*, not the page;
    discard with `_` (mirror of `WriteAt`).
  - Two-value return type → **every** `return` needs two values (`return nil, err` / `return buffer, nil`).
  - **Test conventions:** `_test.go` files compile only under `go test`; `func TestXxx(t *testing.T)`
    — naming IS the wiring; `t.TempDir()` = auto-cleaned temp dir; `t.Fatalf` (stop) vs `t.Errorf` (continue).
  - **`bytes.Equal(a, b)`** — can't compare slices with `==` (won't compile).
  - **`(cached)`** — `go test` replays a cached pass when nothing changed; `-count=1` forces a re-run.

### ✅ Phase-close quiz PASSED — Joey explained (his words): fixed pages = compute vs scan;
`k × PageSize` = byte offset; pager is a dumb byte-mover, the B+tree index decides *which* page.
The "read pager is the mechanism the index will use" aha landed. Captured in `INSIGHTS.md`.

### ▶▶ RESUME HERE — Phase 4 (B+tree). **Cache SKIPPED by decision.**
Buffer-pool cache (`map[int]*page` in the Pager) is deferred — it's an optimization, not a
correctness gap; revisit only if/when query load makes it worth it. Moving to the B+tree: the
"brain" that maps a key → page number so `ReadPage(k)` is called with a *known* k, never a scan.
Still Joey's code (off-limits for Claude to write). Start with a Phase 4 design grill.

---

## 🟡 PHASE 4 — B+tree (DESIGN GRILLING IN PROGRESS)

### Design locked so far
- **Primary/clustering key: synthetic `int` row ID** (auto-assigned at ingest), NOT a natural
  key. Confirmed by checking real `slp -s` output: Slippi *does* have a natural per-game
  identity — `start.match = {id, game, tiebreaker}`, where `(match.id, match.game)` is unique
  per game (`match.id` = the set/series, `game` = game number within it). Rejected as the
  storage key anyway: `match.id` is a variable-length string (~35 chars, reintroduces the
  variable-width problem fixed pages solve), and it's **not always present** — older/offline
  replays may lack `match` entirely (`hash` was also `null` on a sampled file), so it can't
  anchor a primary key. A synthetic int is fixed-width and always present.
  - `(match.id, match.game)` is kept as a **documented natural-key candidate for a future
    secondary index** (Phase 5+) — "look up this exact Slippi game" — not discarded, just not
    the primary. See `INSIGHTS.md` → "A natural key can exist and still be the wrong storage key."

### ✅ ID-generation design DONE — full tradeoffs in `docs/adr/0001-synthetic-row-id-generation.md`
1. **Width: `int64`** (not `int32`) — free headroom against the multi-TB scale target; no
   realistic downside since the field is fixed-width either way.
2. **Persistence: reserved header page** — page 0 stores `nextID`; `Open` reads it to recover
   the counter (O(1), no full-dataset scan on startup); mirrors SQLite reserving page 1 for a
   header. The pager itself stays ignorant that page 0 is special — that knowledge lives in the
   layer above (B+tree/storage-engine), keeping the pager a dumb byte-mover.
3. **Assignment point: storage layer, at write time** — a future `Insert()` reads the counter,
   hands out the ID, bumps + persists it, then stores the row. Ingest code (`toGame()`) never
   computes an ID itself — mirrors Oracle `IDENTITY`/sequence columns (DB mints the key, caller
   doesn't). Single source of truth → structurally impossible to double-assign, even if a second
   ingest path (e.g. the parked `.slp`-upload stretch) is added later.

### ✅ Leaf-node shape DECIDED: clustered (leaf stores the full row, not a pointer)
Weighed clustered (row data lives directly in the leaf, keyed by row ID) vs. non-clustered
(leaf stores `(rowID, page#+slot)`, real row lives in a separate heap). Chose **clustered**:
matches the project's own reference (*Build Your Own Database From Scratch*), avoids designing
two structures (tree + heap) at once, and fits the current access pattern ("fetch by ID," not
heavy secondary-index scanning — that's a Phase 5+ concern where non-clustered secondary
indexes become worth it).

### ✅ File location DECIDED: `storage/slotted.go`, same `storage` package as `pager.go`
Tightly coupled to the page format the pager already defines (`PageSize`, byte layout); zero
external consumers today; Go packages are cheap to split later if the B+tree ever needs it
independently. Splitting into a new package now would be speculative structure — YAGNI.

### Slotted-page format DECIDED so far
Clustered rows are variable-length (`Stage`/`Character` strings etc.), so a fixed 4096-byte
leaf page needs the **slotted-page** layout designed conceptually back in Phase 3 (records pack
from one end, a slot directory grows from the other end) — deliberately NOT built then (no
consumer yet); now the leaf node is that consumer.
1. **Slot entry = `{offset, length}` only — no delete/tombstone flag.** Domain reasoning (Joey's
   own insight, captured in `INSIGHTS.md`): rslp is an archive of historical Slippi replays —
   games that were played don't get "un-played," so there's no delete workflow to design for.
   Slotted pages *can* support tombstones cheaply, but adding one now designs for an operation
   this database will probably never need. Revisit only if a real delete requirement shows up.
2. **Page header = `numSlots` only** (no separately-stored tail/free-space offset). The slot
   directory's start (and thus free space between the two growing ends) is *derived*:
   `tailOffset = PageSize - numSlots × slotEntrySize`. Storing a second redundant offset risks
   the two numbers drifting out of sync for no benefit.
3. **✅ Slot fields (`offset`, `length`) as `uint16`, not `uint32`.** Unlike the row ID (which
   needed multi-TB headroom because it counts *all rows ever*), these values are bounded by
   `PageSize = 4096` no matter how much data loads — `uint16` (max 65,535) can never overflow.
   `uint32` would waste 2 extra bytes per slot for no future benefit.
4. Serialization will use **`encoding/binary`** (e.g. `binary.LittleEndian.PutUint16`/`Uint16`)
   to pack these fixed-width ints into the page's raw `[]byte` — same idea as `make()` for the
   pager: a stdlib tool, not something Joey writes himself.

### ✅ `Slot` struct + `putSlot`/`getSlot` DONE (Joey wrote them) — `storage/slotted.go`
```go
type Slot struct { Offset uint16; Length uint16 }
func putSlot(pageBuffer []byte, position int, slot Slot)  // encode: 2x binary.LittleEndian.PutUint16
func getSlot(pageBuffer []byte, position int) Slot        // decode: 2x binary.LittleEndian.Uint16
```
Builds clean. Go lessons landed:
- **Function signature vs. function call:** the signature (`PutUint16(dst []byte, v uint16)`)
  describes required types; calling it means substituting real values, never retyping the types
  themselves (Joey initially pasted the signature as if it were a call).
- Slice-arithmetic bug caught: `pageBuffer[position:+2]` ≠ `pageBuffer[position+2:]` — unary `+2`
  is just the literal `2` as the slice's end bound, not `position` plus 2; the first form silently
  produces a backwards/invalid slice instead of shifting the start.
- `PutUint16` returns nothing (writes into the slice you hand it); `Uint16` takes one arg and
  **returns** the decoded value — mirror pair, same relationship as `WriteAt`/`ReadAt`.

### ✅ `putNumSlots`/`getNumSlots` DONE (Joey wrote them) — page header (byte 0)
```go
func putNumSlots(pageBuffer []byte, numSlots uint16)
func getNumSlots(pageBuffer []byte) uint16
```
Simpler than `Slot`: one `uint16` at a fixed spot (byte 0), so no `position` param and only one
`PutUint16`/`Uint16` call each. Purpose: every page needs to self-report how many slots it holds
before anything can read its slot directory or find free space — bytes on disk carry no meaning
on their own. Builds clean. Committed `489b97f` (Slot+putSlot/getSlot); header funcs pending commit.

### ▶▶ RESUME HERE — step 5: insert a record into a page
Given a page buffer + a record's raw `[]byte`: write the record bytes in from the front, write a
new `Slot{offset,length}` via `putSlot` at the back, bump `numSlots` via `putNumSlots`. First
function that actually composes everything built so far. Then step 6 (get a record by slot
number), then the full round-trip test (pack N fake records into one page, read every one back
by slot, assert bytes match) — the actual tracer-bullet payoff. Joey writes the code; Claude may
write the test harness.

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
