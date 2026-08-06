// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"github.com/spf13/cobra"
	"go.thesmos.sh/eidos/cli"
)

// runKernel is the eidos command this subcommand drives. Package-level for
// the same reason as the version kernel: flag binding happens at init() and
// execution later, so parsing has to write into the instance RunE executes.
var runKernel = &cli.RunCommand{}

// runCmd executes the pipeline and writes generated files through the sink.
//
// Positional arguments are Go package patterns. Absent any, the resolved
// config's source patterns apply — which is the `//go:generate` case, where
// the working directory already scopes the run.
var runCmd = &cobra.Command{
	Use:   "run [flags] [patterns...]",
	Short: "Execute the pipeline and write generated files",
	Long: "Run reads the named packages, classifies their declarations, and " +
		"writes what each registered generator emits.\n\n" +
		"Output location is not a flag on this command: routing is resolved " +
		"from the config file and the per-plugin routing flags, so the same " +
		"invocation produces the same paths whichever directory it runs from.",
	RunE: func(cmd *cobra.Command, args []string) error {
		env, err := newEnv(cmd)
		if err != nil {
			return err
		}
		cfg, err := loadConfig(env)
		if err != nil {
			return err
		}
		runKernel.Config.File = cfg
		runKernel.Config.Plugins = generators()
		runKernel.Config.Patterns = args
		return exit(runKernel.Execute(cmd.Context(), env))
	},
}

func init() {
	bindKernelFlags(runCmd, "run", runKernel.RegisterFlags)
	rootCmd.AddCommand(runCmd)
}
