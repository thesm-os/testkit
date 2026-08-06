// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"embed"
	"io/fs"
	"text/template"

	refconv "go.thesmos.sh/eidos/lang/golang"
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

// GoFuncMap returns the shared Go-convention helpers.
//
// The plugin contributes no entries of its own: everything the
// templates need is either canonical (`renderType`, `renderExpr`,
// `external`) or comes from the shared bundle, which is the point of
// having a per-language package rather than each plugin restating
// the same conversions.
func GoFuncMap() template.FuncMap { return refconv.FuncMap() }
