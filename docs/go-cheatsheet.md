# Go cheat sheet — the syntax rslp actually uses

Every example is lifted from real rslp code (`main.go`, `storage/`).

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
