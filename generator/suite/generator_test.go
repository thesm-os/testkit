// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/suite"
)

func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Name returns subcommand", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, (&suite.Generator{}).Name(), "suite", "Name")
	})

	t.Run("Generate returns port-pending error", func(t *testing.T) {
		t.Parallel()
		_, err := (&suite.Generator{}).Generate(nil, nil, generator.Config{}, generator.Options{})
		testkit.True(t, err != nil, "placeholder errors")
		testkit.Assert(t, err.Error()).
			Contains("port pending", "diagnostic mentions port status")
	})
}
