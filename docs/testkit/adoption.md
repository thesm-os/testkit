# Adopting testkit

How to integrate testkit into an existing Go project.

## Status

Pre-1.0. Four generators ship today: **`stub`**, **`builder`**, **`sentinel`**, **`enum`**. The remaining generators (`suite`, `model`, `bench`, `sim`, `chaos`, `differential-rollout`, `replay`, `codec`, `smoke`, `pkgdoc`) are designed but not yet implemented; this guide reflects what's available now.

## Prerequisites

- Go 1.24 or later (testkit uses `t.Context()`)
- Project uses `go.mod` (module-aware mode)

## Step 1: Add the dependency

```bash
go get go.thesmos.sh/testkit@latest
```

For primitives only — no code generation — that's everything you need:

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

## Step 3: Create `.testkit.yml` (optional)

Only needed when defaults don't match your project conventions or when configuring validators. Defaults:

```yaml
output:
  test_package_suffix: test
  generated_suffix: .gen.go
  test_package_style: external

stub:
  file-pattern: "{type}_stub"
  type-suffix: Stub
```

See [Configuration](configuration.md) for the full reference. An absent `.testkit.yml` is valid — defaults apply.

## Step 4: Add `//go:generate` directives

Place directives in the package that owns the types. A common pattern is a `generate.go` per package:

```go
// store/generate.go
package store

//go:generate testkit stub      -o storetest/store_stub.gen.go   Store
//go:generate testkit stub      -o storetest/cache_stub.gen.go   Cache
//go:generate testkit builder   -o storetest/builders.gen.go     User Item
//go:generate testkit sentinel  -o errors.gen_test.go
//go:generate testkit enum      -o status_enum.gen_test.go       Status
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

Review the generated files. Commit them to version control.

## Step 6: Wire freshness checks into CI

```makefile
check-generated:
	go generate ./...
	@if ! git diff --quiet -- '*.gen.go' '*.gen_test.go'; then \
	    echo "generated files are stale — run: go generate ./..."; \
	    git diff --stat -- '*.gen.go' '*.gen_test.go'; \
	    exit 1; \
	fi
```

When validators ship, add them similarly:

```makefile
check-testkit:
	testkit validate proto-sync migration depguard wire

check: lint test check-generated check-testkit
```

## Incremental adoption order

testkit does not require all-or-nothing adoption. Recommended order, given today's state:

1. **Primitives.** Import `testkit` for assertions, `MethodStub`, fault injection, recorders. No config file, no CLI. Immediate value.
2. **`sentinel`.** Add `//go:generate testkit sentinel` to packages with `Err*` variables and custom error types. Catches prefix violations and accidental aliasing. Quick win.
3. **`enum`.** Add `//go:generate testkit enum` to packages with iota constants. Catches new constants without test coverage, broken stringers, broken Marshal pairs. Quick win.
4. **`builder`.** Add `//go:generate testkit builder` for fixtures used across many tests. Eliminates brittle inline `Item{...}` construction.
5. **`stub`.** Add `//go:generate testkit stub` for interfaces with multiple implementations. The largest time saver for any codebase with non-trivial interfaces.

When the remaining generators ship, the natural extension order is `suite` → `bench` → `model` → `codec` → `smoke` → `sim` → `chaos`/`replay`/`differential-rollout` → `pkgdoc`.

## Updating generated code

When a source type or proto changes:

```bash
go generate ./...
```

When testkit itself is updated:

```bash
go get go.thesmos.sh/testkit@latest
go generate ./...
```

Review the diff. Generated files include a header noting the testkit subcommand and source location; regeneration after a testkit upgrade may show formatting or template changes — accept and commit.

## Excluding generated files from review

Add to `.gitattributes`:

```
*.gen.go      linguist-generated=true
*.gen_test.go linguist-generated=true
```

This collapses generated files in GitHub PR diffs while keeping them in version control.
