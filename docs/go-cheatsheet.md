# Go cheat sheet — the syntax rslp actually uses

Every example is lifted from real rslp code (`main.go`, `storage/`).

## rslp vocabulary (mirrors `CONTEXT.md`)

| Term | Means | Unit |
|---|---|---|
| **page** | one fixed 4096-byte block; in code, the `page []byte` loaded into memory | 4096 bytes |
| **page number** | which page in the file (page `k` lives at byte `k × 4096`) | a count |
| **header** | bytes 0–1 of a page; holds `numSlots` | 2 bytes |
| **record** | a row's actual bytes, packed in from the **front** of the page | variable |
| **slot** | a 4-byte `{Offset, Length}` entry at the **back** of the page that points to one record | 4 bytes |
| **number** | *which one*: slot #0, #1, #2…; `numSlots` = how many | a count |
| **position** | *where* something is inside the page | byte, 0–4095 |
| **Offset** | slot field: the byte position where its record starts | byte |
| **Length** | slot field: how many bytes its record is | bytes |
| **end** | `start + length` = the first byte **after** something (exclusive, like `page[a:b]`) | byte |
| **size** (`PageSize`) | 4096, one past the last byte (4095); subtract from this, check against 4095 | bytes |

## Syntax

| Concept | Example |
|---|---|
| **Declare + assign** | `offset := int64(k) * PageSize` — `:=` = new variable, type inferred |
| **Declare, zero value** | `var replay rawReplay` |
| **Reassign existing** | `err = json.Unmarshal(out, &replay)` — plain `=` |
| **Struct type** | `type Slot struct { Offset uint16; Length uint16 }` |
| **Struct literal** | `Slot{Offset: offset, Length: length}` |
| **Function** | `func getSlot(buf []byte, position int) Slot { ... }` — params are `name type`, return type last |
| **Multiple returns** | `func Open(path string) (*Pager, error)` → `return &Pager{file: f}, nil` |
| **Error check** (the Go reflex) | `if err != nil { return nil, err }` |
| **Ignore a value** | `_, err := p.file.WriteAt(data, offset)` |
| **Method (receiver)** | `func (p *Pager) ReadPage(k int) ([]byte, error)` — `p` ≈ `this`/`self` |
| **Pointer** | `&Pager{...}` = take address · `*Pager` = pointer type |
| **Allocate a slice** | `make([]byte, PageSize)` |
| **Slice from index** | `buf[position:]` — from `position` to the end |
| **Slice range** | `buf[a:b]` — from `a` up to (not including) `b` |
| **Append** | `players = append(players, p)` — must reassign the result |
| **Loop over a slice** | `for _, entry := range entries { ... }` (`_` = skip the index) |
| **Map literal** | `var stageNames = map[int]string{ 31: "Battlefield" }` |
| **Map lookup** | `characterNames[rp.Character]` |
| **Type conversion** | `int64(k)` · `uint16(n)` · `[]byte(str)` · `string(data)` |
| **Constant** | `const PageSize = 4096` |
| **Exported vs private** | Capitalized (`Open`, `PageSize`) = visible outside the package; lowercase (`putSlot`) = package-private |
| **Binary encode** | `binary.LittleEndian.PutUint16(buf[pos:], v)` — writes into `buf`, returns nothing |
| **Binary decode** | `v := binary.LittleEndian.Uint16(buf[pos:])` — returns the value |

## Gotchas already hit

- **Signature ≠ call.** `PutUint16(b []byte, v uint16)` describes types; calling it is
  `PutUint16(buf[pos:], slot.Offset)` — values, never retyped types.
- **`buf[pos:+2]` ≠ `buf[pos+2:]`.** The first is `buf[pos:2]` (unary `+2` is just `2`).
- **`ReadAt` returns a byte count, not the data.** The bytes land in the buffer you passed in.
- **Slices (and maps) share data when passed.** A slice carries a hidden pointer, so a function that
  writes into a `[]byte` param changes the caller's bytes, with no `&`/`*` needed. Arrays (`[4096]byte`),
  ints and structs are copied. To let a function change those, pass a pointer (`&x` / `*int`) or return the new value.
- **`os.Open` is read-only and won't create a file.** Use
  `os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)` to read + write + create.

## Tests

```go
func TestSomething(t *testing.T) {   // name must start with Test, file must end _test.go
    dir := t.TempDir()                // auto-cleaned temp dir
    if err != nil { t.Fatalf("Open() failed: %v", err) }   // stop the test now
    if !bytes.Equal(got, want) { t.Errorf("mismatch: got %v, want %v", got, want) } // record failure, keep going
}
```

## Terminal (run from repo root)

```sh
go build ./...                        # does everything compile?
go vet ./...                          # catch common mistakes
go test ./storage                     # run the storage package's tests
go test ./storage -run RoundTrip -v   # one test (regex match on name), verbose
go test ./...                         # every package's tests
go run .                              # run main.go (the ingest)
gofmt -w storage/slotted.go           # auto-format a file in place
go doc encoding/binary.LittleEndian   # look up stdlib docs offline
```
