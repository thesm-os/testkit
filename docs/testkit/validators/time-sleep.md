# time.Sleep in Tests

Detects `time.Sleep` calls in test files. Tests that sleep
are flaky, slow, and non-deterministic — they should use
virtual clocks, `RetryUntil`, or `AssertEventually`
instead.

## Command

```
testkit validate time-sleep
```

## What it checks

1. Every `*_test.go` file is scanned for `time.Sleep`
   calls
2. Every `*test/*.go` file (test infrastructure) is
   scanned for `time.Sleep` calls
3. Flagged calls include the file, line, and the sleep
   duration if statically determinable

## Failure output

```
time-sleep: FAIL

  store/store_test.go:42
    time.Sleep(100 * time.Millisecond)
    use testkit.RetryUntil or testkit.AssertEventually

  cache/cachetest/harness.go:87
    time.Sleep(time.Second)
    use a virtual clock or testkit.Timeout
```

## Allowed exceptions

Some test infrastructure legitimately needs real-time
delays (e.g., `GoroutineLeak`'s brief poll window, or
container health-check intervals). Annotate with:

```go
//testkit:allow-sleep — GoroutineLeak teardown poll
time.Sleep(5 * time.Millisecond)
```

## Why

`time.Sleep` in tests causes three problems:

1. **Flakiness** — sleep durations that work on a fast
   machine fail on a slow CI runner
2. **Slowness** — tests that sleep 100ms each add up to
   minutes across a suite
3. **Non-determinism** — sleep-based synchronization
   depends on scheduler timing, making failures
   unreproducible

Every `time.Sleep` in a test has a better alternative:

- Waiting for a condition → `RetryUntil` or
  `AssertEventually`
- Controlling time → virtual clock
- Waiting for a goroutine → channel, `WaitGroup`, or
  `Recorder.WaitForN`

## Configuration

```yaml
# .testkit.yml
validators:
  time_sleep:
    enabled: true
```
