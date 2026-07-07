# rslp — "Ahhh" moments

> A running log of the ideas that *clicked* while building this database from scratch — the
> moments where a concept stopped being words and became something I could reason about.
> Kept both as a review tool (skim it to re-lock the fundamentals) and as raw material for
> describing the project later. Newest at the top. Each entry is in my own words.

---

## Phase 3 — Storage / the Pager

### The page number is arbitrary *to the pager* — the index gives it meaning
The read/write pager work isn't the thing that *finds* your data. It's just the fast
"go fetch page k" machine. The **B+tree index (Phase 4)** is the brain that figures out
*which* page number holds the row you asked for; then it hands that number to the pager.
So the pager is deliberately dumb — it seeks to a page and moves bytes, and knows nothing
about games, characters, or what's inside. Building it first makes sense: you need a way to
grab any page in O(1) *before* you can build the index that decides which page to grab.
Without the index you'd be back to reading records one-by-one until you hit the right one —
which is exactly the slow scan we're trying to kill.

### Fixed-size pages turn "search" into "arithmetic"
Every page is exactly 4096 bytes. Because the size is fixed and known, I can *compute*
where page `k` starts instead of hunting for it: `offset = k × PageSize`. The file is one
long numbered line of bytes; page `k` begins after `k` full pages, so at byte `k × 4096`.
Hand that offset to the OS and it seeks straight there — O(1), no matter how big the file
gets. This is the whole reason we abandoned the Phase 2 flat file: variable-length JSON
lines had no predictable position, so finding row #k meant scanning everything before it.

### A test can *drive out* a design flaw
Writing the round-trip test forced a real change to `Open`. My original `Open` used
`os.Open`, which opens a file **read-only** and won't create a missing one — so the moment
the test tried to *write* a page (and to a fresh temp file that didn't exist yet), it broke.
I didn't spot that gap by staring at the code; the test found it the instant something tried
to actually use it. Fix: `os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)`. That's TDD
earning its keep — RED first, for the *right* reason, then GREEN.

### `ReadAt` fills a buffer you own — the data isn't the return value
This one bit me. `ReadAt(buffer, offset)` returns `(n int, err error)` — the first value is
a byte *count*, not the page. The actual bytes land in the `buffer` I allocated with
`make([]byte, PageSize)` and passed *in*. So reading is: prepare an empty box first, hand it
over, let `ReadAt` pour the page into it, then return the box. Mirror image of `WritePage`,
where I already had the bytes and just handed them off.
