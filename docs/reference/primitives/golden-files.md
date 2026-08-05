# Golden Files

The `golden` package (`testkit/golden`) provides update-or-compare assertions for wire snapshots, fixture files, and expected output.

## AssertGolden

Compares `got` against a golden file under `testdata/golden/`. Flag registration (`-update`) is automatic via `init()`.

```go
golden.AssertGolden(t, "snapshot.bin", got, golden.ShouldUpdate())

// Regenerate goldens:
//   go test -update ./...
```

| Outcome | Behavior |
|---------|----------|
| File missing + update=true | Write, pass |
| File missing + update=false | Fail with regenerate instruction |
| Content matches | Pass |
| Content differs + update=true | Overwrite, pass |
| Content differs + update=false | Fail with line-level diff |

## AssertGoldenAt

Same as `AssertGolden` but takes a literal file path instead of a `testdata/golden/`-relative name. Useful when the golden file lives outside the conventional directory.

```go
golden.AssertGoldenAt(t, "path/to/expected.json", got, golden.ShouldUpdate())
```

## AssertGoldenJSONField

Per-field update-or-compare for JSON golden files. Updates or compares a single field within a JSON object, preserving sibling fields across regenerations.

```go
golden.AssertGoldenJSONField(t, "api.golden.json", "users", got,
    golden.ShouldUpdate(),
    golden.ScrubTimestamps(),
)
```

| Outcome | Behavior |
|---------|----------|
| Field present + matches | Pass |
| Field present + differs + update=false | Fail with `cmp.Diff` over the field |
| Field present + differs + update=true | Replace field value; siblings preserved |
| Field absent + update=true | Add field |
| Field absent + update=false | Fail with regenerate instruction |

Comparison is structural (both sides re-marshaled with the same indent) so whitespace differences don't false-fail.

## Compare

Returns a diff string (empty when equal) without failing the test. Useful for custom assertion logic.

```go
if diff := golden.Compare(want, got, golden.ScrubTimestamps()); diff != "" {
    t.Errorf("mismatch:\n%s", diff)
}
```

## ShouldUpdate

```go
if golden.ShouldUpdate() {
    // -update flag was passed; regenerate fixture
}
```

## Scrubbers

Byte-level transformations applied to golden output before comparison. Use them to normalize volatile fields that are deterministic within a run but irrelevant to the behavioral assertion.

| Scrubber | What it does |
|----------|---|
| `ScrubJSONFields(fields...)` | Replace named JSON field values with `"SCRUBBED"` |
| `ScrubTimestamps()` | Replace ISO-8601 / RFC-3339 timestamps with `"SCRUBBED_TS"` |
| `ScrubHashes()` | Replace hex digests (32-128 chars) with `"SCRUBBED_HASH"` |
| `ScrubRunIDs()` | Replace `run_[a-z0-9]{16}` tokens with `"SCRUBBED_RUN"` |

```go
golden.AssertGolden(t, "response.json", got, golden.ShouldUpdate(),
    golden.ScrubTimestamps(),
    golden.ScrubJSONFields("request_id", "trace_id"),
)
```

`Scrubber` is just `func([]byte) []byte`, so custom scrubbers compose freely.

`ScrubJSONFields` uses a regex approach — not a real JSON parser. Adequate for stable test fixtures; not robust against arbitrary JSON.
