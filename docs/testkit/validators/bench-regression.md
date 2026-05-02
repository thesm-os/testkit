# Benchmark Regression

Compares current benchmark output against a pinned
baseline using `benchstat`. Fails on allocation
regressions or latency regressions exceeding configured
thresholds.

## Command

```
testkit validate bench-regression [--baseline bench/baseline.txt] [--current bench/current.txt]
```

## What it checks

1. **Allocations**: any positive alloc/op increase is a
   failure (allocation contracts are ceilings)
2. **Latency**: p99/mean latency regression >= threshold
   is a failure (default: 5%)
3. **Memory**: B/op changes are reported as warnings but
   don't fail (noisy across struct-padding changes)

## Failure output

```
bench-regression: FAIL

  BenchmarkStore_Put: allocs/op regressed +50% (threshold 0%)
  BenchmarkStore_Get: sec/op-p99 regressed +7.2% (threshold 5%)
```

## Workflow

```bash
# Capture a new baseline (on a release tag or accepted regression):
testkit bench capture --output bench/baseline.txt

# Compare current against baseline (in CI):
testkit bench compare --baseline bench/baseline.txt
# Equivalent to: testkit validate bench-regression
```

## Why

Benchmarks without regression gates are vanity metrics.
A benchmark that reports `0 allocs/op` today silently
becomes `3 allocs/op` tomorrow with no build failure.
The regression validator compares against a pinned
baseline, so any performance degradation is immediately
visible.

## Configuration

```yaml
# .testkit.yml
validators:
  bench_regression:
    enabled: true
    baseline: bench/baseline.txt
    # Latency regression threshold (default: 5%).
    latency_threshold_pct: 5
    # Alloc regression threshold (default: 0 = any increase fails).
    alloc_threshold_pct: 0
```
