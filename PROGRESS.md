# rslp — Progress & Roadmap

> Read `CONTEXT.md` first for the mission and Claude's tutor role.
> This file tracks **where we are**. Update the "YOU ARE HERE" marker as we move.

## Roadmap (one phase at a time — do not jump ahead)
1. **Scaffold + parse a single `.slp` file into structured Go fields.**  ← current phase
2. Define table schema; naively ingest all replays into a flat file; read back.
3. Storage layer: fixed-size pages + a pager that reads/writes pages on demand.
4. B+tree index on top of the pager.
5. Execution operators: scan, filter (WHERE), aggregate (GROUP BY/COUNT/AVG); one query end-to-end.
6. SQL parser (recursive descent) → AST → query plan from the operators.
7. (Optional) Make analytical queries fast — columnar storage / zone maps (OLTP vs OLAP).

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

**▶ Next rep (building blocks already given to Joey):** Slice 1 tracer bullet — serialize
the ONE existing `Game` to disk before scaling.
1. Capture the result in a var: `game := toGame(replay)` (currently `main` prints it inline
   without keeping it — needs a variable).
2. `data, err := json.Marshal(game)` — mirror of `Unmarshal`; struct → `[]byte`. Guard err.
3. `err = os.WriteFile("data/games.jsonl", data, 0644)` — guard err. (`0644` = Unix perms;
   may need to `mkdir data` first. New import: `"os"`.)
4. Verify with `cat data/games.jsonl`. Then scale to the directory walk (Slice 1 step 2).

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
