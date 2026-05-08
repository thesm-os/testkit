// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/generator/enum"
)

var enumCmd = &cobra.Command{
	Use:   "enum [-o path] Type [Type...]",
	Short: "Generate exhaustiveness tests for Go enum types",
	Long:  "Scan const blocks of named types and generate exhaustiveness, distinctness, and stringer tests.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runGenerator(&enum.Generator{}, "enum.output", args)
	},
}

func init() {
	enumCmd.Flags().StringP("output", "o", "", "output file path")

	_ = viper.BindPFlag("enum.output", enumCmd.Flags().Lookup("output"))

	rootCmd.AddCommand(enumCmd)
}
