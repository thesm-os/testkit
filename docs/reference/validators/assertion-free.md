# Assertion-Free Tests

Detects `Test*` functions that call the subject under test
but never call an assertion. These tests exercise code
paths without verifying outcomes — they pass even when the
subject returns wrong results.

## Command

```
testkit validate assertion-free
```

## What it checks

For every `func Test*(t *testing.T)` in `*_test.go` files:

1. The function body contains at least one call to a
   recognized assertion function
2. Subtests (`t.Run`) also contain at least one assertion

Recognized assertion patterns:

- `testkit.*` assertion functions (Equal, ErrorIs, True, etc.)
- `testutil.*` assertion functions
- `t.Fatal`, `t.Fatalf`, `t.Error`, `t.Errorf`
- `require.*`, `assert.*` (testify, if used)
- `rapid.Check` (PBT — rapid does its own assertions)

## Failure output

```
assertion-free: FAIL

  store/store_test.go:42: TestStore_Put
    calls store.Put but contains no assertion —
    add testkit.NoError or testkit.Equal

  store/store_test.go:87: TestStore_List/empty
    subtest calls store.List but contains no assertion
```

## Why

Assertion-free tests are the most dangerous kind of
passing test — they give the illusion of coverage without
verifying anything. A function that calls `store.Put()`
and ignores the error appears in coverage reports as
"covered" but would pass even if `Put` returned
`ErrCorrupt` on every call.

Mutation testing catches many of these, but runs minutes
to hours. This validator runs in seconds as a static scan
and catches them before tests even execute.

## Exceptions

Tests that intentionally have no assertion (e.g., "does
not panic" tests, benchmark setup helpers) can be
annotated:

```go
func TestStore_PutDoesNotPanic(t *testing.T) {
    //testkit:no-assertion — panic absence is the assertion
    store.Put(ctx, item)
}
```

## Configuration

```yaml
# .testkit.yaml
validators:
  assertion_free:
    enabled: true
```
