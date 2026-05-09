// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/stub"
)

func loadFixture(t *testing.T, name string) *generator.Package {
	t.Helper()
	pkg, err := generator.NewLoader().Load("./../testdata/"+name, "")
	testkit.NoError(t, err, "Load testdata/"+name)
	return pkg
}

// analyzeAndProject is the standard helper for unit-testing [stub.Data]
// methods. It runs the full Analyze → Enrich → Project pipeline and
// returns the populated Data.
func analyzeAndProject(t *testing.T, fixture, iface string) *stub.Data {
	t.Helper()
	pkg := loadFixture(t, fixture)
	d, err := stub.Analyze(pkg, []string{iface}, generator.DefaultConfig(), generator.Options{
		Output: "test/out.gen.go",
	})
	testkit.NoError(t, err, "Analyze")
	testkit.NoError(t, stub.Enrich(d, pkg), "Enrich")
	testkit.NoError(t, stub.Project(d, pkg), "Project")
	return d
}

// methodByName finds a MethodView by name in d.Methods. Fails the
// test if not found.
func methodByName(t *testing.T, d *stub.Data, name string) *stub.MethodView {
	t.Helper()
	for i := range d.Methods {
		if d.Methods[i].Name == name {
			return &d.Methods[i]
		}
	}
	t.Fatalf("method %q not found in %d methods", name, len(d.Methods))
	return nil
}

