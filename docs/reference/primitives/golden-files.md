# Golden files

```go
import "go.thesmos.sh/testkit/golden"
```

Comparing generated or serialised output against a checked-in reference. The value is in the diff: a golden file turns "the output changed" into a reviewable line-by-line change rather than a failed equality check with two blobs.

## Asserting

| Function | Compares against |
|---|---|
| `AssertGolden(tb, file string, got []byte, update bool, scrubbers ...Scrubber)` | `testdata/<file>` |
| `AssertGoldenAt(tb, path string, got []byte, update bool, scrubbers ...Scrubber)` | `path`, verbatim |
| `AssertGoldenJSONField(tb, path, field string, got []byte, update bool, scrubbers ...Scrubber)` | one field of the JSON document at `path` |

```go
func TestRenderReport(t *testing.T) {
    got := report.Render(input)
    golden.AssertGolden(t, "report.txt", got, golden.ShouldUpdate())
}
```

`AssertGoldenJSONField` is for the case where one field of a large document is what the test is about. Pinning the whole document there would make every unrelated change fail this test too.

## Updating

`ShouldUpdate()` reports whether the update flag is set, so the same test both checks and regenerates:

```
go test ./... -update
```

Pass `golden.ShouldUpdate()` rather than a hard-coded `false` — a test wired with a literal cannot be regenerated, and someone will eventually edit the golden file by hand instead.

**Review the diff before committing a regenerated file.** An update flag turns any change into a passing test, which is exactly the failure mode golden files exist to prevent. The regenerated file is the assertion; if you did not read it, the test asserts nothing.

## Scrubbers

Output carrying a timestamp, a hash or a run ID differs on every run and would make the file unstable. A `Scrubber` rewrites those to a fixed placeholder before comparison.

```go
type Scrubber func([]byte) []byte
```

| Scrubber | Replaces |
|---|---|
| `ScrubTimestamps()` | timestamps |
| `ScrubHashes()` | hex hash digests |
| `ScrubRunIDs()` | run identifiers |
| `ScrubJSONFields(fields ...string)` | the named JSON fields, whatever they contain |

They compose, and they apply to both sides:

```go
golden.AssertGolden(t, "trace.json", got, golden.ShouldUpdate(),
    golden.ScrubTimestamps(),
    golden.ScrubJSONFields("request_id", "duration_ms"),
)
```

Scrub as little as possible. Every scrubbed field is a field the golden file no longer checks, and a scrubber broad enough to catch the noise usually catches some signal with it.

## Comparing without asserting

```go
func Compare(want, got []byte, scrubbers ...Scrubber) string
```

Returns the diff, or empty when they match after scrubbing. Use it where the comparison feeds something other than a test failure — a report, a summary line, a check that counts differences.

## See also

- [Assertions](assertions.md) — `Equal` uses the same diff machinery for in-memory values.
- [Helpers](helpers.md) — `MustMarshal` for producing the bytes to compare.
