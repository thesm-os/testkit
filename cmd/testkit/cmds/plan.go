// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"github.com/spf13/cobra"
	"go.thesmos.sh/eidos/cli"
)

// planKernel is the eidos command this subcommand drives. Package-level for
// the same reason as the run kernel: flag binding happens at init() and
// execution later, so parsing has to write into the instance RunE executes.
var planKernel = &cli.PlanCommand{}

// planCmd prints the resolved plugin order without running anything.
//
// The order is not obvious from the plugin list: annotators and generators sort
// into priority buckets, and within a bucket a capability topo-sort decides who
// runs first. A plugin whose ordering was silently discarded — the shape
// [cli.PlanCommand] exists to surface — looks identical in the source and
// different here.
//
// No source is read, so this stays useful in a package that does not compile.
var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Print the resolved plugin order without running the pipeline",
	Long: "Plan resolves the pipeline the current config and embedded plugin " +
		"set produce, and prints every frontend, annotator and generator in " +
		"execution order with the backend that renders them.\n\n" +
		"Nothing is read and nothing is written, so this answers " +
		"\"what would run, in what order\" without needing the target packages " +
		"to build.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		env, cfg, err := prepare(cmd)
		if err != nil {
			return err
		}
		planKernel.Config.File = cfg
		planKernel.Config.Plugins = generators()
		return exit(planKernel.Execute(cmd.Context(), env))
	},
}

func init() {
	bindKernelFlags(planCmd, "plan", planKernel.RegisterFlags)
	rootCmd.AddCommand(planCmd)
}
