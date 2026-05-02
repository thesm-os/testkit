# Golden Files

## AssertGolden

Update-or-compare pattern for wire snapshots, fixture
files, and expected output. Flag registration (`-update`)
is automatic via `init()`.

```go
testkit.AssertGolden(t, "snapshot.bin", got)

// To regenerate: go test -update ./...
```

Behaviour:

- File missing + `-update`: write, pass.
- File missing + no flag: fail with instruction.
- Content matches: pass.
- Content differs + `-update`: overwrite, pass.
- Content differs + no flag: fail with line-level diff.

Golden artifacts live in `testdata/golden/` relative to
the test's package directory.

## Scrubbers

Byte-level transformations applied to golden output before
comparison. Normalize volatile fields that are
deterministic within a run but irrelevant to the
behavioural assertion.

| Scrubber | Description |
|----------|-------------|
| `ScrubJSONFields(fields...)` | Replace named JSON fields with "SCRUBBED" |
| `ScrubTimestamps()` | Replace ISO-8601/RFC-3339 timestamps with "SCRUBBED_TS" |
| `ScrubHashes()` | Replace hex digests (32/40/64/96/128 chars) with "SCRUBBED_HASH" |
| `ScrubRunIDs()` | Replace `run_[a-z0-9]{16}` tokens with "SCRUBBED_RUN" |

```go
testkit.AssertGolden(t, "response.json", got,
    testkit.ScrubTimestamps(),
    testkit.ScrubJSONFields("request_id", "trace_id"),
)
```

Custom scrubbers implement `func([]byte) []byte`.