func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Name returns subcommand", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, (&stub.Generator{}).Name(), stub.Name, "Name")
	})

	t.Run("Generate emits two files: impl + auto-test", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		testkit.Len(t, res.Files, 2, "impl + test")

		paths := make([]string, len(res.Files))
		for i, f := range res.Files {
			paths[i] = f.Path
		}
		testkit.Equal(t, paths[0], "storetest/store.gen.go", "impl path")
		testkit.True(t, slices.Contains(paths, "storetest/store.gen_test.go"),
			"test path is auto-derived via TestPathFrom")
	})

	t.Run("impl emits StubX type, per-method stub, dispatch, fault helpers", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		impl := string(res.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("type StoreStub struct", "stub type emitted").
			Contains("*StoreStubGet", "per-method stub field for Get").
			Contains("*StoreStubPut", "per-method stub field for Put").
			Contains("type StoreStubGetCall struct", "Get's Call struct (stub-prefixed for collision-safety)").
			Contains("type StoreStubPutCall struct", "Put's Call struct (stub-prefixed)").
			Contains("type StoreStubGet struct", "per-method stub type").
			Contains("*stub.MethodStub[StoreStubGetCall]",
				"per-method stub embeds runtime MethodStub").
			Contains("func NewStoreStub", "constructor").
			Contains("func StoreStubDelegateTo", "DelegateTo option").
			Contains("func WithStoreGet", "constructor option per method").
			Contains("func (s *StoreStub) Get(ctx context.Context, key string) (basic.Item, error)",
				"Get implements interface").
			Contains("s.OnGet.Record(call)", "Record on dispatch").
			Contains("s.OnGet.FailUnexpectedCall(call)", "fail-unexpected path").
			Contains("func (s *StoreStubGet) FaultNotFound()",
				"sentinel fault helper from //testkit:errors")
	})

	t.Run("test file lives in sibling _test pkg", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		var test string
		for _, f := range res.Files {
			if strings.HasSuffix(f.Path, "_test.go") {
				test = string(f.Content)
			}
		}
		testkit.Assert(t, test).
			Contains("package storetest_test", "external _test package").
			Contains("storetest.NewStoreStub", "calls go through GenQualifier")
	})

	t.Run("non-interface arg surfaces a hard error", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := (&stub.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{
				Output: "x.gen.go",
			})
		testkit.True(t, err != nil, "non-interface rejected")
	})

	t.Run("AllShapes emits all shape stubs and directive helpers", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "interfaces")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"AllShapes"},
			generator.DefaultConfig(), generator.Options{
				Output: "allshapestest/allshapes.gen.go",
			})
		testkit.NoError(t, err, "Generate AllShapes")
		impl := string(res.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("type AllShapesStub struct", "stub type").
			Contains("AllShapesStubGet", "Reader stub").
			Contains("AllShapesStubReset", "VoidLifecycle stub").
			Contains("AllShapesStubTouch", "Mutator stub").
			Contains("AllShapesStubScan", "StreamReader stub").
			Contains("FaultNotFound", "sentinel from //testkit:errors on Get")
	})

	t.Run("Directives emits order-after, partition, retry, deprecated, wrapped-via", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "interfaces")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"Directives"},
			generator.DefaultConfig(), generator.Options{
				Output: "directivestest/directives.gen.go",
			})
		testkit.NoError(t, err, "Generate Directives")
		impl := string(res.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("OrderTracker", "order-after drives OrderTracker field").
			Contains("FaultNotFound", "sentinel from //testkit:errors on Submit").
			Contains("FaultConflict", "second sentinel on Submit").
			Contains("FaultForbidden", "sentinel on Wrap").
			Contains("RetrySchedule", "retry-succeeds-on-attempt helper").
			Contains("Deprecated", "deprecated annotation on Legacy").
			Contains("FaultForPartition", "partition helper on Shard")
	})

	t.Run("generic interface emits type-parameterized stub with concrete test instantiation", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "generics")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"Holder"},
			generator.DefaultConfig(), generator.Options{
				Output: "genericstest/holder.gen.go",
			})
		testkit.NoError(t, err, "Generate Holder")
		impl := string(res.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("HolderStub[V", "generic stub type").
			Contains("HolderStubGetCall[V", "generic call struct")

		// Holder[V any] → position 0 selects "string" from defaults.
		test := string(res.Files[1].Content)
		testkit.Assert(t, test).
			Contains("[string]", "concrete instantiation for auto-test")
	})

	t.Run("two-param generic uses concrete types in test", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "generics")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"KeyMap"},
			generator.DefaultConfig(), generator.Options{
				Output: "genericstest/keymap.gen.go",
			})
		testkit.NoError(t, err, "Generate KeyMap")
		// KeyMap[K comparable, V any] → position 0 = string,
		// position 1 = int from the round-robin defaults.
		test := string(res.Files[1].Content)
		testkit.Assert(t, test).
			Contains("[string, int]", "two concrete type args in test")
	})
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	t.Run("error from spec.Analyze propagates", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := stub.Analyze(pkg, []string{"DoesNotExist"}, generator.DefaultConfig(), generator.Options{
			Output: "x.gen.go",
		})
		testkit.True(t, err != nil, "nonexistent interface rejected")
	})

	t.Run("StubName derived from interface name", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.StubName, "StoreStub", "StubName")
	})

	t.Run("TestTypeArgs empty for non-generic", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.TestTypeArgs, "", "non-generic has no TestTypeArgs")
	})

	t.Run("TestTypeArgs populated for generic", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "generics", "Holder")
		testkit.True(t, d.TestTypeArgs != "", "generic has TestTypeArgs")
		testkit.True(t, strings.HasPrefix(d.TestTypeArgs, "["),
			"TestTypeArgs starts with bracket")
	})
}

func TestProject(t *testing.T) {
	t.Parallel()

	t.Run("HasErrorMethod true when any method returns error", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.True(t, d.HasErrorMethod, "Store.Get returns error")
	})

	t.Run("HasOrderConstraint true when order-after present", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "Directives")
		testkit.True(t, d.HasOrderConstraint,
			"Directives.Read has //testkit:order-after Open")
	})

	t.Run("Methods populated with correct count", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Len(t, d.Methods, 2, "Store has Get + Put")
	})
}

