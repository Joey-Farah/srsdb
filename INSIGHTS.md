# rslp — "Ahhh" moments

> A running log of the ideas that *clicked* while building this database from scratch — the
> moments where a concept stopped being words and became something I could reason about.
> Kept both as a review tool (skim it to re-lock the fundamentals) and as raw material for
> describing the project later. Newest at the top. Each entry is in my own words.

---

## Phase 4 — B+tree / keys

### Deletes are a workload question, not just a data-structure question
Slotted pages are built to handle deleted records gracefully (tombstone a slot, don't
shift bytes) — but rslp is an archive of historical Slippi replays. Games that were
played don't get "un-played." There's no user-facing delete workflow at all. So even
though the data structure *could* support tombstones cheaply, adding one now would be
designing for an operation this database will probably never need. Skipped it —
`{offset, length}` slots only, no delete flag, until a real reason shows up.



### A natural key can exist and still be the wrong storage key
I went looking for a per-game unique id straight from Slippi, sure there had to be one — and
there is: `start.match = { id, game, tiebreaker }`. `match.id` identifies the *set*, `game` is
the game number within it, so `(match.id, match.game)` really is unique per game. My composite-
key instinct was right about the *data model*.

But it's the wrong key for the **storage layer** specifically, for three reasons: (1) `match.id`
is a variable-length string (~35 chars) — exactly the variable-width problem fixed-size pages
exist to kill; (2) it's only present on newer online replays — `hash` was `null` and older/
offline files may lack `match` entirely, so it can't anchor a primary key; (3) string comparison
in a tree is slower than one fixed-width int compare, node after node.

Resolution (standard DB pattern, same move Oracle makes with `GENERATED AS IDENTITY` + a
`UNIQUE` constraint on the natural key): **primary/clustering key = synthetic int row id**
(fixed-width, always present, cheap to compare) — that's what the B+tree orders by. The natural
`(match.id, match.game)` becomes a documented candidate for a *secondary* index later, once
querying "find this exact Slippi game" matters. Two different jobs: uniqueness in the domain
vs. an efficient key to physically order pages by.

---

### `pageBuffer` *is* the page — the page number is how you fetch it
Every slotted-page function takes `pageBuffer` first because it says *which page* to work on —
with many pages in memory at once, `getSlot(page3, …)` and `getSlot(page12, …)` are the same
function on different pages. Two ways to name a page: the **page number** is how the pager
finds it on disk; `pageBuffer` is the page's actual 4096 bytes once loaded. And because a Go
slice is a handle to shared bytes, `putSlot` edits the caller's page directly, so it
returns nothing.

### Slots are *placed* backwards but *written* forwards
The slot directory grows from the end of the page toward the front, but each slot's own 4 bytes
are still written left-to-right from its `position`. So the first slot starts at 4092 (filling
4092–4095), not 4095. Records go at the front: the next one starts at the last slot's
`Offset + Length`. No need to store a free-space pointer, since it can be calculated.

### Slices carry a hidden pointer, so no `&`/`*` is needed
`pageBuffer` looks like an ordinary variable, but a slice is really a small `{pointer, length, capacity}`
struct. Passing it copies that small struct, and the copy's pointer still aims at the same bytes, so a
function that writes through it changes the caller's page. An array (`[4096]byte`) has no hidden pointer
and gets copied in full. Slices and maps share their data when passed; plain ints, structs and arrays are
copied. Explicit pointers (`*Pager`, `&x`) are for sharing or changing a caller's value, or avoiding big
copies. Local variables inside a function never need them.

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
