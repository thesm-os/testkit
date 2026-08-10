# Model

> **Status: planned.** This generator is not implemented. No directive is registered for it, and `testkit run` does not produce the output described below. This page records the intended design so adoption can be planned against a stable target — the directive, the output paths and the generated surface may all differ once it ships.

Tier 2-3 conformance. The `model` generator elevates your testing from "single-call assertions" (Tier 1) to **Property-Based State Machines**.

It reads a Go interface, detects the mathematical *shape* of every method, and emits a rigorous verification harness that attacks your implementation with randomized, concurrent, and adversarial sequences of operations, shrinking failures down to the minimal reproducing trace.

## The Technologies Explained

To understand the `model` generator, you must understand the three formal verification techniques it orchestrates for you:

1. **Property-Based Testing (PBT):** Instead of writing explicit "Arrange-Act-Assert" tests (e.g., `Put("A", 1)`, then `Get("A") == 1`), PBT declares generic "Laws" (e.g., `ReadAfterWrite`). The engine ([`pgregory.net/rapid`](https://pgregory.net/rapid)) generates thousands of random operation sequences. If a sequence breaks a Law, the engine "shrinks" the trace, intelligently removing operations until it finds the exact 3-step sequence that triggered the bug.
2. **Bounded Model Checking (BMC):** A deterministic alternative to PBT's randomized exploration. The harness performs a Depth-First Search (DFS) over the entire state space of your interface, restricted by a configurable boundary (e.g., max 5 steps deep). It exhaustively proves that *no* path within those bounds violates your invariants.
3. **Linearizability Checking:** In concurrent systems, operations overlap. How do you assert correctness when 10 goroutines are calling `Put` and `Get` simultaneously? The harness uses [`anishathalye/porcupine`](https://github.com/anishathalye/porcupine) to record the start and end times of every operation. It then mathematically proves whether that concurrent history could have been serialized into a strict, step-by-step sequence that obeys your sequential Laws.

## Directive

```go
//go:generate testkit model -o storetest/store_model.gen.go Store
```

`model` accepts exactly one type argument and emits a single output file.

## Default output

`<package>test/<subject>_model.gen.go`.

## The Three-Tier Shape Taxonomy

While the `suite` generator relies strictly on method signatures (Tier 1), the `model` generator analyzes the broader topology of your interface to infer multi-method state machines.

### Tier 1: Signature-tier (21 shapes)

The baseline detection. Evaluates a single method isolated from its peers.

* **Reading Band:** `Reader`, `BatchReader`, `Lookup`
* **Writing Band:** `Writer`, `Mutator`, `Deleter`
* **Streaming Band:** `StreamReader`, `StreamConsumer`
* **Lifecycle Band:** `Aggregator`, `Lifecycle`, `Pure`
*(See [Shapes](shapes.md) for the full taxonomy).*

### Tier 2: Contract-tier (12 shapes)

Detected by looking at sibling methods, struct fields, and parameter types across the interface.

* `CompareAndSwap` (takes Version, sibling Reader returns Version).
* `GetOrCompute` (takes a `func() V`).
* `AcquireLease` (sibling `Release` exists).
* `Watcher` / `Publisher` / `Subscriber`.
* *(And 6 others: `TransactionFunc`, `Paginator`, `Persister`, `Updater`, `Upserter`, `Appender`)*

### Tier 3: Composite-tier (4 shapes)

Multi-method shapes that form closed-loop state machines.

* `Pool` (Balanced `Get`/`Put` or `Acquire`/`Release`).
* `Cursor` (Interface has both `Next()` and `Close()`).
* `TwoPhase` (Returns a Tx with `Commit()` and `Rollback()`).
* `Saga` (Chained methods passing results forward; requires `//testkit:saga` directive).

## The Orthogonal Axis: Mixins

Mixins apply orthogonal behavioral properties to any shape via `//testkit:` directives. The `model` generator ships with **31 mixins**, which automatically emit corresponding verification laws.

**Examples:**

* `//testkit:idempotent`: Emits a law proving `F(x); F(x) == F(x)`.
* `//testkit:commutative`: Emits a law proving `A;B == B;A` (for Mutators).
* `//testkit:associative`: Emits a law proving `(A;B);C == A;(B;C)`.
* `//testkit:tamper-evident`: Adds post-write corruption detection.
* `//testkit:leak-free`: Wraps the execution to detect goroutine/FD leaks across cycles.
* `//testkit:point-in-time`: Enforces snapshot isolation semantics for readers.

*(Applying a mixin to an incompatible shape—e.g., `//testkit:commutative` on a `Lifecycle` method—results in a hard codegen error, never a silent fallback).*

## What is Generated

For an interface `Store`, the generator emits four entry points into `<pkg>test`:

```go
// Run as a standard PBT test (default 100 iterations).
func StoreModelTest(t *testing.T, factory func() store.Store, opts ...StoreModelOption)

// Run as a PBT test with custom rapid.T configuration.
func StoreModelAssert(t *testing.T, factory func() store.Store, opts ...StoreModelOption)

// Run as a native Go fuzz target for corpus-driven exploration.
func StoreModelFuzz(f *testing.F, factory func() store.Store, opts ...StoreModelOption)

// Exported property function for advanced harness composition.
func StoreModelProperty(factory func() store.Store, opts ...StoreModelOption) func(*model.T)
```

## State Machine Execution

During execution, `rapid` drives the `StoreModelTest` harness:

1. It instantiates a fresh System Under Test (SUT) via your `factory`.
2. It selects a random `Action` (derived from your interface's methods).
3. It generates fuzzed arguments for that Action.
4. It applies the Action to the SUT.
5. It evaluates all **Auto-Derived Laws** against the new state.
6. It loops this sequence (default 100 times per test run).

If any law fails, `rapid` halts and begins **shrinking**, manipulating the sequence of actions and their arguments to present you with the absolute smallest reproducible failure trace.

### The Law Engine: Auto-Derived Invariants

A "Law" in `testkit` is a formal invariant evaluated after an action executes. The generator auto-derives over 80 distinct laws by cross-referencing your interface's shapes and mixins. Laws are categorized by their statefulness:

#### Stateless Laws (Pre/Post Call)

These laws evaluate the result of a single action against the current state.

* **`AUTO-READ-AFTER-WRITE`:** (Requires `Reader` + `Writer` + `KeyField`). After a successful write, the harness automatically invokes the Reader with the same key and asserts the returned value exactly matches the written value.
* **`AUTO-DELETE-RETURNS-NOT-FOUND`:** (Requires `Reader` + `Deleter` + `//testkit:errors ErrNotFound`). After a successful delete, the harness invokes the Reader and asserts `errors.Is(err, ErrNotFound)`.
* **`AUTO-PURE-DETERMINISM`:** (Requires `Pure` shape). The harness caches the result of the first invocation. All subsequent invocations during the property run must return the exact same value.
* **`AUTO-STREAM-REENTRANCY`:** (Requires `StreamReader`). The harness consumes the iterator twice and asserts both passes yielded identical sequences.

#### Stateful Laws (Temporal & Trace-Aware)

These laws analyze the *entire history* of actions executed so far.

* **`AUTO-APPENDER-MONOTONIC`:** (Requires `Appender`). Scans the trace to ensure every returned offset/index is strictly greater than the previous one.
* **`AUTO-POOL-BALANCED`:** (Requires `Pool` composite shape). Analyzes the trace to ensure every `Acquire` is matched by a `Release` before the run terminates, and that no `Release` is called twice for the same resource.

### Trace Combinators (Temporal Logic)

You are not limited to auto-derived laws. You can compose complex, domain-specific temporal logic by supplying `model.Law[T]` implementations via `StoreModelExtraLaws`. The `model` package provides Trace Combinators that evaluate predicates over the historical action sequence:

* **`model.AfterEvery(trigger, check)`:** A strict causality constraint. Example: After every `BeginTx` action, the *very next* action involving that Tx ID must be `Commit` or `Rollback`.
* **`model.EventuallyAfter(trigger, check, budget)`:** A liveness constraint for AP systems. Example: After a `Publish` action, the message must be observable via `Read` within `budget` subsequent actions (simulating asynchronous replication delay).
* **`model.Never(check)`:** A safety constraint. Example: The state must never reflect a balance below zero.

### Mutation Testing (Runtime Differential)

How do you know if your Laws are strict enough? The `model` generator includes a built-in Mutation Testing engine.

When enabled, the runner instantiates a parallel "Shadow" SUT. It intercepts operations bound for the shadow SUT and injects adversarial mutations:

* `DropWrites`: Silently ignores a `Put` operation.
* `ReturnWrongValue`: Alters a `Get` result.
* `RandomDelay`: Sleeps unpredictably to expose race conditions.
* `OffByOneIndex`: Corrupts paginator cursors.

The runner then verifies that your Law suite **catches** the mutant (i.e., at least one Law fails). If the mutant survives the entire sequence, your Laws are too weak.

## The Reference Pattern (Differential Rollout)

The most powerful way to test a complex system is to compare it to a simple one. The `model` harness supports **Differential Correctness**.

You can supply a "Reference Implementation" (typically a `testkit stub` wrapping a naive in-memory map):

```go
storetest.StoreModelTest(t, factory,
    storetest.StoreModelReference(func() store.Store {
        return storetest.NewStoreStub(nil, storetest.StoreStubDelegateTo(store.NewInMemoryStore()))
    }),
)
```

When a Reference is provided:

1. Every action is applied to *both* the SUT and the Reference simultaneously.
2. The harness asserts that the SUT and the Reference produce the **exact same return values and errors** for every single operation.

This is the ultimate confidence builder for a database migration or refactor: you prove that the highly optimized V2 implementation behaves identically to the naive V1 implementation under chaotic, randomized load.

## Concurrent Linearizability

By default, actions are applied sequentially. When you enable concurrency:

```go
storetest.StoreModelTest(t, factory,
    storetest.StoreModelConcurrent(
        model.Workers(4),
        model.OpsPerWorker(50),
    ),
)
```

The runner spawns multiple goroutines firing randomized actions at a shared SUT. For CRUD shapes, it feeds the resulting execution history into **Porcupine**, proving whether your supposedly thread-safe code actually honors linearizability.

## Layout Conventions

A typical `model` configuration generates into a `<pkg>test/` sub-package.

**What goes where:**

| File | Owner | Contents |
|------|-------|----------|
| `*_model.gen.go` | Generator | The generated state-machine harness and actions (DO NOT EDIT). |
| `model_test.go` | Developer | Hand-written test functions wiring `StoreModelTest`, `StoreModelFuzz`, and `StoreModelOption` injections. |
| `companion.go` | Developer | (Optional) The naive in-memory reference implementation used for Differential Correctness. |

Keep `model_test.go` separated from `spec_test.go` (the Tier 1 suite). Model tests often run longer and require different CI configurations (e.g., `-count=10` or prolonged Fuzzing).

## See also

* [Shape Classification](shapes.md) — The signature-tier foundation.
* [Generators / suite](suite.md) — Tier 1 single-call conformance.
* [Generators / stub](stub.md) — How to generate a stub for use as a Reference implementation.
