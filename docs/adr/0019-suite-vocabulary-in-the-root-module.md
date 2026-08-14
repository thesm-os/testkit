---
adr: 0019
title: The suite package lives in the root module
status: Accepted
date: 2026-08-14
supersedes: none
superseded-by: none
---

# ADR-0019: The suite package lives in the root module

## Status

Accepted

## Context

[RFC-0004](../rfc/0004-the-suite-contract.md) defines a `suite` package
with the types every generated conformance package uses: `ID`, `Class`,
`Caps`, `Check[S]`, `Subject[S]`, `Suite[S]`, `Run`, `Report`. It needs a
home. The choice decides what dependencies every consumer inherits.

The root module `go.thesmos.sh/testkit` is what consumers already import
for assertions. It depends on exactly one module (go-cmp). The engine
module depends on rapid and is only needed for model-tier checks.

## Decision

The `suite` package goes in the root module. It imports only `testing` and
`go.thesmos.sh/testkit/clock`, and it adds no module dependencies. It never
imports `engine/model`; generated model checks import the engine
themselves. A test checks the import list and fails the build if it grows.

## Alternatives Considered

**A separate `suite` module.** One more module to tag and version, and
every consumer of it already imports the root module. No benefit.

**Put it in the engine module.** That would add rapid and the engine to
the dependency graph of every consumer, including those that only use the
cheap tier. Rejected.

**No shared package; generate the types into each package.** Rejected in
RFC-0004: every generated package would have its own copy of the
vocabulary, and tools could not work across packages.

## Consequences

**Positive:**

- Consumers get the vocabulary without any new dependencies.
- One set of types to learn.

**Negative:**

- The first tag freezes the API. To reduce that risk, `suite` stays at v0
  until two consumers use it (RFC-0004, Migration).
- The import-list test must exist from the first commit. Adding it later
  is harder.

**Neutral:**

- [ADR-0018](0018-one-tier-owns-each-classification.md) is unchanged. This
  decision only places the types.
