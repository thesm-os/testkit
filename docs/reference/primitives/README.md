# testkit Primitives

Reusable test utilities provided by the `testkit` module. Every primitive is generic, domain-agnostic, and safe for concurrent use in parallel tests.

testkit has zero knowledge of any consuming project's types. Domain-specific test infrastructure (stubs, fixtures, simulators) lives in each project's `*test` packages — testkit provides the building blocks those packages compose.

## Dependencies

The core `testkit` package depends on:

- `github.com/google/go-cmp/cmp` — structural diff for deep equality assertions
- `pgregory.net/rapid` — property-based testing generators

Optional sub-packages have additional dependencies — see [sub-packages.md](../sub-packages.md).

## Catalog

| Document | Surface |
|----------|---------|
| [Assertions](assertions.md) | Positional + fluent helpers with go-cmp diffs |
| [Directive assertions](directive-assertions.md) | 21 directive-driven contract assertions, cross-method invariants, HookRecorder, suite options |
| [MethodStub](method-stub.md) | Generic per-method test double — recording, faults, clock, strict, verify |
| [Recording](recording.md) | `Recorder[T]` with filtering, waiting, hooks, gating, timestamping, bench mode |
| [Fault injection](fault-injection.md) | `Fault` interface + 5 strategies (counted, retry, probability, windowed, predicate) + `And`/`Or` composition |
| [Clock](clock.md) | `Clock` interface + `RealClock` + `TestClock` for deterministic time |
| [OrderTracker](order-tracker.md) | Cross-method call-order constraints (driven by `//testkit:order-after`) |
| [RandSource](rand.md) | Pluggable RNG for probabilistic faults; defaults to `math/rand/v2`, `FixedRandSource` for tests |
| [Concurrency](concurrency.md) | `ConcurrentStress`, `GoroutineLeak`, `Timeout`, goroutine capture utilities |
| [Benchmarking](benchmarking.md) | `Contract` for allocation and latency gates |
| [Golden files](golden-files.md) | `AssertGolden`, `AssertGoldenAt`, `AssertGoldenJSONField`, `Compare` + scrubbers |
| [Polling](polling.md) | `RetryUntil`, `AssertEventually` |
| [Helpers](helpers.md) | `TestError`, `RequireEnv`, `SeededRand`, `MustMarshal`, `Quiet`, `FailableTB`, `TempFile`, `FreePort`, `SortedKeys`, `TableTest`, `MapDiff`, rapid generators |

## Sub-packages

| Package | Description |
|---------|-------------|
| [`testkit/container`](../sub-packages.md#testkitcontainer) *(not implemented)* | `SharedContainer` via `testcontainers-go` |
| [`testkit/httptest`](../sub-packages.md#testkithttptest) | HTTP response assertions |
| [`testkit/oteltest`](../sub-packages.md#testkitoteltest) | OpenTelemetry metric assertions |
| [`testkit/clitest`](../sub-packages.md#testkitclitest) | CLI binary testing |

## Out of scope

The following live in each project's own `*test` packages because they depend on project-specific types:

- **Pagination cursors** — depend on project-specific page types.
- **Codec harnesses** — depend on project-specific codec interfaces. testkit's `codec` generator produces them from proto descriptors.
- **State fixtures** — depend on project-specific state key types. testkit's `builder` generator produces them.
- **Simulation engines** — projects define their own engine. testkit provides `Recorder` (with hooks, gating, waiting) as the observation layer and the `sim` generator as the harness — but the engine itself is consumer-owned.
