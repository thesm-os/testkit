// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/enum"
)

// withType returns a Data carrying one TypeData with the named flag
// set via mutate. Keeps each rollup subtest focused on one flag
// without the noise of a fully-populated TypeData.
func withType(mutate func(*enum.TypeData)) *enum.Data {
	td := enum.TypeData{TypeName: "X"}
	mutate(&td)
	return &enum.Data{Enums: []enum.TypeData{td}}
}

func TestData(t *testing.T) {
	t.Parallel()

	t.Run("HasContent is false for zero-value Data", func(t *testing.T) {
		t.Parallel()
		var d enum.Data
		testkit.False(t, d.HasContent(), "no enums → skippable")
	})

	t.Run("HasContent is true when any enum is present", func(t *testing.T) {
		t.Parallel()
		d := enum.Data{Enums: []enum.TypeData{{TypeName: "X"}}}
		testkit.True(t, d.HasContent(), "one enum is enough")
	})

	t.Run("HasStringer is false when no type has String", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, withType(func(*enum.TypeData) {}).HasStringer(),
			"no flag set → false")
	})

	t.Run("HasStringer is true when at least one type has String", func(t *testing.T) {
		t.Parallel()
		d := withType(func(td *enum.TypeData) { td.HasString = true })
		testkit.True(t, d.HasStringer(), "any-true rollup")
	})

	t.Run("HasText is false when no type has MarshalText", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, withType(func(*enum.TypeData) {}).HasText(),
			"no flag set → false")
	})

	t.Run("HasText is true when at least one type has MarshalText", func(t *testing.T) {
		t.Parallel()
		d := withType(func(td *enum.TypeData) { td.HasMarshalText = true })
		testkit.True(t, d.HasText(), "any-true rollup")
	})

	t.Run("HasJSON is false when no type has MarshalJSON", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, withType(func(*enum.TypeData) {}).HasJSON(),
			"no flag set → false")
	})

	t.Run("HasJSON is true when at least one type has MarshalJSON", func(t *testing.T) {
		t.Parallel()
		d := withType(func(td *enum.TypeData) { td.HasMarshalJSON = true })
		testkit.True(t, d.HasJSON(), "any-true rollup")
	})

	t.Run("HasBinary is false when no type has MarshalBinary", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, withType(func(*enum.TypeData) {}).HasBinary(),
			"no flag set → false")
	})

	t.Run("HasBinary is true when at least one type has MarshalBinary", func(t *testing.T) {
		t.Parallel()
		d := withType(func(td *enum.TypeData) { td.HasMarshalBinary = true })
		testkit.True(t, d.HasBinary(), "any-true rollup")
	})

	t.Run("any-true rollups don't require every type to set the flag", func(t *testing.T) {
		t.Parallel()
		d := &enum.Data{
			Enums: []enum.TypeData{
				{TypeName: "Stringer", HasString: true},
				{TypeName: "Bare"},
			},
		}
		testkit.True(t, d.HasStringer(), "first type pulls the rollup true")
	})

	t.Run("rollups are independent across encoding flavors", func(t *testing.T) {
		t.Parallel()
		// Set only HasMarshalJSON; every other rollup must stay false.
		d := withType(func(td *enum.TypeData) { td.HasMarshalJSON = true })
		testkit.True(t, d.HasJSON(), "JSON rollup")
		testkit.False(t, d.HasText(), "text not set")
		testkit.False(t, d.HasBinary(), "binary not set")
		testkit.False(t, d.HasStringer(), "stringer not set")
	})
}
