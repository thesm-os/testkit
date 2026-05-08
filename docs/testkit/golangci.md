# Linter Configuration for testkit Consumers

testkit-generated code triggers several linter warnings that are false positives in context. Copy the relevant sections into your `.golangci.yml` to suppress them.

## Recommended `.golangci.yml` additions

```yaml
linters-settings:
  # Generated stub constructors use lambda-wrapped factories that
  # unlambda flags as unnecessary. They're necessary — the lambda
  # captures test-scoped state.
  gocritic:
    disabled-checks:
      - unlambda

  # Generated test specs declare config structs optimized for
  # readability, not field alignment. Suppress fieldalignment
  # on test packages.
  govet:
    disable:
      - fieldalignment

issues:
  exclude-rules:
    # Generated files — suppress all lint warnings.
    - path: '\.gen\.go$'
      linters:
        - gocritic
        - govet
        - revive
        - stylecheck
        - errcheck

    # Generated test files.
    - path: '\.gen_test\.go$'
      linters:
        - gocritic
        - govet
        - revive
        - stylecheck
        - errcheck

    # Test packages may import testkit/model which re-exports rapid.
    # depguard rules that restrict test dependencies should allowlist
    # the testkit module.
    - path: '_test\.go$'
      linters:
        - depguard
      text: 'go.thesmos.sh/testkit'

    # Stub companion files use type assertions for DelegateTo wiring
    # that errcheck flags. The error handling is intentional — stubs
    # panic on misconfiguration rather than returning errors.
    - path: '_stub\.go$'
      linters:
        - errcheck
```

## depguard allowlist

If your project uses `depguard` to restrict imports, add testkit to the allowlist:

```yaml
linters-settings:
  depguard:
    rules:
      main:
        allow:
          - $gostd
          - your.module/...
          - go.thesmos.sh/testkit
```

The `model` package re-exports `pgregory.net/rapid` types so consumers never import rapid directly. If depguard blocks transitive dependencies, allowlist `go.thesmos.sh/testkit` — that's sufficient.

## Per-package granularity

If you prefer targeted suppression over blanket exclusions:

```yaml
issues:
  exclude-rules:
    # Only suppress in *test/ packages that contain generated code.
    - path: 'test/.*\.gen\.go$'
      linters: [gocritic, govet, revive, stylecheck, errcheck]
```

## What each suppression addresses

| Linter | Trigger | Why it's a false positive |
|--------|---------|--------------------------|
| `unlambda` | `func() Store { return NewStore() }` passed to factory params | Lambda captures test-scoped closures; direct function reference loses context |
| `fieldalignment` | Config structs in generated specs | Struct layout optimized for readability and grouping, not memory |
| `errcheck` | Discarded returns in smoke tests, stub calls | Smoke tests intentionally discard results; stubs panic on error |
| `depguard` | `go.thesmos.sh/testkit` import in test files | testkit is test infrastructure; model re-exports rapid to keep it out of consumer go.mod |
| `revive` | Various style warnings on generated code | Generated code follows template conventions, not hand-written style |
