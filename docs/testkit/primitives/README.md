# testkit Primitives

Reusable test utilities provided by the `testkit` module.
Every primitive is generic, domain-agnostic, and safe for
concurrent use in parallel tests.

testkit has zero knowledge of any consuming project's
types. Domain-specific test infrastructure (stubs,
fixtures, simulators) belongs in each project's `*test`
packages — testkit provides the building blocks those
packages compose.

## Dependencies

The core `testkit` package depends on:

- `github.com/google/go-cmp/cmp` — structural diff output
  for deep equality assertions
- `pgregory.net/rapid` — property-based testing generators

Optional sub-packages have additional dependencies:

| Sub-package | Dependency |
|-------------|-----------|
| `testkit/container` | `github.com/testcontainers/testcontainers-go` |
| `testkit/httptest` | stdlib `net/http` only |
| `testkit/oteltest` | `go.opentelemetry.io/otel/sdk` |
| `testkit/clitest` | stdlib `os/exec` only |

## Primitives

| Document | Description |
|----------|-------------|
| [Assertions](assertions.md) | Positional + fluent assertion helpers with go-cmp diffs |
| [Recording](recording.md) | Recorder[T] with filtering, waiting, hooks, and gating |
| [Fault injection](fault-injection.md) | FaultInjector for deterministic Nth-call failures |
| [Concurrency](concurrency.md) | ConcurrentStress, GoroutineLeak |
| [Benchmarking](benchmarking.md) | Contract for allocation and latency gates |
| [Golden files](golden-files.md) | AssertGolden + Scrubbers |
| [Context](context.md) | Timeout (Go 1.24+ `t.Context()` for the rest) |
| [Polling](polling.md) | RetryUntil, AssertEventually |
| [Helpers](helpers.md) | TestError, RequireEnv, SeededRand, MustMarshal, Quiet, FailableTB, TempFile, FreePort, SortedKeys, TableTest, DiffMap, Rapid generators |

## Sub-packages

| Package | Description |
|---------|-------------|
| [testkit/container](../sub-packages.md#testkitcontainer) | SharedContainer via testcontainers-go |
| [testkit/httptest](../sub-packages.md#testkithttptest) | HTTP response assertions |
| [testkit/oteltest](../sub-packages.md#testkitoteltest) | OpenTelemetry metric assertions |
| [testkit/clitest](../sub-packages.md#testkitclitest) | CLI binary testing |

## What testkit does NOT provide

The following live in each project's own `*test` packages
because they depend on project-specific types:

- **Virtual clocks** — projects define their own clock
  abstraction; the test clock lives in a `*test` package.
- **Pagination cursors** — cursor helpers depend on
  project-specific page types.
- **Codec harnesses** — codec test specs depend on
  project-specific codec interfaces.
- **State fixtures** — builders depend on project-specific
  state key types.
- **Chaos marshalers** — implement project-specific codec
  interfaces structurally. The stub generator produces
  equivalent fault injection for any interface.
- **Simulation engines** — projects define their own sim
  engine. testkit provides `Recorder` (with hooks, gating,
  waiting) as the observation layer and generates
  simulation workloads — but the engine is always
  domain-specific.
