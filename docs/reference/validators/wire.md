# Wire Freshness

Verifies that `testdata/wire/*.bin` golden files are
current with respect to their source proto definitions and
codec implementations. A stale golden file means the
snapshot is testing yesterday's format, not today's.

## Command

```
testkit validate wire
```

## What it checks

1. Every codec test spec that references a
   `testdata/wire/<Type>.bin` file has a golden file that
   matches the current codec output
2. No orphaned `.bin` files exist without a corresponding
   codec test spec
3. Wire files are byte-identical to what the current codec
   produces (detects silent encoding changes)

## How it works

1. Scan all `*_codec_test.go` and `*_codec_gen_test.go`
   files for `codectest.Spec` declarations
2. For each spec, marshal the sample value using the
   current codec
3. Compare against the committed `.bin` file
4. Report any mismatches or missing files

## Failure output

```
wire: FAIL

  store/testdata/wire/ItemList.bin
    stale: committed file differs from current codec output
    re-run: testkit wire store

  model/testdata/wire/Entry.bin
    orphaned: no codectest.Spec references this file
```

## Why

Wire snapshots are the primary defence against accidental
wire-format regressions. If a developer changes a proto
field order or type and regenerates the codec but forgets
to update the golden file, the codec test passes (it uses
the stale golden as "expected") but the new wire format is
incompatible with previously-persisted data. The validator
catches this by independently marshalling from the current
codec and comparing.

## Configuration

```yaml
# .testkit.yaml
validators:
  wire:
    enabled: true
```

---
