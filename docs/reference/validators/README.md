# testkit Validators

> **Status: out of scope.** The validator tier is not part of the current
> generator platform and nothing here is implemented. These documents are
> retained as the design record in case the tier returns — see
> [RFC-0001](../../rfc/0001-testkit-as-a-generator-platform.md) for what is in
> scope. Everything below describes an intended design, not shipped behaviour.

CI-time checks that enforce structural invariants across
the codebase. Each validator runs as a `testkit validate`
subcommand and exits non-zero on failure. Wire them into
`make check` or a dedicated CI step.

Validators produce no generated files — they are pure
checks. This distinguishes them from generators, which
produce `.gen.go` files.

## CLI interface

```
testkit validate <check> [flags]

Structural:
    proto-sync           Proto files match generated Go
    migration            SQL migration chain is valid
    depguard             Import graph matches layer rules
    wire                 Wire golden files are fresh
    error-prefix         errors.New uses correct package prefix
    skip-expiry          t.Skip calls have valid expiry dates

Test quality:
    assertion-free       Every Test* function contains an assertion
    test-naming          No fragmented Test<Type>_* top-level functions
    time-sleep           No time.Sleep in test files
    orphaned-doubles     No unused types in *test/ packages
    parallel-safety      No t.Setenv+t.Parallel, no shared mutable state
    contract-completeness Every documented contract has a benchmark

Quality gates:
    benchmarks           Every benchmark has a performance contract
    bench-regression     Benchmark output vs pinned baseline
    coverage             Per-layer line/branch coverage thresholds
    mutation             Per-layer mutation testing thresholds

Compliance:
    audit                Every package has a complete audit doc
    reqs                 REQ-to-test traceability is intact

Global flags:
    -v             Verbose logging (uncovered ranges, mutant locations)
    -config <path> Override .testkit.yaml path
```

## Validators

### Structural

| Validator | Description |
|-----------|-------------|
| [Proto-Sync](proto-sync.md) | Proto files match generated Go codecs |
| [Migration](migration.md) | SQL migration chain is contiguous and valid |
| [Depguard](depguard.md) | Import graph matches layer boundary rules |
| [Wire](wire.md) | Wire golden file freshness |
| [Error Prefix](error-prefix.md) | Every `errors.New` uses `"<pkg>: "` prefix |
| [Skip Expiry](skip-expiry.md) | Every `t.Skip` has a valid, non-expired expiry date |

### Test quality

| Validator | Description |
|-----------|-------------|
| [Assertion-Free](assertion-free.md) | Every `Test*` function contains at least one assertion |
| [Test Naming](test-naming.md) | No fragmented `Test<Type>_*`; subtests read as contracts |
| [time.Sleep](time-sleep.md) | No `time.Sleep` in test files |
| [Orphaned Doubles](orphaned-doubles.md) | No unused types in `*test/` packages |
| [Parallel Safety](parallel-safety.md) | No `t.Setenv` + `t.Parallel`; no shared mutable state |
| [Contract Completeness](contract-completeness.md) | Every documented contract has a gated benchmark |

### Quality gates

| Validator | Description |
|-----------|-------------|
| [Benchmarks](benchmarks.md) | Every benchmark uses `StartContract` with ceilings |
| [Bench Regression](bench-regression.md) | No alloc increase, no latency regression vs baseline |
| [Coverage](coverage.md) | Per-layer line/branch coverage thresholds |
| [Mutation](mutation.md) | Per-layer mutation efficacy + coverage thresholds |

### Compliance

| Validator | Description |
|-----------|-------------|
| [Audit Completeness](audit-completeness.md) | Every package has a complete audit doc with no gaps |
| [REQ Traceability](req-traceability.md) | Every REQ traces to a test, every test traces to a REQ |

## Integration with make check

```makefile
# Structural checks (fast, run on every PR):
check-structural:
    testkit validate proto-sync
    testkit validate migration
    testkit validate depguard
    testkit validate wire
    testkit validate error-prefix
    testkit validate skip-expiry

# Test quality checks (fast, static analysis):
check-test-quality:
    testkit validate assertion-free
    testkit validate test-naming
    testkit validate time-sleep
    testkit validate orphaned-doubles
    testkit validate parallel-safety
    testkit validate contract-completeness

# Quality gates (after tests pass):
check-quality:
    testkit validate benchmarks
    testkit validate bench-regression
    testkit validate coverage
    testkit validate mutation

# Compliance (after quality gates pass):
check-compliance:
    testkit validate audit
    testkit validate reqs

check: lint test check-structural check-test-quality check-quality check-compliance
```

Each validator is independent and can run in parallel
within its group. Exit code is 0 (pass) or 1 (fail) with
human-readable output on stderr.

## Full CI gate example

```yaml
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - name: Install testkit
        run: go install go.thesmos.sh/testkit/tool/cmd/testkit@latest

      - name: Structural checks
        run: |
          go generate ./...
          git diff --exit-code
          testkit validate proto-sync
          testkit validate migration
          testkit validate depguard
          testkit validate wire
          testkit validate error-prefix
          testkit validate skip-expiry

      - name: Quality gates
        run: |
          testkit validate benchmarks
          testkit validate bench-regression
          testkit validate coverage
          testkit validate mutation

      - name: Compliance
        run: |
          testkit validate audit
          testkit validate reqs
```
