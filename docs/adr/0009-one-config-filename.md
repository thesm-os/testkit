---
adr: 0009
title: One config filename
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0009: One config filename

## Status

Accepted

## Context

Project-wide conventions — generated-file suffix, test package style, artifact
directory — live in a config file at the project root. It was documented as
`.testkit.yml` and read as `.testkit.yaml` in places, which is the kind of drift
that only surfaces when someone's config is silently ignored.

That file is also the anchor for artifact resolution. The runtime walks up from
the working directory looking for it, and falls back to `go.mod` when it is
absent. In a multi-module workspace the fallback is wrong: every sub-module has
its own `go.mod`, so artifacts scatter one directory per module instead of
collecting at the project root. Getting the filename wrong therefore does not
fail loudly — it relocates output.

The tempting fix is to accept both names, warn on the old one, and remove it
later. That is the standard compatibility play and it has a standard cost: two
accepted names is permanent ambiguity in every document, every error message,
and every code path that looks the file up, for as long as the window lasts —
and windows are extended more often than they close.

## Decision

One filename: `.testkit.yaml`. No dual-read, no deprecation path, no warning
period.

Existing config files are renamed at the call site.

Two things make this cheap rather than reckless. The file is read by the tool at
build time, not by a running program, so a project that misses the rename gets a
loud failure at `go generate` rather than a subtly relocated artifact months
later. And there are no external consumers yet
([ADR-0002](0002-support-external-consumers-under-semver.md)), so the rename is a
codemod across a known set of repositories.

`.yaml` over `.yml` because the YAML specification names `.yaml` as the
recommended extension and the rest of the repository's config already uses it.

## Alternatives Considered

**Accept both, warn on `.testkit.yml`, drop at the next major.** Rejected: the
warning has to be implemented, tested, documented, and then removed, and the
removal is the step that gets deferred. Two names is ambiguity that outlives the
transition it was meant to smooth.

**Keep `.testkit.yml` and fix the documentation instead.** Rejected: it settles
the drift in the wrong direction. `.yaml` is the recommended extension and
matches everything else in the repository, so keeping `.yml` means the
inconsistency is permanent rather than transitional.

**Support both permanently, no deprecation.** Rejected: it makes the ambiguity
the design. Every lookup path, every error message, and every document has to
name both forever.

## Consequences

**Positive:**

- One name in the documentation, one name in the lookup, one name in error
  messages.
- No warning code to write, test, document, and eventually remove.
- The artifact-directory anchor has exactly one thing to look for, which
  removes a class of silent misplacement.

**Negative:**

- A project that renames late gets a hard failure rather than a warning. That is
  the intent, but it is still an outage for whoever hits it.
- After `v1.0.0` this decision could not be made the same way; a future filename
  change would owe a deprecation cycle under
  [ADR-0002](0002-support-external-consumers-under-semver.md).
- It sets a precedent that "no external consumers yet" justifies hard breaks,
  and that justification expires without an explicit signal.

**Neutral:**

- Quality gates are not in this file. ergon owns them and reads `.ergon.yaml`.
