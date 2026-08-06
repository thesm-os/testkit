// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"github.com/spf13/cobra"
	"go.thesmos.sh/eidos/cli"
)

// pruneKernel is the eidos command this subcommand drives. Package-level for
// the same reason as the run kernel: flag binding happens at init() and
// execution later, so parsing has to write into the instance RunE executes.
var pruneKernel = &cli.PruneCommand{}

// pruneCmd deletes generated files the current run no longer claims.
//
// Renaming a type or dropping a directive leaves its output behind, still
// compiling and still exercised, so a stale double keeps passing tests for an
// interface that no longer exists. Nothing else finds those: they are valid Go
// that nobody references.
//
// Deletion is gated on the generated-file marker in the first line, so a file
// somebody took ownership of survives regardless of what the manifest claims.
// `--dry-run` reports what would go without touching anything, which is the
// form worth reaching for first.
var pruneCmd = &cobra.Command{
	Use:   "prune [flags] [patterns...]",
	Short: "Delete generated files the current run no longer claims",
	Long: "Prune runs the pipeline, then removes files the previous manifest " +
		"claimed and this run does not — the output left behind when a type is " +
		"renamed or a directive removed.\n\n" +
		"Only files carrying the generated marker on their first line are " +
		"deleted, so a file that has been adopted and edited is kept. Use " +
		"--dry-run to see the list before anything is removed.\n\n" +
		"Positional arguments are Go package patterns, as for `run`.",
	RunE: func(cmd *cobra.Command, args []string) error {
		env, cfg, err := prepare(cmd)
		if err != nil {
			return err
		}
		pruneKernel.Config.File = scopedTo(cfg, args)
		pruneKernel.Config.Plugins = generators()
		return exit(pruneKernel.Execute(cmd.Context(), env))
	},
}

func init() {
	bindKernelFlags(pruneCmd, "prune", pruneKernel.RegisterFlags)
	rootCmd.AddCommand(pruneCmd)
}
