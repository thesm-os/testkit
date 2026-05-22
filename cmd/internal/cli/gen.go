// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"go.thesmos.sh/eidos/backend/golang"
	eidoscli "go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/core/diag"
	frontendgolang "go.thesmos.sh/eidos/frontend/golang"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"
)

// eidosBrand drives the generated header marker, config file
// discovery (`.testkit.yaml`), and state directory (`.testkit/`).
const eidosBrand = "testkit"

func init() {
	gen := &cobra.Command{
		Use:   "gen <command> [flags] [args...]",
		Short: "Directive-driven code generation",
		Long: `Scan source packages for //testkit: directives and generate test
infrastructure (stubs, suites, benchmarks, model harnesses).

Subcommands:
  run       Execute the pipeline and write outputs.
  plan      Print the resolved plugin order without running.
  explain   Inspect provenance for an entity, slot, or meta key.
  check     Verify generated files match on-disk state (CI drift gate).
  prune     Delete previously-generated outputs no longer needed.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	for _, sub := range []struct {
		name, use, short string
		setup            func(*eidoscli.Env, *flag.FlagSet, eidosCmd)
	}{
		{"run", "run [flags] [patterns...]", "Execute the pipeline and write outputs", setupRun},
		{"plan", "plan [flags]", "Print the resolved plugin order", nil},
		{"explain", "explain [flags] <selector>", "Inspect provenance for an entity", setupExplain},
		{"check", "check [flags]", "Report drift vs. disk", nil},
		{"prune", "prune [flags]", "Delete orphaned outputs", nil},
	} {
		gen.AddCommand(&cobra.Command{
			Use:                sub.use,
			Short:              sub.short,
			DisableFlagParsing: true,
			SilenceUsage:       true,
			RunE: func(_ *cobra.Command, _ []string) error {
				return dispatch(sub.name, sub.setup)
			},
		})
	}
	rootCmd.AddCommand(gen)
}

// eidosCmd is the subset of eidos CLI commands we dispatch to.
type eidosCmd interface {
	RegisterFlags(fs *flag.FlagSet)
	Execute(ctx context.Context, env *eidoscli.Env) int
}

// newCmd returns the command kernel for the given subcommand name.
func newCmd(name string, plugins []plugin.Plugin) eidosCmd {
	switch name {
	case "run":
		return &eidoscli.RunCommand{Config: eidoscli.RunConfig{Plugins: plugins}}
	case "plan":
		return &eidoscli.PlanCommand{Config: eidoscli.PlanConfig{Plugins: plugins}}
	case "explain":
		return &eidoscli.ExplainCommand{Config: eidoscli.ExplainConfig{Plugins: plugins}}
	case "check":
		return &eidoscli.CheckCommand{Config: eidoscli.CheckConfig{Plugins: plugins}}
	case "prune":
		return &eidoscli.PruneCommand{Config: eidoscli.PruneConfig{Plugins: plugins}}
	default:
		panic("cli: unknown gen subcommand: " + name)
	}
}

// setupRun applies run-specific post-parse config: positional args
// become the frontend's source patterns.
func setupRun(_ *eidoscli.Env, fs *flag.FlagSet, cmd eidosCmd) {
	cmd.(*eidoscli.RunCommand).Config.Patterns = fs.Args()
}

// setupExplain applies explain-specific post-parse config: the first
// positional arg becomes the selector.
func setupExplain(_ *eidoscli.Env, fs *flag.FlagSet, cmd eidosCmd) {
	if fs.NArg() > 0 {
		cmd.(*eidoscli.ExplainCommand).Config.Selector = fs.Arg(0)
	}
}

// dispatch is the single path every `testkit gen <sub>` takes:
// construct env → build command → parse flags → load config →
// apply post-parse setup → execute.
func dispatch(
	name string,
	setup func(*eidoscli.Env, *flag.FlagSet, eidosCmd),
) error {
	env, err := eidoscli.NewEnv(eidosBrand)
	if err != nil {
		return fmt.Errorf("cli: %w", err)
	}
	if env.Diag == nil {
		env.Diag = diag.New()
	}

	cmd := newCmd(name, testkitPlugins())

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	w := io.Writer(io.Discard)
	if env.Stderr != nil {
		w = env.Stderr
	}
	fs.SetOutput(w)
	cfgPath := fs.String(eidoscli.FlagConfig, "", eidoscli.UsageConfig)
	cmd.RegisterFlags(fs)

	if parseErr := fs.Parse(argsAfter(os.Args, name)); parseErr != nil {
		os.Exit(eidoscli.ExitUserError)
	}

	cfg, err := loadEidosConfig(env, *cfgPath)
	if err != nil {
		fmt.Fprintf(env.Stderr, "testkit gen %s: %v\n", name, err)
		os.Exit(eidoscli.ExitUserError)
	}
	setConfigOn(cmd, cfg)

	if setup != nil {
		setup(env, fs, cmd)
	}

	if code := cmd.Execute(context.Background(), env); code != eidoscli.ExitOK {
		os.Exit(code)
	}
	return nil
}

// setConfigOn applies the loaded config to the command. Each command
// type stores it in its own Config.File field.
func setConfigOn(cmd eidosCmd, cfg *eidoscli.Config) {
	switch c := cmd.(type) {
	case *eidoscli.RunCommand:
		c.Config.File = cfg
	case *eidoscli.PlanCommand:
		c.Config.File = cfg
	case *eidoscli.ExplainCommand:
		c.Config.File = cfg
	case *eidoscli.CheckCommand:
		c.Config.File = cfg
	case *eidoscli.PruneCommand:
		c.Config.File = cfg
	}
}

// testkitPlugins returns the static plugin universe. Generator
// plugins are added here as they're implemented.
func testkitPlugins() []plugin.Plugin {
	shapes := shape.New().
		Detectors(detectors.All()...).
		Contracts(contracts.All()...).
		Mixins(mixins.All()...)

	return []plugin.Plugin{
		// Frontend — parse Go source into the store.
		frontendgolang.New(),

		// Annotators — detect shapes, resolve contracts, validate.
		shapes,
		shapes.Resolver(),
		shapes.Validator(),

		// TODO: add testkit generator plugins (stub, suite, bench).

		// Backend — render emit graph to Go source.
		golang.New(),
	}
}

func loadEidosConfig(env *eidoscli.Env, explicit string) (*eidoscli.Config, error) {
	if explicit != "" {
		cfg, err := eidoscli.LoadConfig(explicit)
		if err != nil {
			return nil, fmt.Errorf("cli: load config %s: %w", explicit, err)
		}
		return cfg, nil
	}
	if path, ok := eidoscli.DiscoverConfig(env.Workdir, env.ConfigFileName()); ok {
		cfg, err := eidoscli.LoadConfig(path)
		if err != nil {
			return nil, fmt.Errorf("cli: load config %s: %w", path, err)
		}
		return cfg, nil
	}
	return eidoscli.DefaultConfig(), nil
}

func argsAfter(args []string, needle string) []string {
	for i, a := range args {
		if a == needle {
			return args[i+1:]
		}
	}
	return nil
}
