// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"github.com/spf13/cobra"
	"go.thesmos.sh/eidos/cli"
)

// checkKernel is the eidos command this subcommand drives. Package-level for
// the same reason as the run kernel: flag binding happens at init() and
// execution later, so parsing has to write into the instance RunE executes.
var checkKernel = &cli.CheckCommand{}

// checkCmd re-runs the pipeline into memory and compares every output against
// what is on disk, without writing anything.
//
// This is the CI half of `run`: it answers "is the committed generated code
// what this binary would produce", which otherwise gets answered by running the
// generator and reading `git status` — a check nothing enforces and everyone
// forgets. Drift exits [cli.ExitCheckDrift] rather than the generic error code,
// so a gate can tell "the tree is stale" from "the run failed".
//
// Comparison is byte-equal, whitespace included. A formatter disagreeing with
// the backend is drift, and it is worth knowing about.
var checkCmd = &cobra.Command{
	Use:   "check [flags] [patterns...]",
	Short: "Report whether generated files on disk match what the pipeline produces",
	Long: "Check runs the pipeline against an in-memory sink and compares every " +
		"output byte-for-byte with the file already on disk. Nothing is " +
		"written.\n\n" +
		"Positional arguments are Go package patterns, as for `run`. Absent " +
		"any, the config's sources apply, and absent those, ./... — so an " +
		"invocation from a module root checks that module.",
	RunE: func(cmd *cobra.Command, args []string) error {
		env, cfg, err := prepare(cmd)
		if err != nil {
			return err
		}
		checkKernel.Config.File = scopedTo(cfg, args)
		checkKernel.Config.Plugins = generators()
		return exit(checkKernel.Execute(cmd.Context(), env))
	},
}

func init() {
	bindKernelFlags(checkCmd, "check", checkKernel.RegisterFlags)
	rootCmd.AddCommand(checkCmd)
}
