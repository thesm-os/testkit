// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/generator"
)

// runGenerator executes a generator with the standard CLI pipeline:
// resolve output path, hydrate Config + Options from Viper, load the
// package, run the generator, and write the result.
func runGenerator(g generator.Generator, outputKey string, args []string) error {
	output := viper.GetString(outputKey)
	if output == "" {
		return errors.New("cli: -o flag is required")
	}

	workDir := WorkDir()

	cfg := generator.Config{
		TestPackageSuffix: viper.GetString("test-package-suffix"),
		GeneratedSuffix:   viper.GetString("generated-suffix"),
		TestPackageStyle:  viper.GetString("test-package-style"),
		Stub: generator.StubConfig{
			FilePattern: viper.GetString("stub.file-pattern"),
			TypeSuffix:  viper.GetString("stub.type-suffix"),
		},
	}

	pattern := viper.GetString("package")
	if pattern == "" {
		pattern = "."
	}

	opts := generator.Options{
		Output:     output,
		Check:      viper.GetBool("check"),
		Verbose:    viper.GetBool("verbose"),
		WorkDir:    workDir,
		SourceFile: os.Getenv("GOFILE"),
		Invocation: strings.Join(os.Args[1:], " "),
	}

	// Remote-package mode: load types from elsewhere but emit into
	// the CWD's package. The CWD's import path comes from `go list`
	// — which works even when the directory contains broken
	// generated files (chicken-and-egg on first run).
	if pattern != "." {
		opts.OutputPackage = filepath.Base(workDir)
		out, err := execGoList(workDir)
		if err == nil && out != "" {
			opts.OutputImportBase = out
		}
	}

	pkg, err := generator.NewLoader().Load(pattern, workDir)
	if err != nil {
		return fmt.Errorf("load package: %w", err)
	}

	result, err := g.Generate(pkg, args, cfg, opts)
	if err != nil {
		return err
	}
	return generator.WriteResult(result, workDir, opts.Check)
}

// execGoList runs `go list .` in dir and returns the import path.
// Unlike loader.Load, this doesn't type-check — it works even when
// the package has compilation errors (e.g., broken generated files
// from a previous run, the chicken-and-egg case on first generate).
func execGoList(dir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "list", ".")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go list in %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}
