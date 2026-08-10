# testkit generators

A generator reads Go declarations and writes the test infrastructure you would otherwise write by hand. Every one produces plumbing; the developer supplies domain logic through constructor arguments, delegate implementations, or option closures. Regeneration is safe by construction — adding a field to a struct or a method to an interface changes the generated file and nothing the developer wrote.

testkit supplies generators and annotator configuration. Parsing Go, the intermediate representation, typed metadata, slot ordering, determinism, caching and the file sink all come from [eidos](https://go.thesmos.sh/eidos); see [ADR-0003](../../adr/0003-adopt-eidos-as-the-codegen-substrate.md) and [ADR-0004](../../adr/0004-consume-only-the-annotator-plugin.md). Generators do not inspect Go types directly. They read the three classification axes eidos's shape annotator stamps — detector for the signature, contract for the multi-callable protocol, mixin for the declared guarantees — and decide what to emit from those.

**Status.** Four generators ship: `stub`, `builder`, `enum`, `sentinel`. The rest (`suite`, `bench`, `model`, `sim`, `chaos`, `differential-rollout`, `replay`, `codec`, `smoke`, `pkgdoc`) are designed and not implemented; their pages describe a planned shape and say so at the top.

## Opting in

A declaration opts in with a `//testkit:` directive. There are no `//go:generate` lines and no per-generator subcommands — one pipeline run reads the whole package set and every registered generator sees it.

```go
//testkit:out storetest/ pkg=storetest
package store

//testkit:stub
type Store interface { ... }

//testkit:builder
type Item struct { ... }
```

```
testkit run ./...
```

## CLI

```
testkit <command> [flags] [patterns...]

  run       Execute the pipeline and write generated files
  check     Report whether generated files on disk match what the pipeline produces
  prune     Delete generated files the current run no longer claims
  plan      Print the resolved plugin order without running the pipeline
  explain   Print the provenance trace for an entity, slot, or metadata key
  version   Print brand, emit-contract, generator list, and build
```

`check` is the CI form: it exits non-zero when the tree is stale, without writing. `explain` answers "which plugin put this line here", which is the question a shared output file makes hard to answer by reading.

Project-wide conventions live in `.testkit.yaml`; see [Configuration](../configuration.md).

## Shipped generators

| Generator | Reads | Writes |
|---|---|---|
| [`stub`](stub.md) | An interface carrying `//testkit:stub` | A recording test double plus a companion proving it satisfies the interface |
| [`builder`](builder.md) | A struct carrying `//testkit:builder` | A fluent fixture builder with shape-aware setters, `Mutate`, `Clone` |
| [`enum`](enum.md) | A typed constant block carrying `//testkit:enum` | `String`, `Parse`, `Values`, `IsValid`, the text marshallers, and the checks that hold them to the declaration |
| [`sentinel`](sentinel.md) | A package carrying `//testkit:sentinel` | Checks over the package's error contract — prefix, uniqueness, non-overlap, custom-error invariants |

## Directives

| Directive | Scope | Owner | Purpose |
|---|---|---|---|
| `//testkit:stub` | interface | `stub` | Generate a recording double. Accepts `witness=` for generic interfaces. |
| `//testkit:builder` | struct | `builder` | Generate a fixture builder. Accepts `defaults=companion`. |
| `//testkit:enum` | type | `enum` | Generate the textual surface and its checks. Accepts `methods=off`. |
| `//testkit:value` | enum variant | `enum` | Override one variant's textual form. |
| `//testkit:sentinel` | package | `sentinel` | Generate the package's error checks. Accepts `prefix=<value>` and `prefix=off`. |
| `//testkit:sentinel-no-overlap-with` | package | `sentinel` | Name another package this one's sentinels must stay distinct from. Repeats; each line unions. |
| `//testkit:default` | struct field | `defaults` | Seed one field's value in a generated constructor. |
| `//testkit:fault` | interface method | `fault` | Name the errors a double should offer one-shot helpers for. |

A directive name may be registered once per run, so `//testkit:default` and `//testkit:fault` are owned by their own plugins rather than by the generators that consume them. That is what lets more than one generator read the same stamp without depending on each other being registered.

`//testkit:mixin`, `//testkit:contract` and the detector axis come from eidos's shape annotator, not from testkit. They classify a declaration; a generator decides what that classification is worth.

## Routing the output

Every generator declares a filename suffix, and the layout phase composes `<source-basename><suffix>` beside the source. Nothing computes its own path.

| Generator | Primary | Tagged `test` |
|---|---|---|
| `stub` | `_stub.gen.go` | `_stub.gen_test.go` |
| `builder` | `_builder.gen.go` | `_builder.gen_test.go` |
| `enum` | `.enum_gen.go` | `.enum_test.go` |
| `sentinel` | `.gen_test.go` | — |

Override with `//testkit:out`, usually once at package scope:

```go
//testkit:out storetest/ pkg=storetest
package store
```

`out=` sets the directory, `pkg=` the package clause, and `tag=` scopes the override to one output of a generator that has several:

```go
//testkit:out tag=test ./elsewhere/
```

A suffix ending `_test.go` gets the external test package shift for free — the layout phase appends `_test` to the package and import path, which is what makes a generated test drive its subject across a package boundary the way a consumer does.

**`enum` cannot be routed.** Its output declares methods on the enum's type, and Go permits that only in the type's own package.

## Adoption order

1. **`stub`** — the runtime substrate. Install once per interface.
2. **`builder`** — replaces inline composite-literal construction in tests.
3. **`enum`** and **`sentinel`** — static checks with no runtime dependency; the quickest win in an existing tree.

See [Adoption](../../how-to/adoption.md) for the incremental guide.

## Verifying in CI

```yaml
- name: Verify generated code
  run: testkit check ./...
```

`check` reports staleness without writing, so a pull request that edits a source declaration and forgets to regenerate fails there rather than at merge.
