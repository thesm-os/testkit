// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"github.com/spf13/cobra"
	"go.thesmos.sh/eidos/cli"
)

// explainKernel is the eidos command this subcommand drives. Package-level for
// the same reason as the run kernel: flag binding happens at init() and
// execution later, so parsing has to write into the instance RunE executes.
var explainKernel = &cli.ExplainCommand{}

// explainCmd prints where one entity, slot or metadata key came from.
//
// Generated output is the product of several plugins that never see each other:
// an annotator stamps, a generator reads the stamp, a second generator
// contributes into the first one's file. When the result is wrong, the question
// is which of them did it — and the emit graph records that as provenance,
// which this is the only way to read.
//
// The selector is positional rather than a flag because it is the argument, not
// a modifier: `explain` without one has nothing to do.
var explainCmd = &cobra.Command{
	Use:   "explain [flags] <selector>",
	Short: "Print the provenance trace for an entity, slot, or metadata key",
	Long: "Explain runs the pipeline and reports what produced the selected " +
		"entity: which plugin emitted it, which slot it landed in, and what " +
		"metadata was stamped on the source it came from.\n\n" +
		"The selector names an entity, a slot on one, or a metadata key. This " +
		"is the tool for \"which plugin wrote this line\" when several " +
		"contribute to one file.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		env, cfg, err := prepare(cmd)
		if err != nil {
			return err
		}
		explainKernel.Config.File = cfg
		explainKernel.Config.Plugins = generators()
		explainKernel.Config.Selector = args[0]
		return exit(explainKernel.Execute(cmd.Context(), env))
	},
}

func init() {
	bindKernelFlags(explainCmd, "explain", explainKernel.RegisterFlags)
	rootCmd.AddCommand(explainCmd)
}
