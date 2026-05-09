# Suite

Tier 1 conformance. Reads a Go interface, classifies each method into a *shape* by signature pattern, and emits an `Assert<Iface>Contract(t, factory, opts...)` test harness.

The generated suite evaluates implementations against two layers of contracts:
1. **Baseline contracts:** Auto-detected invariants derived from the method's shape (e.g., a `Reader` must return consistent reads; a `Writer` must succeed on the first try).
2. **Directive contracts:** Orthogonal behaviors triggered by `//testkit:` annotations (e.g., `//testkit:idempotent`, `//testkit:concurrent`, `//testkit:atomic`).

A `factory func() Iface` is the single required injection. Every subtest constructs a fresh implementation per run, ensuring perfect state isolation and safe parallel execution.

## Directive

```go
//go:generate testkit suite -o storetest/store_spec.gen_test.go Store
```

`suite` accepts exactly one type argument and emits a single test file (`_test.go`).

## The Conformance Pattern

The generated entry points support the **Multi-Implementation Conformance Pattern**. This is the load-bearing pattern for database migrations, refactoring, and verifying mocks against real implementations. You write the contract suite wiring *once*, and run it against *N* implementations.

### Single-Implementation Driver
```go
func AssertStoreContract(
    t *testing.T, 
    factory StoreFactory, 
    opts ...suite.Option,
)
```

### Multi-Implementation Driver
```go
func AssertStoreContractAcrossImpls(
    t *testing.T, 
    factories []StoreNamedFactory, 
    opts ...suite.Option,
)
```
This driver runs the exact same contract suite once per provided factory, wrapped in `t.Run(Name, ...)`.

## Auto-detected Baseline Contracts

Each method is classified by `shape.Detect` from its signature (see [Shapes](shapes.md)). The shape dictates which `suite.Assert<Shape>Baseline` is invoked.

Regardless of shape, every method that accepts a `context.Context` automatically receives:
- `<Method> / smoke` — fail-fast bare invocation with sample args.
- `<Method> / respects context` — asserts `ctx.Done()` halts execution and returns a context error.
- `<Method> / respects deadline` — asserts a pre-expired context fails immediately.
- `<Method> / nil context` — asserts `nil` context does not panic.

Shape baselines add deeper semantics:
- **Reader:** Asserts `consistent reads` (repeated calls return identical values).
- **StreamReader:** Asserts it `completes` (terminates), is `reentrant` (two iterators yield equal sequences), and `respects break` (early return from a `for` loop halts production cleanly).
- **Aggregator:** Asserts `count consistency` (stable across immediate re-reads).

### Smoke Test Recovery & `//testkit:sample`

Smoke tests invoke methods with zero-value inputs. If an implementation panics on zero-value inputs (e.g., an ID parameter cannot be empty), the smoke test gracefully `t.Skip`s instead of failing the suite, emitting a diagnostic message to guide the developer.

To eliminate the skip and run the smoke test, use the `//testkit:sample` directive to inject a valid builder function for the parameter:

```go
//testkit:sample SampleRecord
Put(ctx context.Context, item Record) error
```

The generator synthesizes a call to `SampleRecord(impl)` to populate the argument in smoke tests and benchmark hot-paths.

## Directive-driven Subtests

Directives layer orthogonal constraints on top of the shape baseline.

| Directive | Subtest Emitted | Behavior |
|-----------|-----------------|----------|
| `errors ErrX [ErrY...]` | `<Method> / returns <Err>` | Calls method with zero-value, asserts `errors.Is(err, ErrX)`. |
| `wrapped-via ErrX` | `<Method> / wrapped-via` | Asserts the returned error wraps `ErrX` via `errors.Is`. |
| `nilsafe` | `<Method> / nilsafe` | Asserts `nil` pointers don't panic. |
| `pure` | `<Method> / pure` | Asserts the method has no side effects and results match across impls. |
| `idempotent` | `<Method> / idempotent` | Asserts repeated calls produce the identical result. |
| `cacheable` | `<Method> / cacheable` | Asserts deterministic input → output mapping (implies `pure`). |
| `monotonic` | `<Method> / monotonic` | Asserts results are non-decreasing across consecutive calls. |
| `concurrent` | `<Method> / concurrent` | Stress-runs the method from 16x25 goroutines; asserts no race. |
| `concurrent-readers`| `<Method> / concurrent-readers` | Parallel reads, serialized writes fanout. |
| `atomic` | `<Method> / atomic` | Asserts all-or-nothing semantics on failure paths. |
| `timeout D` | `<Method> / timeout` | Asserts completion within duration `D`. |

### Cross-Method Directives

These directives emit paired-method subtests. They declare how a write on one method causally affects a read on another.

| Directive | Subtest Emitted |
|-----------|-----------------|
| `read-after-write <Reader>` | After this writer, the named reader returns the exact written value. |
| `delete-removes <Reader>` | After this deleter, the named reader returns the not-found sentinel. |
| `stream-reflects-mutations <Stream>`| After this writer, the named stream method yields the written value. |
| `lifecycle-after-close <Reader>` | After this close, the named reader returns the closed sentinel. |
| `crdt-merge <Other>` | Two impls applying ops in opposite orders converge to equal state. |

## Injecting State via `suite.Option`

Because `Assert<Iface>Contract` tests your implementation as a black box, it needs help putting the implementation into a state where certain invariants can be tested. For example, you cannot test "Read Consistent" on a database that is entirely empty.

You inject state into the driver via `suite.Option` values:

### `suite.WithPrePopulate(func(impl T))`
The most critical option. The driver wraps your factory: before handing a fresh implementation to any subtest, it calls your `PrePopulate` closure to seed the database with known data. Reader baselines and `pure` subtests rely on this state existing.

```go
AssertStoreContract(t, factory, 
    suite.WithPrePopulate(func(s basic.Store) {
        _ = s.Put(context.Background(), basic.Item{ID: "known-1"})
    }),
)
```

### `suite.WithInvalidFactory(func() T)`
Used by Mutator and Lifecycle shapes to test the "Reject Invalid" baseline. You supply a factory that produces an implementation fundamentally incapable of succeeding (e.g., a database connection with a bad password), and the suite asserts that writes predictably fail.

### `suite.WithRetryFactory(func() T)`
When a method is marked `//testkit:retry-succeeds-on-attempt N`, you must provide an implementation that simulates a transient network failure, returning errors for N-1 calls before succeeding.

## Layout Conventions

A typical interface generates into a `<pkg>test/` sub-package to prevent test dependencies from polluting the production binary.

**What goes where:**

| File | Owner | Contents |
|------|-------|----------|
| `*_spec.gen_test.go` | Generator | The generated suite harness (DO NOT EDIT). |
| `*_stub.gen.go` | Generator | The generated recording stub (DO NOT EDIT). |
| `sample_helpers.go` | Developer | Hand-written `func TestSampleX(_ Iface) X` builder functions for `//testkit:sample`. |
| `spec_test.go` | Developer | Hand-written test functions wiring `Assert<Iface>Contract` with the factory and `suite.Option` injections. |

## See also

- [Shape Classification](shapes.md) — The 21 signature shapes.
- [Generators / bench](bench.md) — Tier 4 performance benchmarking.
- [Generators / stub](stub.md) — How the companion stub interacts with the suite.
