// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"embed"

	"go.thesmos.sh/eidos/sdk"
)

// GoPrimarySuffix is the per-source-basename trailer for the primary output.
// Structs declared in `domain.go` produce `domain_builder.gen.go`.
const GoPrimarySuffix = "_builder.gen.go"

// GoTestSuffix is the trailer for the tagged test output. The `_test.go`
// ending triggers the framework's automatic `<pkg>_test` package shift, so the
// generated checks land outside the package and drive the builder the way a
// consumer does.
const GoTestSuffix = "_builder.gen_test.go"

// GoTestOutputTag is the tag the companion output advertises.
const GoTestOutputTag = "test"

// Module is testkit's module path.
//
// Generated code references the runtime by import path, and a path spelled in
// each template would have to be corrected in every one of them the day the
// module moves. Held here once and carried onto the emit value instead, so a
// template names a package rather than a path.
const Module = "go.thesmos.sh/testkit"

// RuntimePaths is the set of testkit import paths a generated file references,
// embedded in [Tests] so the template can reach them.
//
// The backend's `external` builtin turns a path and a symbol into a qualified
// reference and registers the import on the rendered file, so a path is all a
// template needs — no plugin-registered helper stands between the two. The
// field carries the `Runtime` prefix rather than sitting behind a nested value
// because a template writes it constantly: promoted, `external $.Runtime
// "Equal"` reads as one thought.
//
// Only the companion needs one. The builder itself references nothing outside
// the package it constructs, which is why [Builder] carries no runtime paths
// and its template writes no `external` call.
type RuntimePaths struct {
	// Runtime is the module root, where the assertion helpers the generated
	// checks call live.
	Runtime string
}

// GoRuntime returns the import paths the Go templates reference.
func GoRuntime() RuntimePaths { return RuntimePaths{Runtime: Module} }

//go:embed templates/golang/*.tmpl
var goTemplates embed.FS

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
