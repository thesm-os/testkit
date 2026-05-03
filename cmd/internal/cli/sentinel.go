// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/sentinel"
)

var sentinelCmd = &cobra.Command{
	Use:   "sentinel [-o path]",
	Short: "Generate sentinel error tests",
	Long:  "Scan a package for exported Err* variables and generate tests for error message consistency, uniqueness, and unwrap chain preservation.",
	Args:  cobra.NoArgs,
	RunE:  runSentinel,
}

func init() {
	sentinelCmd.Flags().StringP("output", "o", "", "output file path")

	_ = viper.BindPFlag("sentinel.output", sentinelCmd.Flags().Lookup("output"))

	rootCmd.AddCommand(sentinelCmd)
}

func runSentinel(_ *cobra.Command, _ []string) error {
	output := viper.GetString("sentinel.output")
	if output == "" {
		fmt.Fprintln(os.Stderr, "testkit sentinel: -o flag is required")
		return errors.New("-o flag is required")
	}

	workDir := WorkDir()

	cfg := gen.Config{
		TestPackageSuffix: viper.GetString("test-package-suffix"),
		GeneratedSuffix:   viper.GetString("generated-suffix"),
		TestPackageStyle:  viper.GetString("test-package-style"),
	}

	opts := gen.Options{
		Output:     output,
		Check:      viper.GetBool("check"),
		Verbose:    viper.GetBool("verbose"),
		WorkDir:    workDir,
		SourceFile: os.Getenv("GOFILE"),
	}

	loader := gen.NewLoader()
	pkg, err := loader.Load(".", workDir)
	if err != nil {
		return fmt.Errorf("load package: %w", err)
	}

	g := &sentinel.Generator{}
	result, err := g.Generate(pkg, nil, cfg, opts)
	if err != nil {
		return err
	}

	return gen.WriteResult(result, workDir, opts.Check)
}
