---
adr: 0021
title: Field roles resolve in the generator
status: Accepted
date: 2026-08-14
supersedes: none
superseded-by: none
---

# ADR-0021: Field roles resolve in the generator

## Status

Accepted

## Context

RFC-0004 assigns roles to request-struct fields (drawn, stamped, or
pinned). Something has to decide which field has which role, using either
an explicit `//testkit:role` directive or inference from the field's type.
This needs type information from the consumer's module, a keyword table,
and a record of how each decision was made.

`generator/tiers` is the existing pattern for this kind of table:
hand-written, checked by gate tests in both directions.

## Decision

Roles resolve in a new package, `generator/roles`, built like
`generator/tiers`. The runtime never sees roles: by the time a
`suite.Check` value exists, roles have already been turned into closures
and generator wiring.

The resolver must record how every binding was made (`declared`,
`inferred: type identity`, or `inferred: name+type`). It cannot return a
binding without that record. The record appears in `testkit explain`, the
run report, the generated file header, and in the failure message of any
law whose fields were inferred.

Inference only matches named types declared in the consumer's own module.
It never matches plain `string` or `int`.

## Alternatives Considered

**Resolve in the runtime.** The runtime would need reflection and the
keyword table, and [ADR-0019](0019-suite-vocabulary-in-the-root-module.md)
just fixed its imports at zero. Roles only matter during generation, so
there is nothing for the runtime to do with them. Rejected.

**Resolve in eidos.** eidos classifies methods; roles classify fields.
The keyword set is testkit's own
([ADR-0026](0026-role-keywords-enter-by-registry.md)), and moving it
upstream grows the eidos coupling for a vocabulary only testkit uses.
Rejected.

**Inference without the record.** A wrong inference makes a correct
implementation fail. The record is what lets the consumer see, at the
failure, that inference caused it and how to override it. Rejected.

## Consequences

**Positive:**

- The runtime stays small. The eidos coupling stays in the generator
  module, where it already is.
- The `tiers` pattern is reused instead of invented again.

**Negative:**

- A second table package to maintain, with its own gate tests.
- Inference only works if the consumer uses named types. A codebase that
  uses plain strings everywhere gets no inference and must write every
  role directive by hand.

**Neutral:**

- `testkit explain` and `advise` read the same tables. They are not
  second resolvers.
