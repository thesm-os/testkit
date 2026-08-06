// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"embed"
	"io/fs"
	"text/template"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/gotmpl"
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

// GoTemplates returns the embedded Go template tree. The backend
// reads it once at Build time and registers every `*.tmpl` under it.
// The error is discarded rather than branched on: fs.Sub fails only for a
// malformed path, and this one is a compile-time constant the //go:embed
// directive already validated. A branch for it could not be reached from a
// test — the same call shape eidos's own enum plugin uses.
func GoTemplates() (fs.FS, bool) {
	sub, _ := fs.Sub(goTemplates, "templates/golang")
	return sub, true
}

// GoFuncMap returns the shared list helpers under this plugin's prefix.
//
// Prefixed because the backend rejects two plugins registering the same
// extension name outright: a second testkit generator contributing the same
// unprefixed bundle would fail every run rather than one output. The helpers
// themselves are shared in Go, which is the coupling that survives a rename.
//
// Everything else the templates call is canonical — `renderType`,
// `renderExpr`, `camel` — and comes from the backend.
func GoFuncMap() template.FuncMap { return gotmpl.FuncMap(Name) }
