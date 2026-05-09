// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"embed"
	"errors"
	"go/token"
	"strings"
	"testing"
	"text/template"

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

// skippableData implements [generator.SkippableData] so the
// HasContent short-circuit branch in [generator.Pipeline.Run] is
// reachable from tests.
type skippableData struct{ empty bool }

func (s *skippableData) HasContent() bool { return !s.empty }

// newPipeline returns a Pipeline pre-wired with the embedded smoke
// template. Tests override individual fields to exercise specific
// behaviors without restating the boilerplate.
func newPipeline() generator.Pipeline[*pipelineData] {
	return generator.Pipeline[*pipelineData]{
		Name:      "smoke",
		Kind:      generator.KindInterface,
		Templates: pipelineTmplFS,
		// Narrowed glob: most tests only parse smoke.go.tmpl. The
		// funcs.go.tmpl fixture in the same dir uses a custom func
		// (upperType) that fails parse unless the test wires it via
		// TemplateFuncs — see "TemplateFuncs are available" subtest.
		TemplatePattern: "testdata/templates/smoke*.tmpl",
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

	t.Run("PostEnrich runs after Enrich and CompositionValidator", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		var order []string
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			order = append(order, "analyze")
			return &pipelineData{Type: args[0]}, nil
		}
		pipe.Enrich = func(_ *pipelineData, _ *generator.Package) error {
			order = append(order, "enrich")
			return nil
		}
		pipe.PostEnrich = func(_ *pipelineData, _ *generator.Package) error {
			order = append(order, "post")
			return nil
		}
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.NoError(t, err, "Run")
		testkit.Equal(t, order, []string{"analyze", "enrich", "post"},
			"PostEnrich runs after Enrich")
	})

	t.Run("PostEnrich error propagates and stops the pipeline", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		want := errors.New("post boom")
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		pipe.PostEnrich = func(_ *pipelineData, _ *generator.Package) error { return want }
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.True(t, errors.Is(err, want), "PostEnrich error surfaces")
	})

	t.Run("Enrich error propagates and stops the pipeline", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		want := errors.New("enrich boom")
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		pipe.Enrich = func(_ *pipelineData, _ *generator.Package) error { return want }
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.True(t, errors.Is(err, want), "Enrich error surfaces")
	})

	t.Run("DirectiveValidator error halts before Enrich", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		want := errors.New("unknown directive")
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			iface, _ := pkg.Interface("Store")
			return &pipelineData{Type: args[0], Methods: iface.Methods}, nil
		}
		pipe.Methods = func(d *pipelineData) []generator.MethodInfo { return d.Methods }
		pipe.DirectiveValidator = func(_ []generator.MethodInfo) []error { return []error{want} }
		enrichRan := false
		pipe.Enrich = func(_ *pipelineData, _ *generator.Package) error {
			enrichRan = true
			return nil
		}
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.True(t, errors.Is(err, want), "DirectiveValidator error surfaces")
		testkit.False(t, enrichRan, "Enrich must not run after directive-validation error")
	})

	t.Run("CompositionValidator error halts the pipeline", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		want := errors.New("conflict")
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			iface, _ := pkg.Interface("Store")
			return &pipelineData{Type: args[0], Methods: iface.Methods}, nil
		}
		pipe.Methods = func(d *pipelineData) []generator.MethodInfo { return d.Methods }
		pipe.CompositionValidator = func(_ []directive.Directive, _ token.Position) error { return want }
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.True(t, errors.Is(err, want), "CompositionValidator error surfaces")
	})

	t.Run("SkippableData with no content short-circuits before render", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		pipe := generator.Pipeline[*skippableData]{
			Name:      "skip",
			Kind:      generator.KindInterface,
			Templates: pipelineTmplFS,
			Analyze: func(_ *generator.Package, _ []string, _ generator.Config, _ generator.Options) (*skippableData, error) {
				return &skippableData{empty: true}, nil
			},
			Renderers: []generator.Renderer[*skippableData]{
				{TemplateName: "smoke.go.tmpl", Path: func(o generator.Options) string { return o.Output }},
			},
		}
		res, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.NoError(t, err, "skippable run succeeds")
		testkit.Len(t, res.Files, 0, "no files emitted on empty content")
	})

	t.Run("Renderer.Transform reshapes data per output", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		pipe.Renderers = []generator.Renderer[*pipelineData]{
			{
				TemplateName: "smoke.go.tmpl",
				Path:         func(o generator.Options) string { return o.Output },
				Transform: func(d *pipelineData, _ generator.Options) *pipelineData {
					return &pipelineData{Type: d.Type + "Transformed"}
				},
			},
		}
		res, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.NoError(t, err, "Run")
		testkit.Assert(t, string(res.Files[0].Content)).
			Contains("type StoreTransformed struct", "Transform's output reached the template")
	})

	t.Run("PostRender appends auxiliary files", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		pipe.PostRender = func(_ *pipelineData, _ generator.Options) ([]generator.OutputFile, error) {
			return []generator.OutputFile{{Path: "wire.json", Content: []byte("{}")}}, nil
		}
		res, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.NoError(t, err, "Run")
		testkit.Len(t, res.Files, 2, "primary + auxiliary")
		testkit.Equal(t, res.Files[1].Path, "wire.json", "auxiliary path appended")
	})

	t.Run("PostRender error propagates", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		want := errors.New("post-render boom")
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		pipe.PostRender = func(_ *pipelineData, _ generator.Options) ([]generator.OutputFile, error) {
			return nil, want
		}
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.True(t, errors.Is(err, want), "PostRender error surfaces")
	})

	t.Run("template parse error surfaces with pipeline name", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		pipe := newPipeline()
		// Pattern that matches no files → ParseFS errors.
		pipe.TemplatePattern = "testdata/templates/does-not-exist.tmpl"
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.True(t, err != nil, "missing pattern errors")
		testkit.Assert(t, err.Error()).Contains("parse templates for smoke", "pipeline-named diagnostic")
	})

	t.Run("renderer error surfaces with pipeline + template name", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		pipe := newPipeline()
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		pipe.Renderers = []generator.Renderer[*pipelineData]{
			{TemplateName: "missing.tmpl", Path: func(o generator.Options) string { return o.Output }},
		}
		_, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.True(t, err != nil, "missing template name errors")
		testkit.Assert(t, err.Error()).Contains("render smoke/missing.tmpl",
			"diagnostic carries pipeline + template name")
	})

	t.Run("TemplateFuncs are available to templates", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		pipe := newPipeline()
		// Override the renderer to use a template that calls the
		// data-aware func registered via TemplateFuncs. Using a parsed
		// inline template via the Renderer's TemplateName mechanism
		// won't work — easiest is to use the existing smoke template
		// via a sub-template that calls our func. So instead we
		// register a func and assert it was wired by reading the
		// rendered output for a marker.
		pipe.TemplatePattern = "testdata/templates/funcs*.tmpl"
		pipe.Renderers = []generator.Renderer[*pipelineData]{
			{TemplateName: "funcs.go.tmpl", Path: func(o generator.Options) string { return o.Output }},
		}
		pipe.Analyze = func(_ *generator.Package, args []string, _ generator.Config, _ generator.Options) (*pipelineData, error) {
			return &pipelineData{Type: args[0]}, nil
		}
		pipe.TemplateFuncs = func(d *pipelineData) template.FuncMap {
			return template.FuncMap{
				"upperType": func() string { return strings.ToUpper(d.Type) },
			}
		}
		res, err := pipe.Run(pkg, []string{"Store"}, generator.DefaultConfig(), generator.Options{Output: "x.go"})
		testkit.NoError(t, err, "Run")
		testkit.Assert(t, string(res.Files[0].Content)).
			Contains("// upper: STORE", "TemplateFuncs func ran with data closure")
	})
}
