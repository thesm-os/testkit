// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package all is the umbrella import for every shipped directive
// consumer. Generators that want the full directive vocabulary
// active during enrichment blank-import this package once:
//
//	import _ "go.thesmos.sh/testkit/generator/spec/all"
//
// Each consumer subpackage's init() registers with
// [spec.RegisterConsumer]; the blank import triggers all init()s in
// one line.
//
// Generators that want a narrower set (e.g. a hypothetical
// "stub-lite" that consumes only sample + errors) blank-import the
// individual subpackages instead.
package all

// Imports are alphabetized so goimports stays happy. Sorted by
// pattern (A: markers, C: resolved-symbol) is documentation only;
// see spec/doc.go for the taxonomy.
import (
	_ "go.thesmos.sh/testkit/generator/spec/atomic"
	_ "go.thesmos.sh/testkit/generator/spec/idempotent"
	_ "go.thesmos.sh/testkit/generator/spec/sample"
)
