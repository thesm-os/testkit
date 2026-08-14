---
adr: 0026
title: Role keywords are added by registry row, kinds by RFC
status: Accepted
date: 2026-08-14
supersedes: none
superseded-by: none
---

# ADR-0026: Role keywords are added by registry row, kinds by RFC

## Status

Accepted

## Context

The role keywords came from one interface, the ledger, which forces
`stream`, `seq`, `prev`, and `payload`. RFC-0004 required checking the set
against three unrelated domains before fixing it. The review did the
exercise:

- A queue maps onto the existing keywords (stream→topic, seq→offset,
  payload→message). Nothing new.
- A cache does not map: its lookups are by key, not by position in a
  sequence. This forces `key`. The corpus agrees — cas, cache, and kv are
  all keyed shapes.
- A workflow engine maps onto the existing keywords plus `key`
  (id→key, step→seq, state→payload). Nothing new.

`fence`, `epoch`, and `checkpoint` each appear in only one domain.

Three kinds of change have different costs. A keyword only enables
inference, which a directive can always override and which always carries
a record of how it was made ([ADR-0021](0021-field-roles-resolve-generator-side.md)).
A new *kind* (beyond drawn/stamped/pinned) changes how fields are handled.
A new ID *family* changes strings that consumers have in lock files
([ADR-0020](0020-check-ids-carry-a-case-split-grammar.md)).

## Decision

The first keyword set is `stream`, `seq`, `prev`, `payload`, `key`.
`fence`, `epoch`, and `checkpoint` are not included.

The rule for adding a keyword: a second unrelated domain needs it, or a
law field cannot bind without it.

Adding a keyword is a table row in `generator/roles`, covered by the same
census tests that cover the other tables. Two changes still require an
RFC: a new kind, and a new ID family.

## Alternatives Considered

**An RFC for every keyword.** Too heavy. A keyword enables overridable,
recorded inference; it does not change any contract. The census tests keep
additions visible without a document per word. Rejected.

**Accept any keyword (open set).** A wrong keyword causes wrong inference,
which makes correct code fail. The set must be closed. Rejected.

**Add `fence`, `epoch`, and `checkpoint` now.** Each has one known use.
Adding a keyword later is a row change; removing a wrong one later is a
breaking change. Rejected by the addition rule itself.

## Consequences

**Positive:**

- The set was checked against three domains instead of copied from one.
- The next addition has a clear rule and a cheap mechanism.

**Negative:**

- The registry can still grow carelessly; the census tests check wiring,
  not judgment. The addition rule is prose.
- The line between "keyword" and "kind" needs judgment. A proposal that
  changes stamping behavior is a kind and needs an RFC, and misfiling it
  would dodge that.

**Neutral:**

- Each keyword's inference rule lives in its registry row. Tightening one
  is a row change.
