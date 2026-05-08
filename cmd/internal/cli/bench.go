// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/generator/bench"
)

var benchCmd = &cobra.Command{
	Use:   "bench [-o path] Type",
	Short: "Generate benchmarks for a Go interface",
	Long:  "Generate BenchmarkContract function with per-method hot-path benchmarks and typed plug-in extension points.",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runGenerator(&bench.Generator{}, "bench.output", args)
	},
}

func init() {
	benchCmd.Flags().StringP("output", "o", "", "output file path")

	_ = viper.BindPFlag("bench.output", benchCmd.Flags().Lookup("output"))

	rootCmd.AddCommand(benchCmd)
}
