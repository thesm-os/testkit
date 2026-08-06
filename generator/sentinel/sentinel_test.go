// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel_test

import (
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/pipelinetest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/sentinel"
)

// The framework conformance suites pin the static contract — a stable name,
// deterministic outputs, templates that parse, two directive schemas that do
// not collide — none of which a fixture-driven test would notice.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("satisfies the framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, sentinel.New())
	})

	t.Run("satisfies the generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(t, sentinel.New(), []plugintest.GeneratorFixture{
			{
				Name: "annotated package",
				BuildStore: func(t *testing.T) *store.Store {
					t.Helper()
					return bare("").Build()
				},
			},
			{
				Name: "empty store",
				BuildStore: func(t *testing.T) *store.Store {
					t.Helper()
					return storefixture.New().Build()
				},
			},
		})
	})
}

// The prefix key carries two behaviours behind one name, and the difference
// between them is a check that exists and a check that does not.
func TestPrefix(t *testing.T) {
	t.Parallel()

	t.Run("derives the prefix from the package name", func(t *testing.T) {
		t.Parallel()
		rendered(t, bare("")).Contains(`"cfg: "`)
	})

	t.Run("takes an override over the derived name", func(t *testing.T) {
		t.Parallel()
		// For a package whose errors are named for the subsystem rather than
		// the directory it happens to live in.
		rendered(t, bare("store")).Contains(`"store: "`)
	})

	t.Run("suppresses the check on an empty prefix", func(t *testing.T) {
		t.Parallel()
		// `prefix=` and `prefix=off` say the same thing, and an author who
		// writes the first should not get an assertion they meant to remove.
		rendered(t, bare("=empty")).
			NotContains(`t.Run("every sentinel carries the package prefix"`)
	})

	t.Run("suppresses the check on prefix=off", func(t *testing.T) {
		t.Parallel()
		rendered(t, bare(sentinel.PrefixOff)).
			NotContains(`t.Run("every sentinel carries the package prefix"`)
	})

	t.Run("says why the suppressed check is absent", func(t *testing.T) {
		t.Parallel()
		// A reader looking for the check finds the reason rather than a gap. A
		// generator that quietly dropped a failing one would be worse than one
		// that never had it.
		rendered(t, bare(sentinel.PrefixOff)).Contains("declares prefix=off")
	})
}

// Every sentinel set earns the same four checks, and each has to be able to
// fail — which is what the fixture-driven corpus proves and this pins.
func TestSentinelChecks(t *testing.T) {
	t.Parallel()

	t.Run("ignores a variable outside the naming convention", func(t *testing.T) {
		t.Parallel()
		// The Err prefix is what opts a variable in. One named otherwise is
		// not a sentinel, and treating it as one would assert a message
		// contract over something that never claimed to have it.
		b := bare("")
		b.Variable("Timeout", func(v *storefixture.VariableBuilder) {
			v.Pos(position.At("cfg/errors.go", 1, 1))
		})
		rendered(t, b).NotContains(`"Timeout"`)
	})

	t.Run("ignores an unexported error variable", func(t *testing.T) {
		t.Parallel()
		// A consumer cannot name it, so a check in their external test package
		// would not compile.
		b := bare("")
		b.Variable("errInternal", func(v *storefixture.VariableBuilder) {
			v.Pos(position.At("cfg/errors.go", 1, 1))
		})
		rendered(t, b).NotContains("errInternal")
	})

	t.Run("lists every sentinel it found", func(t *testing.T) {
		t.Parallel()
		// A sentinel named outside the convention is not found, so the list is
		// how an absence becomes visible.
		f := rendered(t, bare(""))
		for _, name := range []string{"ErrEmpty", "ErrFull", "ErrInvalid"} {
			f.Contains(name)
		}
	})

	t.Run("checks every property a sentinel set owes", func(t *testing.T) {
		t.Parallel()
		f := rendered(t, bare(""))
		for _, want := range []string{
			"every sentinel is non-nil",
			"every sentinel has a message",
			"no two sentinels share a message",
			"no sentinel's message is a prefix of another's",
			"no sentinel matches another",
		} {
			f.Contains(`t.Run("` + want + `"`)
		}
	})

	t.Run("omits the checks that cannot fail", func(t *testing.T) {
		t.Parallel()
		// errors.Is compares identity before consulting anything a type
		// declares, so a sentinel survives %w and errors.Join no matter what
		// it is. The assertion would be about the standard library.
		f := rendered(t, bare(""))
		f.NotContains("survives being wrapped")
		f.NotContains("survives being joined")
	})

	t.Run("emits nothing about error types when there are none", func(t *testing.T) {
		t.Parallel()
		// The floor. A corpus holding only the rich case cannot tell "the
		// optional checks were correctly omitted" from "silently dropped".
		rendered(t, bare("")).NotContains("Contract(t *testing.T)")
	})
}

