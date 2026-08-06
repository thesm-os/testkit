// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"os"
	"path/filepath"
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

// sourceTree writes a module holding one annotated interface and makes it the
// working directory.
//
// A real tree rather than a fixture store: the command resolves packages
// through the Go toolchain, so anything short of one exercises the wiring
// around the loader instead of the loader itself.
func sourceTree(t *testing.T) {
	t.Helper()
	dir := t.TempDir()

	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("go.mod", "module example.com/tree\n\ngo 1.26\n")
	write("iface.go", `package tree

//testkit:stub
type Store interface {
	Get(key string) error
}
`)

	t.Chdir(dir)
}
