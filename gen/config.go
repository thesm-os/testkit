// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

const (
	// DefaultTestPackageSuffix is appended to the package name for test
	// packages. "test" produces store → storetest.
	DefaultTestPackageSuffix = "test"

	// DefaultGeneratedSuffix is the file extension suffix for generated files.
	DefaultGeneratedSuffix = ".gen.go"

	// TestPackageStyleExternal produces package foo_test (black-box).
	TestPackageStyleExternal = "external"

	// TestPackageStyleInternal produces package foo (white-box).
	TestPackageStyleInternal = "internal"

	// DefaultTestPackageStyle is the default test package naming convention.
	DefaultTestPackageStyle = TestPackageStyleExternal
)

// Config holds project-wide conventions that influence code generation.
// The CLI is responsible for loading this from .testkit.yml — the gen
// package never touches YAML directly.
type Config struct {
	// TestPackageSuffix is appended to package name for test packages.
	// Default: [DefaultTestPackageSuffix].
	TestPackageSuffix string

	// GeneratedSuffix is the file extension suffix for generated files.
	// Default: [DefaultGeneratedSuffix].
	GeneratedSuffix string

	// TestPackageStyle controls package naming for generated test files.
	// Use [TestPackageStyleExternal] or [TestPackageStyleInternal].
	// Default: [DefaultTestPackageStyle].
	TestPackageStyle string
}

// DefaultConfig returns a Config with all default values.
func DefaultConfig() Config {
	return Config{
		TestPackageSuffix: DefaultTestPackageSuffix,
		GeneratedSuffix:   DefaultGeneratedSuffix,
		TestPackageStyle:  DefaultTestPackageStyle,
	}
}
