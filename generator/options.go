// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

// Config holds project-wide conventions read from .testkit.yml. The
// generator package never reads YAML directly — the CLI is responsible
// for hydrating Config and passing it to generators.
type Config struct {
	// TestPackageSuffix is appended to the package name for test packages.
	// "test" produces store → storetest. Default: "test".
	TestPackageSuffix string

	// GeneratedSuffix is the file extension suffix for generated files.
	// All generated files end with this so they can be excluded from
	// manual review and regenerated without loss. Default: ".gen.go".
	GeneratedSuffix string

	// TestPackageStyle controls package naming for generated test files.
	//   "external" → package foo_test (black-box).
	//   "internal" → package foo (white-box).
	// Default: "external".
	TestPackageStyle string

	// Stub holds naming conventions for the stub generator.
	Stub StubConfig
}

// StubConfig holds naming conventions for the stub generator.
type StubConfig struct {
	// FilePattern is the base name pattern for generated stub files.
	// "{type}" is replaced with the lowercased type name.
	// Default: "{type}_stub".
	FilePattern string

	// TypeSuffix is appended to the type name for generated stub types.
	// E.g. "Store" + "Stub" → "StoreStub". Default: "Stub".
	TypeSuffix string
}

// Defaults for [Config] and [StubConfig].
const (
	DefaultTestPackageSuffix = "test"
	DefaultGeneratedSuffix   = ".gen.go"

	TestPackageStyleExternal = "external"
	TestPackageStyleInternal = "internal"

	DefaultTestPackageStyle = TestPackageStyleExternal

	DefaultStubFilePattern = "{type}_stub"
	DefaultStubTypeSuffix  = "Stub"
)

// DefaultConfig returns a [Config] with all default values.
func DefaultConfig() Config {
	return Config{
		TestPackageSuffix: DefaultTestPackageSuffix,
		GeneratedSuffix:   DefaultGeneratedSuffix,
		TestPackageStyle:  DefaultTestPackageStyle,
		Stub: StubConfig{
			FilePattern: DefaultStubFilePattern,
			TypeSuffix:  DefaultStubTypeSuffix,
		},
	}
}

// Options holds per-invocation flags for a generator run. The CLI
// hydrates Options from command-line flags and the discovered source
// context (working directory, $GOFILE, etc.) before calling
// [Generator.Generate].
type Options struct {
	// Output is the -o flag value. Empty when the generator computes its
	// own output path from convention.
	Output string

	// Check enables dry-run mode — compare output against existing files,
	// return an error with diffs on mismatch, never write.
	Check bool

	// Verbose enables verbose logging.
	Verbose bool

	// BuildTag, when non-empty, is emitted as //go:build <tag> at the top
	// of every generated file.
	BuildTag string

	// WorkDir is the directory //go:generate runs in. Relative paths in
	// Output and other path fields resolve against WorkDir.
	WorkDir string

	// SourceFile is the source .go file containing the //go:generate
	// directive ($GOFILE). Used by file-scoped generators (e.g. sentinel
	// limiting its scan to the declaring file).
	SourceFile string

	// OutputPackage overrides the source package name when computing
	// the package declaration for the generated file. Set by the CLI
	// when the user invokes a generator against a remote package
	// (-p flag): the output should land in the CWD's package, not the
	// remote source's.
	OutputPackage string

	// OutputImportBase is the import path of the working directory.
	// Set by the CLI alongside [OutputPackage] when generating
	// against a remote package — used by [OutputImportPath] so the
	// emitted output references the correct module path.
	OutputImportBase string

	// Invocation is the original CLI invocation as the user typed it,
	// minus the binary name (e.g. "sentinel -o errors.gen_test.go").
	// When set, the generator pipeline prefers it over the
	// reconstructed (subcommand + positional args) form for the
	// "// Source:" attribution line, so flags are preserved verbatim
	// and a reader can re-run the exact command. The CLI runner
	// populates it from os.Args[1:]; tests and direct callers leave
	// it empty and fall back to the legacy formulation.
	Invocation string
}

// Result is the output of a [Generator] invocation. Generators return a
// Result; the caller (CLI or test) is responsible for writing the files.
type Result struct {
	Files []OutputFile
}

// OutputFile is one file produced by a generator.
type OutputFile struct {
	// Path is the file path relative to [Options.WorkDir].
	Path string

	// Content is the formatted Go source.
	Content []byte
}
