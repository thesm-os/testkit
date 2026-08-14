---
adr: 0027
title: Optional features are declared on the role
status: Accepted
date: 2026-08-14
supersedes: none
superseded-by: none
---

# ADR-0027: Optional features are declared on the role

## Status

Accepted

## Context

The harness (RFC-0004, deferred sketch) lets a third-party implementation
claim conformance while skipping features the publisher marked optional.
Conformance mode has no `Without`, so this is the only allowed narrowing.
Two questions needed answers before the harness work starts: where the
publisher marks a feature optional, and how a skipped feature appears in
the conformance statement.

RFC-0004's placement rule says every fact has one home, and a role's
semantics live on the role interface declaration. "A conforming
implementation may lack this capability" is a fact about the role.

## Decision

`//testkit:optional` goes on the role interface, next to the contract
claim it applies to. It applies to roles and mixin claims, never to
individual checks. Embedders inherit it like any other role fact.

The conformance statement reports skipped features as a list
(`"unsupported": ["shred"]`), not as named tiers. Anyone who wants tiers
("core", "full") can define them on top of the list later.

## Alternatives Considered

**A separate file listing optional features.** That is a second home for a
role fact, which the placement rule exists to prevent. Rejected.

**Optionality per check.** That is `Without` under another name, at a
resolution the publisher's vocabulary does not have. A publisher who wants
half a role optional should split the role. Rejected.

**Tiers in the statement.** A tier fixes a ranking of features today, for
all features added later. The list carries the same information and stays
additive. Rejected.

## Consequences

**Positive:**

- The harness work starts with this settled and cannot reopen it per
  publisher.
- Embedders inherit optionality automatically.

**Negative:**

- Role-level granularity is coarse. Marking half a role optional means
  splitting the role, which changes the publisher's interface.
- Inheritance can give an embedder an optional feature it considers
  mandatory. There is no override until a real case shows what the
  override should look like.

**Neutral:**

- The statement's format and versioning are
  [ADR-0025](0025-machine-formats-are-versioned-structs.md)'s. This
  decision only fixes what the `unsupported` field means.
