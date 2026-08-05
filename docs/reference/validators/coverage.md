# Coverage Gates

Enforces per-layer line and branch coverage thresholds
from a config file. Different layers have different
testability ceilings — a single global number doesn't
work.

## Command

```
testkit validate coverage [target ...]
```

A target is a module directory (`model`, `service`,
`plugins`) or a subpath (`service/store`). With no
targets, every layer in the config is exercised.

## What it checks

For each function in the merged coverprofile:

1. Match the function's path against layer definitions
   (longest-prefix match)
2. Compare coverage percentage against the layer's
   threshold
3. Skip structural patterns that are test infrastructure
   (not the system under test)

## Structural skips

Some code is test infrastructure whose assertion branches
only fire when the system under test is broken. Mutating
or coverage-gating this code tests "the test of the test"
— low-value and often unkillable.

| Pattern | Skip reason |
|---------|-------------|
| `*test/*_model.go` | PBT state machine models |
| `*test/*_spec.go` | Conformance suite definitions |
| `Run*Suite` functions | Suite assertion branches |
| `Verify*` functions | Verification assertion branches |

These skips are built into testkit, not configured per
project. They match the naming conventions the generators
produce.

## Failure output

```
coverage: FAIL

  model/   line >= 100%
  ──────────────────────────
  Functions:  42 total · 40 >= threshold · 2 below · 0 skipped
  Verdict:    ✗ FAIL

  Functions below threshold
    66.7%  model/state.go  Apply
    80.0%  model/state.go  Validate
```

With `-v` / `--verbose`, uncovered line ranges are shown:

```
  Uncovered ranges
    model/state.go:45-52 (3 stmts)
    model/state.go:89-91 (1 stmts)
```

## Layer configuration

```yaml
# .testkit.yaml
validators:
  coverage:
    enabled: true
    # Where to find coverprofiles (from `go test -coverprofile`).
    profile_dir: coverage
    layers:
      - path: "model/..."
        line: 100
        branch: 100
      - path: "service/..."
        line: 100
        branch: 100
      - path: "plugins/..."
        line: 90
        branch: 90
    excludes:
      - path: "api/gen/..."
        reason: generated code
      - path: "cmd/.../main.go"
        reason: thin wiring
```

## Why

A single global coverage threshold is either too low
(plugin vendor branches drag it down) or too high
(requires impossible coverage of generated code). Per-
layer thresholds let each layer meet its testability
ceiling: interfaces and value types at 100%, vendor
adapters at 90%, CLI wiring excluded entirely.
