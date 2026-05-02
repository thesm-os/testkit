# testkit

Test infrastructure for Go that generates the code you'd write by hand — stubs, recording wrappers, fixture builders, conformance suites, property-based models — then generates the tests that prove the generated code works. You write domain logic. testkit writes plumbing.

```go
//go:generate testkit stub      -o storetest/in_memory_store.gen.go   Store
//go:generate testkit recording -o storetest/recording_store.gen.go   Store
//go:generate testkit builder   -o storetest/user_builder.gen.go      User
//go:generate testkit suite     -o storetest/store_spec.gen.go        Store
//go:generate testkit model     -o storetest/store_model.gen.go       Store
//go:generate testkit sentinel  -o errors.gen_test.go
//go:generate testkit enum      -o status_enum.gen_test.go            Status
```

```bash
go generate ./...   # generates code + tests for that code
go test ./...       # 100% coverage of the generated plumbing
```

## Why

Every Go project with interfaces ends up writing the same test infrastructure: in-memory stubs with fault injection, recording wrappers for call verification, fixture builders for every struct, conformance suites for every interface contract. It's thousands of lines of mechanical code that follows a predictable pattern — but every team writes it from scratch, introduces subtle bugs in the process, and never tests the plumbing itself.

testkit eliminates that entire class of work. You declare what to generate via `//go:generate` directives. testkit reads your Go types with `go/types`, produces the infrastructure, produces tests for the infrastructure, and hands you an injection point for the one part that requires human judgment: domain logic.

## What it does

### Primitives

