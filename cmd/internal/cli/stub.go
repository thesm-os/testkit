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
	"go.thesmos.sh/testkit/gen/stub"
)

var stubCmd = &cobra.Command{
	Use:   "stub [-o path] Type [Type...]",
	Short: "Generate test stubs for Go interfaces",
	Long:  "Generate configurable test doubles with recording, fault injection, and strict mode.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runStub,
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

func runStub(_ *cobra.Command, args []string) error {
	output := viper.GetString("stub.output")
	if output == "" {
		fmt.Fprintln(os.Stderr, "testkit stub: -o flag is required")
		return errors.New("-o flag is required")
	}

	workDir := WorkDir()

	cfg := gen.Config{
		TestPackageSuffix: viper.GetString("test-package-suffix"),
		GeneratedSuffix:   viper.GetString("generated-suffix"),
		TestPackageStyle:  viper.GetString("test-package-style"),
		Stub: gen.StubConfig{
			FilePattern: viper.GetString("stub.file-pattern"),
			TypeSuffix:  viper.GetString("stub.type-suffix"),
		},
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

	g := &stub.Generator{}
	result, err := g.Generate(pkg, args, cfg, opts)
	if err != nil {
		return err
	}

	return gen.WriteResult(result, workDir, opts.Check)
}
