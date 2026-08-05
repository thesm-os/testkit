# Parallel Safety

Detects test patterns that are unsafe under `t.Parallel()`
or the race detector. Catches bugs that only manifest
under `-race` or in Go 1.24+ where `t.Setenv` +
`t.Parallel` panics.

## Command

```
testkit validate parallel-safety
```

## What it checks

### t.Setenv + t.Parallel

**t.Setenv + t.Parallel:**

Any test or subtest that calls both `t.Setenv` and
`t.Parallel` (panics in Go 1.24+).

```go
// FLAGGED — panics in Go 1.24+
func TestStore_Config(t *testing.T) {
    t.Parallel()
    t.Setenv("STORE_DSN", "postgres://...")  // panic
}
```

**Shared mutable state in parallel subtests:**

A `t.Run` subtest that calls `t.Parallel()` and
references a variable declared in the parent scope
that is also written by another parallel subtest.

```go
// FLAGGED — shared counter across parallel subtests
func TestStore(t *testing.T) {
    count := 0  // shared mutable state
    t.Run("put", func(t *testing.T) {
        t.Parallel()
        count++  // data race
    })
    t.Run("get", func(t *testing.T) {
        t.Parallel()
        count++  // data race
    })
}
```

**Missing t.Parallel on subtests:**

`t.Run` subtests inside a `t.Parallel()` parent that
don't call `t.Parallel()` themselves (likely
oversight — the parent opted in but the subtest forgot).

```go
// FLAGGED — parent is parallel, subtest is not
func TestStore(t *testing.T) {
    t.Parallel()
    t.Run("put", func(t *testing.T) {
        // missing t.Parallel() — runs serially despite parent
    })
}
```

## Failure output

```
parallel-safety: FAIL

  store/store_test.go:42: TestStore_Config
    t.Setenv + t.Parallel — panics in Go 1.24+
    remove t.Setenv or remove t.Parallel

  store/store_test.go:60: TestStore/put
    writes to shared variable "count" from parallel subtest
    use atomic, mutex, or per-subtest state

  store/store_test.go:78: TestStore/get
    parent calls t.Parallel() but subtest does not
    add t.Parallel() to subtest or document why serial
```

## Why

Race conditions in tests are insidious:

- `t.Setenv` + `t.Parallel` worked in Go 1.23 but panics
  in 1.24+ — a silent time bomb in the test suite
- Shared mutable state across parallel subtests produces
  data races that only manifest under `-race` or with
  `go test -count=N` where N > 1
- Missing `t.Parallel` on subtests defeats the parent's
  parallelism, making the test slower than intended
  without any visible error

The race detector catches some of these at runtime, but
only if the race actually fires during the test run. This
validator catches them statically, before execution.

## Configuration

```yaml
# .testkit.yaml
validators:
  parallel_safety:
    enabled: true
    # Warn on missing t.Parallel in subtests of parallel
    # parents (default: true). Set to false if your project
    # intentionally mixes parallel parents with serial
    # subtests.
    warn_missing_parallel: true
```
