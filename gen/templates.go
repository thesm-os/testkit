// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import "embed"

//go:embed templates/*.tmpl
var templateFS embed.FS

// TemplateFile returns the content of an embedded template file.
func TemplateFile(name string) string {
	data, err := templateFS.ReadFile("templates/" + name)
	if err != nil {
		// Templates are embedded at compile time — a missing template
		// is a build-time bug, not a runtime error.
		panic("gen: missing embedded template: " + name) //nolint:forbidigo // compile-time guarantee
	}
	return string(data)
}
