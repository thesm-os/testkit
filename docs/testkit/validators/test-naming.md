# Test Naming

Enforces test naming conventions that keep test files
readable as contract specifications. Catches the most
common Phase 4.2 anti-pattern: multiple top-level
`Test<Type>_*` functions on the same type instead of one
`Test<Type>` with subtests.

## Command

```
testkit validate test-naming
```

## What it checks

### No fragmented top-level tests

1. If a file has two or more `Test<Type>_*` top-level
   functions on the same type, the validator flags them
   for consolidation into one `Test<Type>` with subtests

```go
// FLAGGED — 3 top-level functions on Store
func TestStore_Put(t *testing.T)    { ... }
func TestStore_Get(t *testing.T)    { ... }
func TestStore_Delete(t *testing.T) { ... }

// ACCEPTED — one entry point with subtests
func TestStore(t *testing.T) {
    t.Run("Put persists item", ...)
    t.Run("Get retrieves by ID", ...)
    t.Run("Delete removes item", ...)
}
```

### Allowed top-level splits

These are legitimate orthogonal axes and are not flagged:

- `TestType` + `TestType_Properties` (PBT alongside unit)
- `TestType` + `TestType_Integration`
- `FuzzType_*` (Go toolchain requires top-level)
- `BenchmarkType_*` (convention)
- `TestType` + `TestType_Conformance` (suite wiring)

### Subtest naming

Subtests should read as contract descriptions, not
method names. The validator warns on subtests that
look like bare method names without behavior:

```go
// WARNED — method name, not a contract
t.Run("Put", ...)

// ACCEPTED — describes behavior
t.Run("Put persists item and returns no error", ...)
t.Run("Put/duplicate ID returns ErrConflict", ...)
```

The heuristic: subtests with a single word or
`CamelCase` without spaces are flagged as likely method
names. Subtests with spaces or `/` separators are
accepted.

## Failure output

```
test-naming: FAIL

  store/store_test.go
    TestStore_Put, TestStore_Get, TestStore_Delete —
    consolidate into TestStore with subtests

  store/store_test.go:42
    t.Run("Put", ...) — subtest name looks like a bare
    method name, not a contract description
```

## Why

Test files are the executable specification of a package.
When tests are organized as 17 `TestArena_*` top-level
functions, a reader has to scan 17 function bodies to
understand what Arena guarantees. One `TestArena` with
named subtests collapses that into a one-screen catalog.

## Configuration

```yaml
# .testkit.yml
validators:
  test_naming:
    enabled: true
    # Minimum top-level functions on the same type before
    # flagging (default: 2).
    consolidation_threshold: 2
    # Warn on bare-method-name subtests (default: true).
    warn_bare_subtests: true
```
