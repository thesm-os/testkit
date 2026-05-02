# Helpers

Smaller utilities that don't warrant their own document.

## TestError

Creates a named, deterministic error for use in test
fixtures. Replaces ad-hoc `errors.New("boom")` with a
traceable, `errors.Is`-compatible sentinel.

```go
var errBoom = testkit.TestError("boom")
// errBoom.Error() == "testkit: boom"

stub := NewStubStore(
    testkit.WithPutFault(errBoom, 3),
)
err := stub.Put(ctx, req)
testkit.ErrorIs(t, err, errBoom, "must propagate fault")
```

Every `TestError` is distinct — `TestError("x")` and
`TestError("x")` are two different errors that do not
satisfy `errors.Is`.

## RequireEnv

Skips the test if an environment variable is not set;
returns its value if present.

```go
func TestPostgres_Integration(t *testing.T) {
    dsn := testkit.RequireEnv(t, "DATABASE_URL")
    pool, err := pgxpool.New(ctx, dsn)
}
```

## SeededRand

Deterministic `*rand.Rand` seeded from the test's name
via FNV-1a hash. Two runs of the same test produce
byte-identical sequences.

```go
rng := testkit.SeededRand(t)
idx := rng.IntN(len(items))
```

Underlying source is `math/rand/v2` PCG generator.

## MustMarshal / MustUnmarshal

Fixture construction helpers that fatal the test on
marshal/unmarshal failure.

```go
data := testkit.MustMarshal(t, fixture)
testkit.MustUnmarshal(t, data, &target)
```

Do NOT use for the subject-under-test's marshal behaviour.

## FailingReader / FailingWriter

`io.Reader` / `io.Writer` that return successful bytes
for the first N bytes, then return an error on every
subsequent call. Simulates mid-stream failures.

```go
r := &testkit.FailingReader{
    Source:     bytes.NewReader(payload),
    BeforeFail: 100,
    Err:        io.ErrUnexpectedEOF,
}
```

## Quiet

Suppresses `log/slog` default output for the duration of
a test. Tests using Quiet must NOT call `t.Parallel`.

```go
func TestSubsystem_LogsOnFailure(t *testing.T) {
    defer testkit.Quiet(t)()
    subject.DoThingThatLogs()
}
```

## FailableTB

Stub `testing.TB` that captures the first `Fatalf` /
`Fatal` / `Errorf` without aborting the host goroutine.
For negative-space tests.

```go
ftb := testkit.NewFailableTB().WithName("TestFoo")
someAssertionHelper(ftb, badInput)
testkit.True(t, ftb.Failed(), "must fatal on bad input")
```

| Method | Description |
|--------|-------------|
| `NewFailableTB()` | Create fresh stub |
| `WithName(s)` | Set `Name()` return (fluent) |
| `Failed() bool` | Any fatal/error recorded? |
| `Msg() string` | First captured failure message |
| `Logs() []string` | Captured `Logf` lines |
| `HelperCalls() int` | Number of `Helper()` invocations |

## TempFile

Creates a temp file with content in one call. Cleanup is
automatic via `t.TempDir`.

```go
path := testkit.TempFile(t, "config.json", []byte(`{"key":"val"}`))
```

## FreePort

Finds a free TCP port without races. Binds to `:0`, reads
the assigned port.

```go
port, cleanup := testkit.FreePort(t)
defer cleanup()
startServer(port)
```

## SortedKeys[K, V]

Extracts and sorts map keys for deterministic test output.

```go
keys := testkit.SortedKeys(myMap)
```

## DiffMap[K, V]

Compares two `map[K]V` values and produces a structured
diff (added keys, removed keys, changed values).

```go
diff := testkit.DiffMap(before, after)
testkit.Len(t, diff.Added, 1, "one key added")
```

## TableTest[T]

Generic table-driven test runner with automatic subtest
naming from the `Name` field.

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

## Rapid Generators

Typed wrappers over `pgregory.net/rapid` primitives.

| Function | Description |
|----------|-------------|
| `DrawString(t, prefix)` | `prefix-<random>` |
| `DrawBytes(t, maxLen)` | Random byte slice up to maxLen |
| `DrawEnum[T ~uint8](t, max)` | Random enum value in [1, max] |
| `DrawUint64(t)` | Positive uint64 |
| `DrawInt(t, min, max)` | Random int in [min, max] |
