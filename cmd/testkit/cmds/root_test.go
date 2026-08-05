// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal tests: the command tree, the exit-code bridge, and the config
// resolver are package-level state the exported surface reaches only through
// os.Args and os.Exit. Driving them directly is the only way to assert them
// without spawning a process.
package cmds

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"go.thesmos.sh/eidos/cli"
)

// runRoot drives the shared rootCmd with args and returns stdout, stderr, and
// the exit code [Execute] would hand the process.
//
// Not safe for parallel use: rootCmd, cfgPath, and the kernel Config structs
// are package-level. Every run resets them first so tests do not observe each
// other's flags.
func runRoot(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()

	cfgPath = ""
	versionKernel.Config = cli.VersionConfig{}

	var out, errOut strings.Builder
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		cfgPath = ""
	})

	// Execute is the real entry point, signal handling included. Duplicating
	// its exit-code mapping here would leave that mapping untested, which is
	// the part CI gates depend on.
	code = Execute()
	return out.String(), errOut.String(), code
}

// The command surface is assembled from independent init() functions, so a
// subcommand registered against the wrong parent — or dropped in a refactor —
// is invisible until someone types it.
//
//nolint:paralleltest // cobra's Commands() lazily sorts and caches, so walking the shared rootCmd is a write
func TestCommandTree(t *testing.T) {
	got := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		got[c.Name()] = true
	}
	// Only commands this package registers. cobra grafts `help` and
	// `completion` onto the tree during Execute, so asserting them here would
	// pin an ordering rather than a surface.
	for _, name := range []string{"version"} {
		if !got[name] {
			t.Errorf("rootCmd has no subcommand %q", name)
		}
	}
}

//nolint:paralleltest // drives the shared rootCmd
func TestUnknownCommandFails(t *testing.T) {
	_, _, code := runRoot(t, "definitely-not-a-command")

	if code != cli.ExitUserError {
		t.Fatalf("an unknown command is a user error, got exit %d", code)
	}
}

// A kernel reports outcomes as exit codes and CI gates pin behaviour to
// specific values. Collapsing a non-zero code into a generic error would make
// drift and cache-verification failures indistinguishable from a usage
// mistake.
func TestExitCodeBridge(t *testing.T) {
	t.Parallel()

	t.Run("success is a nil error", func(t *testing.T) {
		t.Parallel()
		if err := exit(cli.ExitOK); err != nil {
			t.Fatalf("ExitOK must not surface as an error, got %v", err)
		}
	})

	t.Run("a non-zero code survives the round trip", func(t *testing.T) {
		t.Parallel()
		err := exit(cli.ExitCheckDrift)
		if err == nil {
			t.Fatal("a non-zero code must surface as an error")
		}
		var ec exitCodeError
		if !asExitCode(err, &ec) {
			t.Fatalf("the code must be recoverable, got %T", err)
		}
		if ec.code != cli.ExitCheckDrift {
			t.Fatalf("expected exit %d, got %d", cli.ExitCheckDrift, ec.code)
		}
		if !strings.Contains(err.Error(), "5") {
			t.Fatalf("the message must name the code, got %q", err.Error())
		}
	})

	t.Run("an ordinary error is not an exit code", func(t *testing.T) {
		t.Parallel()
		var ec exitCodeError
		if asExitCode(errors.New("boom"), &ec) {
			t.Fatal("an unrelated error must not be read as an exit code")
		}
	})
}

// Config resolution anchors artifact paths, so a wrong answer relocates output
// rather than failing.
//
//nolint:paralleltest // t.Chdir cannot be used from a parallel test
func TestLoadConfigResolution(t *testing.T) {
	env, err := cli.NewEnv(brand)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Run("an explicit path wins", func(t *testing.T) { //nolint:paralleltest // mutates package-level cfgPath
		path := filepath.Join(t.TempDir(), "custom.yaml")
		if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		cfgPath = path
		t.Cleanup(func() { cfgPath = "" })

		if _, err := loadConfig(env); err != nil {
			t.Fatalf("an explicit config must be read: %v", err)
		}
	})

	t.Run(
		"a named file that is absent is an error",
		func(t *testing.T) { //nolint:paralleltest // mutates package-level cfgPath
			cfgPath = filepath.Join(t.TempDir(), "absent.yaml")
			t.Cleanup(func() { cfgPath = "" })

			if _, err := loadConfig(env); err == nil {
				t.Fatal("a config the caller named must not be silently skipped")
			}
		},
	)

	t.Run("discovery walks up to the project config", func(t *testing.T) { //nolint:paralleltest // shares env
		// The repo root carries .testkit.yaml and NewEnv anchors on the
		// working directory, so discovery must reach it from here.
		cfgPath = ""
		if _, err := loadConfig(env); err != nil {
			t.Fatalf("discovery must resolve from within the repo: %v", err)
		}
	})

	t.Run("no config anywhere falls back to defaults", func(t *testing.T) { //nolint:paralleltest // t.Chdir
		t.Chdir(t.TempDir())
		bare, err := cli.NewEnv(brand)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		cfgPath = ""
		cfg, err := loadConfig(bare)
		if err != nil {
			t.Fatalf("an absent config is not an error: %v", err)
		}
		if cfg == nil {
			t.Fatal("the default config must be returned, not nil")
		}
	})
}

