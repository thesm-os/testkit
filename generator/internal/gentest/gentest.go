// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package gentest is the grammar every plugin's tests write their
// fixtures in, and the idioms they read the results back through.
//
// The grammar, deliberately, and not a fixture library. Seven plugins
// build store fixtures and the SHAPES are not shared: `Store` means a
// keyed writer to one plugin's tests and a type-parameterised one to
// another's, each minimal for the question its test asks. A shared
// `Store()` would have to be the union of seven plugins' needs, which
// is the catch-all this package exists to avoid, or carry so many
// options that composing it costs more than the four lines it saves.
//
// What IS the same everywhere is the scaffolding around them: the
// position every declaration needs, the context parameter two-thirds of
// methods take, and the drive-the-plugin-and-read-the-diagnostics
// idiom, which existed four times over with the plugin swapped. Those
// are here. A shape stays in the test that asks about it, where a
// reader can see it.
//
// Internal because nothing outside this module needs it, and a
// published test helper is one that owes semver.
package gentest

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sdk"
)

// SourceFile is where a fixture's declarations claim to live.
//
// One path for every fixture that does not care, because several
// generators compose the OUTPUT filename from the source basename — a
// fixture with no position renders to a name derived from nothing, and
// one with a different position per test makes two goldens differ for a
// reason that is not the code under test.
const SourceFile = "kv/iface.go"

// At is the position a declaration needs, at the file every fixture
// shares. Line and column are fixed: nothing reads them, and a fixture
// that varied them would be varying something no assertion is about.
func At() sdk.Pos { return sdk.At(SourceFile, 1, 1) }

// AtFile is [At] for a fixture that genuinely needs its own file —
// a cross-package case, or two interfaces a generator must not merge.
func AtFile(path string) sdk.Pos { return sdk.At(path, 1, 1) }

// Ctx is the context parameter, which two-thirds of fixture methods
// take and none of them spell differently.
func Ctx(m *storefixture.MethodBuilder) {
	m.Param("ctx", storefixture.PkgNamed("context", "Context"))
}

// Err is the error result, the same way.
func Err(m *storefixture.MethodBuilder) { m.Return(storefixture.Named("error")) }

// Diagnostics drives the generator over the store and hands back what
// it reported.
//
// The idiom four plugins had written out, differing only in which
// constructor they named. Taking the plugin rather than being written
// once per package is what stops the fifth copy: a test that wants a
// different plugin passes a different plugin.
func Diagnostics(tb testing.TB, g plugin.Generator, s *sdk.Store) []diag.Diag {
	tb.Helper()
	return plugintest.Generate(tb, g, s).Diagnostics()
}

// About narrows diagnostics to those naming the subject, so an
// assertion about one method's gap is not satisfied by another's.
func About(diags []diag.Diag, subject string) []diag.Diag {
	out := make([]diag.Diag, 0, len(diags))
	for _, d := range diags {
		if strings.Contains(d.Message, subject) {
			out = append(out, d)
		}
	}
	return out
}
