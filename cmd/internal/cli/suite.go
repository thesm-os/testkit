// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/generator/suite"
)

var suiteCmd = &cobra.Command{
	Use:   "suite [-o path] Type",
	Short: "Generate conformance spec for a Go interface",
	Long:  "Generate AssertContract function with per-method directive-derived conformance subtests.",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runGenerator(&suite.Generator{}, "suite.output", args)
	},
}

func init() {
	suiteCmd.Flags().StringP("output", "o", "", "output file path")

	_ = viper.BindPFlag("suite.output", suiteCmd.Flags().Lookup("output"))

	rootCmd.AddCommand(suiteCmd)
}
