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
	"go.thesmos.sh/testkit/gen/builder"
)

var builderCmd = &cobra.Command{
	Use:   "builder [-o path] Type [Type...]",
	Short: "Generate fluent builders for Go structs",
	Long:  "Generate test fixture builders with fluent With* setters for Go struct types.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runBuilder,
}

func init() {
	builderCmd.Flags().StringP("output", "o", "", "output file path")

	_ = viper.BindPFlag("builder.output", builderCmd.Flags().Lookup("output"))

	rootCmd.AddCommand(builderCmd)
}

func runBuilder(_ *cobra.Command, args []string) error {
	output := viper.GetString("builder.output")
	if output == "" {
		fmt.Fprintln(os.Stderr, "testkit builder: -o flag is required")
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

	g := &builder.Generator{}
	result, err := g.Generate(pkg, args, cfg, opts)
	if err != nil {
		return err
	}

	return gen.WriteResult(result, workDir, opts.Check)
}