func TestDataHelpers(t *testing.T) {
	t.Parallel()

	t.Run("FirstContextMethod", func(t *testing.T) {
		t.Parallel()

		t.Run("returns first non-skip method with context", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "basic", "Store")
			m := d.FirstContextMethod()
			testkit.True(t, m != nil, "Store has context methods")
			testkit.True(t, m.HasContext(), "returned method takes ctx")
		})

		t.Run("returns nil when no context methods exist", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "basic", "Minimal")
			m := d.FirstContextMethod()
			testkit.True(t, m == nil,
				"Minimal has only void-no-ctx methods → nil")
		})
	})

	t.Run("FirstErrorMethod", func(t *testing.T) {
		t.Parallel()

		t.Run("returns first non-skip error method", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "basic", "Store")
			m := d.FirstErrorMethod()
			testkit.True(t, m != nil, "Store has error methods")
			testkit.True(t, m.ReturnsError(), "returned method returns error")
		})

		t.Run("returns nil when no error methods exist", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "basic", "Minimal")
			m := d.FirstErrorMethod()
			testkit.True(t, m == nil,
				"Minimal has only void methods → nil")
		})
	})

	t.Run("FirstNonSkipMethod", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a non-skip method", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "interfaces", "Directives")
			m := d.FirstNonSkipMethod()
			testkit.True(t, m != nil, "Directives has non-skip methods")
			testkit.True(t, !m.Skip(), "returned method is not skipped")
		})
	})

	t.Run("FirstNonSkipMethodWithSampleableResults", func(t *testing.T) {
		t.Parallel()

		t.Run("skips void methods and iterators", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "interfaces", "AllShapes")
			m := d.FirstNonSkipMethodWithSampleableResults()
			testkit.True(t, m != nil, "AllShapes has sampleable methods")
			testkit.True(t, m.HasAssertableNonErrorResults(),
				"returned method has assertable results")
		})

		t.Run("returns nil when no sampleable results exist", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "basic", "Minimal")
			m := d.FirstNonSkipMethodWithSampleableResults()
			testkit.True(t, m == nil,
				"Minimal has only void methods → nil")
		})
	})

	t.Run("QualifiedTypeForTest", func(t *testing.T) {
		t.Parallel()

		t.Run("non-generic returns QualifiedType as-is", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "basic", "Store")
			got := d.QualifiedTypeForTest()
			testkit.Equal(t, got, d.Spec.QualifiedType,
				"non-generic: QualifiedTypeForTest == QualifiedType")
		})

		t.Run("generic substitutes concrete type args", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "generics", "Holder")
			got := d.QualifiedTypeForTest()
			testkit.True(t, strings.Contains(got, "generics.Holder"),
				"contains package-qualified name")
			testkit.True(t, !strings.Contains(got, d.Spec.TypeParamArgs),
				"type-param names replaced")
			testkit.True(t, strings.Contains(got, d.TestTypeArgs),
				"concrete test type args substituted")
		})

		t.Run("two-param generic substitutes both args", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "generics", "KeyMap")
			got := d.QualifiedTypeForTest()
			testkit.True(t, strings.Contains(got, "generics.KeyMap"),
				"contains package-qualified name")
			testkit.True(t, strings.Contains(got, d.TestTypeArgs),
				"concrete test type args substituted")
		})
	})
}

