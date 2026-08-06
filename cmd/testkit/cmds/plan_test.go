// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
)

// Plan reads no source, which is what makes it the one command that still
// answers a question when the target packages do not build — so what is under
// test is that it needs nothing and still names the plugins this binary
// carries.
//
//nolint:paralleltest // drives the shared rootCmd and its package-level kernel config
func TestPlanCommand(t *testing.T) {
	t.Run("resolves an order without reading source", func(t *testing.T) {
		// Deliberately run from an empty directory: a plan that needed packages
		// would fail here, and the failure would only show up for the user whose
		// build is already broken.
		t.Chdir(t.TempDir())
		stdout, _, code := runRoot(t, "plan")
		if code != cli.ExitOK {
			t.Fatalf("plan must not need source, got exit %d", code)
		}
		if !strings.Contains(stdout, "backend:") {
			t.Fatalf("the plan must name the backend, got %q", stdout)
		}
	})

	t.Run("names the generators this binary embeds", func(t *testing.T) {
		// The point of the command: which generators a given build carries is
		// otherwise only discoverable by running it and reading the output.
		t.Chdir(t.TempDir())
		stdout, _, _ := runRoot(t, "plan")
		for _, name := range []string{"builder", "stub"} {
			if !strings.Contains(stdout, name) {
				t.Errorf("the plan must name the %s generator, got %q", name, stdout)
			}
		}
	})

	t.Run("takes no positional argument", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if _, _, code := runRoot(t, "plan", "./..."); code == cli.ExitOK {
			t.Fatal("plan reads no source, so a pattern is a usage mistake")
		}
	})

	t.Run("reports an unreadable config", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if _, _, code := runRoot(t, "plan", "--config", "nonexistent.yaml"); code == cli.ExitOK {
			t.Fatal("an unreadable config must not exit 0")
		}
	})
}
