# Adopting testkit

How to integrate testkit into an existing Go project.

## Status

Pre-1.0. Six generators ship today: **`stub`**, **`builder`**, **`sentinel`**, **`enum`**, **`suite`**, **`model`** — plus the `fault` and `defaults` contributors that weave into the double. The remaining generators (`bench`, `sim`, `chaos`, `differential-rollout`, `replay`, `codec`, `smoke`, `pkgdoc`) are designed but not yet implemented; this guide reflects what's available now.

Generators are driven by source directives, not per-generator commands: a `//testkit:<name>` comment on a declaration opts it in, and one `testkit run` regenerates everything the directives claim.

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

The CLI lives in the `cmd` module, built on
[eidos](https://go.thesmos.sh/eidos)
([ADR-0003](../adr/0003-adopt-eidos-as-the-codegen-substrate.md)):

```bash
go install go.thesmos.sh/testkit/cmd/testkit@latest
```

Verify:

```bash
testkit version
```

`version` prints the plugin list with each generator's own version — the
same fingerprints the output headers carry.

## Step 3: Create `.testkit.yaml` (optional)

Only needed when defaults don't match your project conventions or when configuring validators. Defaults:

```yaml
output:
  test_package_suffix: test
  generated_suffix: .gen.go
  test_package_style: external

artifacts:
  dir: .testkit/artifacts
```

See [Configuration](../reference/configuration.md) for the full reference. An absent `.testkit.yaml` is valid — defaults apply.

## Step 4: Add `//testkit:` directives

A directive is a comment on the declaration it opts in. Stack what a type
earns; routing rides the `out=` key:

```go
// store/store.go
package store

// Store is the port your services depend on.
//
//testkit:out storetest/ pkg=storetest
//testkit:stub
//testkit:suite
//testkit:model
type Store interface {
    Get(ctx context.Context, key string) (Item, error)
    Put(ctx context.Context, key string, v Item) error
}
```

```go
// store/status.go
package store

//testkit:enum
type Status uint8

const (
    StatusPending   Status = 1
    StatusConfirmed Status = 2
    StatusCancelled Status = 3
)
```

`//testkit:sentinel` on an error variable's package, `//testkit:builder` on
a struct, `//testkit:fault` and `//testkit:default` where the double and
its zero values need shaping. Each generator's reference page documents its
keys; a wrong key or a misplaced directive is a diagnostic at the line, not
a silent no-op.

## Step 5: Generate

```bash
testkit run ./...
```

One command, every directive. The pipeline parses the packages, classifies
the shapes, runs every generator, and writes only what changed. Run it
twice after upgrading testkit itself: the first pass compile-validates
against the outputs already on disk and may report them stale; the second
converges.

Review the generated files. Commit them to version control.

## Step 6: Wire freshness checks into CI

The CLI carries the drift gate itself — no git-diff scripting:

```makefile
check-generated:
 testkit check ./...
```

`check` runs the full pipeline against an in-memory sink and compares every
output byte-for-byte with the file on disk, exiting non-zero on drift.
`testkit prune` deletes generated files a run no longer claims — the
cleanup after deleting a directive or renaming a type.

## Incremental adoption order

testkit does not require all-or-nothing adoption. Recommended order, given today's state:

1. **Primitives.** Import `testkit` for assertions, `MethodStub`, fault injection, recorders, shape-typed contexts. No config file, no CLI. Immediate value.
2. **`sentinel`.** `//testkit:sentinel` in packages with `Err*` variables and custom error types. Catches prefix violations and accidental aliasing. Quick win.
3. **`enum`.** `//testkit:enum` on iota-constant types. Catches new constants without test coverage, broken stringers, broken Marshal pairs. Quick win.
4. **`builder`.** `//testkit:builder` on fixture structs used across many tests. Eliminates brittle inline `Item{...}` construction.
5. **`stub`.** `//testkit:stub` on interfaces with multiple implementations. The largest time saver for any codebase with non-trivial interfaces; `//testkit:fault` and `//testkit:default` refine the double.
6. **`suite`.** `//testkit:suite` on any interface with a documented contract. Tier 1 conformance — the generated `Assert<Iface>Contract` entry, derived checks per method shape, typed extension points, and a companion that proves every check can fail.
7. **`model`.** `//testkit:model` on stateful interfaces where property-based testing adds value. Tiers 2–3 — rapid-driven state-machine runs against a derived reference, auto-bound laws, the clocked family for time-reading claims, a concurrent Porcupine leg, fuzz targets, and a mutation kill matrix proving the derivation has teeth. Generic interfaces name their types with `witness=`.

When the remaining generators ship, the natural extension order is `bench` → `codec` → `smoke` → `sim` → `chaos`/`replay`/`differential-rollout` → `pkgdoc`.

## Updating generated code

When a source type changes:

```bash
testkit run ./...
```

When testkit itself is updated:

```bash
go get go.thesmos.sh/testkit@latest
go install go.thesmos.sh/testkit/cmd/testkit@latest
testkit run ./... && testkit run ./...
```

Review the diff. Generated files carry a header naming the source file and
every plugin version that produced them, so an upgrade's template changes
surface as a header bump beside the content diff — accept and commit.

## Excluding generated files from review

Add to `.gitattributes`:

```
*.gen.go      linguist-generated=true
*.gen_test.go linguist-generated=true
```

This collapses generated files in GitHub PR diffs while keeping them in version control.
