# srsdb — a relational database engine, built from scratch in Go

> Repo: **srsdb** (Slippi Ranked Stats DB). The engine itself is named **rslp** (the Go module).

A small on-disk relational database engine written by hand in Go — page-based storage,
a B+tree index, a query execution engine, and a SQL parser — built to understand how
databases actually work, not to ship something fast.

> **Status: in development (~15%).** The `.slp` ingest pipeline and a naive flat-file
> baseline are done; the storage engine (pages → B+tree → operators → SQL) is the bulk of
> the work and is still ahead. See [`PROGRESS.md`](./PROGRESS.md) for the live roadmap.

## Why

I work with Oracle Cloud and SQL every day, but always from *above* the database — schemas,
queries, ERDs. This project goes the other direction: down to bytes, pages, and offsets, to
learn how a database engine is built from the ground up. Go was chosen deliberately because
it exposes that systems layer with a small syntax surface.

The reference architecture follows *Build Your Own Database From Scratch in Go*
(B+tree → durability → relational → SQL), cross-checked rather than copied.

## The north star

rslp is meant to become a **queryable companion to [Slippi Ranked
Stats](https://github.com/Joey-Farah/Slippi-Ranked-Stats)** — my desktop app for competitive
*Super Smash Bros. Melee* players. The idea: let power users run their own **read-only SQL**
against their Slippi match data — a corpus of up to ~200,000 `.slp` replay files — to answer
questions the app's UI doesn't surface. That makes the schema design and the SQL layer the
real deliverables.

## Architecture / roadmap

Built bottom-up — tables and SQL sit at the *top* of the stack, not the start.

| Phase | What | Status |
|---|---|---|
| 1 | Parse a `.slp` replay into a clean `Game` struct | ✅ complete |
| 2 | Naive flat-file ingest + read-back + scan (feel the O(n) pain) | ✅ complete |
| 3 | Fixed-size pages + a pager (read/write pages on demand) | ⬜ next |
| 4 | B+tree index over the pager | ⬜ |
| 5 | Execution operators: scan, filter (WHERE), aggregate (GROUP BY) | ⬜ |
| 6 | Recursive-descent SQL parser → AST → query plan | ⬜ |
| 7 | *(optional)* columnar storage / zone maps for analytical queries | ⬜ |

## Stack

- **Go** — the engine, written by hand (no database libraries).
- **[@slippi/slippi-js](https://github.com/project-slippi/slippi-js)** — used as a black-box
  `.slp` parser (Node), shelled out to emit JSON that Go unmarshals into domain structs.

## Running it

```bash
go run .          # build + run the whole package
go vet ./...      # static checks
```

The test dataset is local `.slp` replay files; the generated `data/` directory is
reproducible from those and is not committed.

---

*A learning-first project — kept presentable on purpose. The clarity of the artifact and the
understanding behind it are the point, not just that it runs.*
