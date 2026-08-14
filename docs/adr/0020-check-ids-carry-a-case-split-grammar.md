---
adr: 0020
title: Check IDs have a defined grammar
status: Accepted
date: 2026-08-14
supersedes: none
superseded-by: none
---

# ADR-0020: Check IDs have a defined grammar

## Status

Accepted

## Context

Five things parse check IDs: the `checks.lock` file, the generated check
index, `Without` calls, the conformance statement, and `go test -run`
patterns. The RFC-0004 review found the syntax was never defined, and that
a method scope (`Append/smoke`) and a family scope
(`chain/AUTO-STREAM-CHAIN-LINKS`) shared the first segment with nothing
preventing a method named `chain` from colliding with the family.

## Decision

IDs follow this grammar: `id = scope-segment 1*( "/" sub-segment )`.

The first segment is either an exported Go method name (starts with an
uppercase letter) or a reserved lowercase family word (`chain`, `model`,
`poison`, `cross-role`, `sim`, `x`). Go method names always start with an
uppercase letter, so the two can never collide.

Rules:

- Every ID has at least two segments. A bare scope is a group, not a check.
- New family words require an RFC.
- IDs are unique within one generated package.
- The generator validates the IDs it emits; `Run` validates hand-written
  ones.

## Alternatives Considered

**A struct ID type.** IDs end up as strings in lock files, diffs, and
`-run` patterns anyway. A struct would still need a string form and would
double the API. Rejected.

**Free strings with runtime checks only.** That leaves five parsers
reading an undefined syntax. Rejected; this was the review's finding.

**Hash-based IDs.** Stable under renames, but unreadable in diffs and
`-run` patterns. The lock file exists to be read by people. Rejected.

## Consequences

**Positive:**

- Collisions are impossible because of Go's export rule, not because
  someone maintains a list.
- Any future tool gets the syntax from the package documentation.

**Negative:**

- Renaming a method renames its IDs. Lock files and typed drops break.
  That is intentional (breakage should be visible) but it is work.
- New family words are slow to add because they require an RFC.

**Neutral:**

- Hand-written checks go under their method's scope or under `x/`.
