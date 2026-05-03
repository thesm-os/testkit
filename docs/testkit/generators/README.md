# testkit Generators

A CLI tool that reads Go type definitions and proto descriptors and emits the test infrastructure you would otherwise write by hand. Every generator produces plumbing; the developer injects domain logic via constructor arguments, oracle interfaces, or option closures. The generated code is stable — adding a field to a struct or a method to an interface triggers regeneration; the developer only touches the injection point.

All generators use `go/types` for Go source analysis and `protodesc` for proto schema. Generators are invoked via `//go:generate` directives in the source package — no central type registry. Project-wide conventions (suffix, test package style, directive composition) live in `.testkit.yml`; see [Configuration](../configuration.md).

**Status.** Four generators ship today: `stub`, `builder`, `sentinel`, `enum`. The remaining generators (`suite`, `model`, `bench`, `sim`, `chaos`, `differential-rollout`, `replay`, `codec`, `smoke`, `pkgdoc`) are designed but not yet implemented; their docs document the planned shape and are clearly marked.

## CLI

```
testkit <subcommand> [flags] [Type ...]

Subcommands:
    stub                    Generate per-method test doubles
    builder                 Generate fluent fixture builders
    sentinel                Generate error sentinel tests
    enum                    Generate enum exhaustiveness + stringer tests
    codec                   Generate codec test specs from proto
    suite                   Generate Tier 1 conformance suite
    model                   Generate Tier 2-3 state-machine harness
    bench                   Generate Tier 4 benchmark contracts
    sim                     Generate Tier 5 subsystem simulation harness
    chaos                   Generate Tier 5 fault-injection sim
    differential-rollout    Generate Tier 5 shadow-traffic harness
    replay                  Generate Tier 5 trace-replay harness
    smoke                   Generate CLI command coverage tests
    pkgdoc                  Generate package audit doc skeleton

Per-generator flags:
    -o <file>      Output file path
    -check         Dry-run: exit non-zero if output differs
    -v             Verbose logging
```

## Conformance tiers

Each conformance generator targets a tier. Lower tiers run in seconds; higher tiers run for minutes-to-hours.

| Tier | Generator(s) | What it proves |
|------|--------------|----------------|
| primitives | [`stub`](stub.md) | Runtime substrate for tiers 1-5 |
| fixtures | [`builder`](builder.md) | Domain-value construction |
| static | [`sentinel`](sentinel.md), [`enum`](enum.md) | Compile-time + structural invariants |
| wire | [`codec`](codec.md) | Round-trip + binary fixtures |
| 1 | [`suite`](suite.md) | Single-call contract per directive |
| 2-3 | [`model`](model.md) | Property-based state-machine, differential, workload |
| 4 | [`bench`](bench.md) | Allocation + latency ceilings, complexity shape |
| 5 | [`sim`](sim.md) | Subsystem simulation harness |
| 5 | [`chaos`](chaos.md) | Continuous fault injection on top of sim |
| 5 | [`differential-rollout`](differential-rollout.md) | Shadow-traffic comparison |
| 5 | [`replay`](replay.md) | Production-trace replay |
| CLI | [`smoke`](smoke.md) | Cobra command coverage |
| compliance | [`pkgdoc`](pkgdoc.md) | Audit-doc scaffolding |

## Output file control

Every generator accepts `-o <file>` to set the output path. Multiple types passed as arguments are written into the single output file. When `-o` is omitted, each generator uses its conventional default.

| Generator | Default output |
|-----------|---------------|
| `stub`                  | `<pkg>test/<subject>_stub.gen.go` |
| `builder`               | `<pkg>test/<subject>_builder.gen.go` |
| `suite`                 | `<pkg>test/<subject>_spec.gen.go` |
| `model`                 | `<pkg>test/<subject>_model.gen.go` |
| `bench`                 | `<pkg>test/<subject>_bench.gen.go` |
| `sim`                   | `<pkg>test/<subject>_sim.gen.go` |
| `chaos`                 | `<pkg>test/<subject>_chaos.gen.go` |
| `differential-rollout`  | `<pkg>test/<subject>_diffrollout.gen.go` |
| `replay`                | `<pkg>test/<subject>_replay.gen.go` |
| `codec`                 | `<subject>_codec.gen_test.go` |
| `sentinel`              | `errors.gen_test.go` |
| `enum`                  | `<subject>_enum.gen_test.go` |
| `smoke`                 | `cmd/<binary>/smoke.gen_test.go` |
| `pkgdoc`                | `docs/compliance/package-audit/<pkg>.md` |