A focused set of test utilities that earn their place. Every assertion uses [`go-cmp`](https://github.com/google/go-cmp) for structural diffs — not `reflect.DeepEqual`.

```go
// Assertions with mandatory contract messages.
testkit.Equal(t, got, want, "Get must return the stored item")
testkit.ErrorIs(t, err, store.ErrNotFound, "Get on missing key must return ErrNotFound")

// Fluent chains for multi-property assertions.
testkit.Assert(t, user).
    IsNotNil("must exist").
    HasLen(3, "must have 3 fields populated")
```

```go
// Benchmark contracts that fail the build on regression.
c := testkit.StartContract(b).AllocsMax(0).LatencyMax(5 * time.Microsecond)
for c.Loop() {
    store.Get(b.Context(), key)
}
c.End()
```

```go
// Recorder with filtering, waiting, hooks, and gating.
rec := testkit.NewRecorder[PutCall]()
rec.OnRecord(func(c PutCall) { trace.Append(tick, c) })   // hook into sim
rec.WaitForN(t, 3, 5*time.Second)                          // async wait
gate := rec.NewGate()                                       // block calls for race testing
```

### Generators (14)

Each generator reads your Go types and produces two files: the infrastructure and its tests.

| Generator | What it produces | What you write |
|-----------|-----------------|----------------|
| `stub` | Three-tier in-memory stub (function delegation + fault injection) + tests | Domain state + default functions |
| `recording` | Call-capturing decorator with assertion helpers + tests | Nothing — fully mechanical |
| `builder` | Fluent `With*` builder per exported field + tests | Defaults constructor |
| `suite` | Conformance suite from `//testkit:` contract directives | `Factory` function |
| `model` | `rapid.StateMachine` with oracle interface | Oracle implementation |
| `codec` | `codectest.Spec[T]` from proto descriptors | Sample value overrides |
| `sentinel` | Prefix, non-overlap, uniqueness, unwrap tests | Nothing |
| `enum` | Exhaustiveness, stringer, boundary, round-trip tests | Nothing |
| `differential` | Fan-out wrapper comparing N implementations | Implementation list |
| `smoke` | One-call-per-method baseline test | Spec with sample inputs |
| `simworkload` | Random method dispatch for simulation drivers | Workload hints |
| `integration` | `TestMain` + container + conformance suite wiring | Factory function |
| `wire` | Regenerates `testdata/wire/*.bin` golden files | Nothing |
| `pkgdoc` | Compliance audit skeleton with auto-fill + refresh | Domain analysis |

Plus `testkit scaffold` for one-time companion file generation with TODO markers.

### Validators (18)

Static checks that run in CI. No test execution required — pure code and config analysis.

**Structural** — proto-sync, migration chain, depguard, wire freshness, error prefix, skip expiry

**Test quality** — assertion-free tests, test naming conventions, `time.Sleep` detection, orphaned test doubles, parallel safety (`t.Setenv` + `t.Parallel`), contract-benchmark completeness

**Quality gates** — benchmark contracts, benchmark regression vs baseline, per-layer coverage thresholds, per-layer mutation thresholds

**Compliance** — audit doc completeness, REQ-to-test traceability

## The injection principle

Generated code never contains domain logic. Every generator has a documented injection point — a hand-written companion file where you provide the semantics.

```
                    ┌──────────────────────┐
                    │   //go:generate       │
                    │   testkit stub Store  │
                    └──────────┬───────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                 ▼
    in_memory_store.gen.go   in_memory_store.go  in_memory_store.gen_test.go
    ─────────────────────    ─────────────────   ──────────────────────────
    StoreStub struct         storeState struct   TestStoreStub_Default
    NewStoreStub(...)        NewInMemoryStore()  TestStoreStub_WithFunc
    WithGetFunc option       state.get()         TestStoreStub_WithFault
    WithGetFault option      state.put()         ...per method
    Get/Put/Delete/List      state.delete()
                             state.list()
              │                │                 │
          generated        hand-written       generated
          by testkit       by you             by testkit
```

The generated stub provides the structural skeleton (interface check, option types, fault injection, method dispatch). You write the in-memory state and default functions. testkit generates tests that verify the skeleton works — 100% branch coverage of every `WithFunc`, every `WithFault`, every method delegation path.

## Contract directives

Conformance suites are driven by `//testkit:` directives on interface methods — machine-readable, grep-able, lint-able:

```go
//testkit:errors ErrNotFound ErrConflict
//testkit:idempotent
//testkit:concurrent

// Get retrieves an item by ID.
func (s *Store) Get(ctx context.Context, id string) (Item, error)
```

The suite generator reads these and emits: happy path, nil input, two error-sentinel subtests, an idempotency subtest, and a concurrency subtest. Methods without directives get only happy path and nil input — no vacuous tests.

## Quick start

```bash
# Install
go install go.thesmos.sh/testkit/cmd/testkit@latest

# Add to your project
go get go.thesmos.sh/testkit@latest

# Add directives to your package
cat >> store/generate.go << 'EOF'
package store

//go:generate testkit stub      -o storetest/in_memory_store.gen.go Store
//go:generate testkit recording -o storetest/recording_store.gen.go Store
//go:generate testkit builder   -o storetest/user_builder.gen.go    User
//go:generate testkit suite     -o storetest/store_spec.gen.go      Store
//go:generate testkit sentinel  -o errors.gen_test.go
EOF

# Generate
go generate ./...

# Scaffold the companion file (one-time)
testkit scaffold stub storetest Store

# Fill in domain logic, then test
go test ./...
```

## Output naming conventions

Generated files follow your existing naming conventions — not testkit's:

| Generator | Default output |
|-----------|---------------|
| `stub` | `<pkg>test/in_memory_<subject>.gen.go` |
| `recording` | `<pkg>test/recording_<subject>.gen.go` |
| `builder` | `<pkg>test/<subject>_builder.gen.go` |
| `model` | `<pkg>test/<subject>_model.gen.go` |
| `suite` | `<pkg>test/<subject>_spec.gen.go` |

Override with `-o` when you want a different path or want to combine multiple types into one file:

```go
//go:generate testkit enum -o status_enum.gen_test.go OrderStatus PaymentStatus RefundStatus
```

## CI integration

```makefile
check: lint test check-structural check-test-quality check-quality check-compliance

check-structural:
    go generate ./... && git diff --exit-code
    testkit validate proto-sync
    testkit validate migration
    testkit validate depguard
    testkit validate wire
    testkit validate error-prefix
    testkit validate skip-expiry

check-test-quality:
    testkit validate assertion-free
    testkit validate test-naming
    testkit validate time-sleep
    testkit validate orphaned-doubles
    testkit validate parallel-safety
    testkit validate contract-completeness

check-quality:
    testkit validate benchmarks
    testkit validate bench-regression
    testkit validate coverage
    testkit validate mutation

check-compliance:
    testkit validate audit
    testkit validate reqs
```

## Dependencies

The core `testkit` package:

- [`github.com/google/go-cmp`](https://github.com/google/go-cmp) — structural diffs for assertions
- [`pgregory.net/rapid`](https://pgregory.net/rapid) — property-based testing generators

Optional sub-packages with isolated dependencies:

| Package | Adds |
|---------|------|
| `testkit/container` | `testcontainers-go` |
| `testkit/httptest` | stdlib only |
| `testkit/oteltest` | `go.opentelemetry.io/otel/sdk` |
| `testkit/clitest` | stdlib only |

## Documentation

Full reference documentation lives in [`docs/testkit/`](docs/testkit/README.md):

- [Primitives](docs/testkit/primitives/README.md) — assertions, recording, fault injection, benchmarking, golden files, polling
- [Generators](docs/testkit/generators/README.md) — 14 generators with full code examples and injection point documentation
- [Validators](docs/testkit/validators/README.md) — 18 CI checks organized by tier
- [Configuration](docs/testkit/configuration.md) — `.testkit.yml` reference
- [Adoption](docs/testkit/adoption.md) — incremental adoption guide

## License

[MIT](LICENSE)
