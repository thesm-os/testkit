// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("has expected defaults", func(t *testing.T) {
		t.Parallel()
		cfg := gen.DefaultConfig()
		testkit.Equal(t, cfg.TestPackageSuffix, "test", "suffix default")
		testkit.Equal(t, cfg.GeneratedSuffix, ".gen.go", "generated suffix default")
		testkit.Equal(t, cfg.TestPackageStyle, "external", "test package style default")
	})

	t.Run("has stub naming defaults", func(t *testing.T) {
		t.Parallel()
		cfg := gen.DefaultConfig()
		testkit.Equal(t, cfg.Stub.FilePattern, gen.DefaultStubFilePattern, "stub file pattern")
		testkit.Equal(t, cfg.Stub.TypeSuffix, gen.DefaultStubTypeSuffix, "stub type suffix")
	})
}
