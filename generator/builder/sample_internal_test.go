// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/store"

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

	t.Run("answers both complex widths", func(t *testing.T) {
		t.Parallel()
		sample, _ := samplesFor("complex128", "Wave")
		testkit.Equal(t, sample, "1 + 2i", "a complex takes the complex sample")
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
		for _, name := range []string{"string", "int", "float64", "complex64", "bool"} {
			sample, alternate := samplesFor(name, "Field")
			testkit.NotEqual(t, sample, alternate, "the arms of a pair differ")
		}
	})

	t.Run("declines a name it does not know", func(t *testing.T) {
		t.Parallel()
		// The table answers builtins; a declared type is [resolver]'s job.
		sample, _ := samplesFor("Weekday", "Day")
		testkit.Equal(t, sample, "", "an unknown name admits no sample")
	})
}

// A sample carrying a type has to hand that type over as a reference: a
// rendered file registers an import only for a reference it was given, so a
// value folded into text would name a package the file never imports.
func TestSampleShape(t *testing.T) {
	t.Parallel()

	t.Run("reports an empty sample as absent", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, Sample{}.OK(), "a derived-nothing sample is absent")
	})

	t.Run("carries no reference for a builtin", func(t *testing.T) {
		t.Parallel()
		// `42` stands alone; wrapping it in `int(42)` would be a conversion the
		// linters would rightly object to.
		sample, _ := resolve(t).samples(storefixture.Named("int"), "Count", seen())
		testkit.True(t, sample.Ref == nil, "a builtin needs no type beside it")
		testkit.Equal(t, sample.Text, "42", "the literal stands alone")
	})
}

// A generator reading only field types sees `Weekday` and `Role` as opaque
// names, and every setter taking one goes unchecked. The declarations are in
// the graph, so what is under test is how far reading them reaches — and where
// it stops.
func TestResolverSamples(t *testing.T) {
	t.Parallel()

	t.Run("writes a defined type as a conversion", func(t *testing.T) {
		t.Parallel()
		// A bare 42 would compile against `Weekday` today and stop compiling
		// the moment the field's type moved.
		sample, alternate := resolve(t).samples(named("Weekday"), "Day", seen())
		testkit.Equal(t, sample.Text, "42", "the underlying type supplies the value")
		testkit.False(t, sample.Composite, "a defined type takes a conversion")
		testkit.Equal(t, alternate.Text, "7", "the alternate follows the same route")
	})

	t.Run("writes a struct as a literal setting one of its fields", func(t *testing.T) {
		t.Parallel()
		sample, alternate := resolve(t).samples(named("Role"), "Role", seen())
		testkit.Equal(t, sample.Text, `{Name: "test-name"}`, "the struct sets its first usable field")
		testkit.True(t, sample.Composite, "a struct takes a composite literal")
		testkit.Equal(t, alternate.Text, `{Name: "other-name"}`, "the arms differ inside the literal")
	})

	t.Run("skips a struct field it cannot write a value for", func(t *testing.T) {
		t.Parallel()
		// Opaque's first field is a foreign type; the second is a string, and
		// taking the first would have lost the whole struct's sample.
		sample, _ := resolve(t).samples(named("Opaque"), "Op", seen())
		testkit.Equal(t, sample.Text, `{Tag: "test-tag"}`, "the first usable field wins")
	})

	t.Run("declines a struct with no field it can write", func(t *testing.T) {
		t.Parallel()
		sample, _ := resolve(t).samples(named("Blind"), "Bl", seen())
		testkit.False(t, sample.OK(), "a struct of foreign types admits no sample")
	})

	t.Run("fills one element of an array", func(t *testing.T) {
		t.Parallel()
		// Filling the rest would say nothing more than the first element does.
		sample, _ := resolve(t).samples(storefixture.Array(storefixture.Named("int"), 3), "Arr", seen())
		testkit.Equal(t, sample.Text, "{42}", "one element differs between the arms")
		testkit.True(t, sample.Composite, "an array takes a composite literal")
	})

	t.Run("writes any as a converted string", func(t *testing.T) {
		t.Parallel()
		// `any` admits every value; the conversion is what keeps both sides of
		// the comparison the same type so its parameter can be inferred.
		sample, _ := resolve(t).samples(storefixture.AnonInterface(nil, nil), "Extra", seen())
		testkit.Equal(t, sample.Text, `"test-extra"`, "any takes the string pair")
		testkit.False(t, sample.Composite, "any takes a conversion")
	})

	t.Run("declines an interface that declares a method", func(t *testing.T) {
		t.Parallel()
		// Only the empty interface admits every value. One with a method needs
		// an implementation, and this generator has none to name.
		ref := storefixture.AnonInterface([]*node.Method{{Name: "Read"}}, nil)
		sample, _ := resolve(t).samples(ref, "R", seen())
		testkit.False(t, sample.OK(), "a non-empty interface admits no sample")
	})

	t.Run("declines a type from a package the run never read", func(t *testing.T) {
		t.Parallel()
		// This is the floor: nothing about time.Time is in the graph, so no
		// value of it can be written and the check is dropped rather than faked.
		sample, _ := resolve(t).samples(storefixture.PkgNamed("time", "Time"), "Deadline", seen())
		testkit.False(t, sample.OK(), "an unloaded type admits no sample")
	})

	t.Run("terminates on a struct that reaches itself", func(t *testing.T) {
		t.Parallel()
		// Cyclic's only field is a Cyclic, so a resolver without the guard
		// recurses until the stack ends.
		sample, _ := resolve(t).samples(named("Cyclic"), "C", seen())
		testkit.False(t, sample.OK(), "a cycle yields nothing rather than looping")
	})

	t.Run("declines a slice", func(t *testing.T) {
		t.Parallel()
		// A slice's checks assert on length and on replace-versus-append, which
		// no empty setter can pass, so it needs no pair.
		sample, _ := resolve(t).samples(storefixture.Slice(storefixture.Named("string")), "Tags", seen())
		testkit.False(t, sample.OK(), "a slice needs no sample")
	})

	t.Run("declines an absent type", func(t *testing.T) {
		t.Parallel()
		sample, _ := resolve(t).samples(nil, "X", seen())
		testkit.False(t, sample.OK(), "a field with no recorded type admits no sample")
	})
}

