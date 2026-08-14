# Architecture Decision Records

One decision per record, with the alternatives that were rejected and the
trade-offs accepted. Never edited in place once Accepted — a changed decision
gets a new ADR that supersedes the old one.

For the shape these decisions add up to, read
[RFC-0001](../rfc/0001-testkit-as-a-generator-platform.md).

| # | Title | Status |
|---|---|---|
| [0001](0001-record-decisions-as-adrs.md) | Record decisions as ADRs | Accepted |
| [0002](0002-support-external-consumers-under-semver.md) | Support external consumers under semver | Accepted |
| [0003](0003-adopt-eidos-as-the-codegen-substrate.md) | Adopt eidos as the codegen substrate | Accepted |
| [0004](0004-consume-only-the-annotator-plugin.md) | Consume only eidos's annotator plugin | Accepted |
| [0005](0005-split-into-published-modules.md) | Split into published modules behind a go.work | Accepted · module table superseded by [0014](0014-split-the-cli-from-the-generator-module.md) |
| [0006](0006-tag-published-modules-in-lockstep.md) | Tag published modules in lockstep | Accepted |
| [0007](0007-earn-top-level-packages-by-import.md) | Earn top-level packages by import | Accepted |
| [0008](0008-neutral-directive-form-with-axis-qualifier.md) | Neutral directive form with an axis qualifier | Superseded by [0016](0016-directives-are-positive-only.md) |
| [0009](0009-one-config-filename.md) | One config filename | Accepted |
| [0010](0010-first-stable-release-is-v1.md) | The first stable release is v1.0.0 | Accepted |
| [0011](0011-collapse-ref-packages.md) | Collapse the reference-implementation packages | Accepted |
| [0012](0012-generate-per-shape-helpers-into-the-consumer.md) | Generate per-shape helpers into the consumer | Accepted |
| [0013](0013-defer-codec-pkgdoc-and-smoke.md) | Defer codec, pkgdoc, and smoke | Accepted |
| [0014](0014-split-the-cli-from-the-generator-module.md) | Split the CLI from the generator module | Accepted |
| [0015](0015-subtest-names-carry-the-classification.md) | Subtest names carry the classification | Accepted |
| [0016](0016-directives-are-positive-only.md) | Directives are positive-only | Accepted |
| [0017](0017-every-classification-owes-an-assertion.md) | Every classification owes an assertion | Superseded by [0018](0018-one-tier-owns-each-classification.md) |
| [0018](0018-one-tier-owns-each-classification.md) | One tier owns each classification | Accepted |
| [0019](0019-suite-vocabulary-in-the-root-module.md) | The suite package lives in the root module | Accepted |
| [0020](0020-check-ids-carry-a-case-split-grammar.md) | Check IDs have a defined grammar | Accepted |
| [0021](0021-field-roles-resolve-generator-side.md) | Field roles resolve in the generator | Accepted |
| [0022](0022-canonical-draws-pin-seed-and-version.md) | Canonical draws record their seed and rapid version | Accepted |
| [0023](0023-a-lost-manifest-line-fails-check.md) | Removing a check from the lock file fails testkit check | Accepted |
| [0024](0024-the-veneer-is-slot-composed.md) | The veneer is composed through named slots | Accepted |
| [0025](0025-machine-formats-are-versioned-structs.md) | Machine-read formats are versioned structs | Accepted |
| [0026](0026-role-keywords-enter-by-registry.md) | Role keywords are added by registry row, kinds by RFC | Accepted |
| [0027](0027-optionality-lives-on-the-role.md) | Optional features are declared on the role | Accepted |
