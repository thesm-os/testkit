# Primitives

The runtime half of testkit. Generated code calls into these packages, and a hand-written test can call them directly — nothing here requires a generator.

Every function takes `testing.TB` first and a message last. The message is not decoration: it is what a reader sees when the assertion fires, and the assertion has no other way to say what it was checking.

## Packages

### What a test calls

| Import | Provides | Page |
|---|---|---|
| `go.thesmos.sh/testkit` | Assertions, the fluent chain, contract helpers, benchmark budgets, fixtures | [assertions](assertions.md), [directive assertions](directive-assertions.md), [context](context.md), [benchmarking](benchmarking.md), [helpers](helpers.md) |
| `go.thesmos.sh/testkit/stub` | The dispatch engine, recorder, gates and order tracker generated doubles embed | [method stub](method-stub.md), [recording](recording.md), [order tracker](order-tracker.md), [behaviour suites](behaviour-suites.md) |
| `go.thesmos.sh/testkit/clock` | Deterministic time | [clock](clock.md) |
| `go.thesmos.sh/testkit/fault` | Composable fault-injection strategies | [fault injection](fault-injection.md) |
| `go.thesmos.sh/testkit/rand` | Deterministic random sources | [rand](rand.md) |
| `go.thesmos.sh/testkit/polling` | Wait-for-condition helpers | [polling](polling.md) |
| `go.thesmos.sh/testkit/concurrency` | Stress runs, leak detection, timeouts | [concurrency](concurrency.md) |
| `go.thesmos.sh/testkit/golden` | Golden-file comparison and scrubbing | [golden files](golden-files.md) |

### What a run produces

`core/*` is the substrate behind observation and reporting. Generated code and the engine call into it; a hand-written test rarely does.

| Import | Provides | Page |
|---|---|---|
| `go.thesmos.sh/testkit/core/factory` | Named implementations and the `TESTKIT_SEED` contract | [factory](factory.md) |
| `go.thesmos.sh/testkit/core/trace` | Method-call observations, queryable and causal | [trace](trace.md) |
| `go.thesmos.sh/testkit/core/failure` | The classified failure envelope and its artifacts | [failure](failure.md) |
| `go.thesmos.sh/testkit/core/coverage` | Law, requirement and state-space coverage | [coverage](coverage.md) |
| `go.thesmos.sh/testkit/core/equivalence` | Pluggable equivalence relations for differential comparison | [equivalence](equivalence.md) |
| `go.thesmos.sh/testkit/core/visualize` | Self-contained HTML timelines | [visualize](visualize.md) |
| `go.thesmos.sh/testkit/core/brand` | The project identity every artifact keys off | [brand](brand.md) |

**`core/equivalence`, `core/visualize` and `core/factory` ship ahead of their consumers.** Their package docs name `model`, `sim`, `chaos`, `differential-rollout` and `replay`, none of which is implemented — see the [generator index](../generators/README.md). All three work standalone today; `factory.SeedFromEnv` in particular defines environment variables worth knowing about now.

Each top-level sub-package is a separate module path but ships from this repository ([ADR-0005](../../adr/0005-split-into-published-modules.md)). A package earns its place at the top level by being imported on its own ([ADR-0007](../../adr/0007-earn-top-level-packages-by-import.md)) — which is why `clock` and `rand` are not folded into `stub` despite being consumed mostly by it.

## Fatal, not error

Every assertion calls `tb.Fatalf`. None of them returns a bool for the caller to branch on.

That is deliberate. An assertion that returns leaves the test running against state it has already declared wrong, and the second failure is usually a consequence of the first — so the output names two problems where there is one. Where a test genuinely needs to continue, `Assert` chains, and the chain stops at the first failure for the same reason.

## Testing the assertions themselves

An assertion that never fails is not an assertion, and `tb.Fatalf` makes the failing path hard to reach from a test. [`FailableTB`](helpers.md#failabletb) is the answer: a `testing.TB` that records the first fatal message instead of aborting, so a test can drive an assertion into failure and check what it said.

```go
ft := testkit.NewFailableTB()
testkit.Equal(ft, 1, 2, "counts must match")

testkit.True(t, ft.Failed(), "the assertion must have fired")
testkit.Contains(t, ft.Msg(), "counts must match", "the message must reach the reader")
```
