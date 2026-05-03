# Suite

> **Status:** not yet implemented. Targeted for a subsequent dev cycle. The behavior described below is the design intent — code, flags, and output may differ once shipped.

Tier 1 conformance. Reads an interface and its `//testkit:` directives, emits an `AssertContract(t, factory, opts...)` function that runs one subtest per (method × directive). Single-call assertions only — multi-step state-machine work belongs in [`model`](model.md).

## Planned directive

```go
//go:generate testkit suite -o storetest/store_spec.gen.go Store
```

## Planned default output

`<package>test/<subject>_spec.gen.go`.

## Planned shape

The generated function will accept a factory closure plus domain-input options (e.g. `UnknownID`, `KnownID`, `SampleItem`, `Setup`). Subtests emit only for directives that are actually present on each method; methods without applicable directives generate only happy-path coverage. Missing options skip the affected subtest with a diagnostic message rather than fatal — consumers gradually fill in the option set as they expand contract coverage.

## See also

- [Configuration](../configuration.md) for the directive vocabulary suite consumes
- [Primitives / directive-assertions](../primitives/directive-assertions.md) for the runtime helpers suite calls
