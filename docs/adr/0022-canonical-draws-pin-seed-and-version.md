---
adr: 0022
title: Canonical draws record their seed and rapid version
status: Accepted
date: 2026-08-14
supersedes: none
superseded-by: none
---

# ADR-0022: Canonical draws record their seed and rapid version

## Status

Accepted

## Context

Under RFC-0004, fixture values come from rapid generators. Deterministic
checks use the canonical draw: the first sample at a fixed seed, resolved
when the code is generated. This means the generated values depend on
rapid's internals. A rapid upgrade can change what every generator
produces, which would silently rewrite every fixture value in every
generated file.

## Decision

Every generated file's provenance header records the seed and the rapid
version used for its canonical draws. `testkit check` compares them.

Drift is reported in two categories:

- **Value drift under a version bump**: one summary line per package,
  naming the old and new versions.
- **Check-set drift**: reported line by line.

A CI job generates the corpus's canonical values on two platforms from the
same seed and compares digests. The claim "reproducible across machines"
only ships while that job passes.

## Alternatives Considered

**Keep fixed literals, as today.** No rapid coupling, but this is the
system the audit showed failing: derived literals broke the causal-chain
fixture, generics got zero-value structs, and the suite and model tiers
drew from different pools. Rejected.

**Record nothing and trust rapid to be stable.** Rapid does not guarantee
stream stability across versions. An unrecorded dependency that rewrites
every fixture on upgrade is exactly the kind of silent change the lock
file exists to prevent. Rejected.

**Treat all drift the same.** A version bump regenerates thousands of
fixture lines. A reviewer cannot find a real check change inside that, so
they approve without reading. Splitting the two categories keeps check
changes readable. Rejected.

## Consequences

**Positive:**

- Reproducibility is tested, not assumed.
- A rapid bump has a known procedure: bump, regenerate, read one summary
  line per package, commit locks and headers together. A line-level lock
  change inside a bump commit means someone changed a check at the same
  time, and it shows.

**Negative:**

- A rapid upgrade is now an operator procedure, not a routine dependency
  update.
- The drift checker gains a classifier, which is more code in a tool every
  consumer runs.

**Neutral:**

- Model-tier checks are unaffected. They use the full distribution, and
  rapid already reports seeds for reproducing their failures.
