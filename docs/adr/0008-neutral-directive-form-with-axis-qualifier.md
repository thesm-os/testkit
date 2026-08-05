---
adr: 0008
title: Neutral directive form with an axis qualifier
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0008: Neutral directive form with an axis qualifier

## Status

Accepted

## Context

Directives are how a consumer tells testkit what a type or method guarantees.
They live in the consumer's own source, they are read constantly, and they are
the surface [ADR-0002](0002-support-external-consumers-under-semver.md) promises
to keep stable from `v1.0.0`. The form is worth getting right before then.

eidos's directive parser takes a configurable prefix and supports three forms
per directive: neutral (`//testkit:x`), affirmative (`//+testkit:x`), and
negative (`//-testkit:x`). The negative form suppresses. testkit sets the
namespace; the form is a choice.

The obvious reading is one directive per property:

```go
//+testkit:idempotent
//+testkit:concurrent-safe
//+testkit:bounded
```

Three lines of comment for three properties, on every method that has them. A
method with six properties has six lines above it, and the declaration it
describes is pushed off the screen.

Property names also collide across axes. A name alone does not say whether it is
a shape mixin, a contract, or something else, so the flat namespace has to keep
every name globally distinct forever.

## Decision

Directives take the neutral form with an axis qualifier, and batch several
values per line:

```go
//testkit:mixin idempotent concurrent-safe bounded
//testkit:contract lease
```

The qualifier names the axis; the values are positional. The `-` form stays
available for suppression, which is the case where negation is the whole point:

```go
//-testkit:mixin concurrent-safe
```

The affirmative `+` form is not used. With the neutral form as the default, `+`
would be a synonym, and two spellings of the same thing is ambiguity for no
gain.

This requires eidos's parser to accept extra positional arguments on a directive.
That change is upstream.

## Alternatives Considered

**One directive per property, affirmative form.** Rejected: verbosity at the
call site, which is where directives are read most. Six properties should not
cost six lines.

**Neutral form, one property per line, no axis qualifier.** Rejected: it keeps
the flat namespace, so every mixin name has to stay distinct from every contract
name indefinitely, and a reader cannot tell which axis a name belongs to without
consulting the reference.

**Structured values — `//testkit:mixin(idempotent, bounded)`.** Rejected: it is
not the Go directive idiom. `//go:build`, `//go:generate`, and `//go:embed` all
take positional arguments after a space, and a consumer already reads those.

**Keep the `+` form as the canonical spelling.** Rejected: the neutral form is
shorter and the affirmative marker carries no information when the default is
affirmative. `-` is meaningful precisely because it contrasts with a bare form.

## Consequences

**Positive:**

- A method with six properties carries one or two comment lines, not six.
- The axis is explicit, so mixin and contract namespaces are independent and
  each can grow without cross-checking the other.
- Suppression reads as the exception it is.

**Negative:**

- It depends on an upstream eidos change to accept extra positional arguments.
  Until that lands, this form does not parse.
- Positional values mean a typo is a value, not a syntax error. `//testkit:mixin
  idempotant` is well-formed and silently wrong unless the annotator validates
  against a known set.
- Batching hides diffs: adding a property to a line changes that line, so
  `git blame` attributes all properties on it to the most recent edit.

**Neutral:**

- The `testkit` namespace is unchanged, so existing directive names carry over
  under a qualifier rather than being renamed.