// newEnv routes IO through cobra so tests capture output, and fails only when
// the working directory is gone from under the process.
//
//nolint:paralleltest // t.Chdir cannot be used from a parallel test
func TestNewEnv(t *testing.T) {
	t.Run("IO is routed through the command", func(t *testing.T) { //nolint:paralleltest // mutates shared command IO
		var out, errOut strings.Builder
		versionCmd.SetOut(&out)
		versionCmd.SetErr(&errOut)
		t.Cleanup(func() { versionCmd.SetOut(nil); versionCmd.SetErr(nil) })

		env, err := newEnv(versionCmd)
		if err != nil {
			t.Fatalf("newEnv: %v", err)
		}
		if env.Brand != brand {
			t.Fatalf("expected brand %q, got %q", brand, env.Brand)
		}
		if env.Stdout != &out || env.Stderr != &errOut {
			t.Fatal("IO must come from the cobra command, not the process")
		}
	})

	// A working directory removed from under the process leaves nothing to
	// anchor relative paths against, and that has to surface rather than
	// resolving against an arbitrary root.
	t.Run("a vanished working directory is an error", func(t *testing.T) { //nolint:paralleltest // t.Chdir
		nested := filepath.Join(t.TempDir(), "gone")
		if err := os.Mkdir(nested, 0o750); err != nil {
			t.Fatalf("setup: %v", err)
		}
		t.Chdir(nested)
		if err := os.Remove(nested); err != nil {
			t.Fatalf("setup: %v", err)
		}

		if _, err := newEnv(versionCmd); err == nil {
			t.Fatal("an unresolvable working directory must be reported")
		}
	})
}

// bindKernelFlags is what keeps testkit's flag surface in step with eidos's:
// the kernel declares, this side re-exposes. Losing the wiring would silently
// drop every kernel flag.
//
//nolint:paralleltest // inspects the shared versionCmd flag set
func TestBindKernelFlags(t *testing.T) {
	if versionCmd.Flags().Lookup(cli.FlagDiagFormat) == nil {
		t.Errorf("kernel flag %q is not bound onto the version command", cli.FlagDiagFormat)
	}
	if rootCmd.PersistentFlags().Lookup(cli.FlagConfig) == nil {
		t.Errorf("shared flag %q is not bound onto the root command", cli.FlagConfig)
	}
}

// The generator set is empty until the first generator is ported. Pinning it
// makes the day it changes a deliberate edit rather than a surprise.
func TestGeneratorSetIsEmpty(t *testing.T) {
	t.Parallel()

	if got := generators(); len(got) != 0 {
		t.Fatalf("expected no generators yet, got %d", len(got))
	}
}

// A kernel's exit code has to travel out through cobra to the process. CI
// gates pin behaviour to specific values — drift is 5, cache-verify failure is
// 4 — so collapsing them into a generic failure would make those gates
// indistinguishable from a usage mistake.
//
// The version command cannot produce a non-zero kernel code (its only failure
// needs an empty brand, and brand is a constant), so this drives the wiring
// with a command registered for the duration of the test.
//
//nolint:paralleltest // registers a command on the shared rootCmd
func TestExecuteCarriesKernelExitCode(t *testing.T) {
	probe := &cobra.Command{
		Use:  "exit-code-probe",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return exit(cli.ExitCheckDrift) },
	}
	rootCmd.AddCommand(probe)
	t.Cleanup(func() { rootCmd.RemoveCommand(probe) })

	_, _, code := runRoot(t, "exit-code-probe")

	if code != cli.ExitCheckDrift {
		t.Fatalf("expected the kernel's exit %d to reach the process, got %d",
			cli.ExitCheckDrift, code)
	}
}
