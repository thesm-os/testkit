// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

import (
	"embed"

	"go.thesmos.sh/eidos/sdk"
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

// Module is testkit's module path.
//
// Generated code references the runtime by import path, and a path spelled in
// each template would have to be corrected in every one of them the day the
// module moves. Held here once and carried onto the emit value instead, so a
// template names a package rather than a path.
const Module = "go.thesmos.sh/testkit"

// RuntimePaths is the set of testkit import paths a generated file references,
// embedded in the checks' emit value so the template can reach them.
//
// The backend's `external` builtin turns a path and a symbol into a qualified
// reference and registers the import on the rendered file, so a path is all a
// template needs — no plugin-registered helper stands between the two. The
// field carries the `Runtime` prefix rather than sitting behind a nested value
// because a template writes it constantly: promoted, `external $.Runtime
// "Equal"` reads as one thought.
//
// Only the checks carry it. The API imports nothing of testkit's — its
// sentinel is `errors.New` and its numeric fallback `fmt.Sprintf` — and giving
// it a runtime path it never renders would invite one.
type RuntimePaths struct {
	// Runtime is the module root, where the assertion helpers the generated
	// checks call live.
	Runtime string
}

// GoRuntime returns the import paths the Go templates reference.
func GoRuntime() RuntimePaths { return RuntimePaths{Runtime: Module} }

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
