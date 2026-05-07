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

	"github.com/spf13/viper"

	"go.thesmos.sh/testkit/gen"
)

// execGoList runs `go list .` in dir and returns the import path.
// Unlike loader.Load, this doesn't type-check — it works even when
// the package has compilation errors (e.g., broken generated files).
func execGoList(dir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*1e9)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "list", ".")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go list in %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

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

	pattern := viper.GetString("package")
	if pattern == "" {
		pattern = "."
	}

	opts := gen.Options{
		Output:     output,
		Check:      viper.GetBool("check"),
		Verbose:    viper.GetBool("verbose"),
		WorkDir:    workDir,
		SourceFile: os.Getenv("GOFILE"),
	}

	loader := gen.NewLoader()

	// When loading from a remote package (-p), resolve the CWD's
	// import path so the output gets the right package name and
	// import qualifiers. We use `go list` instead of full package
	// loading because the CWD may contain broken generated files
	// from a previous run (chicken-and-egg on first generation).
	if pattern != "." {
		opts.OutputPackage = filepath.Base(workDir)
		out, err := execGoList(workDir)
		if err == nil && out != "" {
			opts.OutputImportBase = out
		}
	}
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
