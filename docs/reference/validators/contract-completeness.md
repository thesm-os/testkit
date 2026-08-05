# Contract Completeness

Verifies that every documented allocation or latency
contract has a matching `StartContract`-gated benchmark.
The `benchmarks` validator checks existing benchmarks have
contracts; this validator checks the reverse — every
documented contract has a benchmark.

## Command

```
testkit validate contract-completeness
```

## What it checks

1. Scan all `.go` files (non-test) for doc comments
   containing `# Allocation contract` or
   `# Latency contract`
2. For each documented contract, check that a
   `Benchmark*` function exists in the corresponding
   `*_test.go` file with a `StartContract` call that
   includes the matching ceiling (`AllocsMax` for
   allocation, `LatencyMax` for latency)
3. Verify the ceiling in the benchmark matches the
   documented value (e.g., doc says "0 allocs" →
   benchmark has `AllocsMax(0)`)

## Example

Source with documented contract:

```go
// Put stores an item.
//
// # Allocation contract
//
// 0 allocations per call in steady state.
func (s *Store) Put(ctx context.Context, item Item) error {
```

Matching benchmark:

```go
func BenchmarkStore(b *testing.B) {
    b.Run("Put/zero alloc steady state", func(b *testing.B) {
        s := setup()
        c := testkit.StartContract(b).AllocsMax(0)
        for c.Loop() {
            _ = s.Put(ctx, item)
        }
        c.End()
    })
}
```

## Failure output

```
contract-completeness: FAIL

  store/store.go:42: Put
    # Allocation contract: 0 allocs
    no matching BenchmarkStore*/Put* with AllocsMax(0)

  cache/cache.go:87: Get
    # Latency contract: < 5µs p99
    no matching BenchmarkCache*/Get* with LatencyMax
```

## Why

Documentation and benchmarks drift independently. A
method gains an `# Allocation contract` during design
review, but the benchmark is written later — or never.
Without this validator, the contract exists only as prose
in a doc comment, unenforceable by CI.

The `benchmarks` validator ensures benchmarks that exist
have gates. This validator ensures gates that should exist
actually do.

## Configuration

```yaml
# .testkit.yaml
validators:
  contract_completeness:
    enabled: true
    # Doc comment markers to scan for.
    markers:
      - "# Allocation contract"
      - "# Latency contract"
```
