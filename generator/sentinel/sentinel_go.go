// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel

import (
	"embed"

	"go.thesmos.sh/eidos/sdk"
)

// GoSuffix is appended to the anchor declaration's source basename to form the
// output filename.
//
// The `.gen_test.go` ending does two things: `.gen` marks the file generated
// for tooling that skips such files, and `_test.go` triggers the framework's
// external-test-package shift so the checks drive the package the way a
// consumer does rather than reaching inside it.
const GoSuffix = ".gen_test.go"

// Module is testkit's module path.
//
// Generated code references the runtime by import path, and a path spelled in
// each template would have to be corrected in every one of them the day the
// module moves. Held here once and carried onto the emit value instead, so a
// template names a package rather than a path.
const Module = "go.thesmos.sh/testkit"

// RuntimePaths is the set of testkit import paths a generated file references,
// embedded in the emit value so the template can reach them.
//
// The backend's `external` builtin turns a path and a symbol into a qualified
// reference and registers the import on the rendered file, so a path is all a
// template needs — no plugin-registered helper stands between the two. The
// field carries the `Runtime` prefix rather than sitting behind a nested value
// because a template writes it constantly: promoted, `external $.Runtime
// "Equal"` reads as one thought.
type RuntimePaths struct {
	// Runtime is the module root, where the assertion helpers the generated
	// checks call live.
	Runtime string
}

// GoRuntime returns the import paths the Go templates reference.
func GoRuntime() RuntimePaths { return RuntimePaths{Runtime: Module} }

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
