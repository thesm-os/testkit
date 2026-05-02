# testkit Generators

A CLI tool that reads Go type definitions and proto
descriptors and emits mechanical test boilerplate. Every
generator produces plumbing; the developer injects domain
logic via constructor arguments, oracle interfaces, or
configuration. The generated code is stable — adding a
field to a struct or a method to an interface triggers
regeneration; the developer only touches the injection
point.

All generators use `go/types` for Go source analysis and
`protodesc` for proto schema analysis. Generators are
invoked via `//go:generate` directives in the source
package — no central YAML registry of types. Project-wide
conventions (suffix, test package style, validators) live
in `.testkit.yml`; see [Configuration](../configuration.md).

## CLI interface

```
testkit <subcommand> [flags] <Type> [Type...]

Subcommands:
    stub           Generate in-memory stub
    recording      Generate recording wrapper
    builder        Generate fixture builder
    model          Generate state machine model skeleton
    suite          Generate conformance suite skeleton
    codec          Generate codec test spec from proto
    sentinel       Generate error sentinel tests
    enum           Generate enum exhaustiveness + stringer tests
    differential   Generate differential test harness
    smoke          Generate smoke test
    simworkload    Generate simulation workload
    integration    Generate integration test harness
    wire           Regenerate wire snapshot golden files
    pkgdoc         Generate package audit doc skeleton
    scaffold       One-time companion file with TODOs

Per-generator flags:
    -o <file>      Output file path (default: convention per generator)
    -check         Dry-run: exit non-zero if output differs
    -v             Verbose logging
```

## Output file control

Every generator accepts `-o <file>` to set the output
path. When multiple types are passed as arguments, they
are all written into the single output file. When `-o` is
omitted, each generator uses its conventional default.

| Generator | Default output |
|-----------|---------------|
| stub | `<package>test/in_memory_<subject>.gen.go` |
| recording | `<package>test/recording_<subject>.gen.go` |
| builder | `<package>test/<subject>_builder.gen.go` |
| model | `<package>test/<subject>_model.gen.go` |
| suite | `<package>test/<subject>_spec.gen.go` |
| codec | `<subject>_codec.gen_test.go` |
| sentinel | `errors.gen_test.go` |
| enum | `<subject>_enum.gen_test.go` |
| smoke | `smoke_<subject>.gen_test.go` |
| simworkload | `simtest/workload_<subject>.gen.go` |

## Generators

| Generator | Description |
|-----------|-------------|
| [In-Memory Stubs](stub.md) | Three-tier stub with function delegation + fault injection |
| [Recording Wrappers](recording.md) | Call-capturing decorator with assertion helpers |
| [Fixture Builders](builder.md) | Fluent builder with `With*` per field |
| [State Machine Models](model.md) | rapid.StateMachine with oracle interface |
| [Conformance Suites](suite.md) | Per-contract subtest from `//testkit:` directives |
| [Codec Test Specs](codec.md) | `codectest.Spec[T]` from proto descriptors |
| [Error Sentinel Tests](sentinel.md) | Prefix, non-overlap, uniqueness, unwrap |
| [Enum Tests](enum.md) | Exhaustiveness, stringer, boundary, round-trip |
| [Differential Harness](differential.md) | Fan-out wrapper comparing N implementations |
| [Wire Snapshots](wire.md) | Regenerate `testdata/wire/*.bin` golden files |
| [Package Doc Skeleton](pkgdoc.md) | Compliance audit document from exported symbols |
| [Integration Harness](integration.md) | TestMain + container + conformance suite wiring |
| [Smoke Test](smoke.md) | One-call-per-method "does it turn on?" test |
| [Simulation Workload](simworkload.md) | Random method dispatch for sim drivers |
| [Scaffold](scaffold.md) | One-time companion file generation |

## CI integration

Since all generators are declared via `//go:generate`,
standard Go tooling handles regeneration:

```yaml
- name: Verify generated code
  run: |
    go generate ./...
    git diff --exit-code
```

Or via Makefile:

```makefile
generate: generate-proto generate-testkit

generate-testkit:
    go generate ./...

check: lint test check-testkit
```

## Priority order

1. In-memory stubs (4,800 lines, highest error surface)
2. Recording wrappers (900 lines, enables call verification)
3. Fixture builders (2,200 lines, medium frequency)
4. Codec test specs (870 lines, highest correctness risk)
5. Conformance suites (10,900 lines, requires contract directives)
6. State machine models (4,800 lines, complex but high value)
7. Error sentinel tests (500 lines, quick win)
8. Enum tests (1,200 lines, quick win)
9. Smoke tests (1,200 lines, fast onboarding for new impls)
10. Differential harness (1,000 lines, blocks sim phase)
11. Simulation workloads (1,200 lines, sim infrastructure)
12. Integration harnesses (400 lines, plugin wiring)
13. Wire snapshot regen (automation)
14. Package doc skeleton (compliance automation)
