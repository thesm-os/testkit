// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/shape"
	"go.thesmos.sh/testkit/generator/spec"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	t.Run("happy path on basic.Store populates the canonical fields", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		data, err := spec.Analyze(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{Output: "storetest/store.gen.go"})
		testkit.NoError(t, err, "Analyze")
		testkit.Equal(t, data.PackageName, "storetest", "PackageName from output dir")
		testkit.Equal(t, data.Interface.Name, "Store", "Interface attached")
		testkit.Equal(t, data.QualifiedType, "basic.Store", "qualified type for non-generic")
		testkit.Equal(t, data.TypeParamDecl, "", "non-generic decl is empty")
		testkit.Equal(t, data.TypeParamArgs, "", "non-generic args is empty")
		testkit.False(t, data.IsGeneric, "non-generic")
		testkit.True(t, len(data.Methods) > 0, "methods populated")
	})

	t.Run("classifies each method via shape.Classify", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		data, err := spec.Analyze(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{Output: "storetest/store.gen.go"})
		testkit.NoError(t, err, "Analyze")
		// basic.Store: Get(ctx, string) (Item, error) → Reader
		//              Put(ctx, Item) error            → Writer
		byName := make(map[string]shape.Info, len(data.Methods))
		for _, m := range data.Methods {
			byName[m.Name] = m.Shape
		}
		testkit.Equal(t, byName["Get"].Shape, shape.Reader, "Get is Reader")
		testkit.Equal(t, byName["Put"].Shape, shape.Writer, "Put is Writer")
	})

	t.Run("surfaces error for missing interface", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := spec.Analyze(pkg, []string{"DoesNotExist"},
			generator.DefaultConfig(), generator.Options{Output: "x.gen.go"})
		testkit.True(t, err != nil, "missing interface errors")
	})

	t.Run("rejects empty args", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := spec.Analyze(pkg, nil,
			generator.DefaultConfig(), generator.Options{Output: "x.gen.go"})
		testkit.True(t, err != nil, "no args errors")
	})

	t.Run("rejects multi-arg invocation (one interface per generation)", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := spec.Analyze(pkg, []string{"Store", "Other"},
			generator.DefaultConfig(), generator.Options{Output: "x.gen.go"})
		testkit.True(t, err != nil, "multi-arg errors")
	})

	t.Run("Tracker is shared and primed with the source package", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		data, err := spec.Analyze(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{Output: "storetest/store.gen.go"})
		testkit.NoError(t, err, "Analyze")
		testkit.True(t, data.Tracker != nil, "Tracker present")
		// Tracker has registered the source pkg already (via QualifyType).
		testkit.Equal(t, data.Tracker.AddPath(pkg.Path()), "basic",
			"source pkg already tracked under its base name")
	})

	t.Run("FinalizeImports captures Tracker.Imports() into Data.Imports", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		data, err := spec.Analyze(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{Output: "storetest/store.gen.go"})
		testkit.NoError(t, err, "Analyze")
		testkit.True(t, len(data.Imports) == 0, "Imports empty before Finalize")
		data.FinalizeImports()
		testkit.True(t, len(data.Imports) > 0, "Imports populated after Finalize")
	})
}