// An optional method earns a check, and a type without it must get no check
// rather than one that cannot fail.
func TestErrorTypes(t *testing.T) {
	t.Parallel()

	t.Run("holds an error type to the same prefix as a sentinel", func(t *testing.T) {
		t.Parallel()
		// A custom error reaches the same logs and is read the same way, so
		// checking only the Err* vars leaves half a package unexamined.
		rendered(t, rich()).Contains("carries the package prefix like a sentinel does")
	})

	t.Run("checks that a zero value reports rather than panics", func(t *testing.T) {
		t.Parallel()
		// A message dereferencing a field the zero value leaves nil crashes
		// inside whatever was already going wrong.
		rendered(t, rich()).Contains("reports a message for its zero value")
	})

	t.Run("checks a type against the package's other error types", func(t *testing.T) {
		t.Parallel()
		rendered(t, rich()).Contains("does not match the package's other error types")
	})

	t.Run("omits an errors.As recovery check", func(t *testing.T) {
		t.Parallel()
		// As finds a value by assignability while walking the chain, so it
		// succeeds for any type reachable through it and fails for none.
		rendered(t, rich()).NotContains("must be recoverable with errors.As")
	})

	t.Run("checks a declared Is against errors.Is", func(t *testing.T) {
		t.Parallel()
		// An Is on the wrong receiver form is never consulted, and the type
		// then silently matches nothing.
		rendered(t, rich()).Contains("errors.Is must agree with NotFoundError.Is")
	})

	t.Run("omits the Is check for a type declaring none", func(t *testing.T) {
		t.Parallel()
		rendered(t, rich()).NotContains("errors.Is must agree with PlainError.Is")
	})

	t.Run("checks unwrap for a type carrying a cause", func(t *testing.T) {
		t.Parallel()
		rendered(t, rich()).Contains("WrappedError must expose its cause")
	})

	t.Run("omits the unwrap check for a type carrying none", func(t *testing.T) {
		t.Parallel()
		// Without a field to put a cause in there is nothing to hand the type,
		// so the check is dropped rather than run against a nil.
		rendered(t, rich()).NotContains("PlainError must expose its cause")
	})

	t.Run("writes a value into every string field it checks", func(t *testing.T) {
		t.Parallel()
		rendered(t, rich()).Contains(`Key: "test-key"`)
	})

	t.Run("ignores an unexported type declaring Error", func(t *testing.T) {
		t.Parallel()
		// Same reason as an unexported sentinel: the checks live outside the
		// package and cannot name it.
		b := rich()
		b.Struct("hiddenError", func(st *storefixture.StructBuilder) {
			st.Pos(position.At("cfg/errors.go", 1, 1))
			st.Field("Detail", storefixture.Named("string"), nil)
		})
		methods(b, "hiddenError", "Error")
		rendered(t, b).NotContains("hiddenError")
	})

	t.Run("ignores an unexported field of an error type", func(t *testing.T) {
		t.Parallel()
		// A literal naming it would not compile from the test package.
		b := rich()
		for _, st := range b.PackageNode().Structs {
			if st.Name == "NotFoundError" {
				st.Fields = append(st.Fields,
					&node.Field{Name: "secret", Type: storefixture.Named("string")},
					&node.Field{Name: "Untyped"})
			}
		}
		rendered(t, b).NotContains("secret:")
	})

	t.Run("leaves a non-string field out of the message check", func(t *testing.T) {
		t.Parallel()
		// A message renders a number through a verb whose width and base are
		// not visible here, so asserting that "42" appears would fail against
		// %03d for a field that is perfectly well reported.
		rendered(t, rich()).NotContains("Attempts must reach the message")
	})
}

