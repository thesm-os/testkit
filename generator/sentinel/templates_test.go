// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel_test

import (
	"bytes"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/sentinel"
)

// render parses every .tmpl in the embedded FS and executes the named
// partial against data, returning the rendered output. Wrapping the
// boilerplate keeps each subtest focused on "this template should
// emit these markers."
func render(t *testing.T, name string, data any) string {
	t.Helper()
	tmpl, err := generator.NewTemplateSet().ParseFS(sentinel.TemplateFS(), "templates/*.tmpl")
	testkit.NoError(t, err, "ParseFS")
	var buf bytes.Buffer
	testkit.NoError(t, tmpl.ExecuteTemplate(&buf, name, data), "ExecuteTemplate "+name)
	return buf.String()
}

// sampleData builds a [sentinel.Data] populated enough to exercise
// every partial. Two error types — one with Is, one with Unwrap —
// and two sentinel vars.
func sampleData() *sentinel.Data {
	return &sentinel.Data{
		PackageName: "basic_test",
		ImportPath:  "example.com/basic",
		Qualifier:   "basic.",
		TestName:    "BasicSentinelErrors",
		Prefix:      "basic: ",
		Sentinels: []sentinel.ErrorVar{
			{Name: "ErrNotFound"},
			{Name: "ErrConflict"},
		},
		ErrorTypes: []sentinel.ErrorType{
			{
				Name:      "NotFoundError",
				Qualifier: "basic.",
				HasIs:     true,
				Fields: []sentinel.FieldData{
					{Name: "Entity", TypeStr: "string", SampleValue: `"test-entity"`, FormatCheckValue: "test-entity"},
				},
				OtherTypes:       []string{"WrappedError"},
				FormatCheckOrder: []string{"test-entity"},
			},
			{
				Name:        "WrappedError",
				Qualifier:   "basic.",
				HasUnwrap:   true,
				UnwrapField: "Cause",
				Fields: []sentinel.FieldData{
					{Name: "Msg", TypeStr: "string", SampleValue: `"test-msg"`, FormatCheckValue: "test-msg"},
					{
						Name: "Cause", TypeStr: "error",
						SampleValue:      `errors.New("test-cause")`,
						FormatCheckValue: "test-cause",
						IsError:          true,
					},
				},
				OtherTypes:       []string{"NotFoundError"},
				FormatCheckOrder: []string{"test-msg", "test-cause"},
			},
		},
	}
}

func TestTemplateFS(t *testing.T) {
	t.Parallel()

	t.Run("embeds all six partial templates", func(t *testing.T) {
		t.Parallel()
		entries, err := sentinel.TemplateFS().ReadDir("templates")
		testkit.NoError(t, err, "ReadDir templates/")
		want := map[string]bool{
			"sentinel.go.tmpl":          false,
			"header.go.tmpl":            false,
			"sentinel_vars.go.tmpl":     false,
			"error_type.go.tmpl":        false,
			"error_type_is.go.tmpl":     false,
			"error_type_unwrap.go.tmpl": false,
		}
		for _, e := range entries {
			if _, ok := want[e.Name()]; ok {
				want[e.Name()] = true
			}
		}
		for name, found := range want {
			testkit.True(t, found, "missing template: "+name)
		}
	})

	t.Run("ParseFS succeeds for the full set", func(t *testing.T) {
		t.Parallel()
		_, err := generator.NewTemplateSet().ParseFS(sentinel.TemplateFS(), "templates/*.tmpl")
		testkit.NoError(t, err, "ParseFS")
	})
}

