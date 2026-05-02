# Orphaned Test Doubles

Detects types in `*test/` packages that are not imported
by any test file. Dead test infrastructure clutters the
codebase and misleads engineers into thinking coverage
exists where it doesn't.

## Command

```
testkit validate orphaned-doubles
```

## What it checks

For every exported type in `*test/` packages:

1. Check whether any `*_test.go` file (in the same module)
   references the type
2. Types referenced only by other `*test/` types (but
   never by an actual test) are also flagged — transitive
   orphans

## What it scans

| Type pattern | Example |
|-------------|---------|
| `InMemory*` | `InMemoryStore` — stub with no test using it |
| `Recording*` | `RecordingCache` — recording wrapper with no test using it |
| `*Builder` | `UserBuilder` — fixture builder with no test using it |
| `Static*` | `StaticResolver` — canned impl with no test using it |
| `*Spec` | `StoreSpec` — conformance suite spec with no wiring test |
| `*Oracle` | `StoreOracle` — PBT oracle with no model test |

## Failure output

```
orphaned-doubles: FAIL

  storetest.RecordingCache
    defined at store/storetest/recording_cache.go:12
    not imported by any *_test.go file

  storetest.StoreOracle
    defined at store/storetest/store_model_oracle.go:8
    not imported by any *_test.go file
    (referenced only by storetest.StoreModel, which is also unused)
```

## Why

Test doubles accumulate during development — an interface
gains a stub, then the interface changes, and the old
stub is never deleted because nothing breaks (it's not
production code). Over time these orphans:

- Inflate line counts and give false confidence in test
  infrastructure coverage
- Confuse engineers who read the `*test/` package and
  assume every type is exercised
- Drift from the interface they were meant to implement
  (no compile error because nothing calls them)

## Configuration

```yaml
# .testkit.yml
validators:
  orphaned_doubles:
    enabled: true
    # Patterns to scan in *test/ packages.
    # Default: all exported types.
    patterns:
      - "InMemory*"
      - "Recording*"
      - "*Builder"
      - "Static*"
      - "*Spec"
      - "*Oracle"
```
