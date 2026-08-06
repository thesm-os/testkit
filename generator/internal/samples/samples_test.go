// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package samples_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/samples"
)

// The pair is what stops a generated check from passing against a setter that
// assigns nothing, so what is under test is that both values exist, that they
// differ, and that a type which cannot honestly supply them says so.
func TestFor(t *testing.T) {
	t.Parallel()

	t.Run("carries the field's name in a string", func(t *testing.T) {
		t.Parallel()
		// A value appearing in a failure message says which setter produced it,
		// which matters when several string setters fail at once.
		sample, _ := samples.For("string", "Host")
		testkit.Equal(t, sample, `"test-host"`, "the sample names its field")
	})

	t.Run("renders a string sample as a quoted literal", func(t *testing.T) {
		t.Parallel()
		// The value renders straight into a call, so one that lost its quoting
		// would emerge as an undeclared identifier.
		sample, alternate := samples.For("string", "Host")
		testkit.HasPrefix(t, sample, `"`, "the sample is quoted")
		testkit.HasSuffix(t, alternate, `"`, "the alternate is quoted")
	})

	t.Run("answers every integer width from one arm", func(t *testing.T) {
		t.Parallel()
		// The literals are untyped, so one arm serves every width — and a width
		// the arm forgot would silently lose its check rather than fail here.
		for _, name := range []string{
			"int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"byte", "rune",
		} {
			sample, _ := samples.For(name, "Count")
			testkit.Equal(t, sample, "42", "an integer type takes the integer sample")
		}
	})

	t.Run("answers both float widths", func(t *testing.T) {
		t.Parallel()
		sample, _ := samples.For("float32", "Ratio")
		testkit.Equal(t, sample, "3.14", "a float takes the float sample")
	})

	t.Run("answers both complex widths", func(t *testing.T) {
		t.Parallel()
		sample, _ := samples.For("complex128", "Wave")
		testkit.Equal(t, sample, "1 + 2i", "a complex takes the complex sample")
	})

	t.Run("exhausts a bool", func(t *testing.T) {
		t.Parallel()
		// The only type whose pair covers every value it has, which makes its
		// check the strictest: no seed can equal both arms.
		sample, alternate := samples.For("bool", "Active")
		testkit.Equal(t, sample, "true", "the sample is one arm")
		testkit.Equal(t, alternate, "false", "the alternate is the other")
	})

	t.Run("returns two distinct values for every type it answers", func(t *testing.T) {
		t.Parallel()
		// A pair whose arms agreed would be one value written twice, and the
		// check would pass against a constructor that seeded it.
		for _, name := range []string{"string", "int", "float64", "complex64", "bool"} {
			sample, alternate := samples.For(name, "Field")
			testkit.NotEqual(t, sample, alternate, "the arms of a pair differ")
		}
	})

	t.Run("declines a name it does not know", func(t *testing.T) {
		t.Parallel()
		// The table answers builtins; a declared type is [resolver]'s job.
		sample, _ := samples.For("Weekday", "Day")
		testkit.Equal(t, sample, "", "an unknown name admits no sample")
	})
}
