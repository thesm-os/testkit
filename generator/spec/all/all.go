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
// pattern (A: markers, B: simple-value, C: resolved-symbol) is
// documentation only; see spec/doc.go for the taxonomy.
import (
	_ "go.thesmos.sh/testkit/generator/spec/allocs"
	_ "go.thesmos.sh/testkit/generator/spec/atomic"
	_ "go.thesmos.sh/testkit/generator/spec/bounded"
	_ "go.thesmos.sh/testkit/generator/spec/cacheable"
	_ "go.thesmos.sh/testkit/generator/spec/concurrent"
	_ "go.thesmos.sh/testkit/generator/spec/concurrentreaders"
	_ "go.thesmos.sh/testkit/generator/spec/crdtmerge"
	_ "go.thesmos.sh/testkit/generator/spec/deleteremoves"
	_ "go.thesmos.sh/testkit/generator/spec/deprecated"
	_ "go.thesmos.sh/testkit/generator/spec/errors"
	_ "go.thesmos.sh/testkit/generator/spec/eventually"
	_ "go.thesmos.sh/testkit/generator/spec/hooks"
	_ "go.thesmos.sh/testkit/generator/spec/idempotent"
	_ "go.thesmos.sh/testkit/generator/spec/integrationonly"
	_ "go.thesmos.sh/testkit/generator/spec/latency"
	_ "go.thesmos.sh/testkit/generator/spec/lease"
	_ "go.thesmos.sh/testkit/generator/spec/lifecycleafterclose"
	_ "go.thesmos.sh/testkit/generator/spec/monotonic"
	_ "go.thesmos.sh/testkit/generator/spec/nilsafe"
	_ "go.thesmos.sh/testkit/generator/spec/orderafter"
	_ "go.thesmos.sh/testkit/generator/spec/pagination"
	_ "go.thesmos.sh/testkit/generator/spec/partition"
	_ "go.thesmos.sh/testkit/generator/spec/pure"
	_ "go.thesmos.sh/testkit/generator/spec/readafterwrite"
	_ "go.thesmos.sh/testkit/generator/spec/retrysucceeds"
	_ "go.thesmos.sh/testkit/generator/spec/sample"
	_ "go.thesmos.sh/testkit/generator/spec/scope"
	_ "go.thesmos.sh/testkit/generator/spec/sideeffect"
	_ "go.thesmos.sh/testkit/generator/spec/streamreflectsmutations"
	_ "go.thesmos.sh/testkit/generator/spec/timeout"
	_ "go.thesmos.sh/testkit/generator/spec/validates"
	_ "go.thesmos.sh/testkit/generator/spec/wrappedvia"
)
