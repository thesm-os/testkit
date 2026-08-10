// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault

import (
	"embed"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/stub"
)

// GoTestOutputTag is the tag the companion output advertises. It matches the
// stub generator's for the same reason the suffixes do — routing overrides
// select on the tag, and a contribution answering to a different one would
// travel separately from the double it belongs to.
const GoTestOutputTag = stub.GoTestOutputTag

// RuntimePaths is the set of testkit import paths a generated file references,
// embedded in both emit values so the templates can reach them.
//
// The paths are [stub.Module]'s rather than a second copy: this plugin renders
// into the stub generator's file, so the two name one runtime and a divergence
// would be invisible — both spellings resolve, and the file would simply import
// two packages where it means one.
//
// The backend's `external` builtin turns a path and a symbol into a qualified
// reference and registers the import, so a path is all a template needs. The
// fields carry the `Runtime` prefix rather than sitting behind a nested value
// because a template writes them constantly: promoted, `external $.RuntimeFault
// "And"` reads as one thought.
type RuntimePaths struct {
	// Runtime is the module root, where the assertion helpers the generated
	// checks call live.
	Runtime string

	// RuntimeFault is the fault package, where the injectors the generated
	// helpers construct live.
	RuntimeFault string
}

// GoRuntime returns the import paths the Go templates reference.
func GoRuntime() RuntimePaths {
	return RuntimePaths{Runtime: stub.Module, RuntimeFault: stub.Module + "/fault"}
}

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
