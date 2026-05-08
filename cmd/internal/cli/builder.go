// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/generator/builder"
)

var builderCmd = &cobra.Command{
	Use:   "builder [-o path] Type [Type...]",
	Short: "Generate fluent builders for Go structs",
	Long:  "Generate test fixture builders with fluent With* setters for Go struct types.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runGenerator(&builder.Generator{}, "builder.output", args)
	},
}

func init() {
	builderCmd.Flags().StringP("output", "o", "", "output file path")

	_ = viper.BindPFlag("builder.output", builderCmd.Flags().Lookup("output"))

	rootCmd.AddCommand(builderCmd)
}
