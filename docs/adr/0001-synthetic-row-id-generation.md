# ADR 0001 — Synthetic row-ID generation for the B+tree primary key

**Status:** Accepted
**Context date:** Phase 4 design grill

## Context

The B+tree (Phase 4) needs a key to order rows by. Slippi replays do carry a natural
per-game identity — `start.match = {id, game, tiebreaker}`, where `(match.id, match.game)`
is unique per game. Joey's plan is to eventually load **multiple terabytes** of `.slp`
replays (well beyond the current ~200k-file sample), so whatever scheme is chosen has to
hold up at that scale, not just today's dataset.

Three sub-decisions had to be made: the key itself, the ID width, where the counter is
persisted, and who assigns it. Each is hard to reverse (baked into the on-disk record
format and the B+tree ordering), each was surprising without the storage-layer context
(the natural key looks like the obvious choice at first), and each was a genuine trade-off
between real options — hence one ADR covering all three.

## Decision 1 — Primary key: synthetic `int64` row ID, not `(match.id, match.game)`

| Option | Pros | Cons |
|---|---|---|
| **Natural key: `(match.id, match.game)`** | Meaningful without a lookup; matches how Slippi itself identifies a game; no "minting" logic needed. | `match.id` is a variable-length string (~35 chars) — reintroduces the variable-width problem fixed pages exist to solve. **Not always present** — older/offline replays may lack `match` entirely (confirmed: `hash` was `null` on a sampled file). Slower tree comparisons (string vs. one int). |
| **Synthetic `int64` row ID** ✅ | Fixed-width — trivial to pack into page bytes and compare in the tree. Always present — minted by us, never missing. Cheap comparisons at scale (millions–tens of millions of rows). Mirrors standard DB practice (Oracle `GENERATED AS IDENTITY`). | Meaningless on its own — a lookup by Slippi identity requires a secondary index (deferred to Phase 5+). Adds a "who assigns it" responsibility (see Decision 3). |

**Decision:** synthetic `int64` row ID is the primary/clustering key. `(match.id, match.game)`
is kept as a documented natural-key candidate for a future secondary index, not discarded.

## Decision 2 — ID width: `int64`, not `int32`

| Option | Pros | Cons |
|---|---|---|
| **`int32`** | 4 bytes instead of 8 — marginally smaller records. | Ceiling ~2.1B. At multi-TB scale (millions–tens of millions of games, but growing indefinitely as more replays are added over time) this is real risk, not theoretical. No path to recover without a format migration. |
| **`int64`** ✅ | Effectively no realistic ceiling. Costs 4 extra bytes per row — negligible against a 4096-byte page. | None material — the field is fixed-width either way, so there's no "pay for what you use" argument here. |

**Decision:** `int64`. No close call — free headroom against a plan that explicitly targets
scale growth.

## Decision 3 — Where the counter is persisted: a reserved header page

| Option | Pros | Cons |
|---|---|---|
| **A — reserved header page (page 0 stores `nextID`)** ✅ | O(1) recovery on restart — one `ReadPage(0)`. Durable across crashes (bump + write-back on every assign). Establishes a reusable "metadata page" pattern for later needs (free list, page count) — same trick real engines use (e.g. SQLite reserves page 1 for a header). | The pager itself must stay ignorant of this — page 0's special meaning has to live in the layer above it, not the pager, or the pager stops being a dumb byte-mover. |
| **B — derive `nextID` from existing data on `Open`** | No extra reserved page; nothing to keep in sync. | Requires a full scan (`max(id)+1`) on *every* startup — exactly the O(n) cost the whole pager/B+tree project exists to eliminate. Gets worse as the dataset grows, not better — directly conflicts with the multi-TB goal. |
| **C — in-memory counter only, no persistence** | Trivial to implement. | Breaks on restart: a fresh process starts back at 0/whatever seed, colliding with IDs already on disk. Not viable past a toy/demo. |

**Decision:** A. Reserved header page.

## Decision 4 — Who assigns the ID: storage layer, at write time

| Option | Pros | Cons |
|---|---|---|
| **A — ingest code (`toGame()`/`main.go`) assigns it** | Keeps storage layer simpler (just accepts a fully-formed record). | Puts a *storage* concern in ingest code. If more than one ingest path ever exists (e.g. the parked `.slp`-upload stretch goal), two independent counters can hand out the same ID — silent corruption. |
| **B — storage layer assigns it at insert time** ✅ | Single source of truth — whoever owns the header page is the only thing allowed to increment it, so duplicate IDs are structurally impossible. Mirrors Oracle `IDENTITY`/sequence columns: the DB mints the key, the caller doesn't compute it. | Slightly more work up front — needs a real `Insert()`-shaped operation (not written yet) that owns "read counter → hand out ID → bump + persist → write row" as one unit. |

**Decision:** B. A future storage-layer `Insert` mints and persists the ID; ingest code never
computes one itself.

## Consequences

- The pager (`Open`/`ReadPage`/`WritePage`) is unaffected — it still just moves bytes by page
  number and stays ignorant that page 0 is special.
- The B+tree/storage-engine layer being built next owns: reading/writing the page-0 header,
  minting IDs, and the eventual `Insert` operation.
- `(match.id, match.game)` is not lost — it's recorded here and in `INSIGHTS.md` as the future
  secondary-index candidate.