// A package may carry an error type and no sentinel at all, which decides both
// what is emitted and — because the output file is named after a declaration —
// where it lands.
func TestErrorTypesWithoutSentinels(t *testing.T) {
	t.Parallel()

	t.Run("emits the per-type checks", func(t *testing.T) {
		t.Parallel()
		rendered(t, typesOnly()).Contains("TestPlainErrorContract")
	})

	t.Run("emits no sentinel umbrella", func(t *testing.T) {
		t.Parallel()
		// There is no set to assert anything about, and a check over an empty
		// one would read as though the package had been examined.
		rendered(t, typesOnly()).NotContains("Sentinels(t *testing.T)")
	})

	t.Run("omits the Is check with no sentinel to compare against", func(t *testing.T) {
		t.Parallel()
		// The check asks whether errors.Is reaches the declared method, and
		// with nothing to pass it there is no question to ask.
		rendered(t, typesOnly()).NotContains("errors.Is must agree with")
	})
}

// The cross-package check needs two packages by construction, so a fixture set
// with one leaves the directive parsing, naming nothing, and passing.
func TestNoOverlap(t *testing.T) {
	t.Parallel()

	t.Run("checks against the named package", func(t *testing.T) {
		t.Parallel()
		rendered(t, neighbouring()).Contains("no sentinel matches one in other")
	})

	t.Run("names the neighbour's sentinels rather than its own", func(t *testing.T) {
		t.Parallel()
		rendered(t, neighbouring()).Contains("ErrGone")
	})
}

// The diagnostics are what a corpus cannot show: a fixture provoking one would
// fail the run that generates every other fixture.
func TestDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("reports a package with no error contract to check", func(t *testing.T) {
		t.Parallel()
		// The directive says the package's errors are a contract. A file
		// asserting nothing about an empty set would read as though they had
		// been checked.
		empty := storefixture.New().Package("cfg", "example.com/cfg")
		annotate(empty, storefixture.Directive("sentinel"))
		testkit.Len(t, diagnostics(t, empty), 1, "a package with nothing to check is reported")
	})

	t.Run("ignores a no-overlap directive naming nothing", func(t *testing.T) {
		t.Parallel()
		// The schema declares the argument, so a bare line is a malformed
		// directive rather than a package to check against.
		b := bare("")
		annotate(b, storefixture.Directive("sentinel-no-overlap-with"))
		testkit.Len(t, diagnostics(t, b), 0, "a directive naming nothing is skipped")
	})

	t.Run("reports a package declaring non-overlap with itself", func(t *testing.T) {
		t.Parallel()
		// Every sentinel matches itself, so the check would fail for a package
		// behaving exactly as intended.
		b := bare("")
		annotate(b, storefixture.Directive("sentinel-no-overlap-with",
			storefixture.Arg("example.com/cfg")))
		testkit.Len(t, diagnostics(t, b), 1, "a self-reference is reported")
	})
}

// rendered drives the plugin and the Go backend over b through a synthetic
// pipeline, so routing and rendering both participate, and returns the file.
func rendered(t *testing.T, b *storefixture.Builder) *pipelinetest.FileAssertion {
	t.Helper()
	return pipelinetest.New(t).
		WithFrontend(pipelinetest.FromNodes(b.PackageNode())).
		WithGenerator(sentinel.New()).
		WithBackend(backendgolang.New()).
		Build().
		Run().
		AssertFile("errors" + sentinel.GoSuffix)
}

