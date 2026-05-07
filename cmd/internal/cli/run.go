// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/gen"
)

// runGenerator executes a generator with the standard CLI pipeline:
// resolve output path, build config + options from Viper, load the
// package, generate, and write the result.
func runGenerator(g gen.Generator, outputKey string, args []string) error {
	output := viper.GetString(outputKey)
	if output == "" {
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

	pattern := viper.GetString("package")
	if pattern == "" {
		pattern = "."
	}

	loader := gen.NewLoader()
	pkg, err := loader.Load(pattern, workDir)
	if err != nil {
		return fmt.Errorf("load package: %w", err)
	}

	result, err := g.Generate(pkg, args, cfg, opts)
	if err != nil {
		return err
	}

	return gen.WriteResult(result, workDir, opts.Check)
}
