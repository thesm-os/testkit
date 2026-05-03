# Golden Files

## AssertGolden

Update-or-compare pattern for wire snapshots, fixture files, and expected output. Flag registration (`-update`) is automatic via `init()`.

```go
testkit.AssertGolden(t, "snapshot.bin", got)

// Regenerate goldens:
//   go test -update ./...
```

| Outcome | Behavior |
|---------|----------|
| File missing + `-update` | Write, pass |
| File missing + no flag | Fail with regenerate instruction |
| Content matches | Pass |
| Content differs + `-update` | Overwrite, pass |
| Content differs + no flag | Fail with line-level diff |

Golden artifacts live under `testdata/golden/` relative to the test's package directory.

## ShouldUpdate

```go
if testkit.ShouldUpdate() {
    // -update flag was passed; regenerate fixture
}
```

For test code that needs to know whether the user passed `-update` (e.g., when generating fixtures alongside the assertion).

## Scrubbers

Byte-level transformations applied to golden output before comparison. Use them to normalize volatile fields that are deterministic within a run but irrelevant to the behavioral assertion.

| Scrubber | What it does |
|----------|---|
| `ScrubJSONFields(fields...)` | Replace named JSON field values with `"SCRUBBED"` |
| `ScrubTimestamps()` | Replace ISO-8601 / RFC-3339 timestamps with `"SCRUBBED_TS"` |
| `ScrubHashes()` | Replace hex digests (32-128 chars) with `"SCRUBBED_HASH"` |
| `ScrubRunIDs()` | Replace `run_[a-z0-9]{16}` tokens with `"SCRUBBED_RUN"` |

```go
testkit.AssertGolden(t, "response.json", got,
    testkit.ScrubTimestamps(),
    testkit.ScrubJSONFields("request_id", "trace_id"),
)
```

`Scrubber` is just `func([]byte) []byte`, so custom scrubbers compose freely.

`ScrubJSONFields` uses a regex approach — not a real JSON parser. Adequate for stable test fixtures; not robust against arbitrary JSON.