// diagnostics drives the plugin over b and returns what it reported.
func diagnostics(t *testing.T, b *storefixture.Builder) []diag.Diag {
	t.Helper()
	s := b.Build()
	sink := diag.New()
	ctx := &plugin.GeneratorContext{Store: s, Reader: store.NewReader(s), Diag: sink}
	if err := sentinel.New().Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return sink.Diagnostics()
}

// annotate attaches a directive to the fixture's package node, which is where
// this plugin's schemas are scoped.
func annotate(b *storefixture.Builder, d *directive.Directive) {
	pkg := b.PackageNode()
	pkg.DirectiveList = append(pkg.DirectiveList, d)
}

// bare returns the floor: three sentinels, no error types, with prefix set to
// value when non-empty.
func bare(prefix string) *storefixture.Builder {
	b := storefixture.New().Package("cfg", "example.com/cfg")
	for _, name := range []string{"ErrEmpty", "ErrFull", "ErrInvalid"} {
		b.Variable(name, func(v *storefixture.VariableBuilder) {
			v.Pos(position.At("cfg/errors.go", 1, 1))
		})
	}
	dir := storefixture.Directive("sentinel")
	switch prefix {
	case "":
	case "=empty":
		dir = storefixture.Directive("sentinel", storefixture.KV("prefix", ""))
	default:
		dir = storefixture.Directive("sentinel", storefixture.KV("prefix", prefix))
	}
	annotate(b, dir)
	return b
}

// typesOnly declares an error type and no sentinel, so the anchor the output
// is named after has to come from the type rather than from a variable.
func typesOnly() *storefixture.Builder {
	b := storefixture.New().Package("cfg", "example.com/cfg")
	b.Struct("PlainError", func(s *storefixture.StructBuilder) {
		s.Pos(position.At("cfg/errors.go", 1, 1))
		s.Field("Detail", storefixture.Named("string"), nil)
	})
	methods(b, "PlainError", "Error", "Is")
	annotate(b, storefixture.Directive("sentinel"))
	return b
}

// rich adds the three error-type shapes: one declaring Is, one declaring Unwrap
// with a cause, and one declaring neither.
func rich() *storefixture.Builder {
	b := bare("")
	b.Struct("NotFoundError", func(s *storefixture.StructBuilder) {
		s.Pos(position.At("cfg/errors.go", 1, 1))
		s.Field("Key", storefixture.Named("string"), nil)
		s.Field("Attempts", storefixture.Named("int"), nil)
	})
	b.Struct("WrappedError", func(s *storefixture.StructBuilder) {
		s.Pos(position.At("cfg/errors.go", 1, 1))
		s.Field("Cause", storefixture.Named("error"), nil)
	})
	b.Struct("PlainError", func(s *storefixture.StructBuilder) {
		s.Pos(position.At("cfg/errors.go", 1, 1))
		s.Field("Detail", storefixture.Named("string"), nil)
	})
	methods(b, "NotFoundError", "Error", "Is")
	methods(b, "WrappedError", "Error", "Unwrap")
	methods(b, "PlainError", "Error")
	return b
}

// neighbouring returns a package declaring non-overlap with another that the
// same run also loaded.
func neighbouring() *storefixture.Builder {
	b := bare("")
	pkg := b.PackageNode()
	pkg.Variables = append(pkg.Variables, &node.Variable{
		Name: "ErrGone", Package: "example.com/other",
	})
	annotate(b, storefixture.Directive("sentinel-no-overlap-with",
		storefixture.Arg("example.com/other")))
	return b
}

// methods attaches pointer-receiver methods to a named struct, which is the
// receiver form a custom error conventionally uses.
func methods(b *storefixture.Builder, name string, names ...string) {
	for _, s := range b.PackageNode().Structs {
		if s.Name != name {
			continue
		}
		for _, m := range names {
			s.Methods = append(s.Methods, &node.Method{
				Name:     m,
				Receiver: storefixture.Pointer(storefixture.Named(name)),
			})
		}
	}
}
