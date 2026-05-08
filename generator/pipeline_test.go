// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"embed"
	"errors"
	"go/token"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
)

//go:embed testdata/templates/*.tmpl
var pipelineTmplFS embed.FS

type pipelineData struct {
	Type    string
	Methods []generator.MethodInfo
}

// newPipeline returns a Pipeline pre-wired with the embedded smoke
// template. Tests override individual fields to exercise specific
// behaviors without restating the boilerplate.
func newPipeline() generator.Pipeline[*pipelineData] {
	return generator.Pipeline[*pipelineData]{
		Name:            "smoke",
		Kind:            generator.KindInterface,
		Templates:       pipelineTmplFS,
		TemplatePattern: "testdata/templates/*.tmpl",
		Renderers: []generator.Renderer[*pipelineData]{
			{TemplateName: "smoke.go.tmpl", Path: func(o generator.Options) string { return o.Output }},
		},
	}
}

func TestPipeline(t *testing.T) {
	t.Parallel()

	t.Run("Run produces a file with template output", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		res, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "out.go"})
		testkit.NoError(t, err, "Run")
		testkit.Len(t, res.Files, 1, "single output file")
		testkit.Equal(t, res.Files[0].Path, "out.go", "Path matches Options.Output")
		testkit.Assert(t, string(res.Files[0].Content)).Contains("type Store struct", "rendered Type")
	})

	t.Run("Options.Invocation overrides reconstructed Source attribution", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		res, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{
			Output:     "out.go",
			Invocation: "smoke -o out.go --some-flag value Store",
		})
		testkit.NoError(t, err, "Run")
		testkit.Assert(t, string(res.Files[0].Content)).
			Contains("// Source: ", "Source line present").
			Contains("smoke -o out.go --some-flag value Store",
				"verbatim invocation preserved (flags + args)")
	})

	t.Run("Run falls back to reconstructed Args when Invocation is empty", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		res, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(),
			generator.Options{Output: "out.go"})
		testkit.NoError(t, err, "Run")
		testkit.Assert(t, string(res.Files[0].Content)).
			Contains("testkit smoke Store", "fallback uses subcommand + args")
	})

	t.Run("Analyze errors propagate through Run", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		want := errors.New("analyze boom")
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, _ []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return nil, want
		}
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{})
		testkit.True(t, errors.Is(err, want), "Analyze error propagates")
	})

	t.Run("Run rejects pipelines without Analyze", func(t *testing.T) {
		t.Parallel()
		pipe := newPipeline()
		_, err := pipe.Run(loadBasic(t), nil, generator.DefaultConfig(), generator.Options{})
		testkit.True(t, err != nil, "missing Analyze must error")
		testkit.Assert(t, err.Error()).Contains("no Analyze", "diagnostic mentions Analyze")
	})

	t.Run("Run rejects pipelines without Renderers", func(t *testing.T) {
		t.Parallel()
		pipe := generator.Pipeline[*pipelineData]{
			Name: "bad",
			Analyze: func(_ *generator.Package, _ []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
				return &pipelineData{}, nil
			},
		}
		_, err := pipe.Run(loadBasic(t), nil, generator.DefaultConfig(), generator.Options{})
		testkit.True(t, err != nil, "missing Renderers must error")
		testkit.Assert(t, err.Error()).Contains("no Renderers", "diagnostic mentions Renderers")
	})

	t.Run("Run fails when args don't match Kind", func(t *testing.T) {
		t.Parallel()
		pipe := newPipeline()
		pipe.Kind = generator.KindStruct
		pipe.Analyze = func(_ *generator.Package, _ []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{}, nil
		}
		// Store is an interface — KindStruct rejects it.
		_, err := pipe.Run(loadBasic(t), []string{"Store"}, generator.DefaultConfig(), generator.Options{})
		testkit.True(t, err != nil, "wrong kind must error")
	})

	t.Run("CompositionValidator runs once per method", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		called := 0
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			iface, _ := pkg.Interface("Store")
			return &pipelineData{Type: args[0], Methods: iface.Methods}, nil
		}
		pipe.Methods = func(d *pipelineData) []generator.MethodInfo { return d.Methods }
		pipe.CompositionValidator = func(_ []directive.Directive, _ token.Position) error {
			called++
			return nil
		}
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.NoError(t, err, "Run")
		testkit.Equal(t, called, 2, "called once per method (Get + Put)")
	})

	// Compile-time guard so unused-import warnings stay quiet when
	// other subtests are stubbed out.
	_ = strings.TrimSpace
}
