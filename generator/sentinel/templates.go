// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel

import "embed"

// templateFS holds the sentinel template files. Embedded at build
// time so the generator binary is self-contained.
//
//go:embed templates/*.tmpl
var templateFS embed.FS

// TemplateFS exposes the embedded template filesystem. Tests use this
// to assert templates load cleanly; the generator itself reads from
// templateFS directly via its [generator.Pipeline] config.
func TemplateFS() embed.FS { return templateFS }
