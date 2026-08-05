# Mutation Gates

Enforces per-layer mutation testing thresholds. Two
metrics gate independently: test efficacy (do tests catch
mutations?) and mutator coverage (do tests reach the
mutatable code?).

## Command

```
testkit validate mutation [target ...]
```

A target is a module directory or subpath. With no
targets, every layer in the config is exercised.

## What it checks

For each layer:

1. Run mutation testing (via `gremlins` or configured
   mutator)
2. Parse output for efficacy and coverage percentages
3. Compare against per-layer thresholds

## Two metrics

| Metric | Formula | What it catches |
|--------|---------|-----------------|
| **Score** (efficacy) | KILLED / (KILLED + LIVED) | Tests exercise the path but assert nothing |
| **Coverage** (mutator) | (KILLED + LIVED) / TOTAL | Tests don't reach the mutatable code at all |

A package can score 100% efficacy on a single KILLED
mutant while leaving 99% of the surface untouched —
coverage catches that case. Both must pass.

## Structural excludes

Same philosophy as coverage gates — test infrastructure
is excluded from mutation because mutating it tests "the
test of the test."

| Pattern | Reason |
|---------|--------|
| `*_string.go` | Stringer-generated; identity mutants are unkillable |
| `*.codec.go` | Generated codecs; offset arithmetic doesn't map to defects |
| `*test/*_spec.go` | Conformance suites are the test, not the SUT |
| `*test/*_model.go` | PBT models are the test, not the SUT |
| `sim/` | Simulation engine is test infrastructure |

## Failure output

```
mutation: FAIL

  model/   score >= 100% / coverage >= 99%
  ──────────────────────────
  Mutants:   142 killed · 3 lived · 2 not covered · 0 timed out
  Score:      97.9%   (>= 100%)   ✗ FAIL
  Coverage:   99.3%   (>= 99%)    ✓ PASS

  Contributing files   (L=lived / NC=not covered / TO=timed out)
       2   model/state.go   (L:2 / NC:0 / TO:0)
       1   model/apply.go   (L:1 / NC:0 / TO:0)
```

With `-v`, every non-killed mutant is listed with
file:line:col for direct navigation.

## Mutation equivalents

Some mutations are mathematically equivalent to the
original code and cannot be killed without contrived
tests. These are declared with inline comments:

```go
//mutation:equivalent: boundary-tight — x > 0 vs x >= 1 on unsigned
```

Equivalent declarations must be signed off in the
package audit document. The validator counts them as
KILLED for threshold purposes.

## Configuration

```yaml
# .testkit.yaml
validators:
  mutation:
    enabled: true
    # Mutator tool (default: gremlins).
    tool: gremlins
    layers:
      - path: "model/..."
        score: 100
        coverage: 99
      - path: "service/..."
        score: 100
        coverage: 99
      - path: "plugins/..."
        score: 80
        coverage: 80
    # Gremlins-specific settings.
    timeout_coefficient: 30
    workers: 4
    test_cpu: 2
```
