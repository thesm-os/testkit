// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"embed"
	"io/fs"
	"text/template"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/gotmpl"
)

// GoPrimarySuffix is the per-source-basename trailer for the primary output.
// Structs declared in `domain.go` produce `domain_builder.gen.go`.
const GoPrimarySuffix = "_builder.gen.go"

//go:embed templates/golang/*.tmpl
var goTemplates embed.FS

// GoTestSuffix is the trailer for the tagged test output. The `_test.go`
// ending triggers the framework's automatic `<pkg>_test` package shift, so the
// generated checks land outside the package and drive the builder the way a
// consumer does.
const GoTestSuffix = "_builder.gen_test.go"

// GoTestOutputTag is the tag the companion output advertises.
const GoTestOutputTag = "test"

// GoOutputs returns the Go adapter's output set: the primary builder plus its
// checks.
//
// The untagged entry comes first because the framework reserves the empty tag
// for a plugin's primary output and requires it at index 0.
func GoOutputs() []sdk.Output {
	return []sdk.Output{
		{Tag: "", Suffix: GoPrimarySuffix},
		{Tag: GoTestOutputTag, Suffix: GoTestSuffix},
	}
}

// GoTemplates returns the embedded Go template tree. The error is discarded
// rather than branched on: fs.Sub fails only for a malformed path, and this one
// is a compile-time constant the //go:embed directive already validated.
func GoTemplates() (fs.FS, bool) {
	sub, _ := fs.Sub(goTemplates, "templates/golang")
	return sub, true
}

// GoFuncMap returns the shared list helpers under this plugin's prefix.
//
// Prefixed because the backend binds a plugin's funcmap at parse time from that
// plugin's own entries and rejects two plugins registering one name. The
// helpers are shared in Go; only the namespace cannot be.
func GoFuncMap() template.FuncMap { return gotmpl.FuncMap(Name) }
