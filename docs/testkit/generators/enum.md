# Enum

The `enum` generator is a "Static" conformance tier tool. It scans `const` blocks of named integer types and emits a test file (`_test.go`) along with a wire-compat JSON golden file (`_wire.json`) to aggressively verify the safety and stability of your hand-written enums. Optional methods (`String`, `Parse<Type>`, `MarshalText`, `MarshalJSON`) are detected automatically; their round-trip tests are appended when the methods exist.

## Directive

```go
//go:generate testkit enum -o enum.gen_test.go Status

// Multiple enums in one file:
//go:generate testkit enum -o enum.gen_test.go Status Priority Region
```

## Default output

The generator emits two files in the source package directory:
1. `<subject>_enum.gen_test.go` (or your chosen `-o` path) — The test file.
2. `<subject>_wire.json` — A committed golden file mapping constant names to their exact integer values.

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
    t.Run("wire compat", ...)             // const→int matches _wire.json golden
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
    t.Run("wire compat", ...)
}
```

The struct does not contain `wantStr` or anything else only useful for stringers.

## What it locks in

| Property | Why it matters |
|----------|----------------|
| Exhaustiveness | If a constant is added but the test is not regenerated, the count mismatch fails immediately |
| Zero value | The zero value is the first iota declaration — explicit assertion catches reordering |
| Distinct values | No two constants share an integer value (catches accidental `iota` arithmetic bugs) |
| Wire compat | Catch accidental integer value shifting that breaks database/API serialization |
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

## Updating the Wire Golden File

If you deliberately change an integer value or add a new constant, the wire-compat test will fail. To accept the new schema, run:

```bash
go test -update
```

This overwrites the `_wire.json` file with the new mapping, which you then commit to source control.

## Why

Enum types accumulate silent bugs: a new constant added without test coverage, a stringer not regenerated so the new value prints as `Status(4)` in logs, a Marshal pair where one side was hand-written and drifted from the other, or an `iota` shift that silently breaks your database. The generator catches all of these at `go test` time, in seconds. Adding a new constant to the source forces regeneration, and the generated test asserts every property against the new constant set.

## Layout Conventions

Unlike the `suite` and `stub` generators which write to a `<pkg>test/` sub-package, the `enum` generator writes its output directly into the source package directory, but scoped as a `_test.go` file.

**What goes where:**

| File | Owner | Contents |
|------|-------|----------|
| `types.go` | Developer | The source file containing the `type Status int` and `const (...)` block. |
| `*_enum.gen_test.go` | Generator | The static verification suite asserting the enum's invariants. |
| `*_wire.json` | Generator | The committed golden file tracking exact integer allocations. |

Because `_test.go` files and `_wire.json` files are excluded from the final production binary, this layout provides 100% test coverage without bloating the compiled artifact.

## See also
- [Generators / Overview](README.md)
