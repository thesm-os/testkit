// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import "embed"

//go:embed templates/*.tmpl
var templateFS embed.FS

// TemplateFS exposes the embedded templates for per-partial
// regression tests in templates_test.go.
func TemplateFS() embed.FS { return templateFS }
