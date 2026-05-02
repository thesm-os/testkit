# Wire Snapshot Regeneration

Regenerates all `testdata/wire/*.bin` golden files for
every `codectest.Spec` found in a package's test files.

## go:generate directive

```go
//go:generate testkit wire
```

Wire takes no arguments — it scans all `codectest.Spec`
declarations in the current package's test files.

## Output

Overwrites `testdata/wire/*.bin` files in the target
package. No new Go files generated.

## What it does

1. Scan all `*_codec_test.go` and `*_codec_gen_test.go`
   files for `codectest.Spec` declarations
2. For each spec, marshal the `Sample` value using the
   current codec
3. Write the result to `testdata/wire/<Type>.bin`
4. Report which files were updated

Equivalent to `go test -update-wire-snapshots` but runs
in one pass across all packages, validates every proto
message has a corresponding spec, and reports missing
specs as errors.

## Why

Wire snapshots are the primary defence against accidental
wire-format regressions. Regenerating them is a mechanical
step that should never require human judgment. The
generator ensures every proto type has a snapshot and
every snapshot matches the current codec.

## Scale

33 existing wire snapshots. Automation, not line savings.

---
