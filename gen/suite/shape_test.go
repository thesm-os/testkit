// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directive"
	"go.thesmos.sh/testkit/gen/suite"
)

func TestDetectShape(t *testing.T) {
	t.Parallel()

	t.Run("basic.Store", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must load Store")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method     string
			directives []gen.Directive
			wantShape  suite.MethodShape
		}{
			{"Get", nil, suite.ShapeReader},
			{"Put", nil, suite.ShapeWriter},
			{"Delete", nil, suite.ShapeWriter},
			{"Delete", []gen.Directive{{Name: directive.DirDeleter}}, suite.ShapeDeleter},
			{"Count", nil, suite.ShapePure},
			{"Ping", nil, suite.ShapeLifecycle},
			{"LegacyPut", nil, suite.ShapeWriter},
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
				info := suite.DetectShape(m, tracker, tc.directives)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})

	t.Run("basic.Store Reader key/val types", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
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
		info := suite.DetectShape(get, tracker, nil)
		testkit.Equal(t, info.Shape, suite.ShapeReader, "Get must be Reader")
		testkit.Assert(t, info.KeyType).Contains("string", "key type must be string")
		testkit.Assert(t, info.ValType).Contains("Item", "val type must be Item")
	})

	t.Run("iterators.Scanner", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "iterators")
		iface, err := pkg.Interface("Scanner")
		testkit.NoError(t, err, "must load Scanner")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method    string
			wantShape suite.MethodShape
		}{
			{"Keys", suite.ShapeStreamReader},
			{"Scan", suite.ShapeStreamReader},
			{"Count", suite.ShapeAggregator},
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
				info := suite.DetectShape(m, tracker, nil)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})

	t.Run("mixed.Processor", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "mixed")
		iface, err := pkg.Interface("Processor")
		testkit.NoError(t, err, "must load Processor")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method    string
			wantShape suite.MethodShape
		}{
			{"Run", suite.ShapeLifecycle},
			{"Process", suite.ShapeWriter},
			{"Describe", suite.ShapePure},
			{"LegacyProcess", suite.ShapeWriter},
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
				info := suite.DetectShape(m, tracker, nil)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})

	t.Run("nocontext.Cache", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "nocontext")
		iface, err := pkg.Interface("Cache")
		testkit.NoError(t, err, "must load Cache")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method    string
			wantShape suite.MethodShape
		}{
			{"Get", suite.ShapeUnknown},
			{"Set", suite.ShapeUnknown},
			{"Len", suite.ShapePure},
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
				info := suite.DetectShape(m, tracker, nil)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})

	t.Run("erroronly.Closer", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "erroronly")
		iface, err := pkg.Interface("Closer")
		testkit.NoError(t, err, "must load Closer")
		tracker := gen.NewImportTracker("example.com/test")

		for _, tc := range []struct {
			method    string
			wantShape suite.MethodShape
		}{
			{"Open", suite.ShapeLifecycle},
			{"Close", suite.ShapeLifecycle},
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
				info := suite.DetectShape(m, tracker, nil)
				testkit.Equal(t, info.Shape, tc.wantShape,
					tc.method+" must be "+tc.wantShape.String())
			})
		}
	})
}
