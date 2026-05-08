// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/generator/model"
)

var modelCmd = &cobra.Command{
	Use:   "model [-o path] Type",
	Short: "Generate property-based model tests for a Go interface",
	Long:  "Generate AssertModel function with shape-derived state machine actions, auto-laws, and reference synthesis.",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runGenerator(&model.Generator{}, "model.output", args)
	},
}

func init() {
	modelCmd.Flags().StringP("output", "o", "", "output file path")

	_ = viper.BindPFlag("model.output", modelCmd.Flags().Lookup("output"))

	rootCmd.AddCommand(modelCmd)
}
