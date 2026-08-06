// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// The run command's body is the binary's whole reason to exist, and every
// failure it can reach is a failure a user meets on their first invocation.
// Driving it through the real command tree is what proves the wiring — the
// environment, the config, the plugin set, and the exit-code bridge — rather
// than proving that a function nobody calls returns what it says.
//
//nolint:paralleltest // drives the shared rootCmd and its package-level kernel config
func TestRunCommand(t *testing.T) {
	t.Run("reports a pattern that matches nothing", func(t *testing.T) {
		sourceTree(t)
		_, _, code := runRoot(t, "run", "./nonexistent/...")
		if code == 0 {
			t.Fatal("a pattern matching no package must not exit 0")
		}
	})

	t.Run("reports an unreadable config", func(t *testing.T) {
		// The config is resolved before any package is loaded, so a broken one
		// has to fail here rather than surfacing as a generator that emitted
		// nothing.
		sourceTree(t)
		_, _, code := runRoot(t, "run", "--config", "nonexistent.yaml", "./...")
		if code == 0 {
			t.Fatal("an unreadable config must not exit 0")
		}
	})
}

// sourceTree writes a module holding one annotated interface, makes it the
// working directory, and returns its path.
//
// A real tree rather than a fixture store: the command resolves packages
// through the Go toolchain, so anything short of one exercises the wiring
// around the loader instead of the loader itself.
func sourceTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// The generated double imports testkit's runtime packages, so the module
	// has to be able to resolve them or the *second* load of the tree — the one
	// `check` performs, with the generated file now part of the package —
	// fails to typecheck. A replace to the repository under test also pins what
	// is being exercised to this checkout rather than to a published version.
	write("go.mod", "module example.com/tree\n\ngo 1.26\n\n"+
		"require go.thesmos.sh/testkit v0.0.0\n\n"+
		"replace go.thesmos.sh/testkit => "+repoRoot(t)+"\n")
	t.Setenv("GOFLAGS", "-mod=mod")
	write("iface.go", `package tree

//testkit:stub
type Store interface {
	Get(key string) error
}
`)

	t.Chdir(dir)
	return dir
}

// repoRoot returns the checkout this test binary was built from, which is what
// the fixture module replaces testkit with.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the source file this test was built from")
	}
	// .../cmd/testkit/cmds/run_test.go -> the repository root.
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
}