func TestMethodView(t *testing.T) {
	t.Parallel()

	t.Run("naming conventions", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")

		t.Run("StubType", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, get.StubType(), "StoreStubGet", "StubType")
		})
		t.Run("QualStubType non-generic", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, get.QualStubType(), "StoreStubGet",
				"QualStubType has no suffix for non-generic")
		})
		t.Run("CallType", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, get.CallType(), "StoreStubGetCall", "CallType")
		})
		t.Run("QualCallType non-generic", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, get.QualCallType(), "StoreStubGetCall",
				"QualCallType has no suffix for non-generic")
		})
	})

	t.Run("ReturnType and QualReturnType", func(t *testing.T) {
		t.Parallel()

		t.Run("method with results", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "basic", "Store")
			get := methodByName(t, d, "Get")
			testkit.Equal(t, get.ReturnType(), "storeStubGetReturn",
				"ReturnType is lowercased stub+method+Return")
			testkit.Equal(t, get.QualReturnType(), "storeStubGetReturn",
				"QualReturnType has no suffix for non-generic")
		})

		t.Run("void method returns empty", func(t *testing.T) {
			t.Parallel()
			d := analyzeAndProject(t, "interfaces", "AllShapes")
			touch := methodByName(t, d, "Touch")
			testkit.Equal(t, touch.ReturnType(), "",
				"ReturnType empty for void method")
			testkit.Equal(t, touch.QualReturnType(), "",
				"QualReturnType empty for void method")
		})
	})

	t.Run("Skip reports integration-only", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "Directives")

		closeM := methodByName(t, d, "Close")
		testkit.True(t, closeM.Skip(), "Close is integration-only")

		open := methodByName(t, d, "Open")
		testkit.True(t, !open.Skip(), "Open is not skipped")
	})

	t.Run("directive accessors on Directives fixture", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "Directives")

		// Convenience lookup; methodByName fatals on miss so each
		// subtest is safe to dereference.
		byName := make(map[string]*stub.MethodView, len(d.Methods))
		for i := range d.Methods {
			byName[d.Methods[i].Name] = &d.Methods[i]
		}
		// Sanity: confirm the expected methods exist.
		for _, name := range []string{"Open", "Close", "Submit", "Wrap", "Legacy", "Retry", "Read", "Shard"} {
			testkit.True(t, byName[name] != nil, name+" method found")
		}

		t.Run("Errors returns sentinels", func(t *testing.T) {
			t.Parallel()
			errs := byName["Submit"].Errors()
			testkit.True(t, len(errs) >= 2,
				"Submit has //testkit:errors ErrNotFound ErrConflict")
		})

		t.Run("Errors empty when not set", func(t *testing.T) {
			t.Parallel()
			errs := byName["Open"].Errors()
			testkit.Len(t, errs, 0, "Open has no //testkit:errors")
		})

		t.Run("Deprecated returns replacement", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, byName["Legacy"].Deprecated(), "Submit",
				"Legacy is deprecated in favor of Submit")
		})

		t.Run("Deprecated empty when not set", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, byName["Open"].Deprecated(), "",
				"Open is not deprecated")
		})

		t.Run("RetryN returns attempt count", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, byName["Retry"].RetryN(), 3,
				"Retry has //testkit:retry-succeeds-on-attempt 3")
		})

		t.Run("RetryN zero when not set", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, byName["Open"].RetryN(), 0,
				"Open has no retry")
		})

		t.Run("OrderAfter returns prerequisite", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, byName["Read"].OrderAfter(), "Open",
				"Read has //testkit:order-after Open")
		})

		t.Run("OrderAfter empty when not set", func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, byName["Open"].OrderAfter(), "",
				"Open has no order-after")
		})

		t.Run("Partition returns payload", func(t *testing.T) {
			t.Parallel()
			p := byName["Shard"].Partition()
			testkit.True(t, p != nil,
				"Shard has //testkit:partition ID")
		})

		t.Run("Partition nil when not set", func(t *testing.T) {
			t.Parallel()
			testkit.True(t, byName["Open"].Partition() == nil,
				"Open has no partition")
		})

		t.Run("WrappedVia returns payload", func(t *testing.T) {
			t.Parallel()
			w := byName["Submit"].WrappedVia()
			testkit.True(t, w != nil,
				"Submit has //testkit:wrapped-via ErrInternal")
		})

		t.Run("WrappedVia nil when not set", func(t *testing.T) {
			t.Parallel()
			testkit.True(t, byName["Open"].WrappedVia() == nil,
				"Open has no wrapped-via")
		})
	})

	t.Run("generic method naming includes type args", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "generics", "KeyMap")
		get := methodByName(t, d, "Get")
		testkit.True(t, strings.Contains(get.QualStubType(), "["),
			"QualStubType includes type args for generic")
		testkit.True(t, strings.Contains(get.QualCallType(), "["),
			"QualCallType includes type args for generic")
		testkit.True(t, strings.Contains(get.QualReturnType(), "["),
			"QualReturnType includes type args for generic")
	})
}
