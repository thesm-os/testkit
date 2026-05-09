# testkit Generators

A CLI tool that reads Go type definitions and proto descriptors and emits the test infrastructure you would otherwise write by hand. Every generator produces plumbing; the developer injects domain logic via constructor arguments, oracle interfaces, or option closures. The generated code is stable — adding a field to a struct or a method to an interface triggers regeneration; the developer only touches the injection point.

All generators use `go/types` for Go source analysis and `protodesc` for proto schema. Generators are invoked via `//go:generate` directives in the source package — no central type registry. Project-wide conventions (suffix, test package style, directive composition) live in `.testkit.yml`; see [Configuration](../configuration.md).

**Status.** Six generators ship today: `stub`, `builder`, `sentinel`, `enum`, `suite`, `bench`. The remaining generators (`model`, `sim`, `chaos`, `differential-rollout`, `replay`, `codec`, `smoke`, `pkgdoc`) are designed but not yet implemented; their docs document the planned shape and are clearly marked.

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
| 2-3 | [`model`](model.md) | Property-based state-machine, differential, workload (planned) |
| 4 | [`bench`](bench.md) | Allocation, latency, and per-percentile budgets across 21 shapes |
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

Conformance generators are driven by `//testkit:` directives on interface methods. Directives are validated at codegen time — unknown directives error, conflicting combinations error, redundant pairs warn. The full registry lives in `generator/directive/known.go`; the top-level [`README`](../../../README.md#directives) carries the per-generator consumption matrix.

What each shipped generator consumes:

- **`stub`** — `errors`, `wrapped-via`, `deprecated`, `integration-only`, `retry-succeeds-on-attempt`, `partition`, `order-after`.
- **`suite`** — every behavioral and cross-method directive: `errors`, `wrapped-via`, `nilsafe`, `pure`, `idempotent`, `cacheable`, `monotonic`, `concurrent`, `concurrent-readers`, `atomic`, `bounded`, `timeout`, `sideeffect`, `validates`, `hooks`, `eventually`, `scope`, `pagination`, `lease`, `partition`, `order-after`, `retry-succeeds-on-attempt`, `wrapped-via`, `deprecated`, `read-after-write`, `delete-removes`, `stream-reflects-mutations`, `lifecycle-after-close`, `crdt-merge`, `sample`, `integration-only`.
- **`bench`** — `allocs`, `latency`, `percentiles`, `sample`, `integration-only`.
- **`builder`** — `default` only (field-scoped).
- **`sentinel`** — `sentinel-no-overlap-with` (declares additional packages for non-overlap verification).

Planned generators (`model`, `sim`, `chaos`, `differential-rollout`, `replay`, `smoke`, `codec`, `pkgdoc`) document their planned directive surface in their respective doc files. `enum` is directive-free — it derives its assertions from the type's stringer-tagged constants.

Shape hints (`deleter`, `mutator`, `not-mutator`, `keyfield`) influence shape detection during analysis but don't add subtests directly. The `req` directive carries requirement IDs into generated subtests' names so REQ-to-test traceability surfaces in `go test -v` output.

## Generators

| Generator | Tier | Status | What it produces |
|-----------|------|--------|------------------|
| [`stub`](stub.md) | primitives | ready | Per-method test doubles (recording, faults, clock, gates, strict) — substrate for tiers 1-5 |
| [`builder`](builder.md) | fixtures | ready | Fluent `With*` builders, `Append*`, `WithEntry`, `Mutate`, `Clone` |
| [`sentinel`](sentinel.md) | static | ready | Prefix, uniqueness, non-overlap, unwrap-chain, custom-error round-trip |
| [`enum`](enum.md) | static | ready | Exhaustiveness, stringer round-trip, out-of-range, optional Marshal/Parse |
| [`codec`](codec.md) | wire | planned | `codectest.Spec[T]`, round-trip suite + bench + fuzz seeds + `testdata/wire/*.bin` fixtures |
| [`suite`](suite.md) | 1 | ready | `Assert<Iface>Contract(t, factory, opts...)` with 21-shape-detected subtests + typed plug-in points |
| [`model`](model.md) | 2-3 | planned | rapid property-based state-machine with differential SUT/ref testing, shape-specific laws, concurrent stress, trace combinators |
| [`bench`](bench.md) | 4 | ready | `Benchmark<Iface>Contract(b, factory, opts...)` with 21-shape-detected hot-path / concurrent benchmarks, opt-in `allocs`/`latency`/`percentiles` budget gates, typed plug-in points |
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
