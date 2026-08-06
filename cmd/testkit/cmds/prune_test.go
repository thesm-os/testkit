// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
)

// Prune is the only command that deletes, so what is under test is what it
// refuses to delete: a dry run must leave the tree alone, and a file somebody
// has taken ownership of must survive being no longer claimed.
//
//nolint:paralleltest // drives the shared rootCmd and its package-level kernel config
func TestPruneCommand(t *testing.T) {
	t.Run("removes output the current run no longer claims", func(t *testing.T) {
		// Deleting the directive is how a type stops being generated for, and
		// the output left behind still compiles — so nothing but this finds it.
		dir := sourceTree(t)
		if _, _, code := runRoot(t, "run", "./..."); code != cli.ExitOK {
			t.Fatalf("the fixture must generate cleanly, got exit %d", code)
		}
		stale := generated(t, dir)
		undirective(t, dir)

		if _, _, code := runRoot(t, "prune", "./..."); code != cli.ExitOK {
			t.Fatalf("prune must exit 0, got %d", code)
		}
		if _, err := os.Stat(stale); !os.IsNotExist(err) {
			t.Fatal("prune must remove output the run no longer claims")
		}
	})

	t.Run("leaves the tree alone under --dry-run", func(t *testing.T) {
		// The flag exists so a gate can report what would go without being the
		// thing that removed it.
		dir := sourceTree(t)
		if _, _, code := runRoot(t, "run", "./..."); code != cli.ExitOK {
			t.Fatalf("the fixture must generate cleanly, got exit %d", code)
		}
		stale := generated(t, dir)
		undirective(t, dir)

		stdout, _, code := runRoot(t, "prune", "--dry-run", "./...")
		if code != cli.ExitOK {
			t.Fatalf("a dry run must exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "would delete") {
			t.Fatalf("a dry run must report what it would remove, got %q", stdout)
		}
		if _, err := os.Stat(stale); err != nil {
			t.Fatalf("a dry run must not delete: %v", err)
		}
	})

	t.Run("keeps a file that lost its generated marker", func(t *testing.T) {
		// The marker is what separates "this is ours to remove" from "somebody
		// adopted this". Losing an adopted file is unrecoverable, so the gate
		// errs towards keeping.
		dir := sourceTree(t)
		if _, _, code := runRoot(t, "run", "./..."); code != cli.ExitOK {
			t.Fatalf("the fixture must generate cleanly, got exit %d", code)
		}
		adopted := generated(t, dir)
		body, err := os.ReadFile(adopted) //nolint:gosec // path composed from the test's own TempDir
		if err != nil {
			t.Fatalf("read %s: %v", adopted, err)
		}
		_, rest, _ := strings.Cut(string(body), "\n")
		//nolint:gosec // G703: path composed from this test's own TempDir
		if err := os.WriteFile(adopted, []byte("// mine now\n"+rest), 0o600); err != nil {
			t.Fatalf("rewrite %s: %v", adopted, err)
		}
		undirective(t, dir)

		if _, _, code := runRoot(t, "prune", "./..."); code != cli.ExitOK {
			t.Fatalf("prune must exit 0, got %d", code)
		}
		if _, err := os.Stat(adopted); err != nil {
			t.Fatalf("an adopted file must survive prune: %v", err)
		}
	})

	t.Run("reports an unreadable config", func(t *testing.T) {
		sourceTree(t)
		if _, _, code := runRoot(t, "prune", "--config", "nonexistent.yaml", "./..."); code == cli.ExitOK {
			t.Fatal("an unreadable config must not exit 0")
		}
	})
}

// undirective rewrites the fixture's source with its directive removed, which
// is what makes the previous run's output unclaimed.
func undirective(t *testing.T, dir string) {
	t.Helper()
	body := `package tree

type Store interface {
	Get(key string) error
}
`
	if err := os.WriteFile(filepath.Join(dir, "iface.go"), []byte(body), 0o600); err != nil {
		t.Fatalf("rewrite iface.go: %v", err)
	}
}
