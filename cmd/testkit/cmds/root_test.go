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
	"go.thesmos.sh/eidos/plugin"

	"go.thesmos.sh/testkit/core/brand"
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
	// Every kernel is package-level and retains per-invocation state, so a
	// second run in the same process would otherwise inherit the first's —
	// including a context the first run's signal handler already cancelled.
	*runKernel = cli.RunCommand{}
	*checkKernel = cli.CheckCommand{}
	*explainKernel = cli.ExplainCommand{}
	*planKernel = cli.PlanCommand{}
	*pruneKernel = cli.PruneCommand{}
	// cobra fills a subcommand's context only when it is nil, so a second
	// Execute in one process would dispatch under the first invocation's
	// context — which that invocation's signal handler cancelled on its way
	// out. Clearing it puts the fresh, signal-aware context back in play. A
	// real process runs one command, so only a test binary meets this.
	for _, c := range rootCmd.Commands() {
		// nil is the value with the effect wanted: cobra refills a nil context
		// and leaves a non-nil one alone, so anything else here pins the stale
		// context rather than clearing it.
		c.SetContext(nil) //nolint:staticcheck // SA1012: nil is what triggers cobra's refill
	}

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
	for _, name := range []string{"check", "explain", "plan", "prune", "run", "version"} {
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
	env, err := cli.NewEnv(brand.Name)
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
		bare, err := cli.NewEnv(brand.Name)
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
		if env.Brand != brand.Name {
			t.Fatalf("expected brand %q, got %q", brand.Name, env.Brand)
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

// The binary is useless without all four plugin roles: a missing frontend
// parses nothing, a missing backend renders nothing, and either failure
// reports as a build error at run time rather than here.
//
// Pinning the roles rather than the count means adding a generator does not
// touch this test, while dropping the frontend does.
func TestPluginSetCarriesEveryRole(t *testing.T) {
	t.Parallel()

	plugins := generators()
	if len(plugins) == 0 {
		t.Fatal("the binary registers no plugins at all")
	}

	var frontends, annotators, gens, backends int
	for _, p := range plugins {
		if _, ok := p.(plugin.Frontend); ok {
			frontends++
		}
		if _, ok := p.(plugin.Annotator); ok {
			annotators++
		}
		if _, ok := p.(plugin.Generator); ok {
			gens++
		}
		if _, ok := p.(plugin.Backend); ok {
			backends++
		}
	}
	for _, c := range []struct {
		role string
		n    int
	}{
		{"frontend", frontends},
		{"annotator", annotators},
		{"generator", gens},
		{"backend", backends},
	} {
		if c.n == 0 {
			t.Errorf("the plugin set registers no %s", c.role)
		}
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

// Only run carries a Patterns field, so check, prune and explain would ignore
// the arguments run accepts. Two commands that must agree about what they
// looked at cannot read the same argument differently — a check that silently
// examined the whole module while the user scoped it to one package would pass
// or fail for reasons unrelated to what they asked about.
func TestScopedTo(t *testing.T) {
	t.Parallel()

	t.Run("replaces the configured patterns", func(t *testing.T) {
		t.Parallel()
		cfg := &cli.Config{Sources: []cli.ConfigSource{{Frontend: "golang", Patterns: []string{"./..."}}}}
		got := scopedTo(cfg, []string{"./corpus/..."})
		if len(got.Sources) != 1 || len(got.Sources[0].Patterns) != 1 {
			t.Fatalf("expected one source with one pattern, got %+v", got.Sources)
		}
		if got.Sources[0].Patterns[0] != "./corpus/..." {
			t.Fatalf("the argument must win, got %q", got.Sources[0].Patterns[0])
		}
	})

	t.Run("keeps the configured frontend", func(t *testing.T) {
		t.Parallel()
		// Routing the patterns to a different frontend than the config named
		// would hand them to a loader that cannot read them.
		cfg := &cli.Config{Sources: []cli.ConfigSource{{Frontend: "golang"}}}
		got := scopedTo(cfg, []string{"./x/..."})
		if got.Sources[0].Frontend != "golang" {
			t.Fatalf("the configured frontend must survive, got %q", got.Sources[0].Frontend)
		}
	})

	t.Run("leaves the config alone without arguments", func(t *testing.T) {
		t.Parallel()
		// No arguments means "whatever the config says", which is what makes a
		// bare invocation from a module root work.
		cfg := &cli.Config{Sources: []cli.ConfigSource{{Frontend: "golang", Patterns: []string{"./a/..."}}}}
		got := scopedTo(cfg, nil)
		if len(got.Sources) != 1 || got.Sources[0].Patterns[0] != "./a/..." {
			t.Fatalf("the configured patterns must survive, got %+v", got.Sources)
		}
	})

	t.Run("names no frontend when the config declared no source", func(t *testing.T) {
		t.Parallel()
		// An empty name is what the pipeline reads as "every frontend", which
		// is the same default a bare run gets.
		got := scopedTo(&cli.Config{}, []string{"./x/..."})
		if got.Sources[0].Frontend != "" {
			t.Fatalf("expected no frontend, got %q", got.Sources[0].Frontend)
		}
	})

	t.Run("tolerates an absent config", func(t *testing.T) {
		t.Parallel()
		if got := scopedTo(nil, []string{"./x/..."}); got != nil {
			t.Fatalf("a nil config has nothing to scope, got %+v", got)
		}
	})
}
