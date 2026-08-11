# Fuzz

> **Status: designed, not implemented.**
> [RFC-0003](../../rfc/0003-the-projection-consumers.md) fixes this
> generator's design. No directive is registered yet and `testkit run` does
> not produce the outputs below; where this page and the RFC differ, the RFC
> is the authority.

The `fuzz` generator turns a `//testkit:fuzz` method into a seeded target
for Go's coverage-guided fuzzer — the only conformance mode that witnesses
many values rather than one. It reads the projection the
[suite generator](suite.md) queues, so the corpus starts from the same
fixture values the assertions use and a crashing input replays against the
harness with no translation.

## The directive

```go
//testkit:fuzz
Store(ctx context.Context, v Payload) error
```

Method-scoped, no keys, negation denied — a target exists where one is
declared, and deleting the line is the suppression. A fuzz target is CI
minutes and a corpus per method, so the method is the grain; the record of
why default-on lost is in
[RFC-0002](../../rfc/0002-the-suite-generator.md).

**Feasibility is Go's constraint, checked at codegen.** `f.Fuzz` accepts
strings, `[]byte`, booleans, integers and floats; an annotated method
qualifies iff every non-context parameter is one of those or a struct whose
exported fields recursively decompose to them. An annotated method that
does not decompose is a **diagnostic at the directive**, never a silent
skip.

## The corpus

Seeds are derived, four sources deep, each `f.Add` line carrying a comment
naming its rule:

- **The fixture pair** — `Sample` and `Other` per parameter, for replay
  symmetry with the harness.
- **The boundary alphabet, varied one parameter at a time** — empty,
  unicode and format-hostile strings; zero, ±1 and extreme integers; `NaN`
  and infinities; one whole-tuple zero-value seed. Vary-one, never the
  cross-product: every seed runs as a regression subtest per subject in
  plain `go test`.
- **Stamp-derived seeds** — a `bounded` parameter seeds `min`, `max`,
  `min-1`, `max+1`; an enum-typed parameter seeds one value per variant,
  read from the enum generator's queued projection.
- **Promotion** — `make fuzz-promote` copies what the per-machine cache
  corpus learned into `testdata/fuzz/<Target>/`, where it becomes committed
  regression input. The generated corpus is the floor; the promoted corpus
  is the compounding interest.

## What it generates

The per-input property is exported into `_fuzz.gen.go`; the discoverable
target lands in `_fuzz.gen_test.go` and ranges the suite's subject
registry, so the consumer writes no fuzz shims:

```go
// _fuzz.gen.go — the body is selected by the stamps.
func FuzzMixedStoreInput(t *testing.T, s validates.Mixed, key, body string)

// _fuzz.gen_test.go — every registered subject sees every input.
func FuzzMixedStore(f *testing.F)
```

| Stamp | The fuzzed property |
|---|---|
| *(none)* | the call returns; no panic, no hang |
| `nilsafe` | zero-valued inputs included; still no panic |
| `pure` | same input twice, same output twice |
| `bounded` | the result stays inside the declared range |
| `validates` | what the named validator refuses, the method refuses |

The subjects are real implementations — a stub answers regardless of input,
so fuzzing it proves nothing. Cross-subject agreement is deliberately not
asserted: substitutability is defined by the contract's laws, not bytewise
agreement.

## Why targets stay per-method

Go has no aggregate form: `testing.F` has no sub-targets, `f.Fuzz` runs
once per target, the corpus format is defined by the one callback's
parameter tuple, and `go test -fuzz` refuses a pattern matching more than
one target. The sequence-space counterpart — one fuzz target driving whole
action sequences — exists and belongs to the model tier as
`Fuzz<Iface>Model`; see [Model](model.md).

## Layout conventions

| Tag | Suffix | Contents |
|---|---|---|
| *(primary)* | `_fuzz.gen.go` | The exported per-input properties. |
| `test` | `_fuzz.gen_test.go` | One `Fuzz<Iface><Method>` entry per annotated method. |

The growing corpus lands in `testdata/fuzz/<Target>/` beside the target, in
the consumer's tree; `prune` never enters `testdata/`.

## See also

- [Suite](suite.md) — the projection, fixture and registry this generator
  reads.
- [Model](model.md) — sequence-space fuzzing over the same registry.
- [RFC-0003](../../rfc/0003-the-projection-consumers.md) — the design
  record, including the corpus derivation rules.