## Directive vocabulary

Conformance generators are driven by `//testkit:` directives on interface methods. Directives are validated at codegen time — unknown directives error, conflicts error, redundancies warn. See [`Configuration`](../configuration.md) for the full vocabulary and composition rules.

The generators that consume directives:

- **`stub`** — `errors`, `deprecated`, `integration-only`, `retry-succeeds-on-attempt`, `partition`, `order-after`
- **`suite`** — most behavioral directives (`nilsafe`, `ctx`, `timeout`, `pure`, `validates`, `bounded`, `errors`, `deprecated`, `req`, `integration-only`)
- **`model`** — property directives (`idempotent`, `cacheable`, `monotonic`, `concurrent`, `atomic`, `invariant`, `consistency`, `lease`, `pagination`, `eventually`, `sideeffect`, `partition`, `order-after`, `errors`, `req`)
- **`bench`** — `allocs`, `latency`, `complexity`, `concurrent`, `timeout`, `req`
- **`sim`** — directives that influence simulation behavior (most of the matrix)
- **`chaos`** / **`differential-rollout`** / **`replay`** — directives that map onto each tier-5 lens (fault scheduling, equivalence relations, replay ordering)
- **`smoke`** — CLI-specific (`exit-code`, `golden-output`, `signal`, `flag-validation`, etc.)
- **`builder`** — `default` only

`sentinel`, `enum`, `codec`, and `pkgdoc` are directive-free or consume their own narrow vocabulary.

## Generators

| Generator | Tier | Status | What it produces |
|-----------|------|--------|------------------|
| [`stub`](stub.md) | primitives | ready | Per-method test doubles (recording, faults, clock, gates, strict) — substrate for tiers 1-5 |
| [`builder`](builder.md) | fixtures | ready | Fluent `With*` builders, `Append*`, `WithEntry`, `Mutate`, `Clone` |
| [`sentinel`](sentinel.md) | static | ready | Prefix, uniqueness, non-overlap, unwrap-chain, custom-error round-trip |
| [`enum`](enum.md) | static | ready | Exhaustiveness, stringer round-trip, out-of-range, optional Marshal/Parse |
| [`codec`](codec.md) | wire | planned | `codectest.Spec[T]`, round-trip suite + bench + fuzz seeds + `testdata/wire/*.bin` fixtures |
| [`suite`](suite.md) | 1 | planned | `AssertContract(t, factory, opts...)` — one subtest per (method × directive) |
| [`model`](model.md) | 2-3 | planned | rapid state-machine: `RunStateMachine` + `RunDifferential` + `RunWorkload` |
| [`bench`](bench.md) | 4 | planned | `BenchmarkContract(b, factory, opts...)` with `AllocsMax(N)` / `LatencyMax(X)` per directive |
| [`sim`](sim.md) | 5 | planned | Subsystem simulation harness — engine clock + rand + capture-on-failure + workloads + invariants |
| [`chaos`](chaos.md) | 5 | planned | Continuous fault simulation: random schedules, partitions, skew, restarts |
| [`differential-rollout`](differential-rollout.md) | 5 | planned | Shadow-traffic harness with response comparison and divergence reporting |
| [`replay`](replay.md) | 5 | planned | Production-trace replay across versions |
| [`smoke`](smoke.md) | CLI | planned | Cobra command coverage with sampled flag combinations and golden-output diffs |
| [`pkgdoc`](pkgdoc.md) | compliance | planned | Audit-doc skeleton with REQ table, refactor banner, evidence section |

## CI integration

Standard `//go:generate` flow:

```yaml
- name: Verify generated code
  run: |
    go generate ./...
    git diff --exit-code
```

## Adoption order

When adopting incrementally, follow the dependency order:

1. **`stub`** — runtime substrate; install once per interface.
2. **`builder`** — replaces inline `Item{...}` construction.
3. **`sentinel`** + **`enum`** — quick-win static checks.
4. **`suite`** — Tier 1 conformance for any interface with documented contracts.
5. **`bench`** — once interfaces have allocation/latency contracts.
6. **`model`** — for stateful interfaces with property contracts.
7. **`codec`** — for wire-format types.
8. **`smoke`** — for CLI binaries.
9. **`sim`** — once you have multiple interfaces composed in a subsystem.
10. **`chaos`** / **`replay`** / **`differential-rollout`** — pre-prod tier; install per workload.
11. **`pkgdoc`** — once a package's compliance audit is needed.

See [Adoption](../adoption.md) for the full incremental-adoption guide.
