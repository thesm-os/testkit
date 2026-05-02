// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"go.thesmos.sh/testkit/cmd/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print testkit version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(version.String())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
