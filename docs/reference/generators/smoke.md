# Smoke

> **Status:** not yet implemented. Targeted for a subsequent dev cycle. The behavior described below is the design intent — code, flags, and output may differ once shipped.

CLI command coverage generator. Invokes each declared `cobra.Command` (or equivalent) with sampled flag combinations, asserts exit code and stdout/stderr shape per command. Auto-detects the subcommand tree, flag types, and required-flag validation. Captures golden output for stable commands; diffs on regression.

This is distinct from interface-shaped conformance — `smoke` operates on the binary's command surface, not on Go interfaces.

## Planned directive

```go
//go:generate testkit smoke -o smoke.gen_test.go
```

(Run inside `cmd/<binary>` package; the generator scans for `cobra.Command` value declarations or `RootCmd()` factories.)

## Planned default output

`cmd/<binary>/smoke.gen_test.go`.

## Consumed directives

CLI-specific vocabulary: `exit-code`, `golden-output`, `signal`, `flag-validation`, `command-effect`, `command-budget`, `subcommand-order`, `all-or-nothing`, `rerun`, plus skip-on `integration-only` and `deprecated`. See the directive matrix in the top-level README.

## See also

- [`testkit/clitest`](../sub-packages.md#testkitclitest) — the runtime helpers smoke generates against
