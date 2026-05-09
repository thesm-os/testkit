# Shape Classification

Shape classification is the analytical engine of the `testkit` generators. The `generator/shape` package uses signature-driven analysis to categorize Go interface methods into **21 canonical Shapes**. Downstream generators (`suite`, `bench`, `model`) use this classification to automatically apply the correct mathematical laws, performance budgets, and state-machine transitions without requiring you to write custom test logic.

## Architecture: The Priority Registry

Classification is handled by a priority-ordered registry of detectors. The registry walks the 21 detectors from highest priority to lowest; the **first match wins**. This cascade allows highly specialized shapes to claim a signature before a more generic fallback detector catches it.

| Priority Band | Characteristics | Example Shapes |
|---------------|-----------------|----------------|
| **1000 - 900** | **High-Signal:** Unambiguous patterns like `iter.Seq` returns or interface-typed parameters. | `StreamReader`, `BatchReader` |
| **850 - 800** | **Exact-Match:** Strict, no-argument signatures. | `Predicate`, `Pure`, `Lookup` |
| **750 - 500** | **Directive-Driven / Multi-Arg:** Shapes where arity or a `//testkit:` directive disambiguates intent. | `Deleter`, `MultiArgWriter` |
| **450 - 200** | **Generic Catch-alls:** The standard single-key/single-value read/write patterns. | `Reader`, `Writer`, `Mutator` |

---

## The Shape Taxonomy

Every shape expects its leading `context.Context` parameter to be optional. When `ctx?` is listed below, it means the shape matches both `func(ctx context.Context, ...)` and `func(...)`.

### Reading Band
Idempotent operations that retrieve data without mutating system state.

| Shape | Signature Pattern | Description | Suite Baseline Law | Example |
|-------|-------------------|-------------|--------------------|---------|
| **Reader** | `func(ctx?, K) (V, error)` | Standard single-key fetch. | **Consistent Reads:** Repeated calls return identical values. | `Get(ctx, string) (Record, error)` |
| **ReaderNoError** | `func(ctx?, K) V` | Infallible fetch. | **Consistent Reads:** Repeated calls return identical values. | `Lookup(ctx, string) Record` |
| **ReaderWithBool** | `func(ctx?, K) (V, bool)` | Map-style fetch. | **Missing Returns False:** The boolean accurately reflects presence. | `Load(ctx, string) (Record, bool)` |
| **Lookup** | `func(ctx?, K) (V, R, bool)` | Fetch returning a value + metadata. | **Metadata Consistency:** `V` and `R` are stable across reads. | `Inspect(ctx, string) (Record, string, bool)` |
| **PointerReader** | `func(ctx?, K) *V` | Fetch returning a pointer. | **Nil On Missing:** Returns `nil` instead of an error when not found. | `Find(ctx, string) *Record` |
| **MultiReader** | `func(ctx?, K) (V1, V2, error)` | Fetch returning multiple discrete values. | **Atomic Read:** Both `V1` and `V2` are read at the same snapshot. | `Fetch(ctx, string) (Record, string, error)` |
| **BatchReader** | `func(ctx?, ...K) ([]V, error)` | Fetch for a slice/variadic list of keys. | **Partial Failure:** Batch queries don't fail entirely if one key is missing. | `Many(ctx, ...string) ([]Record, error)` |

### Writing Band
Operations that mutate system state.

| Shape | Signature Pattern | Description | Suite Baseline Law | Example |
|-------|-------------------|-------------|--------------------|---------|
| **Writer** | `func(ctx?, V) error` <br> `func(ctx?, V) (R, error)` | Single-value insert/update. Can optionally return a result `R`. | **Write Success:** Ensures the system accepts valid samples. | `Put(ctx, Record) error` |
| **CompositeWriter** | `func(ctx?, K, V) error` | Insert/update parameterized by an explicit key. | **Keyed Atomicity:** The write is bound to the given key. | `Set(ctx, string, Record) error` |
| **Mutator** | `func(ctx?, V)` | Infallible, fire-and-forget state mutation. | **Non-Observable Error:** Never panics on valid samples. | `Touch(ctx, string)` |
| **Deleter** | `func(ctx?, K) error` | Removes state. | **Delete Removes Value:** A subsequent `Reader` must fail/return false. | `Remove(ctx, string) error` |
| **MultiArgWriter** | `func(ctx, P1, P2, ...) error` | Complex mutation with multiple distinct parameters. | **Parameter Validation:** Respects context and rejects zero-values. | `Schedule(ctx, string, Record, int) error` |

