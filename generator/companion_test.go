// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"go/types"
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestLoadOutputPackage(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when Output is empty", func(t *testing.T) {
		t.Parallel()
		got := generator.LoadOutputPackage(generator.Options{WorkDir: "/tmp"})
		testkit.True(t, got == nil, "no Output → nil")
	})

	t.Run("returns nil when WorkDir is empty", func(t *testing.T) {
		t.Parallel()
		got := generator.LoadOutputPackage(generator.Options{Output: "x/y.gen.go"})
		testkit.True(t, got == nil, "no WorkDir → nil")
	})

	t.Run("returns nil when output dir equals \".\" (same-pkg)", func(t *testing.T) {
		t.Parallel()
		got := generator.LoadOutputPackage(generator.Options{
			Output:  "y.gen.go",
			WorkDir: "/tmp",
		})
		testkit.True(t, got == nil, "same-pkg short-circuits")
	})

	t.Run("returns nil when the directory cannot be loaded", func(t *testing.T) {
		t.Parallel()
		got := generator.LoadOutputPackage(generator.Options{
			Output:  "does-not-exist/y.gen.go",
			WorkDir: "./testdata",
		})
		testkit.True(t, got == nil, "load failure → nil, not error")
	})

	t.Run("loads existing sibling output package", func(t *testing.T) {
		t.Parallel()
		// testdata/defaults/defaultstest exists — load via the
		// (Output, WorkDir) convention used by every generator.
		abs, err := filepath.Abs("./testdata/defaults")
		testkit.NoError(t, err, "abs")
		got := generator.LoadOutputPackage(generator.Options{
			Output:  "defaultstest/x.gen.go",
			WorkDir: abs,
		})
		testkit.True(t, got != nil, "sibling output pkg loaded")
		testkit.Equal(t, got.Name(), "defaultstest", "package name")
	})
}

func TestLookupCompanionFunc(t *testing.T) {
	t.Parallel()

	t.Run("found in source package returns (true, false)", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./testdata/basic", "")
		testkit.NoError(t, err, "load basic")
		// basic declares ParseStatus(string) (Status, bool).
		found, fromOutput := generator.LookupCompanionFunc(
			pkg, generator.Options{}, "ParseStatus", generator.ParseSig("Status"),
		)
		testkit.True(t, found, "found")
		testkit.True(t, !fromOutput, "from source pkg, not output")
	})

	t.Run("missing in source pkg, no output pkg → (false, false)", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./testdata/basic", "")
		testkit.NoError(t, err, "load basic")
		found, fromOutput := generator.LookupCompanionFunc(
			pkg, generator.Options{}, "DoesNotExist", generator.StringerSig,
		)
		testkit.True(t, !found, "missing")
		testkit.True(t, !fromOutput, "no fallback either")
	})

	t.Run("found in output package returns (true, true)", func(t *testing.T) {
		t.Parallel()
		// testdata/defaults declares no RequestDefaults; the sibling
		// defaultstest package does — exercising the output-pkg
		// fallback branch.
		pkg, err := generator.NewLoader().Load("./testdata/defaults", "")
		testkit.NoError(t, err, "load defaults")
		abs, err := filepath.Abs("./testdata/defaults")
		testkit.NoError(t, err, "abs")
		found, fromOutput := generator.LookupCompanionFunc(
			pkg,
			generator.Options{
				Output:  "defaultstest/x.gen.go",
				WorkDir: abs,
			},
			"RequestDefaults",
			generator.DefaultsFuncSig("Request"),
		)
		testkit.True(t, found, "found in output pkg")
		testkit.True(t, fromOutput, "fromOutput=true")
	})

	t.Run("signature mismatch in source skips to output search", func(t *testing.T) {
		t.Parallel()
		// ParseStatus exists in basic with the ParseSig("Status")
		// shape. Asking for a different sig must fall through; with no
		// output pkg configured, that yields (false, false).
		pkg, err := generator.NewLoader().Load("./testdata/basic", "")
		testkit.NoError(t, err, "load basic")
		mismatch := func(_ *types.Signature) bool { return false }
		found, fromOutput := generator.LookupCompanionFunc(
			pkg, generator.Options{}, "ParseStatus", mismatch,
		)
		testkit.True(t, !found, "sig mismatch → not found")
		testkit.True(t, !fromOutput, "no output pkg")
	})
}

func TestFieldDirective(t *testing.T) {
	t.Parallel()

	pkg, err := generator.NewLoader().Load("./testdata/defaults", "")
	testkit.NoError(t, err, "load defaults")

	t.Run("returns matching directive when present", func(t *testing.T) {
		t.Parallel()
		// Config.Host carries //testkit:default "localhost".
		d, ok := generator.FieldDirective(pkg, "Config", "Host", directive.Default)
		testkit.True(t, ok, "directive found")
		testkit.Equal(t, d.Name, directive.Default, "name matches")
		testkit.Len(t, d.Args, 1, "one arg")
		testkit.Equal(t, d.Args[0], `"localhost"`, "arg captured verbatim")
	})

	t.Run("returns false when field has no directives", func(t *testing.T) {
		t.Parallel()
		// Config.Name carries no directive.
		_, ok := generator.FieldDirective(pkg, "Config", "Name", directive.Default)
		testkit.True(t, !ok, "no directive → false")
	})

	t.Run("returns false when directive name does not match", func(t *testing.T) {
		t.Parallel()
		_, ok := generator.FieldDirective(pkg, "Config", "Host", directive.Sample)
		testkit.True(t, !ok, "different name → false")
	})

	t.Run("returns false for unknown struct or field", func(t *testing.T) {
		t.Parallel()
		_, ok := generator.FieldDirective(pkg, "DoesNotExist", "X", directive.Default)
		testkit.True(t, !ok, "unknown struct → false")
		_, ok = generator.FieldDirective(pkg, "Config", "DoesNotExist", directive.Default)
		testkit.True(t, !ok, "unknown field → false")
	})
}
