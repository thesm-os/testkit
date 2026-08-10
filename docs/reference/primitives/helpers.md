# Helpers

```go
import "go.thesmos.sh/testkit"
```

Fixtures, I/O doubles, and the `testing.TB` stub that makes an assertion's failing path reachable from a test.

## FailableTB

An assertion that calls `tb.Fatalf` cannot be tested with a real `*testing.T` — the failure aborts the test doing the checking. `FailableTB` records the first fatal message and returns instead.

```go
func NewFailableTB() *FailableTB
```

```go
ft := testkit.NewFailableTB()
testkit.Equal(ft, 1, 2, "counts must match")

testkit.True(t, ft.Failed(), "the assertion must have fired")
testkit.Contains(t, ft.Msg(), "counts must match", "the message must reach the reader")
```

| Method | Returns |
|---|---|
| `Failed() bool` | whether `Fatal`, `Fatalf`, `Error` or `Errorf` was called |
| `Msg() string` | the first fatal message, or the most recent non-fatal one |
| `Logs() []string` | a copy of everything passed to `Log`/`Logf` |
| `HelperCalls() int` | how many times `Helper` was called |
| `Context() context.Context` | a context cancelled when the stub fails |
| `RunCleanups()` | runs registered cleanups, LIFO |

Two configuration methods, both returning the stub so they chain:

`WithGoexit()` makes `Fatal`, `Fatalf` and `FailNow` call `runtime.Goexit` — matching what a real `*testing.T` does. Use it when the code under test relies on the fatal *not* returning. Without it, execution continues past the fatal, which is what makes the message inspectable.

`WithName(name)` sets what `Name()` returns; it defaults to `"FailableTB"`.

`Skip`, `Skipf` and `SkipNow` are no-ops. `TempDir` is unsupported and fails.

`HelperCalls` exists so a test can assert an assertion marked itself as a helper — without it, a failure reports the line inside the assertion rather than the caller's line, and nobody notices until they are reading a confusing failure.

## Fixtures

| Function | Does |
|---|---|
| `TestError(name string) error` | Returns a distinct error whose message is `testkit: <name>`, matching itself under `errors.Is` |
| `RequireEnv(tb, key) string` | Returns the environment variable, or **skips** the test when unset |
| `SeededRand(tb) *rand.Rand` | A `*rand.Rand` seeded from the FNV-1a hash of the test's name |
| `TempFile(tb, name string, content []byte) string` | Writes a file under `tb.TempDir()` and returns its path |
| `FreePort(tb) int` | An available TCP port on localhost |

`SeededRand` is deterministic per test and different between tests. The same test replays identically; two tests do not accidentally share a sequence.

`RequireEnv` skips rather than fails. A test needing a credential is not broken on a machine that has none — but a skip that outlives the reason for it is deferred work with no owner, so pair it with a CI job that sets the variable.

## Collections

| Function | Returns |
|---|---|
| `SortedKeys[K, V](m map[K]V) []K` | The keys, sorted |
| `DiffMap[K, V](before, after map[K]V) MapDiff[K, V]` | What changed between two maps |

```go
type MapDiff[K comparable, V any] struct {
    Added   map[K]V    // in after, not before
    Removed map[K]V    // in before, not after
    Changed map[K][2]V // in both, different — [before, after]
}
```

`SortedKeys` is what makes a map-ranging assertion deterministic. Ranging a map directly produces a different order every run, which turns a golden file or a rendered message into a flake.

## Table tests

```go
func TableTest[T any](t *testing.T, cases []T, run func(t *testing.T, tc T))
```

Runs each case as a subtest. The case type supplies the name through a `Name` field or a `String` method.

```go
testkit.TableTest(t, []struct {
    Name string
    In   string
    Want int
}{
    {"empty", "", 0},
    {"one", "a", 1},
}, func(t *testing.T, tc struct{ Name, In string; Want int }) {
    testkit.Equal(t, len(tc.In), tc.Want, "length must match")
})
```

## I/O

| Function | Does |
|---|---|
| `MustMarshal(tb, v any) []byte` | JSON-marshals, failing the test on error |
| `MustUnmarshal(tb, data []byte, v any)` | JSON-unmarshals, failing the test on error |
| `MustDecodeHex(tb, s string) []byte` | Decodes hex, failing the test on error |
| `Quiet(tb) func()` | Suppresses `log/slog` output; call the returned function to restore |

The `Must*` forms exist so a fixture line stays one line. `MustMarshal(t, v)` says the marshalling is setup, not the thing under test — an `if err != nil` there is noise the reader has to skip past.

### Failing readers and writers

```go
type FailingReader struct {
    Source     io.Reader
    BeforeFail int
    Err        error
}

type FailingWriter struct {
    BeforeFail int
    Err        error
}
```

Each succeeds for the first `BeforeFail` bytes and then returns `Err`. This is how a partial-write or truncated-read path gets exercised — the branch that handles `n < len(p)` alongside a non-nil error, which is the one hand-written tests almost never reach.

```go
w := &testkit.FailingWriter{BeforeFail: 10, Err: io.ErrShortWrite}
err := report.WriteTo(w)
testkit.ErrorIs(t, err, io.ErrShortWrite, "a short write must propagate")
```

## See also

- [Assertions](assertions.md) — what `FailableTB` is for testing.
- [Golden files](golden-files.md) — `MustMarshal` feeds the bytes a golden comparison takes.