> **Note on Deleter:** Because `func(ctx, string) error` is ambiguous (is it `Writer(V)` or `Deleter(K)`?), `Deleter` detection requires the `//testkit:deleter` directive on the method.
> **Note on Mutator:** Automatically detected from any `func(ctx?, V)` signature that has no returns. Use `//testkit:not-mutator` to opt out.

### Streaming Band
High-throughput or unbounded data sequences.

| Shape | Signature Pattern | Description | Suite Baseline Law | Example |
|-------|-------------------|-------------|--------------------|---------|
| **StreamReader** | Returns `iter.Seq[V]` or `iter.Seq2[K, V]` | Go 1.23+ iterator. | **Reentrancy Safety:** The iterator can be broken early without leaking. | `All(ctx) iter.Seq[Record]` |
| **StreamConsumer** | `func(ctx, io.Reader) (V, error)` | Consumes a stream interface. | **Backpressure Respect:** Stops consuming if context is canceled. | `ReadFrom(ctx, io.Reader) (int, error)` |

### Aggregation & Lifecycle
System-level state, counting, or management.

| Shape | Signature Pattern | Description | Suite Baseline Law | Example |
|-------|-------------------|-------------|--------------------|---------|
| **Aggregator** | `func(ctx?) (T, error)` | Computes a global metric (e.g., `Count()`, `Sum()`). | **Count Consistency:** Aggregation is stable across immediate re-reads. | `Count(ctx) (int, error)` |
| **MultiAggregator** | `func(ctx?) (T1, T2, error)` | Computes multiple global metrics. | **Atomic Aggregation:** Metrics are computed from the same snapshot. | `Stats(ctx) (int, int, error)` |
| **Lifecycle** | `func(ctx) error` | State machine progression (e.g., `Start`, `Stop`). | **Order Awareness:** Idempotent if called twice; respects teardown. | `Init(ctx) error` |
| **VoidLifecycle** | `func()` <br> `func(ctx)` | Infallible state machine progression. | **Resource Cleanup:** Does not panic on repeated teardowns. | `Reset()` |
| **PoisonAccessor** | `func() error` | Checks health/error state (e.g., `Err() error`). | **Sticky Error:** Once an error is returned, it never clears. | `Err() error` |

> **Note on Lifecycle:** A signature like `func() error` could be an `Aggregator`, `Lifecycle`, or `PoisonAccessor`. To disambiguate, `Lifecycle` requires a `context.Context` parameter, while `PoisonAccessor` strictly forbids it.

### Stateless Band
Utility methods that do not rely on or modify the system's internal state.

| Shape | Signature Pattern | Description | Suite Baseline Law | Example |
|-------|-------------------|-------------|--------------------|---------|
| **Pure** | `func() T` | Deterministic computation without inputs. | **Determinism:** Always returns the exact same value. | `Description() string` |
| **Predicate** | `func() bool` | Binary state check (e.g., `IsReady()`). | **Constant Return:** Value does not flap between reads. | `IsHealthy() bool` |

---

## The "Universal Laws"

In addition to shape-specific laws, the `suite` generator automatically enforces **Universal Laws** on every method, based solely on the presence of `context.Context` and `error`:

1.  **Context Cancellation:** If `ctx` is present, it is canceled immediately; the method MUST return a context error.
2.  **Context Deadline Exceeded:** If `ctx` is present, a pre-expired context is passed; the method MUST return a deadline error.
3.  **Nil Context Safety:** If `ctx` is present, `nil` is passed; the method MUST NOT panic.
4.  **Smoke Test:** Automatically generates zero-value or sampled inputs to ensure the method does not panic on basic invocation.

## Overriding Detection

The priority registry is robust, but sometimes Go's type system isn't expressive enough to capture your intent. You can use **Shape Hints** (directives) to force or prevent specific classification:

*   `//testkit:deleter` — Elevates a `func(ctx?, string) error` signature. Without this, the registry detects it as a generic `Writer(V)`; with it, it becomes a `Deleter(K)`, enabling delete-removes validation.
*   `//testkit:mutator` — Explicit marker for state-changing methods (though `func(ctx?, V)` is auto-detected as a `Mutator` even without it).
*   `//testkit:not-mutator` — Prevents `Mutator` auto-detection for a void-return method (e.g., `Log(v)`). It falls back to `Unknown`.
*   `//testkit:directive mutator=off` — The bundle-syntax equivalent of `not-mutator`.
*   `//testkit:keyfield FieldName` — While not changing the shape, this hint tells the registry which struct field to extract when synthesizing a `K` from a `V`.

## See also

- [Generators / suite](suite.md) — How the Tier 1 test suite utilizes shapes.
- [Generators / bench](bench.md) — How benchmarks use shapes to determine hot-paths.
- [Generators / model](model.md) — How the state-machine uses shapes to infer transitions.
