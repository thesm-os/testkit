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
	"go.thesmos.sh/testkit/gen/enum"
)

var enumCmd = &cobra.Command{
	Use:   "enum [-o path] Type [Type...]",
	Short: "Generate exhaustiveness tests for Go enum types",
	Long:  "Scan const blocks of named types and generate exhaustiveness, distinctness, and stringer tests.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runEnum,
}

func init() {
	enumCmd.Flags().StringP("output", "o", "", "output file path")

	_ = viper.BindPFlag("enum.output", enumCmd.Flags().Lookup("output"))

	rootCmd.AddCommand(enumCmd)
}

func runEnum(_ *cobra.Command, args []string) error {
	output := viper.GetString("enum.output")
	if output == "" {
		fmt.Fprintln(os.Stderr, "testkit enum: -o flag is required")
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

	g := &enum.Generator{}
	result, err := g.Generate(pkg, args, cfg, opts)
	if err != nil {
		return err
	}

	return gen.WriteResult(result, workDir, opts.Check)
}
