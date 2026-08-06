// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/emit"

	"go.thesmos.sh/testkit"
)

// The pair is what stops a generated check from passing against a setter that
// assigns nothing, so what is under test is that both values exist, that they
// differ, and that a type which cannot honestly supply them says so.
func TestSamplesFor(t *testing.T) {
	t.Parallel()

	t.Run("carries the field's name in a string", func(t *testing.T) {
		t.Parallel()
		// A value appearing in a failure message says which setter produced it,
		// which matters when several string setters fail at once.
		sample, _ := samplesFor("string", "Host")
		testkit.Equal(t, sample, `"test-host"`, "the sample names its field")
	})

	t.Run("renders a string sample as a quoted literal", func(t *testing.T) {
		t.Parallel()
		// The value renders straight into a call, so one that lost its quoting
		// would emerge as an undeclared identifier.
		sample, alternate := samplesFor("string", "Host")
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
			sample, _ := samplesFor(name, "Count")
			testkit.Equal(t, sample, "42", "an integer type takes the integer sample")
		}
	})

	t.Run("answers both float widths", func(t *testing.T) {
		t.Parallel()
		sample, _ := samplesFor("float32", "Ratio")
		testkit.Equal(t, sample, "3.14", "a float takes the float sample")
	})

	t.Run("exhausts a bool", func(t *testing.T) {
		t.Parallel()
		// The only type whose pair covers every value it has, which makes its
		// check the strictest: no seed can equal both arms.
		sample, alternate := samplesFor("bool", "Active")
		testkit.Equal(t, sample, "true", "the sample is one arm")
		testkit.Equal(t, alternate, "false", "the alternate is the other")
	})

	t.Run("returns two distinct values for every type it answers", func(t *testing.T) {
		t.Parallel()
		// A pair whose arms agreed would be one value written twice, and the
		// check would pass against a constructor that seeded it.
		for _, name := range []string{"string", "int", "float64", "bool"} {
			sample, alternate := samplesFor(name, "Field")
			testkit.NotEqual(t, sample, alternate, "the arms of a pair differ")
		}
	})

	t.Run("declines a defined type", func(t *testing.T) {
		t.Parallel()
		// `Weekday int` is recorded by name, so there is no way to learn it is
		// an integer and a literal written for it would not compile.
		sample, _ := samplesFor("Weekday", "Day")
		testkit.Equal(t, sample, "", "a defined type admits no sample")
	})

	t.Run("declines a type with no literal but its zero value", func(t *testing.T) {
		t.Parallel()
		// Comparing a zero value against itself is exactly the vacuity the pair
		// exists to prevent, so the caller is told to omit the check instead.
		for _, name := range []string{"error", "any", "complex128"} {
			sample, _ := samplesFor(name, "Field")
			testkit.Equal(t, sample, "", "an unwritable type admits no sample")
		}
	})
}

// The two derivations run at different points — one over the source graph, one
// after substitution — and a disagreement between them would give a generic
// field a different check from the concrete field beside it.
func TestSamplesOfNode(t *testing.T) {
	t.Parallel()

	t.Run("derives from a builtin", func(t *testing.T) {
		t.Parallel()
		sample, _ := samplesOfNode(storefixture.Named("string"), "Host")
		testkit.Equal(t, sample, `"test-host"`, "a builtin field takes a sample")
	})

	t.Run("declines a type in another package", func(t *testing.T) {
		t.Parallel()
		// A named type carries an import path and nothing this generator can
		// read about its underlying shape.
		ref := storefixture.Named("Duration")
		ref.Package = "time"
		sample, _ := samplesOfNode(ref, "Timeout")
		testkit.Equal(t, sample, "", "a foreign type admits no sample")
	})

	t.Run("declines a type parameter", func(t *testing.T) {
		t.Parallel()
		// A parameter is package-less like a builtin, so it reaches the table
		// and falls through it; its pair arrives once a witness resolves it.
		sample, _ := samplesOfNode(storefixture.Named("T"), "Value")
		testkit.Equal(t, sample, "", "an unresolved parameter admits no sample")
	})

	t.Run("declines an absent type", func(t *testing.T) {
		t.Parallel()
		sample, _ := samplesOfNode(nil, "Value")
		testkit.Equal(t, sample, "", "a field with no recorded type admits no sample")
	})
}

// TestSamplesOfRef covers the post-substitution derivation, which is what gives
// a parameterised field the same check its concrete neighbour gets.
func TestSamplesOfRef(t *testing.T) {
	t.Parallel()

	t.Run("derives from a witness", func(t *testing.T) {
		t.Parallel()
		sample, _ := samplesOfRef(emit.Builtin("int"), "Count")
		testkit.Equal(t, sample, "42", "a resolved parameter takes a sample")
	})

	t.Run("declines a reference that is not a builtin", func(t *testing.T) {
		t.Parallel()
		sample, _ := samplesOfRef(emit.SliceOf(emit.Builtin("string")), "Tags")
		testkit.Equal(t, sample, "", "a composite admits no sample")
	})

	t.Run("declines an absent reference", func(t *testing.T) {
		t.Parallel()
		sample, _ := samplesOfRef(nil, "Value")
		testkit.Equal(t, sample, "", "an absent reference admits no sample")
	})
}

// A pointer's setter takes the pointee, so its check has to name the pointee's
// values rather than the field's own type.
func TestSampleSource(t *testing.T) {
	t.Parallel()

	t.Run("takes the pointee for a pointer", func(t *testing.T) {
		t.Parallel()
		f := Field{Shape: Pointer, Type: emit.Ptr(emit.Builtin("int")), Elem: emit.Builtin("int")}
		sample, _ := samplesOfRef(sampleSource(f), "Retries")
		testkit.Equal(t, sample, "42", "a pointer takes its pointee's sample")
	})

	t.Run("takes the key for a set", func(t *testing.T) {
		t.Parallel()
		// A set's entry setter takes a key and nothing else, so the pair its
		// check needs is a pair of keys.
		f := Field{Shape: Set, Type: emit.MapOf(emit.Builtin("string"), nil), Key: emit.Builtin("string")}
		sample, _ := samplesOfRef(sampleSource(f), "Tags")
		testkit.Equal(t, sample, `"test-tags"`, "a set takes its key's sample")
	})

	t.Run("takes the field's own type otherwise", func(t *testing.T) {
		t.Parallel()
		f := Field{Type: emit.Builtin("string")}
		sample, _ := samplesOfRef(sampleSource(f), "Host")
		testkit.Equal(t, sample, `"test-host"`, "a scalar takes its own sample")
	})

	t.Run("yields nothing for a shape with its own checks", func(t *testing.T) {
		t.Parallel()
		// A slice, byte slice or map already asserts on length and on
		// merge-versus-replace, which no empty setter can pass.
		f := Field{Shape: Slice, Type: emit.SliceOf(emit.Builtin("string")), Elem: emit.Builtin("string")}
		sample, _ := samplesOfRef(sampleSource(f), "Tags")
		testkit.Equal(t, sample, "", "a composite needs no sample")
	})
}
