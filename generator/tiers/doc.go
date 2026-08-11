// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package tiers assigns every classification eidos registers to the half of
// testkit's evidence that covers it, and selects the laws a classified method
// owes.
//
// Three modules read it and none of them owns it. `generator/suite` reads the
// tier, to say in a harness header what that file does not check and where the
// evidence comes from instead. `generator/model` reads the selection and the
// field manifests, to emit the bindings. The conformance gate reads both, to
// hold the table to the two registries it stands between — eidos's
// classifications on one side, [engine/model/law]'s laws on the other.
//
// # Why it is not in `suite`
//
// It was, while `suite` was the only reader. The field manifests are the model
// tier's material and `suite` has no use for them, and the conformance gate
// cannot reach into another generator's package to check them. A shared table
// with three readers is its own package.
//
// # Why it is not in `engine`
//
// `generator` must not depend on `engine`, which would put rapid and porcupine
// in the build-time module's dependency graph for a table of strings — the
// isolation docs/adr/0005 exists to preserve. The bridge is
// [go.thesmos.sh/testkit/core/lawid], in the root module both already depend
// on, so a law identifier is a constant on both sides rather than a literal on
// each.
//
// # Selection
//
// A method's classifications are its detector shape, its mixins, and the roles
// it fills in any contract. [Select] returns every rule those satisfy — plural,
// because one stamp can owe more than one law: `cursor` owes both "Next after
// Close reports the sentinel" and "a second Close is a no-op", and a table
// keyed on the classification could only ever name one of them.
package tiers
