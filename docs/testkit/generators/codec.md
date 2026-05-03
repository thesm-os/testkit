# Codec

> **Status:** not yet implemented. Targeted for a subsequent dev cycle. The behavior described below is the design intent — code, flags, and output may differ once shipped.

Wire-correctness generator. Reads a proto descriptor and emits a `codectest.Spec[T]` round-trip suite plus per-spec benchmark, fuzz seeds, and binary wire fixtures (`testdata/wire/*.bin`) regenerated when codec semantics change. Single source of truth across the spec and the wire artifact — no hand-written field-classification tables.

## Planned directive

```go
//go:generate testkit codec -o snapshot_codec.gen_test.go ../../api/proto/myapp/types/v1/model/snapshot.proto
```

## Planned modes

- **Spec emission** (default) — generate the spec, suite, bench, fuzz seeds.
- **`-update-wire`** — regenerate `testdata/wire/<Type>.bin` from the current codec output.

## Planned injection point

Sample-value overrides via `<Type>Sample()` convention in the test package.

## See also

- [Primitives / Golden files](../primitives/golden-files.md) — the assertion engine for `.bin` fixtures
