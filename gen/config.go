// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

// Config holds project-wide conventions that influence code generation.
// The CLI is responsible for loading this from .testkit.yml — the gen
// package never touches YAML directly.
type Config struct {
	// TestPackageSuffix is appended to package name for test packages.
	// "test" produces store → storetest. Default: "test".
	TestPackageSuffix string

	// GeneratedSuffix is the file extension suffix for generated files.
	// Default: ".gen.go".
	GeneratedSuffix string

	// TestPackageStyle controls package naming for generated test files.
	// "external" produces package foo_test (black-box).
	// "internal" produces package foo (white-box).
	// Default: "external".
	TestPackageStyle string
}

// DefaultConfig returns a Config with all default values.
func DefaultConfig() Config {
	return Config{
		TestPackageSuffix: "test",
		GeneratedSuffix:   ".gen.go",
		TestPackageStyle:  "external",
	}
}
