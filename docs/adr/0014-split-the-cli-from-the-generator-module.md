---
adr: 0014
title: Split the CLI from the generator module
status: Accepted
date: 2026-08-05
supersedes: 0005 (module table only)
superseded-by: none
---

# ADR-0014: Split the CLI from the generator module

## Status

Accepted

## Context

[ADR-0005](0005-split-into-published-modules.md) split the repository into
published modules by what a consumer pays for, and named three: runtime, engine,
and a `tool` module holding "the CLI and the fourteen generator plugins".

Its reasoning holds. Its table does not survive contact with two facts.

**`tool` is not what the things in it are called.** The catalogue, the
reference documentation, and every conversation about them says *generator* —
"the stub generator", "fourteen generators". `plugin` is eidos's word for the
mechanism they are implemented with, and `tool` is a word for the binary. An
import path of `tool/plugin/stub` names the mechanism twice and the thing
itself never.

**A binary and a library want different homes.** Go convention puts a command
under `cmd/`, and both sibling repositories follow it. Nesting that inside a
module whose other contents are importable packages — `tool/cmd/testkit`
alongside `tool/plugin/stub` — produces an install path that says "tool", "cmd",
and "testkit" in a row, where only the last carries information.

The obvious fix, `cmd/testkit` at the repository root, does not work: the root
is the runtime module, so its `go.mod` would then require eidos, and eidos
enters the module graph of every consumer who imports testkit for an assertion.
That is the coupling [ADR-0005](0005-split-into-published-modules.md) exists to
remove. A `cmd/` directory carrying its own `go.mod` avoids it entirely.

## Decision

The `tool` module is replaced by two:

| Path | Module | Holds |
|---|---|---|
| `generator/` | `go.thesmos.sh/testkit/generator` | The generators, one package each |
| `cmd/` | `go.thesmos.sh/testkit/cmd` | The binary |

Four published modules in total. Lockstep tagging
([ADR-0006](0006-tag-published-modules-in-lockstep.md)) covers any count, so the
extra module costs a tag and no coordination.

Installation is `go install go.thesmos.sh/testkit/cmd/testkit@latest`.

Everything else in [ADR-0005](0005-split-into-published-modules.md) stands: the
split is still by dependency weight, `go.work` still joins the modules, and
sibling checkouts are still added locally rather than committed.

## Alternatives Considered

**Keep `tool/cmd/testkit` as originally recorded.** Rejected: it names the same
thing three times in the install path and buries the generators under a word
that describes the binary rather than them.

**One `cmd` module holding the binary and the generators under
`cmd/internal/plugin/`.** Rejected on naming — `cmd/` conventionally holds main
packages, and readers and tooling both assume it. It had one real merit, since
`internal/` would make the plugin API structurally unpublishable, which the
chosen layout gives up: `generator/stub` is importable, so its surface is
public by default and will owe stability from `v1.0.0`
([ADR-0002](0002-support-external-consumers-under-semver.md)). That is a cost
accepted, not overlooked.

**`cmd/testkit` inside the runtime module.** Rejected: it puts eidos in the
`go.mod` of the module every consumer imports.

**Name the generator module `plugin`.** Rejected: it describes how the
generators are built rather than what they are, and the implementation
mechanism is eidos's business rather than the consumer's.

## Consequences

**Positive:**

- The import path matches the vocabulary: `generator/stub` is the stub
  generator.
- The binary lives where Go convention puts it, and the install path carries
  one redundant segment instead of two.
- `cmd` depends on `generator` rather than containing it, so a consumer
  embedding testkit's generators in their own binary takes the generators
  without the CLI.

**Negative:**

- A fourth `go.mod` to keep aligned, and a fourth tag on every release, in a
  repository that already pays that cost three times.
- `generator/stub` and its siblings are importable, so the plugin surface is
  public whether or not it was designed to be. Nothing marks it unstable.
- `generator/` is the path the removed hand-rolled pipeline occupied, so
  `git log --follow` across that boundary is misleading. The name is right and
  the contents share nothing with what was there before.

**Neutral:**

- The module boundary is enforced by depguard rather than by review: generator
  and cmd may import eidos, runtime and engine may not, and cobra is confined
  to cmd.
