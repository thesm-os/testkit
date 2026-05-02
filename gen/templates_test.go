// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestTemplateFile(t *testing.T) {
	t.Parallel()

	t.Run("loads existing template", func(t *testing.T) {
		t.Parallel()
		content := gen.TemplateFile("builder.go.tmpl")
		testkit.Assert(t, content).
			IsNotEmpty("must return template content").
			Contains("package", "must contain Go package declaration")
	})

	t.Run("panics on missing template", func(t *testing.T) {
		t.Parallel()
		testkit.Panics(t, func() {
			gen.TemplateFile("nonexistent.tmpl")
		}, "must panic on missing template")
	})
}
