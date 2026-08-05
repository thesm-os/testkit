# Skip Expiry

Verifies that every `t.Skip` call carries a TODO owner
and an expiry date, and that no skip has expired. Catches
deferred flaky-test fixes that were forgotten.

## Command

```
testkit validate skip-expiry
```

## What it checks

1. Every `t.Skip(...)` call in `*_test.go` files contains
   `expires YYYY-MM-DD`
2. No expiry date is in the past (compared to today)
3. Every skip has a `TODO(owner)` attribution

## Required format

```go
t.Skip("TODO(alice): flaky under -race — expires 2026-06-01")
```

## Failure output

```
skip-expiry: FAIL

  store/store_test.go:42
    EXPIRED: expires 2026-04-15 (today is 2026-05-02)
    t.Skip("TODO(alice): flaky under -race — expires 2026-04-15")

  cache/cache_test.go:87
    MISSING EXPIRY: t.Skip has no "expires YYYY-MM-DD"
    t.Skip("TODO: fix later")
```

## Why

`t.Skip` is a pressure valve — it lets a flaky test be
temporarily disabled without blocking the pipeline. But
without an expiry, "temporary" becomes permanent. This
validator turns every skip into a ticking clock: fix the
underlying issue before the expiry, or the build fails.

## Configuration

```yaml
# .testkit.yaml
validators:
  skip_expiry:
    enabled: true
    # Maximum days into the future a skip can be set
    # (prevents "expires 2099-12-31" evasion).
    max_future_days: 90
```
