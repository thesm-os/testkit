// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

import "embed"

//go:embed templates/*.tmpl
var templateFS embed.FS

// TemplateFS exposes the embedded template directory for use by the
// pipeline and by per-partial template tests.
func TemplateFS() embed.FS { return templateFS }
