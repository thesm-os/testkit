# Fault Injection

## FaultInjector

Embeddable, deterministic, counter-based fault injector.
Fires an error on every Nth call. Thread-safe via mutex.

```go
fi := testkit.NewFaultInjector(errBoom, 3) // fires on 3rd, 6th, 9th...
if fi.FaultShouldFire() {
    return fi.FaultErr
}
```

All exported names are prefixed with `Fault` to prevent
collisions when embedded alongside interfaces that define
methods like `Err()`, `Reset()`, or `Count()`.

| Method | Description |
|--------|-------------|
| `NewFaultInjector(err, n)` | Create injector; n <= 0 disables |
| `FaultShouldFire() bool` | Increment counter, return true on Nth |
| `FaultCount() int` | Total calls observed |
| `FaultReset()` | Zero the counter |

Used by generated stubs for Tier 3 fault injection. The
zero value is safe but never fires.
