# rslp — Architecture

> Visual companion to `CONTEXT.md` (mission) and `PROGRESS.md` (roadmap & status).
> Diagrams use [Mermaid](https://mermaid.js.org/), which GitHub renders natively.
>
> **Two things are drawn here:**
> 1. **The engine stack** — the final layered design, built bottom-up.
> 2. **The two data paths** — how data flows *in* (ingest) and how a query flows *down* and back *up*.
> 3. **Where we are today** — the slice of the above that actually runs right now.

---

## 1. The engine stack (final design, built bottom-up)

A database is a stack of layers. Each layer only talks to the one directly below it, and
hides its complexity from the one above. We build **from the bottom up** — you can't have a
table until pages, a B+tree, and records exist beneath it.

```mermaid
flowchart TB
    SQL["<b>SQL parser</b> — Phase 6<br/>text query → AST → query plan"]
    OPS["<b>Execution operators</b> — Phase 5<br/>scan · filter (WHERE) · aggregate (GROUP BY/COUNT/AVG)"]
    TREE["<b>B+tree index</b> — Phase 4<br/>ordered keys → fast lookup, no full scan"]
    PAGER["<b>Pager</b> — Phase 3  ◀ WE ARE HERE<br/>reads/writes fixed 4KB pages by number"]
    DISK[("<b>Disk file</b><br/>one file, carved into 4096-byte pages")]

    SQL --> OPS --> TREE --> PAGER --> DISK

    classDef done fill:#1f6f43,stroke:#0d3,color:#fff
    classDef now fill:#8a6d0b,stroke:#fc0,color:#fff
    classDef todo fill:#333,stroke:#777,color:#ccc
    class PAGER now
    class TREE,OPS,SQL todo
```

**Why bottom-up:** the pager knows nothing about games or tables — it just moves bytes in
fixed blocks. The B+tree sits on top and decides how records are laid out *inside* those
bytes. Operators sit on the B+tree. SQL sits on the operators. Each layer is a **deep
module**: a tiny interface (the pager is just `Open` / `ReadPage` / `WritePage`) hiding
significant complexity. Swap-ability comes from this — e.g. the B+tree will depend on a
pager *interface*, so a fake in-memory pager can stand in during tests.

---

## 2. The two data paths

### 2a. Ingest path — getting `.slp` replays onto disk (the "write" story)

```mermaid
flowchart LR
    SLP[".slp files<br/>(~200k Melee replays)"]
    JS["slippi-js black box<br/>(shell out to 'slp -s')"]
    JSON["JSON<br/>settings + metadata"]
    TOGAME["toGame()<br/>anti-corruption seam:<br/>raw peppi format → clean Game"]
    REC["Game record<br/>(bytes)"]
    PAGER["Pager.WritePage(k, bytes)"]
    DISK[("pages on disk")]

    SLP --> JS --> JSON --> TOGAME --> REC --> PAGER --> DISK
```

`.slp` parsing is a deliberate **black box** — no good Go parser exists, so we shell out to
`@slippi/slippi-js` and consume its JSON. `toGame()` is the seam that translates *their*
data model into *ours*. Everything left of the pager is Phases 1–2 (done); the pager is
Phase 3.

### 2b. Query path — answering a question (the "read" story)

```mermaid
flowchart TB
    Q["SQL text<br/>e.g. SELECT stage, COUNT(*) FROM games<br/>WHERE character = 'Falco' GROUP BY stage"]
    PARSE["Parser → AST → query plan"]
    OPS["Operators execute the plan:<br/>scan → filter → aggregate"]
    TREE["B+tree: jump to matching keys<br/>(instead of scanning all 200k rows)"]
    PAGER["Pager.ReadPage(k)<br/>fetch the pages holding those rows"]
    DISK[("pages on disk")]
    RESULT["Result rows → back up to the user"]

    Q --> PARSE --> OPS --> TREE --> PAGER --> DISK
    DISK -. bytes .-> PAGER -. rows .-> TREE -. rows .-> OPS -. results .-> RESULT
```

A query flows **down** the stack (SQL → plan → operators → index → pager → disk) and the
data flows **back up** (bytes → rows → filtered/aggregated results). The whole reason for
the B+tree (Phase 4) is to avoid the full O(n) scan we deliberately suffered in Phase 2.

---

## 3. Where we are today

Only the shaded parts exist. The ingest pipeline (Phases 1–2) runs; we're mid-build on the
pager (Phase 3). Everything above the pager is not yet written.

```mermaid
flowchart TB
    subgraph DONE["✅ Done — Phases 1–2 (ingest, naive flat file)"]
        SLP[".slp files"] --> JS["slp -s (slippi-js)"] --> TOGAME["toGame()"] --> JSONL["data/games.jsonl<br/>(throwaway flat file)"]
    end

    subgraph NOW["🟡 Phase 3 (storage/ package) — core done, cache next"]
        OPEN["Open(path) ✅ (read-write + create)"]
        WRITE["WritePage(k, data) ✅"]
        READ["ReadPage(k) ✅"]
        TEST["round-trip test ✅"]
        CACHE["buffer-pool cache — next"]
        OPEN --- WRITE --- READ --- TEST --- CACHE
    end

    subgraph LATER["⬜ Not started — Phases 4–6"]
        TREE["B+tree"] --> OPSX["operators"] --> SQLX["SQL parser"]
    end

    DONE -.->|"replaces the flat file with real pages"| NOW
    NOW -.-> LATER
```

> The Phase 2 flat file (`data/games.jsonl`) is a **throwaway** stepping stone — Phase 3's
> pager replaces it with real page-based storage. Keeping it in the diagram shows the
> migration, which is part of the learning story.
