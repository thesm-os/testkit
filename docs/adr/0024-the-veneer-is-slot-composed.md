---
adr: 0024
title: The veneer is composed through named slots
status: Accepted
date: 2026-08-14
supersedes: none
superseded-by: none
---

# ADR-0024: The veneer is composed through named slots

## Status

Accepted

## Context

Three generator plugins write into one consumer surface: `suite` (the
entry points and deterministic checks), `model` (law and property checks
plus their config fields), and `builder` (request builders).

Today the model plugin adds its checks by appending closures to a private
config struct inside the suite plugin's output. Because the struct is
private, no test can inspect what was added. The audit found two defects
at exactly this spot: `ContractModelConcurrent` ignoring its options, and
the clocked-law door that nothing armed.

The RFC-0004 review asked whether merging the suite and model plugins
would remove the seam entirely.

## Decision

The veneer file is composed through three named slots, each carrying
public data: `checks` (`[]suite.Check[S]` from model), `config-fields`
(fields on the generated `Config`, from model), and `fixtures` (request
builders, from builder). `generator/suite` owns the file. The private
config hook is deleted in the same change that adds the slots.

A gate test compiles each slot's contribution against the `suite` package
alone. If a contribution needs anything private, the build fails.

## Alternatives Considered

**Merge the suite and model plugins.** This would delete the seam.
Rejected for three reasons (RFC-0004, Alternatives): the builder plugin is
a third contributor either way, so slots are needed regardless; generating
a suite without the model tier is a supported configuration, and a merged
plugin would force rapid into the dependency graph of consumers who never
use it; and [ADR-0018](0018-one-tier-owns-each-classification.md)'s gates
check tier ownership along exactly this plugin boundary.

**Keep the private hook and validate it better.** The problem is
structural: private data cannot be inspected by tests. Better validation
of an uninspectable seam does not fix the class. Rejected.

**Each plugin writes its own file, no composition.** The consumer would
get three entry points and three configs, which recreates the
fragmentation this RFC removes. Rejected.

## Consequences

**Positive:**

- The seam is testable. Any future plugin (a sim generator, for example)
  is held to the same rule automatically.
- Tier ownership stays checkable by the existing gates.

**Negative:**

- Slot composition at this granularity has not been done before in this
  codebase. A spike (two toy plugins writing one file) is scheduled before
  the dependent work starts.
- Three slot contracts to version.

**Neutral:**

- `ModelProperty` and the self-proof tests are unchanged. Slots govern how
  plugins contribute, not what they prove.
