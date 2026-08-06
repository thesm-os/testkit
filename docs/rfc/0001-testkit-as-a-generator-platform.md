---
rfc: 0001
title: testkit as a generator platform
status: Draft
date: 2026-08-05
---

# RFC-0001: testkit as a generator platform

## Summary

testkit is a platform for fourteen code generators that read Go interfaces and
types and emit the test doubles, fixtures, conformance suites, and benchmarks a
team would otherwise hand-write — plus the tests that prove the generated code
is correct.

This RFC states what that platform is shaped like and indexes the decisions that
fix the shape. It specifies no generator: those are documented individually in
[reference/generators/](../reference/generators/). It is a map, not a manual.

## Problem

testkit owned its entire code-generation pipeline: frontend, intermediate
representation, slot ordering, determinism, caching, and sink. Six of fourteen
generators shipped. The remaining eight were blocked behind pipeline work rather
than behind anything to do with testing.

Three things were wrong with that.

**The pipeline was not the product.** Fourteen generators is the product; the
machinery that walks a Go package and writes a file is a means. Every hour spent
on slot topology was an hour not spent on a generator.

**The pipeline was already owned elsewhere.** Shape classification — twenty
detectors, twenty-four contracts, twenty-eight mixins, seventy-two
classifications in total — exists in [eidos](https://go.thesmos.sh/eidos) and is
maintained there.
testkit reimplemented thirty-nine of forty-four of those packages. Two
independent implementations of the same classification vocabulary is not
redundancy, it is drift waiting to happen.

**Consumers paid for everything.** A caller who wanted `testkit.Equal` and a
`MethodStub` took the property-testing engine, the linearizability checker, and
the code generator's dependency graph along with it.

## Shape

### Four published modules, split by what a consumer pays for

| Module | Contains | Dependency cost |
|---|---|---|
| `go.thesmos.sh/testkit` | Runtime primitives — assertions, stubs, recording, fault injection, clocks, golden files, polling | go-cmp |
| `go.thesmos.sh/testkit/engine` | Model checking — property runner, law library, linearizability models, reference implementations, bounded model checking | + rapid, porcupine |
| `go.thesmos.sh/testkit/generator` | The generators, one package each | + eidos |
| `go.thesmos.sh/testkit/cmd` | The binary | + eidos, cobra |

A consumer who wants assertions and stubs takes the first and none of the rest.
A consumer who wants generated conformance suites installs the binary at build
time and ships nothing extra at runtime. A consumer embedding the generators in
a binary of their own takes `generator` without the CLI. The split is enforced by module boundaries
rather than by convention, so it cannot erode.

A fourth module, `conformance`, is unpublished: it holds the generated corpus
that proves the generators work. Unpublished modules cost no tags.

See [ADR-0005](../adr/0005-split-into-published-modules.md),
[ADR-0006](../adr/0006-tag-published-modules-in-lockstep.md), and
[ADR-0014](../adr/0014-split-the-cli-from-the-generator-module.md).

### One boundary with eidos

eidos supplies the frontend, the intermediate representation, typed metadata
with provenance, slot ordering, determinism, caching, and the sink. testkit
supplies annotator *configuration* and every generator plugin.

The boundary is narrow and deliberate: testkit consumes `plugins/annotator` and
nothing else. It writes no annotators of its own, and it adopts none of eidos's
generators — not even the three whose names collide.
See [ADR-0003](../adr/0003-adopt-eidos-as-the-codegen-substrate.md) and
[ADR-0004](../adr/0004-consume-only-the-annotator-plugin.md), and
[the classification map](../internal/classification-map.md) for the audit that
confirmed the vocabulary maps across without gaps.

```
Go source
    │
    ▼
eidos frontend ──► eidos annotators ──► testkit plugins ──► eidos backend ──► sink
                   (shape: 72                (14 generators)
                    classifications)
```

### Fourteen generators, three tiers of maturity

The catalogue is documented per generator in
[reference/generators/](../reference/generators/). Three are deferred with a
stated reason ([ADR-0013](../adr/0013-defer-codec-pkgdoc-and-smoke.md)); the
rest are in scope.

Directive vocabulary is shared across the catalogue and namespaced to `testkit`.
Directives take the neutral form with an axis qualifier, so several properties
batch onto one line, and there is no negated form
([ADR-0016](../adr/0016-directives-are-positive-only.md)):

```go
//testkit:mixin idempotent concurrent-safe
//testkit:contract lease
```

## What is decided

Every load-bearing decision has its own record. Read them for the alternatives
that were rejected and the trade-offs accepted.

| ADR | Decision |
|---|---|
| [0001](../adr/0001-record-decisions-as-adrs.md) | Record decisions as ADRs |
| [0002](../adr/0002-support-external-consumers-under-semver.md) | Support external consumers under semver |
| [0003](../adr/0003-adopt-eidos-as-the-codegen-substrate.md) | Adopt eidos as the codegen substrate |
| [0004](../adr/0004-consume-only-the-annotator-plugin.md) | Consume only eidos's annotator plugin |
| [0005](../adr/0005-split-into-published-modules.md) | Split into published modules behind a go.work |
| [0006](../adr/0006-tag-published-modules-in-lockstep.md) | Tag published modules in lockstep |
| [0007](../adr/0007-earn-top-level-packages-by-import.md) | Earn top-level packages by import |
| [0008](../adr/0008-neutral-directive-form-with-axis-qualifier.md) | Neutral directive form with an axis qualifier |
| [0009](../adr/0009-one-config-filename.md) | One config filename |
| [0010](../adr/0010-first-stable-release-is-v1.md) | The first stable release is v1.0.0 |
| [0011](../adr/0011-collapse-ref-packages.md) | Collapse the reference-implementation packages |
| [0012](../adr/0012-generate-per-shape-helpers-into-the-consumer.md) | Generate per-shape helpers into the consumer |
| [0013](../adr/0013-defer-codec-pkgdoc-and-smoke.md) | Defer codec, pkgdoc, and smoke |
| [0014](../adr/0014-split-the-cli-from-the-generator-module.md) | Split the CLI from the generator module |
| [0015](../adr/0015-subtest-names-carry-the-classification.md) | Subtest names carry the classification |
| [0016](../adr/0016-directives-are-positive-only.md) | Directives are positive-only |
| [0017](../adr/0017-every-classification-owes-an-assertion.md) | Every classification owes an assertion |

## Execution

The restructure is a sequence of individually-buildable commits on the `v2`
branch, not a cleared tree and not an orphan branch. The branch name is a
working label; the release it produces is `v1.0.0`
([ADR-0010](../adr/0010-first-stable-release-is-v1.md)). Roughly 44,000 lines move unchanged, so
starting fresh would mean restoring most of them immediately; an orphan branch
additionally breaks goreleaser's `.PreviousTag`, the release tooling, and
`git bisect`.

Every commit is a buildable artifact. Rename detection survives the moves
because an import rewrite changes about three lines in a two-hundred-line file,
far above the similarity threshold.

| Step | State |
|---|---|
| Split the runtime and engine modules; delete the hand-rolled pipeline | Done — `09b3e8b` |
| Drive both modules to their coverage gates — runtime 99.6%, engine 96.3% at its lowest package | Done — `aa91066` |
| Record the design as RFC and ADRs | Done — `1adfd52` |
| Audit the eidos classifications against what the catalogue needs | Done — [the classification map](../internal/classification-map.md) |
| Stand up the `generator` and `cmd` modules | Done — `ee57412` |
| Stand up the `conformance` module and its gate | In progress |
| Port the generator plugins | Not started |

## Open questions

- **Directive schema ownership.** The classification audit settles the
  vocabulary but not the schemas. Which directive definitions survive as
  testkit's own, and which are parser machinery eidos already provides, is a
  second audit over the deleted `generator/directive` tree.
- **`core/registry`.** The plugin registry's shape is unfixed. It is the
  smallest self-contained unit left and is contract-gated on its own.
- **Simulation.** `sim` and the tier-5 generators built on it (`chaos`,
  `replay`, `differential-rollout`) are in the catalogue but have no
  implementation plan. They may warrant their own RFC.
