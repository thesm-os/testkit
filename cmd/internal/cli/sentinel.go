// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/gen/sentinel"
)

var sentinelCmd = &cobra.Command{
	Use:   "sentinel [-o path]",
	Short: "Generate sentinel error tests",
	Long:  "Scan a package for exported Err* variables and custom error types, and generate tests for consistency, uniqueness, and unwrap chain preservation.",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runGenerator(&sentinel.Generator{}, "sentinel.output", nil)
	},
}

func init() {
	sentinelCmd.Flags().StringP("output", "o", "", "output file path")

	_ = viper.BindPFlag("sentinel.output", sentinelCmd.Flags().Lookup("output"))

	rootCmd.AddCommand(sentinelCmd)
}
