// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault

import (
	"embed"
	"io/fs"
	"text/template"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/gotmpl"
	"go.thesmos.sh/testkit/generator/stub"
)

// GoTestOutputTag is the tag the companion output advertises. It matches the
// stub generator's for the same reason the suffixes do — routing overrides
// select on the tag, and a contribution answering to a different one would
// travel separately from the double it belongs to.
const GoTestOutputTag = stub.GoTestOutputTag

// langGo is the backend language identifier the per-language adapters key on.
const langGo = "golang"

//go:embed templates/golang/*.tmpl
var goTemplates embed.FS

// GoOutputs returns the same output set the stub generator declares.
//
// This is not duplication to be tidied away: Layout composes a contributor's
// Target from the contributing plugin's own suffix table, so declaring the
// host's suffixes is the mechanism by which two plugins converge on one file.
// A suffix that drifted from the stub generator's would silently produce a
// second file rather than an error.
func GoOutputs() []sdk.Output {
	return []sdk.Output{
		{Tag: "", Suffix: stub.GoPrimarySuffix},
		{Tag: GoTestOutputTag, Suffix: stub.GoTestSuffix},
	}
}

// GoTemplates returns the embedded Go template tree. The error is discarded
// rather than branched on: fs.Sub fails only for a malformed path, and this
// one is a compile-time constant the //go:embed directive already validated.
func GoTemplates() (fs.FS, bool) {
	sub, _ := fs.Sub(goTemplates, "templates/golang")
	return sub, true
}

// GoFuncMap returns the shared list helpers under this plugin's prefix.
//
// Prefixed, and therefore spelled `faultArgs` where the stub generator's
// templates say `stubArgs`, for the same one function. The backend binds a
// plugin's funcmap at parse time from that plugin's own entries and rejects
// two plugins registering one name, so a shared bundle has to be published
// once per plugin. The sharing that matters happens in Go, in
// [gotmpl] — the namespace is the part that cannot be shared.
func GoFuncMap() template.FuncMap { return gotmpl.FuncMap(Name) }

// Outputs dispatches to the per-language adapter.
func (*Plugin) Outputs(lang string) []sdk.Output {
	if lang == langGo {
		return GoOutputs()
	}
	return nil
}

// Templates dispatches to the per-language adapter's template tree.
func (*Plugin) Templates(lang string) (fs.FS, bool) {
	if lang == langGo {
		return GoTemplates()
	}
	return nil, false
}

// TemplateFuncs dispatches to the per-language adapter's funcmap.
func (*Plugin) TemplateFuncs(lang string) template.FuncMap {
	if lang == langGo {
		return GoFuncMap()
	}
	return nil
}

// TemplateOverrides returns nil — the plugin replaces no canonical funcmap
// entry.
func (*Plugin) TemplateOverrides(string) template.FuncMap { return nil }
