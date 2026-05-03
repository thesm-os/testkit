# Enum

Reads `iota` constants of a named type and emits a single `Test<Type>Enum` function with the full battery of enum tests. Optional methods (`String`, `Parse<Type>`, `MarshalText`, `MarshalJSON`) are detected automatically; their tests are emitted only when the methods exist.

## Directive

```go
//go:generate testkit enum -o enum.gen_test.go Status

// Multiple enums in one file:
//go:generate testkit enum -o enum.gen_test.go Status Priority Region
```

## Default output

`<subject>_enum.gen_test.go` (or your chosen `-o` path) in the source package directory.

## What is generated

### Always-emitted subtests

For every enum:

```go
func TestStatusEnum(t *testing.T) {
    type enumEntry struct {
        value   basic.Status
        name    string
        wantStr string  // only when stringer is detected
    }
    all := []enumEntry{
        {value: basic.StatusActive,  name: "StatusActive",  wantStr: "Active"},
        {value: basic.StatusClosed,  name: "StatusClosed",  wantStr: "Closed"},
        {value: basic.StatusPending, name: "StatusPending", wantStr: "Pending"},
    }

    t.Run("exhaustive", ...)              // len(all) matches the declared count
    t.Run("zero value is StatusPending", ...)  // first iota declaration
    t.Run("all values are distinct", ...) // no two constants alias the same int
}
```

The constant list is sorted alphabetically by name.

### Stringer-conditional subtests

Emitted when the enum has a `String() string` method:

```go
t.Run("stringer", ...)                         // every name → expected string
t.Run("out of range uses fallback format", ...) // Status(N) for N == len(all)
```

The "out of range" subtest constructs a value just past the last declared constant and asserts the stringer falls back to `"<Type>(<int>)"` format.

### ParseX round-trip

Emitted when a top-level `Parse<Type>(string) (<Type>, error)` exists:

```go
t.Run("parse round-trip", ...)            // Parse(value.String()) → value
t.Run("parse rejects unknown string", ...) // Parse("<invalid-Type>") → error
```

### MarshalText round-trip

Emitted when both `MarshalText` and `UnmarshalText` are defined:

```go
t.Run("marshal text round-trip", ...)         // MarshalText → UnmarshalText
t.Run("unmarshal rejects unknown text", ...)  // UnmarshalText("<invalid>") → error
```

### JSON round-trip

Emitted when both `MarshalJSON` and `UnmarshalJSON` are defined:

```go
t.Run("json round-trip", ...)                // json.Marshal → json.Unmarshal
t.Run("json unmarshal rejects unknown", ...) // unknown string → error
```

### Plain-iota enums

When an enum has no String/Parse/Marshal methods (a bare `iota` block), the generator emits only the always-emitted subtests:

```go
func TestPriorityEnum(t *testing.T) {
    all := []enumEntry{ /* sorted by name */ }
    t.Run("exhaustive", ...)
    t.Run("zero value is PriorityLow", ...)
    t.Run("all values are distinct", ...)
}
```

The struct does not contain `wantStr` or anything else only useful for stringers.

## What it locks in

| Property | Why it matters |
|----------|----------------|
| Exhaustiveness | If a constant is added but the test is not regenerated, the count mismatch fails immediately |
| Zero value | The zero value is the first iota declaration — explicit assertion catches reordering |
| Distinct values | No two constants share an integer value (catches accidental `iota` arithmetic bugs) |
| Stringer | Every named value renders to the expected string |
| Out-of-range fallback | Strings that aren't named values render as `"<Type>(N)"` |
| Parse round-trip | `Parse(v.String()) == v` for every named value |
| Parse rejection | Unknown strings return an error |
| Text/JSON round-trip | Marshal-Unmarshal preserves identity |
| Text/JSON rejection | Unknown encoded values return an error |

## Detection rules

- **Stringer**: method `String() string` on the enum type
- **Parse**: top-level function `Parse<Type>(s string) (<Type>, error)` in the same package
- **MarshalText**: both `MarshalText() ([]byte, error)` on the value receiver and `UnmarshalText([]byte) error` on the pointer receiver
- **MarshalJSON**: both `MarshalJSON() ([]byte, error)` and `UnmarshalJSON([]byte) error`

If only one half of a Marshal pair is present, neither subtest is emitted — partial round-trip is not testable.

## Why

Enum types accumulate silent bugs: a new constant added without test coverage, a stringer not regenerated so the new value prints as `Status(4)` in logs, a Marshal pair where one side was hand-written and drifted from the other. The generator catches all of these at `go test` time, in seconds. Adding a new constant to the source forces regeneration, and the generated test asserts every property against the new constant set.
