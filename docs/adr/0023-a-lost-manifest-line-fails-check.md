---
adr: 0023
title: Removing a check from the lock file fails testkit check
status: Accepted
date: 2026-08-14
supersedes: none
superseded-by: none
---

# ADR-0023: Removing a check from the lock file fails testkit check

## Status

Accepted

## Context

The platform audit found that a check which stops being emitted after a
generator upgrade disappears silently: the loss is buried in a generated
diff thousands of lines long that nobody reads. RFC-0004 adds
`checks.lock`: a file with one line per check (ID, class, claim,
tab-separated, sorted), so the set of checks is reviewable on its own.

A list alone changes nothing. There has to be a rule that fires when a
check disappears. The first draft said the lock must change "in the same
commit", but `testkit check` re-runs the pipeline in memory and knows
nothing about commits, so that rule could not be enforced.

## Decision

`testkit check` fails when a regeneration's check set is missing an ID
that is present in the on-disk lock file, and lists the missing IDs. The
failure is suppressed only when the lock file itself changed in the same
run's write set. Additions never fail.

Keeping the lock change in the same commit as the regeneration is review
practice, not something the tool checks.

The file starts with a `testkit-checks v1` header. Claims may not contain
tabs or newlines; the generator rejects them. Any format change gets a new
version header.

## Alternatives Considered

**Enforce "same commit".** The tool cannot see commits. A rule the tool
cannot check is worthless. Rejected.

**No rule; rely on reviewing regeneration diffs.** That is the current
situation, and the audit showed it fails: the check loss drowns in the
generated code. Rejected.

**Store only a digest per package.** Tamper-evident but unreadable. The
point of the file is that a reviewer can read which checks changed.
Rejected.

## Consequences

**Positive:**

- Removing a check requires editing the lock file, which shows up as a
  small readable diff and can be questioned in review.
- Reviewers read the list of assertions, not the generated code.

**Negative:**

- Every intended removal or rename costs a lock edit, and lock files can
  have merge conflicts.
- The tool guarantees the change is visible, not that anyone reads it. A
  consumer can regenerate and update the lock in one run without looking.

**Neutral:**

- Version bumps interact through
  [ADR-0022](0022-canonical-draws-pin-seed-and-version.md): value changes
  are reported separately, so a bump commit has a clean lock diff.
