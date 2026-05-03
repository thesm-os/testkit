# Sentinel

Reads the package's exported error declarations and emits tests that lock in error-handling discipline. Two layers:

1. **Package-level sentinels** (`var ErrX = errors.New(...)`) get prefix, uniqueness, non-overlap, and unwrap-chain tests as one umbrella `Test<Pkg>SentinelErrors` function.
2. **Custom error types** (structs implementing `error`) get a per-type `Test<Type>` function with `errors.As` extraction, wrapping survival (`errors.Join`, `fmt.Errorf %w`), `Error()` content, and optional `Is`/`Unwrap` exercises.

The generator scans the source package — no type arguments needed.

## Directive

```go
//go:generate testkit sentinel -o errors.gen_test.go
```

## Default output

`errors.gen_test.go` in the source package directory.

## What is generated

For a package with sentinels `ErrNotFound`, `ErrConflict`, `ErrForbidden`, plus custom types `ValidationError`, `NotFoundError` (with `Is`), `WrappedError` (with `Unwrap`):

### Test\<Pkg\>SentinelErrors

Single function with subtests covering the package-level sentinel set:

```go
func TestBasicSentinelErrors(t *testing.T) {
    type errEntry struct { name string; err error }
    all := []errEntry{
        {"ErrConflict",  basic.ErrConflict},
        {"ErrForbidden", basic.ErrForbidden},
        {"ErrNotFound",  basic.ErrNotFound},
    }

    t.Run("prefix",        func(t *testing.T) { /* each Error() starts with "<pkg>: " */ })
    t.Run("uniqueness",    func(t *testing.T) { /* no two have identical Error() */ })
    t.Run("non-overlap",   func(t *testing.T) { /* errors.Is asymmetric — ErrA != ErrB */ })
    t.Run("unwrap chain",  func(t *testing.T) { /* errors.Join wrap survives errors.Is */ })
}
```

The prefix derived from the package name (e.g. `"basic: "`). The sentinel list is sorted alphabetically — generation is deterministic.

### Per-type tests for custom error types

For each struct that implements `error()`, a per-type test function with shared subtests plus optional ones based on detected methods:

```go
func TestNotFoundError(t *testing.T) {
    t.Run("errors.As extracts type", ...)              // round-trip via errors.As
    t.Run("survives errors.Join wrapping", ...)        // wrap + As
    t.Run("survives fmt.Errorf wrapping", ...)         // wrap + As (%w)
    t.Run("Error format includes all fields", ...)     // every exported field appears in Error()
    t.Run("Is matches same type", ...)                 // only if Is() is defined
    t.Run("Is matches across instances ...", ...)      // only if Is() is defined
    t.Run("Is rejects different error types", ...)     // only if Is() is defined
}

func TestWrappedError(t *testing.T) {
    // ... shared subtests ...
    t.Run("Unwrap returns cause", ...)                 // only if Unwrap() is defined
    t.Run("errors.Is traverses Unwrap chain", ...)     // only if Unwrap() is defined
}
```

The "Error format includes all fields" subtest constructs a value with sentinel field values (e.g. `"test-entity"`) and asserts every value appears in the rendered `Error()` string — catching broken format strings.

## What it locks in

| Property | Why it matters |
|----------|----------------|
| Prefix | Consistent log grep — `"store: not found"` is greppable by package |
| Uniqueness | No two sentinels share `.Error()` — ambiguous logs are bugs |
| Non-overlap | `errors.Is(a, b) == false` for every pair — prevents alias bugs in `switch` |
| Unwrap chain | Sentinels survive `errors.Join` so callers can match through wrappers |
| `errors.As` round-trip | Custom types survive `errors.Join` and `fmt.Errorf("%w", ...)` |
| `Error()` format | Every exported field appears in the rendered string |
| Optional `Is` | Custom matchers are self-consistent (matches same type, rejects other types) |
| Optional `Unwrap` | Returns the documented cause and is found via `errors.Is` traversal |

## Why

Sentinel discipline silently rots: a renamed package leaves old prefixes, copy-paste produces aliases, a switch on `errors.Is` matches the wrong branch when sentinels accidentally overlap, a format-string typo drops a field. The generator catches all of these at `go test` time — no runtime detection, no production incident.

## See also

- [`error-prefix` validator](../validators/error-prefix.md) — catches the prefix issue at CI time across packages that haven't yet adopted the generator.
