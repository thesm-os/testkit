// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
)

// The selector is the argument rather than a flag, so the arity is part of the
// command's contract — and the trace it prints is only useful if it names both
// the plugin that produced an output and where the output landed.
//
//nolint:paralleltest // drives the shared rootCmd and its package-level kernel config
func TestExplainCommand(t *testing.T) {
	t.Run("traces a selected entity to its output", func(t *testing.T) {
		sourceTree(t)
		stdout, _, code := runRoot(t, "explain", "tree.Store")
		if code != cli.ExitOK {
			t.Fatalf("explain must resolve a declared entity, got exit %d", code)
		}
		if !strings.Contains(stdout, "Store") {
			t.Fatalf("the trace must name the subject, got %q", stdout)
		}
	})

	t.Run("names the plugin that produced each output", func(t *testing.T) {
		// The question the command exists for: several plugins contribute to
		// one file, and provenance is the only record of which wrote what.
		sourceTree(t)
		stdout, _, _ := runRoot(t, "explain", "tree.Store")
		if !strings.Contains(stdout, "stub") {
			t.Fatalf("the trace must name the producing plugin, got %q", stdout)
		}
	})

	t.Run("requires a selector", func(t *testing.T) {
		sourceTree(t)
		if _, _, code := runRoot(t, "explain"); code == cli.ExitOK {
			t.Fatal("explain without a selector has nothing to do")
		}
	})

	t.Run("rejects a second argument", func(t *testing.T) {
		// The selector is singular. A second argument is likelier to be a
		// package pattern the user expected to scope the run, which this
		// command does not take — silently ignoring it would explain the wrong
		// thing convincingly.
		sourceTree(t)
		if _, _, code := runRoot(t, "explain", "tree.Store", "./..."); code == cli.ExitOK {
			t.Fatal("a second argument must be rejected")
		}
	})

	t.Run("reports an unreadable config", func(t *testing.T) {
		sourceTree(t)
		if _, _, code := runRoot(t, "explain", "--config", "nonexistent.yaml", "tree.Store"); code == cli.ExitOK {
			t.Fatal("an unreadable config must not exit 0")
		}
	})
}
