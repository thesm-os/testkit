# Adopting testkit

How to integrate testkit into an existing Go project.

## Prerequisites

- Go 1.22 or later
- Proto files use `buf` for generation (for codec
  generators)
- Project uses `go.mod` (module-aware mode)

## Step 1: Add the dependency

```bash
go get go.thesmos.sh/testkit@latest
```

For primitives only (no code generation), this is the
only step needed. Import and use directly:

```go
import "go.thesmos.sh/testkit"

func TestSomething(t *testing.T) {
    testkit.Equal(t, got, want, "result mismatch")
}
```

## Step 2: Install the CLI

```bash
go install go.thesmos.sh/testkit/cmd/testkit@latest
```

Verify:

```bash
testkit version
```

## Step 3: Create .testkit.yml (optional)

Only needed if you want non-default conventions or
validators. Create manually:

```yaml
proto_root: api/proto

output:
  test_package_suffix: test
  generated_suffix: .gen.go
```

See [Configuration](configuration.md) for the full
reference. If omitted, all defaults apply.

## Step 4: Add //go:generate directives

Add directives to the packages that own the types. A
common pattern is a `generate.go` file per package:

```go
// store/generate.go

package store

//go:generate testkit stub -o storetest/in_memory_store.gen.go Store
//go:generate testkit recording -o storetest/recording_store.gen.go Store
//go:generate testkit builder -o storetest/builders.gen.go User Entry
//go:generate testkit suite -o storetest/store_spec.gen.go Store
//go:generate testkit model -o storetest/store_model.gen.go Store
//go:generate testkit sentinel -o errors.gen_test.go
//go:generate testkit enum -o status_enum.gen_test.go Status
```

Or place directives alongside the types they reference:

```go
// store/status.go

package store

//go:generate testkit enum -o status_enum.gen_test.go Status

type Status uint8

const (
    StatusPending   Status = 1
    StatusConfirmed Status = 2
    StatusCancelled Status = 3
)
```

## Step 5: Generate

```bash
go generate ./...
```

Review the generated files. Commit them to version
control.

## Step 6: Wire into CI

Add freshness checks:

```makefile
check-generated:
    go generate ./...
    @if ! git diff --quiet -- '*.gen.go' '*.gen_test.go'; then \
        echo "generated files are stale — run: go generate ./..."; \
        git diff --stat -- '*.gen.go' '*.gen_test.go'; \
        exit 1; \
    fi
```

Add validators (if `.testkit.yml` is configured):

```makefile
check-testkit:
    testkit validate proto-sync
    testkit validate migration
    testkit validate depguard
    testkit validate wire

check: lint test check-generated check-testkit
```

## Incremental adoption

testkit does not require all-or-nothing adoption.
Recommended order:

1. **Primitives first.** Import `testkit` for assertions,
   fault injection, containers, and recorders. No config
   file needed, no CLI needed. Immediate value.

2. **Error sentinels.** Add `//go:generate testkit sentinel`
   to packages with `Err*` variables. Catches prefix
   violations and accidental aliasing. Quick win.

3. **Enum tests.** Add `//go:generate testkit enum` to
   packages with iota constants. Catches additions without
   test coverage. Quick win.

4. **Codec specs.** Add `//go:generate testkit codec` to
   packages with proto types. Replaces hand-written field
   classification tables.

5. **Fixture builders.** Add `//go:generate testkit builder`
   for types used across many test files. Eliminates
   brittle inline construction.

6. **In-memory stubs.** Add `//go:generate testkit stub`
   for interfaces with multiple implementations. The
   biggest time saver for large codebases.

7. **Recording wrappers.** Add
   `//go:generate testkit recording` for interfaces where
   you need to verify call patterns. Independent of stubs
   — can wrap any implementation.

8. **Conformance suites.** Add `//go:generate testkit suite`
   for interfaces with documented contracts. Ensures every
   implementation meets the same bar.

9. **Smoke tests.** Add `//go:generate testkit smoke` for
   new implementations before full conformance wiring.
   The "does it turn on?" check.

10. **State machine models.** Add
    `//go:generate testkit model` for stateful interfaces.
    The highest-value generator.

11. **Differential harnesses.** Add
    `//go:generate testkit differential` when you have
    multiple implementations of the same interface.

12. **Simulation workloads.** Add
    `//go:generate testkit simworkload` for interfaces
    that need random-operation sim drivers.

13. **Validators.** Create `.testkit.yml` with validator
    config and wire into CI once the codebase generates
    enough code to benefit from freshness checks.

## Updating generated code

When the source type or proto changes:

```bash
go generate ./...
```

When testkit itself is updated:

```bash
go get go.thesmos.sh/testkit@latest
go generate ./...
```

Review the diff. Generated files include a header with
the testkit version; version bumps that change output
format will show in the diff.

## Excluding generated files from review

Add to `.gitattributes`:

```
*.gen.go linguist-generated=true
*.gen_test.go linguist-generated=true
```

This collapses generated files in GitHub pull request
diffs while keeping them in version control.
