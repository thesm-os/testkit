# Bench

> **Status: designed, not implemented.**
> [RFC-0003](../../rfc/0003-the-projection-consumers.md) fixes this
> generator's design. No directive is registered yet and `testkit run` does
> not produce the outputs below; where this page and the RFC differ, the RFC
> is the authority.

The `bench` generator turns a `//testkit:bench` method into a measured loop
holding the budgets the directive declares, backed by the shipped
[`testkit.Contract`](../primitives/benchmarking.md) runtime — `StartContract`,
chained ceilings, `Loop`, `End`. It reads the projection the
[suite generator](suite.md) queues, so the benchmark's subject is seeded the
same way and fed the same fixture values the assertions use: a budget
regression and an assertion failure point at the same call with the same
input.

## The directive

```go
//testkit:bench allocs=0 p99=500us
Read(ctx context.Context, key string) (Payload, error)
```

Method-scoped, because a budget is a property of one hot path. The
directive's presence is the opt-in: a bare `//testkit:bench` measures and
reports, and each key present becomes a ceiling. There is no default budget
— a budget nobody declared is a number the generator invented.

| Key | Gates | Backed by |
|---|---|---|
| `allocs=N` | allocations per operation | `Contract.AllocsMax` — shipped |
| `p99=D` | 99th-percentile latency per operation | `Contract.LatencyMax` — shipped |
| `mean=D` | mean latency per operation | `Contract.MeanMax` — commissioned |
| `mem=B` | bytes allocated per operation | `Contract.BytesMax` — commissioned |

Percentile ceilings carry a sample floor: below one hundred iterations the
metric is reported, not enforced, so a short `-benchtime` run cannot fail a
budget the data cannot support.

## What it generates

Per annotated method, an exported measured-loop body into `_bench.gen.go` —
seeded through the interface's own writer, arguments from the fixture, no
zero values:

```go
func BenchmarkMixedRead(b *testing.B, factory func() validates.Mixed) {
    subject := factory()
    fx := DefaultMixedFixture()
    if err := subject.Store(b.Context(), fx.V); err != nil {
        b.Fatalf("seed: %v", err)
    }
    c := testkit.StartContract(b).AllocsMax(0).LatencyMax(500 * time.Microsecond)
    for c.Loop() {
        _, _ = subject.Read(b.Context(), fx.Key)
    }
    c.End()
}
```

And into `_bench.gen_test.go`, the entry point `go test -bench` discovers,
ranging the suite's subject registry — the consumer registers
implementations once and writes no benchmark shims:

```go
func BenchmarkMixedContract(b *testing.B) // b.Run per subject per annotated method
```

`-bench 'MixedContract/in-memory/Read'` scopes to one subject's one method.
An empty registry skips with the one-line instruction that fills it.

There is no per-method option surface: with the entry generated and the
bodies exported, a custom benchmark is a plain `Benchmark*` function in the
consumer's own file.

## The double

The measured subject is the consumer's implementation — a stub answers from
a table and benchmarking it prices nothing. In delegate mode the generated
helper constructs the double with `<Iface>StubBenchMode()`, which disables
call recording, so the wrapped run prices the delegation and not the
ledger; the delta between plain and wrapped runs is the double's overhead,
measured rather than asserted.

## Layout conventions

| Tag | Suffix | Contents |
|---|---|---|
| *(primary)* | `_bench.gen.go` | The exported measured-loop bodies. |
| `test` | `_bench.gen_test.go` | The registry-ranging `Benchmark<Iface>Contract` entry. |

## See also

- [Primitives / Benchmarking](../primitives/benchmarking.md) — the
  `Contract` runtime the generated bodies drive.
- [Suite](suite.md) — the projection and registry this generator reads.
- [RFC-0003](../../rfc/0003-the-projection-consumers.md) — the design
  record, including the commissioned `Contract` additions.
