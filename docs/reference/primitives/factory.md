# Factory

```go
import "go.thesmos.sh/testkit/core/factory"
```

Two primitives generated tests share: a tagged constructor so a multi-implementation run names which implementation failed, and the seed-resolution contract every property-based run uses.

`SeedFromEnv` defines environment variables a **user** sets. If you read one page in `core/`, read [that section](#seedfromenv).

## Named

```go
func NewNamed[T any](name string, fn func() T) Named[T]

func (n Named[T]) Name() string
func (n Named[T]) Construct() T
```

A constructor closure with a stable identifier attached.

```go
storetest.AssertStoreModelAcrossImpls(t,
    []storetest.StoreModelOption{...},
    factory.NewNamed("in-memory", newInMem),
    factory.NewNamed("redis",     newRedis),
)
```

The name flows into trace events, classified-failure JSON and the coverage-summary header, so a CI artifact says which implementation produced an observation without anyone recovering the call site. A positional index would say "impl 2", which is useless three weeks later.

`Construct` calls the closure once per invocation. The closure must return an isolated value — the runner's per-iteration isolation guarantee depends on it, and a closure returning a shared instance turns a property run into a test of accumulated state.

`NewNamed` **panics** on an empty name or a nil closure. Both are usage errors no runner can recover from, and the panic fires at test setup rather than partway through a property run, so the diagnostic lands at the call site.

## SeedFromEnv

```go
func SeedFromEnv(tb testing.TB, pkgPrefix, generator string) int64
```

Resolves the seed for a deterministic run. Three sources, in order:

1. **The generator-specific variable** — `<PKGPREFIX>_<GENERATOR>_SEED`, e.g. `STORETEST_MODEL_SEED`, `LEDGERTEST_SIM_SEED`, `LEDGERTEST_CHAOS_SEED`.
2. **`TESTKIT_SEED`** — the global fallback, for pinning every run in a process at once.
3. **A wall-clock-derived seed**, logged via `tb.Logf` so a failing run can be replayed.

```
TESTKIT_SEED=0x5eed go test ./...
STORETEST_MODEL_SEED=42 go test ./storetest/
```

Both decimal and `0x`-prefixed hex are accepted.

**An invalid value fails via `tb.Fatalf` rather than falling back.** A typo'd seed that silently became a random one would make a "reproduction" that reproduces nothing, and the person running it would have no way to tell.

The third source is why the seed is always logged. A run that found something is only useful if the seed that found it is recoverable from the output.

| Constant | Value |
|---|---|
| `EnvSeedSuffix` | `_SEED` |
| `EnvFallbackSeed` | the global variable name |
| `EnvVarName(pkgPrefix, generator)` | composes the generator-specific name |

## Status

`Named` is consumed by the `model`, `sim` and `differential-rollout` generators, none of which is implemented — see the [generator index](../generators/README.md). The package ships ahead of them and both functions work standalone today, which is why `SeedFromEnv` is worth knowing about now: any hand-written property test can adopt the same seed contract and get the same replay story.

## See also

- [Helpers](helpers.md) — `testkit.SeededRand` for per-test deterministic data, which is a different job: it derives from the test name rather than from configuration.
- [Trace](trace.md) — where the `Named` identifier surfaces in recorded events.
