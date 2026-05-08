// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/enum"
)

func runAnalyze(t *testing.T, args []string, opts generator.Options) *enum.Data {
	t.Helper()
	if opts.Output == "" {
		opts.Output = "enum.gen_test.go"
	}
	data, err := enum.Analyze(loadFixture(t, "basic"), args, generator.DefaultConfig(), opts)
	testkit.NoError(t, err, "Analyze")
	return data
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	t.Run("populates one TypeData per requested type", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Status", "Priority"}, generator.Options{})
		testkit.Len(t, data.Enums, 2, "two types requested")
	})

	t.Run("Status carries every method-presence flag", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Status"}, generator.Options{})
		s := data.Enums[0]
		testkit.True(t, s.HasString, "Status has stringer")
		testkit.True(t, s.HasParse, "Status has ParseStatus")
		testkit.Equal(t, s.ParseFunc, "ParseStatus", "ParseFunc derived from type name")
		testkit.True(t, s.HasMarshalText, "Status has MarshalText/UnmarshalText")
		testkit.True(t, s.HasMarshalJSON, "Status has MarshalJSON/UnmarshalJSON")
		testkit.True(t, s.HasMarshalBinary, "Status has MarshalBinary/UnmarshalBinary")
	})

	t.Run("Priority has none of the method flags", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Priority"}, generator.Options{})
		p := data.Enums[0]
		testkit.False(t, p.HasString, "no stringer")
		testkit.False(t, p.HasParse, "no parse")
		testkit.False(t, p.HasMarshalText, "no text")
		testkit.False(t, p.HasMarshalJSON, "no json")
		testkit.False(t, p.HasMarshalBinary, "no binary")
	})

	t.Run("values preserve source-iota order", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Status"}, generator.Options{})
		names := make([]string, len(data.Enums[0].Values))
		for i, v := range data.Enums[0].Values {
			names[i] = v.Name
		}
		testkit.Equal(t, names,
			[]string{"StatusPending", "StatusActive", "StatusClosed"},
			"iota declaration order")
	})

	t.Run("ExpectedStr uses inline comment when present", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Status"}, generator.Options{})
		want := []string{"Pending", "Active", "Closed"}
		got := make([]string, len(data.Enums[0].Values))
		for i, v := range data.Enums[0].Values {
			got[i] = v.ExpectedStr
		}
		testkit.Equal(t, got, want, "inline comments captured")
	})

	t.Run("ExpectedStr falls back to prefix-stripped name when no comment", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Priority"}, generator.Options{})
		want := []string{"Low", "Medium", "High"}
		got := make([]string, len(data.Enums[0].Values))
		for i, v := range data.Enums[0].Values {
			got[i] = v.ExpectedStr
		}
		testkit.Equal(t, got, want, "prefix stripped from const names")
	})

	t.Run("MaxValue is highest int across constants", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Status"}, generator.Options{})
		testkit.Equal(t, data.Enums[0].MaxValue, int64(2), "iota peaks at 2")
	})

	t.Run("ZeroValueName picks the first iota[0]", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Status"}, generator.Options{})
		testkit.Equal(t, data.Enums[0].ZeroValueName, "StatusPending", "iota[0]")
	})

	t.Run("GoldenFile derives from output basename (one combined per generation)", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Status", "Priority"},
			generator.Options{Output: "status.gen_test.go"})
		testkit.Equal(t, data.GoldenFile, "status.gen_wire.json",
			"single combined golden mirrors test file basename")
	})

	t.Run("missing constants → hard error", func(t *testing.T) {
		t.Parallel()
		_, err := enum.Analyze(loadFixture(t, "basic"), []string{"DoesNotExist"},
			generator.DefaultConfig(), generator.Options{Output: "enum.gen_test.go"})
		testkit.True(t, err != nil, "hard error on missing constants")
		testkit.Assert(t, err.Error()).
			Contains("DoesNotExist", "diagnostic names the type").
			Contains("no constants", "explains the failure")
	})

	t.Run("empty args list → hard error", func(t *testing.T) {
		t.Parallel()
		_, err := enum.Analyze(loadFixture(t, "basic"), nil, generator.DefaultConfig(),
			generator.Options{Output: "enum.gen_test.go"})
		testkit.True(t, err != nil, "no types specified")
	})

	t.Run("Data rollups respond to per-type flags", func(t *testing.T) {
		t.Parallel()
		// Status alone: every rollup is true.
		s := runAnalyze(t, []string{"Status"}, generator.Options{})
		testkit.True(t, s.HasContent(), "non-empty")
		testkit.True(t, s.HasStringer(), "stringer rollup")
		testkit.True(t, s.HasText(), "text rollup")
		testkit.True(t, s.HasJSON(), "json rollup")
		testkit.True(t, s.HasBinary(), "binary rollup")

		// Priority alone: every method rollup is false.
		p := runAnalyze(t, []string{"Priority"}, generator.Options{})
		testkit.False(t, p.HasStringer(), "no stringer")
		testkit.False(t, p.HasText(), "no text")
		testkit.False(t, p.HasJSON(), "no json")
		testkit.False(t, p.HasBinary(), "no binary")
	})

	t.Run("subdir output sets ImportPath + Qualifier", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, []string{"Status"},
			generator.Options{Output: "subpkg/enum.gen_test.go"})
		testkit.True(t, data.ImportPath != "", "imports source pkg from subdir")
		testkit.Equal(t, data.Qualifier, "basic.", "dotted qualifier")
		testkit.Equal(t, data.Enums[0].Qualifier, "basic.",
			"per-type qualifier propagated")
	})

	// --- corner-case fixture tests below ---

	t.Run("wrong-signature String() rejected → HasString false", func(t *testing.T) {
		t.Parallel()
		data, err := enum.Analyze(loadFixture(t, "invalid"), []string{"WrongSig"},
			generator.DefaultConfig(), generator.Options{Output: "wrong.gen_test.go"})
		testkit.NoError(t, err, "Analyze")
		testkit.False(t, data.Enums[0].HasString,
			"String(int) does not satisfy fmt.Stringer")
	})

	t.Run("wrong-signature Parse rejected → HasParse false", func(t *testing.T) {
		t.Parallel()
		data, err := enum.Analyze(loadFixture(t, "invalid"), []string{"WrongSig"},
			generator.DefaultConfig(), generator.Options{Output: "wrong.gen_test.go"})
		testkit.NoError(t, err, "Analyze")
		testkit.False(t, data.Enums[0].HasParse,
			"ParseWrongSig returns (string, error) not (WrongSig, error)")
	})

	t.Run("wrong-signature MarshalText rejected → HasMarshalText false", func(t *testing.T) {
		t.Parallel()
		data, err := enum.Analyze(loadFixture(t, "invalid"), []string{"WrongSig"},
			generator.DefaultConfig(), generator.Options{Output: "wrong.gen_test.go"})
		testkit.NoError(t, err, "Analyze")
		testkit.False(t, data.Enums[0].HasMarshalText,
			"MarshalText returning string is not encoding.TextMarshaler")
	})

	t.Run("explicit non-iota values are surfaced with their declared values", func(t *testing.T) {
		t.Parallel()
		data, err := enum.Analyze(loadFixture(t, "enums"), []string{"Color"},
			generator.DefaultConfig(), generator.Options{Output: "color.gen_test.go"})
		testkit.NoError(t, err, "Analyze")
		c := data.Enums[0]
		testkit.Len(t, c.Values, 4, "four colors")
		valueByName := make(map[string]int64, len(c.Values))
		for _, v := range c.Values {
			valueByName[v.Name] = v.IntValue
		}
		testkit.Equal(t, valueByName["ColorUnknown"], int64(-1), "negative value preserved")
		testkit.Equal(t, valueByName["ColorRed"], int64(10), "explicit 10")
		testkit.Equal(t, valueByName["ColorBlue"], int64(20), "explicit 20")
		testkit.Equal(t, valueByName["ColorChartreus"], int64(999), "explicit 999")
		testkit.Equal(t, c.MaxValue, int64(999), "MaxValue from declared values")
	})

	t.Run("ZeroValueName empty when no declared constant has value 0", func(t *testing.T) {
		t.Parallel()
		data, err := enum.Analyze(loadFixture(t, "enums"), []string{"Color"},
			generator.DefaultConfig(), generator.Options{Output: "color.gen_test.go"})
		testkit.NoError(t, err, "Analyze")
		testkit.Equal(t, data.Enums[0].ZeroValueName, "",
			"no constant equals 0 → empty (template suppresses zero-value subtest)")
	})

	t.Run("constants spanning multiple files preserve cross-file declaration order", func(t *testing.T) {
		t.Parallel()
		data, err := enum.Analyze(loadFixture(t, "enums"), []string{"Region"},
			generator.DefaultConfig(), generator.Options{Output: "region.gen_test.go"})
		testkit.NoError(t, err, "Analyze")
		names := make([]string, len(data.Enums[0].Values))
		for i, v := range data.Enums[0].Values {
			names[i] = v.Name
		}
		// multifile.go declares RegionUS, RegionEU; multifile_more.go
		// declares RegionAP, RegionLA. Source position sorts by
		// filename first, then offset — multifile.go < multifile_more.go
		// lexically.
		testkit.Equal(t, names,
			[]string{"RegionUS", "RegionEU", "RegionAP", "RegionLA"},
			"declaration order across files")
	})

	t.Run("string-typed enum surfaces a clear rejection", func(t *testing.T) {
		t.Parallel()
		// Tag is `type Tag string` — wire-compat assumes integer
		// values, so analyze must reject up front. The diagnostic
		// names the type and explains why.
		_, err := enum.Analyze(loadFixture(t, "invalid"), []string{"Tag"},
			generator.DefaultConfig(), generator.Options{Output: "tag.gen_test.go"})
		testkit.True(t, err != nil, "string-typed enum rejected")
		testkit.Assert(t, err.Error()).
			Contains("Tag", "diagnostic names the type").
			Contains("only integer enums", "explains the constraint")
	})

	t.Run("unexported constants are skipped", func(t *testing.T) {
		t.Parallel()
		data, err := enum.Analyze(loadFixture(t, "enums"), []string{"Region"},
			generator.DefaultConfig(), generator.Options{Output: "region.gen_test.go"})
		testkit.NoError(t, err, "Analyze")
		for _, v := range data.Enums[0].Values {
			testkit.False(t, v.Name == "_internal",
				"unexported _internal must not appear")
		}
	})
}
