# Pkgdoc

> **Status:** not yet implemented. Targeted for a subsequent dev cycle. The behavior described below is the design intent — code, flags, and output may differ once shipped.

Compliance audit-doc generator. Emits a per-package skeleton at `docs/compliance/package-audit/<pkg>.md` with REQ table, refactor-history banner, and evidence section. Auto-fills mechanical parts (REQ rows from `//testkit:req` annotations, evidence rows from CI output); refreshes when source changes; validates REQ IDs against source directives.

## Planned directive

```go
//go:generate testkit pkgdoc -o ../../docs/compliance/package-audit/store.md
```

## Planned modes

- **Generate** (default) — emit the skeleton with auto-fillable sections populated, manual sections marked with `<!-- manual -->`.
- **`--refresh`** — re-scan the package and update auto-fill sections without touching manual prose.

## Planned injection point

Domain analysis (design notes, refactor narrative, exceptions) lives in the manual sections.

## See also

- [Validators / audit-completeness](../validators/audit-completeness.md) — enforces every package has a complete audit doc
- [Validators / req-traceability](../validators/req-traceability.md) — validates REQ IDs round-trip between code and audit
