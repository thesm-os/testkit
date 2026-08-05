# Documentation

`go.thesmos.sh/testkit` reads your interfaces and types, generates the test
doubles, builders, fixtures, conformance suites, and benchmarks you would
otherwise write by hand, and generates the tests that prove the generated code
works.

## For users

| I want to | Read |
|---|---|
| Use assertions, stubs, fault injection, clocks | [Primitives](reference/primitives/README.md) |
| Know what a generator emits and where it plugs in | [Generators](reference/generators/README.md) |
| Look up a config key | [Configuration](reference/configuration.md) |
| Lay out my test packages | [Layout](reference/layout.md) |
| Configure my linter to match | [golangci](reference/golangci.md) |
| Adopt testkit in an existing project | [Adoption](how-to/adoption.md) |
| Understand the tier framing | [Conformance tiers](explanation/conformance-tiers.md) |

## For contributors

| I want to know | Read |
|---|---|
| What the platform is and how it is shaped | [RFCs](rfc/README.md) |
| Why a specific thing is the way it is | [Architecture decisions](adr/README.md) |

Documents under `rfc/` and `adr/` are numbered and never renumbered. Rejected,
withdrawn, and superseded records stay on disk — they are the record of why not.
Everything under `reference/`, `how-to/`, and `explanation/` is the opposite: it
is edited freely to stay current, and describes the present rather than the
reasoning that produced it.

## Principles

- **Zero domain knowledge.** testkit knows Go types, not business logic. Domain
  semantics are always injected by the caller.
- **Dependency weight is a consumer choice.** The runtime module costs go-cmp
  and nothing else; the engine and tool modules are opt-in
  ([ADR-0005](adr/0005-split-into-published-modules.md)).
- **Generated plumbing has its own tests.** Every generator emits both the
  artifact and the test file that exercises it.
- **Stubs are the substrate.** One `MethodStub[Call]` primitive composes
  recording, fault injection, latency, virtual clock, strict mode, and
  call-count expectations. Every conformance tier builds on it.
