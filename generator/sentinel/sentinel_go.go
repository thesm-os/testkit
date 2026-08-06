// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel

import (
	"embed"
	"io/fs"
	"text/template"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/gotmpl"
)

// GoSuffix is appended to the anchor declaration's source basename to form the
// output filename.
//
// The `.gen_test.go` ending does two things: `.gen` marks the file generated
// for tooling that skips such files, and `_test.go` triggers the framework's
// external-test-package shift so the checks drive the package the way a
// consumer does rather than reaching inside it.
const GoSuffix = ".gen_test.go"

//go:embed templates/golang/*.tmpl
var goTemplatesFS embed.FS

// GoOutputs returns the Go adapter's output set: one file per annotated
// package.
//
// A single output, unlike the double and the builder. Those generate something
// and then check it; this only checks, so there is no production half to route
// separately.
func GoOutputs() []sdk.Output {
	return []sdk.Output{{Suffix: GoSuffix}}
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
// plugin's own prefix.
//
// The prefix is not decoration. Funcmap entries merge across every plugin in a
// run and a duplicate name fails the whole build, while the binding happens per
// plugin at parse time — so plugins share the Go function and each registers it
// under a name of its own.
func GoFuncMap() template.FuncMap { return gotmpl.FuncMap(Name) }
