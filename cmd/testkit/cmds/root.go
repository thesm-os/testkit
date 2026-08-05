// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cmds defines the cobra command tree for the testkit CLI.
//
// Each subcommand lives in its own file and registers itself with [rootCmd]
// from init(); the files here hold command definitions and flag wiring only.
// The work belongs to eidos's command kernels ([cli.RunCommand],
// [cli.VersionCommand], and the rest), which every RunE constructs and
// executes.
//
// # Flag wiring
//
// Flags come from eidos rather than being declared here. Each kernel binds its
// own flags into a stdlib [flag.FlagSet] through RegisterFlags, and
// [bindKernelFlags] folds that set into the cobra command via
// pflag's AddGoFlagSet. A flag eidos adds to a kernel therefore appears on the
// testkit command with no change here — declaring flags by hand against
// eidos's exported name constants would drift the moment upstream added one.
//
// The cost of that bridge is POSIX parsing: pflag reads a single dash as a
// shorthand cluster, so `--config` works and `-config` does not. testkit has
// never shipped a binary, so nothing breaks, and it matches ergon.
//
// # Hazards
//
// Kernels return a process exit code rather than an error, and the codes are
// load-bearing — CI gates pin behaviour to specific values ([cli.ExitCheckDrift]
// for drift, [cli.ExitCacheVerifyFailed] for cache mismatch). RunE must
// therefore carry the code out rather than collapsing it into a non-nil error,
// which is what [exitCodeError] exists for.
package cmds

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/plugin"

	"go.thesmos.sh/testkit/cmd/internal/version"
	"go.thesmos.sh/testkit/core/brand"
)

// rootCmd is the top-level `testkit` command. Subcommand files register
// themselves against it from init().
var rootCmd = &cobra.Command{
	Use:   brand.Name,
	Short: "Generate test doubles, suites, and benchmarks from Go types",
	Long: "testkit reads your interfaces and types, generates the test doubles, \n" +
		"builders, fixtures, conformance suites, and benchmarks you would \n" +
		"otherwise write by hand, and generates the tests that prove the " +
		"generated code works.",
	Version:       version.Full(),
	SilenceUsage:  true,
	SilenceErrors: true,
}

// cfgPath captures --config. Empty means "walk up from the working directory
// looking for .testkit.yaml"; a missing file is not an error, because the
// defaults apply.
var cfgPath string

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, cli.FlagConfig, "",
		"path to the config file (default: nearest "+brand.ConfigFile+")")
}

// Execute runs the command tree and returns the process exit code.
//
// Signal handling covers the whole tree: SIGINT and SIGTERM cancel the context
// the kernels run under, so a pipeline in flight unwinds through its own
// cleanup rather than leaving a partially-written output tree behind.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := rootCmd.ExecuteContext(ctx)
	if err == nil {
		return cli.ExitOK
	}
	var ec exitCodeError
	if ok := asExitCode(err, &ec); ok {
		return ec.code
	}
	fmt.Fprintf(rootCmd.ErrOrStderr(), "%s: %v\n", brand.Name, err)
	return cli.ExitUserError
}

// exit converts a kernel's exit code into the error RunE returns: nil for
// success, an [exitCodeError] otherwise. Every subcommand ends with this, so
// the non-zero codes reach the process while success stays a nil error.
func exit(code int) error {
	if code == cli.ExitOK {
		return nil
	}
	return exitCodeError{code: code}
}

// exitCodeError carries a kernel's exit code out through cobra's error return
// without cobra printing anything. The kernels have already written their own
// diagnostics by the time they return a code.
type exitCodeError struct{ code int }

func (e exitCodeError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// asExitCode reports whether err is an [exitCodeError] and copies it out.
func asExitCode(err error, target *exitCodeError) bool {
	ec, ok := err.(exitCodeError) //nolint:errorlint // sentinel is never wrapped: it is constructed at the RunE boundary
	if ok {
		*target = ec
	}
	return ok
}

// newEnv builds the eidos environment for one invocation, routing IO through
// cobra so tests can capture output.
func newEnv(cmd *cobra.Command) (*cli.Env, error) {
	env, err := cli.NewEnv(brand.Name)
	if err != nil {
		return nil, fmt.Errorf("cmds: environment unavailable: %w", err)
	}
	env.Stdin, env.Stdout, env.Stderr = cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()
	return env, nil
}

// loadConfig resolves the config file: --config wins when set, otherwise the
// binary walks up from the working directory. A missing file is not an error.
func loadConfig(env *cli.Env) (*cli.Config, error) {
	path := cfgPath
	if path == "" {
		found, ok := cli.DiscoverConfig(env.Workdir, env.ConfigFileName())
		if !ok {
			return cli.DefaultConfig(), nil
		}
		path = found
	}
	cfg, err := cli.LoadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("cmds: load config %s: %w", path, err)
	}
	return cfg, nil
}

// bindKernelFlags folds a kernel's stdlib flag registrations into cobraCmd.
//
// register is the kernel's RegisterFlags method. It binds flags to fields of
// the kernel's Config struct; AddGoFlagSet re-exposes each as a long-form
// cobra flag pointing at the same field, so parsing populates the kernel
// directly.
func bindKernelFlags(cobraCmd *cobra.Command, name string, register func(*flag.FlagSet)) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	register(fs)
	cobraCmd.Flags().AddGoFlagSet(fs)
}

// generators returns the plugin set this binary embeds. Empty until the first
// generator is ported; `testkit version` reports the set, so an empty result is
// visible rather than silent.
func generators() []plugin.Plugin { return nil }
