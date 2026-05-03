# Helpers

Smaller utilities that don't warrant their own document.

## TestError

Creates a named, deterministic sentinel error for fixtures. Replaces ad-hoc `errors.New("boom")` with a traceable, `errors.Is`-compatible value.

```go
var errBoom = testkit.TestError("boom")
// errBoom.Error() == "testkit: boom"

stub.OnPut.Faults(errBoom, 3)
err := stub.Put(ctx, req)
testkit.ErrorIs(t, err, errBoom, "must propagate fault")
```

`TestError("x")` always returns the same error for the same name; two calls with different names return distinct errors.

## RequireEnv

Skips the test if the named environment variable is not set; returns its value if present.

```go
func TestPostgres_Integration(t *testing.T) {
    dsn := testkit.RequireEnv(t, "DATABASE_URL")
    pool, err := pgxpool.New(t.Context(), dsn)
    // ...
}
```

## SeededRand

Returns a deterministic `*rand.Rand` seeded from the FNV-1a hash of `tb.Name()`. Two runs of the same test produce byte-identical sequences.

```go
rng := testkit.SeededRand(t)
idx := rng.IntN(len(items))
```

Backed by `math/rand/v2` PCG.

## MustMarshal / MustUnmarshal

Fixture construction helpers that fatal on JSON error.

```go
data := testkit.MustMarshal(t, fixture)
testkit.MustUnmarshal(t, data, &target)
```

Do **not** use these for the subject under test's marshal behavior — that's what the `codec` generator is for.

## FailingReader / FailingWriter

`io.Reader` / `io.Writer` that succeeds for the first N bytes, then fails on every subsequent call.

```go
r := &testkit.FailingReader{
    Source:     bytes.NewReader(payload),
    BeforeFail: 100,
    Err:        io.ErrUnexpectedEOF,
}
```

Used to simulate mid-stream failures.

## Quiet

Suppresses `log/slog` default-handler output for the duration of the test by replacing the default with a discard handler. Tests using `Quiet` must NOT call `t.Parallel()` — they mutate process-global state.

```go
func TestSubsystem_LogsOnFailure(t *testing.T) {
    defer testkit.Quiet(t)()
    subject.DoThingThatLogs()
}
```

## FailableTB

Stub `testing.TB` that captures the first `Fatalf`/`Fatal`/`Errorf` without aborting the host goroutine. Used to verify that an assertion helper actually fails when expected.

```go
ftb := testkit.NewFailableTB().WithName("TestFoo")
testkit.Equal(ftb, 1, 2, "must fail")
testkit.True(t, ftb.Failed(), "must record failure")
testkit.Contains(t, ftb.Msg(), "must fail", "captured message")
```

| Method | Purpose |
|--------|---------|
| `NewFailableTB()` | Construct |
| `WithName(s)` | Set `Name()` (fluent) |
| `Failed()` / `Msg()` | Inspect captured failure |
| `Logs()` | Captured `Logf` lines |
| `HelperCalls()` | Number of `Helper()` invocations |
| `Cleanup(fn)` / `RunCleanups()` | LIFO cleanup simulation |
| `Context()` | Context cancelled on Fatal |

## TempFile

Creates a temp file in `tb.TempDir()` with content. Cleanup is automatic.

```go
path := testkit.TempFile(t, "config.json", []byte(`{"key":"val"}`))
```

## FreePort

Finds a free TCP port without races. Binds to `:0`, reads the assigned port, closes the listener.

```go
port := testkit.FreePort(t)
startServer(port)
```

## SortedKeys

Extracts and sorts map keys for deterministic iteration in tests.

```go
for _, k := range testkit.SortedKeys(myMap) {
    ...
}
```

## MapDiff / DiffMap

Compares two maps and returns a structured difference (added keys, removed keys, changed values). Values are compared using `cmp.Equal`.

```go
diff := testkit.DiffMap(before, after)
testkit.Len(t, diff.Added, 1, "one key added")
```

`MapDiff` is the result type; `DiffMap` is the constructor.

## TableTest

Generic table-driven test runner. Subtest names come from each case's `Name` field (or a `Name() string` method).

```go
testkit.TableTest(t, []struct {
    Name  string
    Input int
    Want  int
}{
    {"zero", 0, 0},
    {"positive", 5, 25},
}, func(t *testing.T, tc struct{ Name string; Input, Want int }) {
    testkit.Equal(t, square(tc.Input), tc.Want, "square")
})
```

## Rapid generators

Typed wrappers over `pgregory.net/rapid` primitives.

| Function | Returns |
|----------|---------|
| `DrawString(t, prefix)` | `prefix-<random>` |
| `DrawBytes(t, maxLen)` | Random byte slice up to `maxLen` |
| `DrawEnum[T ~uint8](t, max)` | Random enum value in `[0, max]` |
| `DrawUint64(t)` | Random uint64 |
| `DrawInt(t, lo, hi)` | Random int in `[lo, hi]` |
