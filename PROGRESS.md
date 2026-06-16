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

## ▶ YOU ARE HERE — Phase 1, step: run `slp` from Go and decode its JSON

**DONE:** `Game`/`Player` domain structs (built/printed earlier); `raw*` decode structs that
mirror peppi's JSON (`rawReplay`/`rawStart`/`rawPlayer`/`rawMetadata`) — `go vet` clean.
Parser chosen: `slp` CLI (peppi-slp), invoked as `slp -s <file>` (skips frames).
Char/stage stored as **int IDs** (peppi external IDs: Marth=9, Falco=20, stage 28=Dream Land).

**Now (Joey's rep) — 3 sub-steps, currently at START of A:**
- **A (next up):** From Go, run `slp -s <path>` and print its raw output. Not started yet.
- **B:** `json.Unmarshal` that output into a `rawReplay`; print to confirm decode works.
- **C:** map `rawReplay` → clean `Game` (re-add Game/Player structs). That ends Phase 1.

Sample file: `/Users/joeyfarah/Documents/slp replays/Game_20260409T184304.slp`

### Resume notes for Step A (building blocks already explained)
New imports needed: `"os/exec"`, `"log"` (plus existing `"fmt"`).
1. Run a command: `out, err := exec.Command("slp", "-s", path).Output()`
   — `out` is stdout as `[]byte`; args bypass the shell, so the space in the path is fine.
2. Go error idiom (THE core pattern — value, not exceptions):
   `if err != nil { log.Fatal(err) }`
3. Print bytes as text: `fmt.Println(string(out))`
Task: in `main()`, set `path`, run `slp -s`, handle err, print raw output.

### Current `main.go` state
- `raw*` decode structs done + `go vet` clean.
- `func main()` is just `fmt.Println()` (blank) — replace its body in Step A.
- Domain `Game`/`Player` structs were removed earlier; re-add them in Step C for mapping.
- Habit to keep: run `go vet ./...` alongside `go run .` (it caught the unexported-field bug).

**Done so far**
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
