// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/eidos/cli"
)

// Check is a CI gate, so the exit code is the interface: a caller distinguishes
// "the tree is stale" from "the run failed" by the number alone. Asserting on
// the message would pin prose; asserting on the code pins the contract.
//
//nolint:paralleltest // drives the shared rootCmd and its package-level kernel config
func TestCheckCommand(t *testing.T) {
	t.Run("reports drift as its own exit code", func(t *testing.T) {
		// A generated file edited after the fact is the case the gate exists
		// for, and the one a generic non-zero code would make indistinguishable
		// from a broken pipeline.
		dir := sourceTree(t)
		if _, _, code := runRoot(t, "run", "./..."); code != cli.ExitOK {
			t.Fatalf("the fixture must generate cleanly, got exit %d", code)
		}
		appendTo(t, generated(t, dir), "\n// edited by hand\n")

		_, _, code := runRoot(t, "check", "./...")
		if code != cli.ExitCheckDrift {
			t.Fatalf("drift must exit %d, got %d", cli.ExitCheckDrift, code)
		}
	})

	t.Run("passes on a tree the pipeline would reproduce", func(t *testing.T) {
		sourceTree(t)
		if _, _, code := runRoot(t, "run", "./..."); code != cli.ExitOK {
			t.Fatalf("the fixture must generate cleanly, got exit %d", code)
		}
		if _, _, code := runRoot(t, "check", "./..."); code != cli.ExitOK {
			t.Fatalf("a current tree must exit 0, got %d", code)
		}
	})

	t.Run("writes nothing", func(t *testing.T) {
		// The whole value of a gate is that it can run on a clean checkout in
		// CI. One that generated as a side effect would report success by
		// having just fixed the thing it was asked to detect.
		dir := sourceTree(t)
		if _, _, code := runRoot(t, "check", "./..."); code != cli.ExitCheckDrift {
			t.Fatalf("an ungenerated tree is drift, got exit %d", code)
		}
		if _, err := os.Stat(generated(t, dir)); !os.IsNotExist(err) {
			t.Fatal("check must not write the file it was asked to compare")
		}
	})

	t.Run("reports an unreadable config", func(t *testing.T) {
		sourceTree(t)
		if _, _, code := runRoot(t, "check", "--config", "nonexistent.yaml", "./..."); code == cli.ExitOK {
			t.Fatal("an unreadable config must not exit 0")
		}
	})
}

// generated returns the path of the double the fixture tree produces. The
// fixture declares no routing directive, so the output lands beside its source.
func generated(t *testing.T, dir string) string {
	t.Helper()
	return filepath.Join(dir, "iface_stub.gen.go")
}

// appendTo adds text to an existing file, which is how a hand edit to generated
// output looks to the gate.
func appendTo(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("close %s: %v", path, err)
		}
	}()
	if _, err := f.WriteString(text); err != nil {
		t.Fatalf("append to %s: %v", path, err)
	}
}
