# Benchmark Contracts

Verifies that every `Benchmark*` function in the codebase
uses a `testkit.StartContract` with explicit allocation
and/or latency ceilings. Prevents benchmarks from
silently regressing when someone adds a `Benchmark*`
function without a performance contract.

## Command

```
testkit validate benchmarks
```

## What it checks

1. Every `func Benchmark*(b *testing.B)` function contains
   a `testkit.StartContract` call — the [bench
   generator](../generators/bench.md)'s outputs satisfy
   this by construction
2. Every `StartContract` chain includes at least one of
   `AllocsMax` or `LatencyMax` (a contract without
   ceilings is vacuous)
3. Every `StartContract` chain ends with `.End()` (a
   contract without `End` never asserts)

## Failure output

```
benchmarks: FAIL

  service/store/tamper_evident_test.go
    BenchmarkTamperEvident_Verify: no StartContract — add allocation/latency ceilings

  model/state_test.go
    BenchmarkState_Get: StartContract without AllocsMax or LatencyMax — contract is vacuous
```

## Why

Benchmarks without contracts are vanity metrics — they
measure but never assert. A benchmark that reports
`0 allocs/op` today will silently become `3 allocs/op`
tomorrow with no test failure. The validator ensures
every benchmark has a contract that will fail the build
when the performance budget is exceeded.

## Configuration

```yaml
# .testkit.yaml
validators:
  benchmarks:
    enabled: true
    # Packages to exclude (e.g., third-party wrappers
    # where you don't control allocations).
    exclude:
      - artifacts/v1
```

---
