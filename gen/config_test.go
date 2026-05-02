// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"testing"

	"go.thesmos.sh/testkit"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("has expected defaults", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		testkit.Equal(t, cfg.TestPackageSuffix, "test", "suffix default")
		testkit.Equal(t, cfg.GeneratedSuffix, ".gen.go", "generated suffix default")
		testkit.Equal(t, cfg.TestPackageStyle, "external", "test package style default")
	})
}
