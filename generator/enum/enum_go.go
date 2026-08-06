// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

import (
	"embed"
	"io/fs"
	"text/template"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/gotmpl"
)

// GoPrimarySuffix and GoTestSuffix are appended to the source basename.
//
// The API cannot be routed elsewhere: it declares methods on the enum's type,
// and Go permits that only in the type's own package. The checks travel with
// it and take the external test package the `_test.go` ending gives them.
const (
	GoPrimarySuffix = ".enum_gen.go"
	GoTestSuffix    = ".enum_test.go"
)

// GoTestOutputTag is the tag the checks' output advertises.
const GoTestOutputTag = "test"

//go:embed templates/golang/*.tmpl
var goTemplatesFS embed.FS

// GoOutputs returns the Go adapter's output set. The empty tag is the primary
// and must come first.
func GoOutputs() []sdk.Output {
	return []sdk.Output{
		{Suffix: GoPrimarySuffix},
		{Tag: GoTestOutputTag, Suffix: GoTestSuffix},
	}
}

// GoTemplates returns the embedded Go template tree.
func GoTemplates() (fs.FS, bool) {
	sub, err := fs.Sub(goTemplatesFS, "templates/golang")
	if err != nil {
		return nil, false
	}
	return sub, true
}

// GoFuncMap returns the helpers the Go templates call, registered under this
// plugin's own prefix — funcmap entries merge across every plugin in a run and
// a duplicate name fails the build.
func GoFuncMap() template.FuncMap { return gotmpl.FuncMap(Name) }
