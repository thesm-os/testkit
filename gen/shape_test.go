// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directives"
)

func suiteTestdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	return filepath.Join(filepath.Dir(file), "suite", "testdata")
}

func loadSuiteTestPackage(t *testing.T, subdir string) *gen.Package {
	t.Helper()
	loader := gen.NewLoader()
	dir := filepath.Join(suiteTestdataDir(t), subdir)
	pkg, err := loader.Load(".", dir)
	testkit.NoError(t, err, "must load package")
	return pkg
}

func TestDetectShape(t *testing.T) {
	t.Parallel()

	t.Run("basic.Store", func(t *testing.T) {
		t.Parallel()
		pkg := loadSuiteTestPackage(t, "basic")
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must load Store")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method     string
			directives []gen.Directive
			wantShape  gen.MethodShape
		}{
			{"Get", nil, gen.ShapeReader},
			{"Put", nil, gen.ShapeWriter},
			{"Delete", nil, gen.ShapeWriter},
			{"Delete", []gen.Directive{{Name: directives.Deleter}}, gen.ShapeDeleter},
			{"Count", nil, gen.ShapeUnknown},
			{"Ping", nil, gen.ShapeLifecycle},
			{"LegacyPut", nil, gen.ShapeWriter},
		} {
			t.Run(tc.method, func(t *testing.T) {
				t.Parallel()
				var m gen.MethodInfo
				for _, method := range iface.Methods {
					if method.Name == tc.method {
						m = method
						break
					}
				}
				if m.Name == "" {
					t.Fatalf("method %s not found", tc.method)
				}
				info := gen.DetectShape(m, tracker, tc.directives)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})

	t.Run("basic.Store Reader key/val types", func(t *testing.T) {
		t.Parallel()
		pkg := loadSuiteTestPackage(t, "basic")
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must load Store")
		tracker := gen.NewImportTracker("example.com/test")

		var get gen.MethodInfo
		for _, m := range iface.Methods {
			if m.Name == "Get" {
				get = m
				break
			}
		}
		info := gen.DetectShape(get, tracker, nil)
		testkit.Equal(t, info.Shape, gen.ShapeReader, "Get must be Reader")
		testkit.Assert(t, info.KeyType).Contains("string", "key type must be string")
		testkit.Assert(t, info.ValType).Contains("Item", "val type must be Item")
	})

	t.Run("iterators.Scanner", func(t *testing.T) {
		t.Parallel()
		pkg := loadSuiteTestPackage(t, "iterators")
		iface, err := pkg.Interface("Scanner")
		testkit.NoError(t, err, "must load Scanner")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method    string
			wantShape gen.MethodShape
		}{
			{"Keys", gen.ShapeStreamReader},
			{"Scan", gen.ShapeStreamReader},
			{"Count", gen.ShapeAggregator},
		} {
			t.Run(tc.method, func(t *testing.T) {
				t.Parallel()
				var m gen.MethodInfo
				for _, method := range iface.Methods {
					if method.Name == tc.method {
						m = method
						break
					}
				}
				info := gen.DetectShape(m, tracker, nil)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})

	t.Run("mixed.Processor", func(t *testing.T) {
		t.Parallel()
		pkg := loadSuiteTestPackage(t, "mixed")
		iface, err := pkg.Interface("Processor")
		testkit.NoError(t, err, "must load Processor")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method    string
			wantShape gen.MethodShape
		}{
			{"Run", gen.ShapeLifecycle},
			{"Process", gen.ShapeWriter},
			{"Describe", gen.ShapePure},
			{"LegacyProcess", gen.ShapeWriter},
		} {
			t.Run(tc.method, func(t *testing.T) {
				t.Parallel()
				var m gen.MethodInfo
				for _, method := range iface.Methods {
					if method.Name == tc.method {
						m = method
						break
					}
				}
				info := gen.DetectShape(m, tracker, nil)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})

	t.Run("nocontext.Cache", func(t *testing.T) {
		t.Parallel()
		pkg := loadSuiteTestPackage(t, "nocontext")
		iface, err := pkg.Interface("Cache")
		testkit.NoError(t, err, "must load Cache")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method    string
			wantShape gen.MethodShape
		}{
			{"Get", gen.ShapeUnknown},
			{"Set", gen.ShapeUnknown},
			{"Len", gen.ShapePure},
		} {
			t.Run(tc.method, func(t *testing.T) {
				t.Parallel()
				var m gen.MethodInfo
				for _, method := range iface.Methods {
					if method.Name == tc.method {
						m = method
						break
					}
				}
				info := gen.DetectShape(m, tracker, nil)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})

	t.Run("voidctx.Counter", func(t *testing.T) {
		t.Parallel()
		pkg := loadSuiteTestPackage(t, "voidctx")
		iface, err := pkg.Interface("Counter")
		testkit.NoError(t, err, "must load Counter")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method    string
			wantShape gen.MethodShape
		}{
			{"Add", gen.ShapeUnknown},
			{"Name", gen.ShapePure},
		} {
			t.Run(tc.method, func(t *testing.T) {
				t.Parallel()
				var m gen.MethodInfo
				for _, method := range iface.Methods {
					if method.Name == tc.method {
						m = method
						break
					}
				}
				info := gen.DetectShape(m, tracker, nil)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})

	t.Run("erroronly.Closer", func(t *testing.T) {
		t.Parallel()
		pkg := loadSuiteTestPackage(t, "erroronly")
		iface, err := pkg.Interface("Closer")
		testkit.NoError(t, err, "must load Closer")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method    string
			wantShape gen.MethodShape
		}{
			{"Open", gen.ShapeLifecycle},
			{"Close", gen.ShapeLifecycle},
		} {
			t.Run(tc.method, func(t *testing.T) {
				t.Parallel()
				var m gen.MethodInfo
				for _, method := range iface.Methods {
					if method.Name == tc.method {
						m = method
						break
					}
				}
				info := gen.DetectShape(m, tracker, nil)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})
}
