# Directive Assertions

testkit ships assertion helpers in `suite/directives.go` that match the semantics of `//testkit:` directives. The `suite` generator emits one call per directive; consumers can also call them directly when wiring suite logic by hand.

Each assertion returns `func(*testing.T, func() T)` — a closure that the generated driver invokes with the test and factory. The closures passed in are minimal translators that adapt the impl's specific signature to the runtime's expected shape. No control flow lives in generated code.

## Context-safety assertions

These run by default on every ctx-taking method without requiring a directive.

| Helper | Directive | Contract |
|--------|-----------|----------|
| `AssertNilSafe` | `nilsafe` | No panic on zero-value / nil inputs |
| `AssertCtxCancellation` | (auto) | Cancelled ctx returns `context.Canceled` |
| `AssertCtxDeadline` | (auto) | Past-deadline ctx returns deadline error |
| `AssertNilCtx` | (auto) | Nil ctx does not panic |

## Directive-driven assertions

Each corresponds to a `//testkit:` directive. The generator emits one call per directive occurrence.

### Behavioral directives

| Helper | Directive | Contract |
|--------|-----------|----------|
| `AssertDeprecatedSmoke` | `deprecated` | Deprecated method doesn't panic; logs replacement |
| `AssertRetrySucceedsOnAttempt` | `retry-succeeds-on-attempt` | First N-1 calls error, Nth succeeds |
| `AssertOrderAfter` | `order-after` | Carrier errors before prerequisite, succeeds after |
| `AssertPartitionIsolation` | `partition` | Two sequential partition writes succeed |
| `AssertWrappedVia` | `wrapped-via` | Error satisfies `errors.Is` for both wrap target and sentinel |
| `AssertIdempotentSecondCall` | `idempotent` | Second call doesn't panic |
| `AssertNilSafeNoPanic` | `nilsafe` | No panic on zero inputs (void-return variant) |

### Property directives

| Helper | Directive | Contract |
|--------|-----------|----------|
| `AssertPureImplIndependent` | `pure` | Two independent impls return equal results |
| `AssertCacheableRepeatedReads` | `cacheable` | Three reads return pairwise-equal results |
| `AssertMonotonicNonDecreasing` | `monotonic` | N samples are non-decreasing (`cmp.Ordered` result) |
| `AssertBoundedRange` | `bounded` | Result is in `[min, max]` inclusive |
| `AssertAtomicNoTrace` | `atomic` | Failed mutation leaves state equal to pre-call |

### Concurrency directives

| Helper | Directive | Contract |
|--------|-----------|----------|
| `AssertConcurrentStrict` | `concurrent` | 16 workers x 25 iters under race detector |
| `AssertConcurrentReadersParallel` | `concurrent-readers` | 32 reader goroutines under race detector |

### Timing directives

| Helper | Directive | Contract |
|--------|-----------|----------|
| `AssertTimeoutWithin` | `timeout` | Call completes within the specified duration |
| `AssertEventuallyConverges` | `eventually` | Polling converges (two consecutive equal results) within deadline |

### Observation directives

| Helper | Directive | Contract |
|--------|-----------|----------|
| `AssertSideEffectObservable` | `side-effect` | Observable state differs before vs after mutation |
| `AssertValidatesZeroInput` | `validates` | Zero/invalid input returns non-nil error |
| `AssertHooksFire` | `hooks` | Named hooks fire during method invocation (via `HookRecorder`) |

### Resource directives

| Helper | Directive | Contract |
|--------|-----------|----------|
| `AssertScopeAuthRequired` | `scope` | Unauthorized call returns sentinel; authorized call succeeds |
| `AssertLeaseAcquireRelease` | `lease` | Acquire/release/acquire works; double-acquire-without-release fails |

## Cross-method assertions

These verify invariants spanning two methods on the same interface.

| Helper | Directive | Contract |
|--------|-----------|----------|
| `AssertReadAfterWrite` | `read-after-write` | Write then read returns the written value |
| `AssertReadAfterWriteByKey` | `read-after-write` | Key-parameterized variant |
| `AssertDeleteRemovesValue` | `delete-removes` | Delete then read returns sentinel |
| `AssertDeleteRemovesByKey` | `delete-removes` | Key-parameterized variant |
| `AssertStreamReflectsMutations` | `stream-reflects-mutations` | Stream yields values written via paired writer |
| `AssertStreamReflectsValueWritten` | `stream-reflects-mutations` | Value-parameterized variant |
| `AssertLifecycleAfterClose` | `lifecycle-after-close` | Reader errors or returns zero after lifecycle method |
| `AssertLifecycleAfterCloseReflective` | `lifecycle-after-close` | Reflective variant using method name |
| `AssertCRDTMerge` | `crdt-merge` | Merge is commutative: merge(a,b) == merge(b,a) |

## HookRecorder

Used by `AssertHooksFire`. Production hook-firing code extracts the recorder from context and calls `Record(name)`.

```go
recorder := suite.NewHookRecorder()
ctx := suite.ContextWithRecorder(t.Context(), recorder)
subject.DoWork(ctx)
assert(recorder.Count("BeforeWrite") > 0)
```

| Method | Purpose |
|--------|---------|
| `NewHookRecorder()` | Create empty recorder |
| `Record(name)` | Increment fire count (goroutine-safe, nil-safe) |
| `Count(name)` | Query fire count |
| `ContextWithRecorder(ctx, r)` | Attach recorder to context |
| `RecorderFromContext(ctx)` | Extract recorder (nil when absent) |

## Suite options

Consumer-supplied configuration passed to the generated driver via `...suite.Option`.

| Option | Purpose |
|--------|---------|
| `WithInvalidFactory(factory)` | Factory producing an impl that must fail RejectInvalid assertions |
| `WithPoisonedFactory(factory)` | Factory producing a poisoned impl for PoisonAccessor tests |
| `WithPrePopulate(seed)` | Seed callback applied to every fresh impl before subtests |
| `WithObservableVia(method)` | Reader method name for paired-method observation |
| `WithAggregatorBounds(lower, upper)` | Bounds for Aggregator bounded assertions |
| `WithAggregatorBoundsAt(i, lower, upper)` | Per-slot bounds for MultiAggregator |
| `WithStreamSample(factory)` | Factory producing a fresh `io.Reader` for StreamConsumer tests |
| `WithScopeContext(fn)` | Context factory for scope authorization tests |
| `WithScopeUnauthorized(err)` | Sentinel error for unauthorized scope access |
| `WithLeaseRelease(method)` | Release method name for lease lifecycle tests |
| `WithStateEqual(eq)` | Custom equality function for atomic assertions |

## See also

- [Configuration](../configuration.md) — directive vocabulary reference
- [Generators / suite](../generators/suite.md) — how `suite` consumes directives to call these helpers