func TestPartial_Header(t *testing.T) {
	t.Parallel()

	t.Run("emits package + canonical imports", func(t *testing.T) {
		t.Parallel()
		out := render(t, "header", sampleData())
		testkit.Assert(t, out).
			Contains("package basic_test", "package decl").
			Contains(`"errors"`, "errors import").
			Contains(`"strings"`, "strings import").
			Contains(`"testing"`, "testing import").
			Contains(`"go.thesmos.sh/testkit"`, "testkit import").
			Contains(`"example.com/basic"`, "source pkg import")
	})

	t.Run("emits doc comment summarizing what the file tests", func(t *testing.T) {
		t.Parallel()
		out := render(t, "header", sampleData())
		testkit.Assert(t, out).
			Contains("// This file verifies", "doc lead-in").
			Contains(`sentinels share a "basic: " prefix`, "prefix bullet uses Data.Prefix").
			Contains("Is/Unwrap methods", "method-coverage bullet")
	})

	t.Run("emits Directives consumed block when directives are set", func(t *testing.T) {
		t.Parallel()
		data := sampleData()
		data.Directives = []string{"//testkit:sentinel-no-overlap-with example.com/peer"}
		out := render(t, "header", data)
		testkit.Assert(t, out).
			Contains("Directives consumed:", "directives heading").
			Contains("//testkit:sentinel-no-overlap-with example.com/peer", "directive line")
	})

	t.Run("omits Directives consumed block when none apply", func(t *testing.T) {
		t.Parallel()
		out := render(t, "header", sampleData()) // sampleData has no Directives
		testkit.Assert(t, out).
			NotContains("Directives consumed:", "no directives → no heading")
	})

	t.Run("emits cross-package bullet only when CrossPackages is set", func(t *testing.T) {
		t.Parallel()
		without := render(t, "header", sampleData())
		testkit.Assert(t, without).
			NotContains("do not alias any peer package", "no cross-pkg bullet without peers")

		with := *sampleData()
		with.CrossPackages = []sentinel.CrossPackage{{ImportPath: "example.com/peer", Alias: "peer"}}
		testkit.Assert(t, render(t, "header", &with)).
			Contains("do not alias any peer package", "cross-pkg bullet with peers")
	})

	t.Run("imports fmt only when error types are present", func(t *testing.T) {
		t.Parallel()
		empty := *sampleData()
		empty.ErrorTypes = nil
		testkit.Assert(t, render(t, "header", &empty)).
			NotContains(`"fmt"`, "no fmt without error types")

		full := sampleData()
		testkit.Assert(t, render(t, "header", full)).
			Contains(`"fmt"`, "fmt imported when error types present")
	})

	t.Run("source-pkg import omitted when same-package", func(t *testing.T) {
		t.Parallel()
		samePkg := *sampleData()
		samePkg.ImportPath = ""
		testkit.Assert(t, render(t, "header", &samePkg)).
			NotContains("example.com/basic", "no extra import for same-pkg output")
	})
}

func TestPartial_SentinelVars(t *testing.T) {
	t.Parallel()

	t.Run("emits TestXxxSentinelErrors with all five subtests", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, render(t, "sentinel-vars", sampleData())).
			Contains("func TestBasicSentinelErrors", "top-level test fn").
			Contains(`"prefix"`, "prefix subtest").
			Contains(`"uniqueness"`, "uniqueness subtest").
			Contains(`"non-overlap"`, "non-overlap subtest").
			Contains(`"errors.Join chain"`, "errors.Join chain subtest").
			Contains(`"fmt.Errorf chain"`, "fmt.Errorf chain subtest").
			Contains(`{"ErrNotFound", basic.ErrNotFound}`, "qualified entry").
			Contains(`{"ErrConflict", basic.ErrConflict}`, "qualified entry")
	})

	t.Run("emits doc comment + per-subtest comments", func(t *testing.T) {
		t.Parallel()
		out := render(t, "sentinel-vars", sampleData())
		testkit.Assert(t, out).
			Contains("// TestBasicSentinelErrors verifies", "function doc").
			Contains(`// "prefix":`, "prefix subtest doc").
			Contains(`// "uniqueness":`, "uniqueness subtest doc").
			Contains(`// "non-overlap":`, "non-overlap subtest doc").
			Contains(`// "errors.Join chain":`, "errors.Join subtest doc").
			Contains(`// "fmt.Errorf chain":`, "fmt.Errorf subtest doc")
	})

	t.Run("uses bare names for source-pkg refs when same package", func(t *testing.T) {
		t.Parallel()
		samePkg := *sampleData()
		samePkg.Qualifier = ""
		got := render(t, "sentinel-vars", &samePkg)
		testkit.Assert(t, got).
			Contains(`{"ErrNotFound", ErrNotFound}`, "no qualifier when same pkg")
	})
}