// The post-substitution derivation is what gives a parameterised field the same
// check its concrete neighbour gets.
func TestSamplesOfRef(t *testing.T) {
	t.Parallel()

	t.Run("derives from a witness", func(t *testing.T) {
		t.Parallel()
		sample, _ := samplesOfRef(emit.Builtin("int"), "Count")
		testkit.Equal(t, sample.Text, "42", "a resolved parameter takes a sample")
	})

	t.Run("declines a reference that is not a builtin", func(t *testing.T) {
		t.Parallel()
		// Only a witness reaches here, and a witness is always a builtin — the
		// source types are gone by this point, so there is nothing to resolve
		// a named type against.
		sample, _ := samplesOfRef(emit.SliceOf(emit.Builtin("string")), "Tags")
		testkit.False(t, sample.OK(), "a composite admits no sample")
	})

	t.Run("declines an absent reference", func(t *testing.T) {
		t.Parallel()
		sample, _ := samplesOfRef(nil, "Value")
		testkit.False(t, sample.OK(), "an absent reference admits no sample")
	})
}

// A field's pair has to describe whatever its setter takes, which for two
// shapes is not the field's own type.
func TestSampleSource(t *testing.T) {
	t.Parallel()

	t.Run("takes the pointee for a pointer", func(t *testing.T) {
		t.Parallel()
		f := Field{Shape: Pointer, Type: emit.Ptr(emit.Builtin("int")), Elem: emit.Builtin("int")}
		sample, _ := samplesOfRef(sampleSource(f), "Retries")
		testkit.Equal(t, sample.Text, "42", "a pointer takes its pointee's sample")
	})

	t.Run("takes the key for a set", func(t *testing.T) {
		t.Parallel()
		// A set's entry setter takes a key and nothing else, so the pair its
		// check needs is a pair of keys.
		f := Field{Shape: Set, Type: emit.MapOf(emit.Builtin("string"), nil), Key: emit.Builtin("string")}
		sample, _ := samplesOfRef(sampleSource(f), "Tags")
		testkit.Equal(t, sample.Text, `"test-tags"`, "a set takes its key's sample")
	})

	t.Run("takes the field's own type otherwise", func(t *testing.T) {
		t.Parallel()
		f := Field{Type: emit.Builtin("string")}
		sample, _ := samplesOfRef(sampleSource(f), "Host")
		testkit.Equal(t, sample.Text, `"test-host"`, "a scalar takes its own sample")
	})
}

// seen returns a fresh recursion guard.
func seen() map[string]bool { return make(map[string]bool) }

// named returns a reference to a type declared by [resolve]'s fixture.
func named(name string) *node.TypeRef {
	return storefixture.PkgNamed("example.com/cfg", name)
}

// resolve returns a resolver over a package declaring one type of each shape
// the derivation has an arm for.
func resolve(t *testing.T) *resolver {
	t.Helper()
	foreign := func(name string) *node.TypeRef { return storefixture.PkgNamed("time", name) }
	s := storefixture.New().
		Package("cfg", "example.com/cfg").
		Alias("Weekday", func(b *storefixture.AliasBuilder) {
			b.Target(storefixture.Named("int"))
		}).
		Struct("Role", func(b *storefixture.StructBuilder) {
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		Struct("Opaque", func(b *storefixture.StructBuilder) {
			b.Field("At", foreign("Time"), nil)
			b.Field("Tag", storefixture.Named("string"), nil)
		}).
		Struct("Blind", func(b *storefixture.StructBuilder) {
			b.Field("At", foreign("Time"), nil)
		}).
		Struct("Cyclic", func(b *storefixture.StructBuilder) {
			b.Field("Self", cyclicRef(), nil)
		}).
		Build()
	return newResolver(store.NewReader(s))
}

// cyclicRef names the struct that declares it, which is the shape a resolver
// without a recursion guard never returns from.
func cyclicRef() *node.TypeRef { return storefixture.PkgNamed("example.com/cfg", "Cyclic") }
