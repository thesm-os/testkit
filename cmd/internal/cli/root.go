// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cli implements the testkit command-line interface using
// Cobra for subcommand dispatch and Viper for config file + flag merging.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configFileName = ".testkit"
	configFileType = "yml"
)

var rootCmd = &cobra.Command{
	Use:          "testkit",
	Short:        "Generate test infrastructure for Go projects",
	Long:         "testkit generates stubs, suites, builders, and other test infrastructure from Go type information.",
	SilenceUsage: true,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("config", "", "path to .testkit.yml (default: auto-discover)")
	rootCmd.PersistentFlags().StringP(
		"package", "p", ".", "source package to load types from (import or relative path)",
	)
	rootCmd.PersistentFlags().Bool("check", false, "dry-run mode — compare output, error if different")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")

	_ = viper.BindPFlag("package", rootCmd.PersistentFlags().Lookup("package"))
	_ = viper.BindPFlag("check", rootCmd.PersistentFlags().Lookup("check"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// Execute runs the root command.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	cfgFile, _ := rootCmd.PersistentFlags().GetString("config")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(configFileName)
		viper.SetConfigType(configFileType)
		viper.AddConfigPath(".")
		addParentConfigPaths()
	}

	// Defaults matching gen.DefaultConfig().
	viper.SetDefault("generated-suffix", ".gen.go")
	viper.SetDefault("test-package-style", "external")
	viper.SetDefault("test-package-suffix", "test")
	viper.SetDefault("stub.file-pattern", "{type}_stub")
	viper.SetDefault("stub.type-suffix", "Stub")

	_ = viper.ReadInConfig() // missing file is OK — defaults apply
}

func addParentConfigPaths() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		viper.AddConfigPath(parent)
		if isRepoRoot(parent) {
			break
		}
		dir = parent
	}
}

func isRepoRoot(dir string) bool {
	for _, marker := range []string{".git", "go.work"} {
		_, err := os.Stat(filepath.Join(dir, marker))
		if err == nil {
			return true
		}
	}
	return false
}

// WorkDir returns the working directory for code generation.
func WorkDir() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "testkit: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	return dir
}
