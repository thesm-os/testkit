// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/gen/stub"
)

var stubCmd = &cobra.Command{
	Use:   "stub [-o path] Type [Type...]",
	Short: "Generate test stubs for Go interfaces",
	Long:  "Generate configurable test doubles with recording, fault injection, and strict mode.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runGenerator(&stub.Generator{}, "stub.output", args)
	},
}

func init() {
	stubCmd.Flags().StringP("output", "o", "", "output file path")
	stubCmd.Flags().String("test-package-style", "", "test package style: external or internal")
	stubCmd.Flags().String("stub-file-pattern", "", "file base name pattern (default: {type}_stub)")
	stubCmd.Flags().String("stub-type-suffix", "", "generated type suffix (default: Stub)")

	_ = viper.BindPFlag("stub.output", stubCmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("test-package-style", stubCmd.Flags().Lookup("test-package-style"))
	_ = viper.BindPFlag("stub.file-pattern", stubCmd.Flags().Lookup("stub-file-pattern"))
	_ = viper.BindPFlag("stub.type-suffix", stubCmd.Flags().Lookup("stub-type-suffix"))

	rootCmd.AddCommand(stubCmd)
}
