// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"embed"

	"go.thesmos.sh/eidos/sdk"
)

// GoPrimarySuffix is the per-source-basename trailer for the primary
// output. Interfaces declared in `store.go` produce `store_stub.gen.go`.
const GoPrimarySuffix = "_stub.gen.go"

// GoTestSuffix is the trailer for the tagged test output. The
// `_test.go` ending triggers the framework's automatic `<pkg>_test`
// package shift, so the generated test lands in an external test
// package and cannot reach package-private state.
const GoTestSuffix = "_stub.gen_test.go"

// GoTestOutputTag is the tag the companion output advertises.
// Source-side `//testkit:out tag=test …` directives, project config under
// the plugin's `tags:` block, and CLI `-o stub:test=…` overrides
// all match against this value.
const GoTestOutputTag = "test"

// Module is testkit's module path.
//
// Generated code references the runtime by import path, and a path spelled in
// each template would have to be corrected in every one of them the day the
// module moves. Held here once and carried onto the emit values instead, so a
// template names a package rather than a path.
const Module = "go.thesmos.sh/testkit"

// RuntimePaths is the set of testkit import paths a generated file
// references, embedded in both emit values so the templates can reach them.
//
// The backend's `external` builtin turns a path and a symbol into a qualified
// reference and registers the import on the rendered file, so a path is all a
// template needs — no plugin-registered helper stands between the two. The
// fields carry the `Runtime` prefix rather than sitting behind a nested value
// because a template writes them constantly: promoted, `external
// $.RuntimeStub "Answer"` reads as one thought.
type RuntimePaths struct {
	// Runtime is the module root, where the assertion helpers the companion
	// calls live.
	Runtime string

	// RuntimeStub is the double's own runtime: MethodStub, Answer, the
	// order tracker, and the behaviour suites the companion drives.
	RuntimeStub string

	// RuntimeClock is the virtual clock latency and time-windowed faults run
	// against.
	RuntimeClock string

	// RuntimeRand is the seeded source that makes probabilistic fault
	// injection replayable.
	RuntimeRand string
}

// GoRuntime returns the import paths the Go templates reference.
func GoRuntime() RuntimePaths {
	return RuntimePaths{
		Runtime:      Module,
		RuntimeStub:  Module + "/stub",
		RuntimeClock: Module + "/clock",
		RuntimeRand:  Module + "/rand",
	}
}

//go:embed templates/golang/*.tmpl
var goTemplates embed.FS

// GoOutputs returns the Go adapter's output set: the primary
// `<basename>_stub.gen.go` plus the `test`-tagged companion.
//
// The untagged entry comes first because the framework reserves the
// empty tag for a plugin's primary output and requires it at index 0
// when present.
func GoOutputs() []sdk.Output {
	return []sdk.Output{
		{Tag: "", Suffix: GoPrimarySuffix},
		{Tag: GoTestOutputTag, Suffix: GoTestSuffix},
	}
}
