// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import "embed"

//go:embed templates/*.tmpl
var templateFS embed.FS

// TemplateFS returns the embedded template filesystem for testing.
func TemplateFS() embed.FS {
	return templateFS
}