func TestPartial_ErrorType(t *testing.T) {
	t.Parallel()

	t.Run("emits the four always-present subtests", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[1] // WrappedError, has IsError field
		out := render(t, "error-type", &et)
		testkit.Assert(t, out).
			Contains("func TestWrappedError", "top-level test fn").
			Contains(`"errors.As extracts type"`, "errors.As subtest").
			Contains(`"survives errors.Join wrapping"`, "errors.Join subtest").
			Contains(`"survives fmt.Errorf wrapping"`, "fmt.Errorf subtest").
			Contains(`"Error format includes all fields in source order"`, "format subtest").
			Contains("ContainsInOrder", "ContainsInOrder strictness check")
	})

	t.Run("includes Is partial when HasIs is true", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[0] // NotFoundError (HasIs)
		testkit.Assert(t, render(t, "error-type", &et)).
			Contains(`"Is matches same type"`, "Is partial included")
	})

	t.Run("includes Unwrap partial when HasUnwrap is true", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[1] // WrappedError (HasUnwrap)
		out := render(t, "error-type", &et)
		testkit.Assert(t, out).
			Contains(`"Unwrap returns cause"`, "Unwrap partial included").
			Contains(`"errors.Is traverses Unwrap chain"`, "errors.Is chain subtest")
	})

	t.Run("emits doc comment + per-subtest comments", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[1] // WrappedError, has Is + Unwrap
		out := render(t, "error-type", &et)
		testkit.Assert(t, out).
			Contains("// TestWrappedError verifies", "function doc").
			Contains("// errors.As must extract the concrete error type", "errors.As subtest doc").
			Contains("errors.Join multi-error", "Join subtest doc").
			Contains("canonical Go wrapping idiom", "fmt.Errorf subtest doc").
			Contains("source declaration order", "format subtest doc").
			Contains("Each custom error type must be distinct", "non-overlap subtest doc")
	})

	t.Run("error-typed field branches use ErrorIs not Equal", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[1] // WrappedError, has IsError field Cause
		out := render(t, "error-type", &et)
		testkit.Assert(t, out).
			Contains("testkit.ErrorIs(t, target.Cause", "Cause uses ErrorIs").
			Contains("testkit.Equal(t, target.Msg", "Msg uses Equal")
	})
}

func TestPartial_ErrorTypeIs(t *testing.T) {
	t.Parallel()

	t.Run("emits all three Is subtests when fields are present", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[0] // NotFoundError, has fields
		testkit.Assert(t, render(t, "error-type-is", &et)).
			Contains(`"Is matches same type"`, "matches same").
			Contains(`"Is matches across instances with different fields"`, "matches across instances").
			Contains(`"Is rejects different error types"`, "rejects different")
	})

	t.Run("emits per-subtest comments", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[0]
		out := render(t, "error-type-is", &et)
		testkit.Assert(t, out).
			Contains("type-level equivalence", "matches-same comment").
			Contains("not be field-sensitive", "across-instances comment").
			Contains("must reject foreign error types", "rejects comment")
	})

	t.Run("skips per-instance subtest when type has no fields", func(t *testing.T) {
		t.Parallel()
		et := sentinel.ErrorType{Name: "Empty", Qualifier: "basic.", HasIs: true}
		testkit.Assert(t, render(t, "error-type-is", &et)).
			Contains(`"Is matches same type"`, "matches same").
			NotContains(`"Is matches across instances with different fields"`, "no per-instance subtest").
			Contains(`"Is rejects different error types"`, "rejects different")
	})
}

func TestPartial_ErrorTypeUnwrap(t *testing.T) {
	t.Parallel()

	t.Run("emits both Unwrap subtests with cause assertions", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[1] // WrappedError with UnwrapField=Cause
		out := render(t, "error-type-unwrap", &et)
		testkit.Assert(t, out).
			Contains(`"Unwrap returns cause"`, "first subtest").
			Contains(`"errors.Is traverses Unwrap chain"`, "second subtest").
			Contains("Cause: cause", "Cause field populated").
			Contains("Cause: sentinel", "sentinel populated")
	})

	t.Run("emits per-subtest comments", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[1]
		out := render(t, "error-type-unwrap", &et)
		testkit.Assert(t, out).
			Contains("must return the cause field's value", "Unwrap-cause comment").
			Contains("must traverse the Unwrap chain", "errors.Is chain comment").
			Contains("zero-value error must return nil from Unwrap", "Unwrap-nil comment")
	})

	t.Run("emits Unwrap-nil subtest when UnwrapField is identified", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[1] // WrappedError with UnwrapField=Cause
		out := render(t, "error-type-unwrap", &et)
		testkit.Assert(t, out).
			Contains(`"Unwrap returns nil when cause is unset"`, "Unwrap-nil subtest").
			Contains("err.Unwrap() == nil", "asserts nil on zero-value error")
	})

	t.Run("omits cause-field assignments when no UnwrapField is identified", func(t *testing.T) {
		t.Parallel()
		et := sentinel.ErrorType{Name: "BareUnwrap", Qualifier: "basic.", HasUnwrap: true}
		out := render(t, "error-type-unwrap", &et)
		testkit.Assert(t, out).
			NotContains("Cause: cause", "no field write without UnwrapField").
			NotContains("testkit.ErrorIs(t, got, cause", "no cause assertion either").
			NotContains(`"Unwrap returns nil when cause is unset"`, "Unwrap-nil subtest gated on UnwrapField")
	})
}

