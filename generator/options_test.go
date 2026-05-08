// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("DefaultConfig has expected values", func(t *testing.T) {
		t.Parallel()
		c := generator.DefaultConfig()
		testkit.Equal(t, c.TestPackageSuffix, generator.DefaultTestPackageSuffix, "TestPackageSuffix")
		testkit.Equal(t, c.GeneratedSuffix, generator.DefaultGeneratedSuffix, "GeneratedSuffix")
		testkit.Equal(t, c.TestPackageStyle, generator.DefaultTestPackageStyle, "TestPackageStyle")
		testkit.Equal(t, c.Stub.FilePattern, generator.DefaultStubFilePattern, "Stub.FilePattern")
		testkit.Equal(t, c.Stub.TypeSuffix, generator.DefaultStubTypeSuffix, "Stub.TypeSuffix")
	})

	t.Run("external and internal styles are distinct", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, generator.TestPackageStyleExternal != generator.TestPackageStyleInternal,
			"styles must differ")
	})
}
