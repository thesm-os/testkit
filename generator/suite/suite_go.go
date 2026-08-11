// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"embed"

	"go.thesmos.sh/eidos/sdk"
)

// GoPrimarySuffix is the per-source-basename trailer for the harness.
//
// `<source-basename>_suite.gen.go`, matching what stub and builder compose and
// what reference/layout.md documents.
const GoPrimarySuffix = "_suite.gen.go"

// GoTestSuffix is the trailer for the companion output, which drives every
// generated check against a subject that complies and one that violates.
//
// A suffix ending `_test.go` earns the external-test-package shift from Layout,
// which is what makes the companion reach the harness across a package boundary
// the way a consumer does.
const GoTestSuffix = "_suite.gen_test.go"

// GoTestOutputTag names the companion output. Routing overrides and CLI
// selection address an output by tag, so the two files can be routed apart.
const GoTestOutputTag = "test"

// GoIntegrationEnv is the variable a run sets to include integration-only
// checks.
//
// One name for every generated suite, so a consumer sets it once rather than
// per interface. `//testkit:mixin integrationonly` names no variable — the
// classification says the method reaches something outside the process, and
// which switch turns that on is a fact about how a team runs their tests.
//
// Unset is a skip rather than a pass, which is the whole point: a check that
// silently succeeded because its dependency was absent is a check that reports
// coverage it did not earn.
const GoIntegrationEnv = "TESTKIT_INTEGRATION"

// Module is testkit's own import path, which the generated harness references
// for its assertion helpers.
//
// Declared here rather than imported from a sibling generator: this plugin does
// not depend on any of them, and taking a constant from one would make it look
// as though it did.
const Module = "go.thesmos.sh/testkit"

// The template tree, embedded through the recursive directory form rather than
// a `*.tmpl` glob.
//
// The glob reaches one level only, and this tree is nested by axis: one file
// per classification is the only arrangement in which seventy-two of them stay
// readable, and the backend's loader walks the filesystem depth-first, so the
// nesting costs nothing on the reading side.
//
//go:embed templates/golang
var goTemplatesFS embed.FS

// GoOutputs returns the Go output set: the harness and its companion.
//
// The empty tag is the primary and sits at index 0, which the pipeline
// validates at Build.
func GoOutputs() []sdk.Output {
	return []sdk.Output{
		{Tag: "", Suffix: GoPrimarySuffix},
		{Tag: GoTestOutputTag, Suffix: GoTestSuffix},
	}
}