func TestPartial_ErrorTypeOtherTypes(t *testing.T) {
	t.Parallel()

	t.Run("emits per-other-type non-overlap subtests", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[0] // NotFoundError, OtherTypes=[WrappedError]
		out := render(t, "error-type", &et)
		testkit.Assert(t, out).
			Contains(`"does not match other error types in package"`, "outer subtest").
			Contains(`"not WrappedError"`, "per-other subtest").
			Contains("&basic.WrappedError{}", "uses zero-value other type")
	})

	t.Run("omits the section when there are no other types", func(t *testing.T) {
		t.Parallel()
		et := sentinel.ErrorType{
			Name:      "Solo",
			Qualifier: "basic.",
		}
		out := render(t, "error-type", &et)
		testkit.Assert(t, out).
			NotContains("does not match other error types in package",
				"no section without OtherTypes")
	})
}

func TestPartial_ErrorTypeFormatStrictness(t *testing.T) {
	t.Parallel()

	t.Run("emits ContainsInOrder when FormatCheckOrder is non-empty", func(t *testing.T) {
		t.Parallel()
		et := sampleData().ErrorTypes[1] // WrappedError, FormatCheckOrder=[test-msg, test-cause]
		out := render(t, "error-type", &et)
		testkit.Assert(t, out).
			Contains("ContainsInOrder", "uses ordered assertion").
			Contains(`"test-msg"`, "first marker").
			Contains(`"test-cause"`, "second marker")
	})

	t.Run("falls back to non-empty assertion when no FormatCheckOrder", func(t *testing.T) {
		t.Parallel()
		et := sentinel.ErrorType{
			Name:      "Naked",
			Qualifier: "basic.",
			Fields: []sentinel.FieldData{
				{Name: "Code", TypeStr: "int", SampleValue: "42"},
			},
		}
		out := render(t, "error-type", &et)
		testkit.Assert(t, out).
			NotContains("ContainsInOrder", "no order check without identifiable fields").
			Contains(`IsNotEmpty("Error() must return non-empty string"`, "still asserts non-empty")
	})
}

func TestPartial_CrossPackage(t *testing.T) {
	t.Parallel()

	t.Run("emits cross-package non-overlap test function", func(t *testing.T) {
		t.Parallel()
		data := sampleData()
		data.CrossPackages = []sentinel.CrossPackage{
			{
				ImportPath: "example.com/storage",
				Alias:      "storage",
				Sentinels:  []sentinel.ErrorVar{{Name: "ErrMissing"}},
			},
		}
		out := render(t, "cross-pkg", map[string]any{
			"Local": data,
			"Peer":  data.CrossPackages[0],
		})
		testkit.Assert(t, out).
			Contains("func TestBasicSentinelErrorsNoOverlapWithStorage", "test fn").
			Contains(`{"ErrNotFound", basic.ErrNotFound}`, "local entry").
			Contains(`{"ErrMissing", storage.ErrMissing}`, "peer entry").
			Contains("testkit.ErrorIsNot(t, l.err, p.err", "non-overlap assertion")
	})

	t.Run("emits doc comment naming the driving directive", func(t *testing.T) {
		t.Parallel()
		data := sampleData()
		data.CrossPackages = []sentinel.CrossPackage{
			{
				ImportPath: "example.com/storage",
				Alias:      "storage",
				Sentinels:  []sentinel.ErrorVar{{Name: "ErrMissing"}},
			},
		}
		out := render(t, "cross-pkg", map[string]any{
			"Local": data,
			"Peer":  data.CrossPackages[0],
		})
		testkit.Assert(t, out).
			Contains("// TestBasicSentinelErrorsNoOverlapWithStorage asserts", "function doc").
			Contains("Driven by:", "directive attribution").
			Contains("//testkit:sentinel-no-overlap-with example.com/storage", "directive line includes peer path").
			Contains("// One subtest per (local × peer) pair", "loop comment")
	})
}
