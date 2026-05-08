// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"embed"
	"fmt"
	"go/token"
	"strings"
	"text/template"

	"go.thesmos.sh/testkit/generator/directive"
)

// Pipeline is the canonical Generate sequence shared by every generator.
// Each generator constructs a Pipeline at package init, configures the
// strategy slots (Analyze, Enrich, Renderers, etc.), and calls Run from
// its [Generator.Generate] method.
//
// The pipeline owns:
//
//   - type validation (gates non-conforming args before analysis)
//   - directive validation (unknown-directive errors)
//   - enrichment dispatch (calls into the generator's Enrich, when set)
//   - composition validation (conflict/required-pair checks)
//   - template parsing
//   - source attribution (filename:minLine-maxLine for the header)
//   - header construction
//   - rendering
//   - output assembly into a [Result]
//
// Generators provide the strategy: which Analyze function, which
// Templates FS, which Renderers, etc.
//
// The type parameter D is the generator-specific data type returned by
// Analyze and consumed by Enrich and Renderers. D is typically a
// pointer-to-struct (e.g. *stub.Data, *suite.SpecData).
type Pipeline[D any] struct {
	// Name is the generator's subcommand name. Surfaces in the file
	// header and error messages.
	Name string

	// Kind constrains the args this generator accepts.
	Kind TypeKind

	// Templates is the embedded filesystem containing the generator's
	// templates/. Parsed once per Run via NewTemplateSet().ParseFS.
	Templates embed.FS

	// TemplatePattern is the glob passed to ParseFS. Defaults to
	// "templates/*.tmpl" when empty.
	TemplatePattern string

	// Analyze builds the data model from the loaded package. Required.
	Analyze func(pkg *Package, args []string, cfg Config, opts Options) (D, error)

	// Enrich runs directive-driven mutation on the data model. Optional —
	// generators that don't consume directives (sentinel, enum) leave
	// this nil.
	Enrich func(data D, pkg *Package) error

	// Positions extracts source positions from the data model so the
	// pipeline can compute the SourceFile attribution. Optional —
	// pipelines without method-bearing data leave this nil.
	Positions func(data D) []token.Position

	// Methods extracts MethodInfo slices from the data model so the
	// pipeline can run directive validation. Optional — generators
	// without method directives (sentinel scanning whole packages)
	// leave this nil.
	Methods func(data D) []MethodInfo

	// DirectiveValidator validates unknown directives. Optional — when
	// nil, no directive validation runs. Most conformance generators
	// pass a directive.Registry's Validate method.
	DirectiveValidator func(methods []MethodInfo) []error

	// CompositionValidator validates directive composition rules
	// (conflicts, required pairs). Optional. Receives all directives
	// from one method per call.
	CompositionValidator func(directives []directive.Directive, pos token.Position) error

	// Renderers list the templates to execute and the paths to write.
	// Order is preserved: a stub's primary file is rendered before its
	// test file.
	Renderers []Renderer[D]

	// PostRender runs after the Renderers produce their output and
	// returns additional output files to append to the [Result].
	// Optional — generators that emit only template output leave this
	// nil. Useful for auxiliary artifacts (e.g. enum's per-type wire
	// JSON goldens) whose count and content depend on data shape and
	// thus can't be expressed as a fixed [Renderer] slice.
	PostRender func(data D, opts Options) ([]OutputFile, error)
}

// Renderer is one (template, output-path) pairing. The Path function
// computes the output path from the run options — typically returning
// opts.Output for the primary file or [TestPathFrom](opts.Output) for
// the test file.
type Renderer[D any] struct {
	// TemplateName is the name passed to ExecuteTemplate. Empty means
	// execute the root template.
	TemplateName string

	// Path computes the output file path from opts. Required.
	Path func(opts Options) string
}

// SkippableData is an opt-in interface a generator's data type can
// implement to short-circuit pipeline rendering when the analyzed
// package has nothing worth emitting (e.g. sentinel against a package
// with no Err* vars and no error types).
//
// When [Analyze] returns a value satisfying SkippableData and
// HasContent reports false, [Pipeline.Run] returns an empty
// [*Result] without invoking any further pipeline step. Generators
// that always emit output (stub, suite, bench, model) leave this
// unimplemented and pay no overhead.
type SkippableData interface {
	HasContent() bool
}

// Run executes the pipeline. Returns the assembled [Result] or an error.
//
// The sequence:
//
//  1. ValidateTypes(args, p.Kind)
//  2. p.Analyze(...)
//  3. p.DirectiveValidator(p.Methods(data))   if both set
//  4. p.Enrich(data)                          if set
//  5. p.CompositionValidator per method       if set
//  6. parse templates
//  7. compute SourceAttribution from p.Positions(data)
//  8. for each Renderer: render and append to Result.Files
func (p *Pipeline[D]) Run(pkg *Package, args []string, cfg Config, opts Options) (*Result, error) {
	if p.Analyze == nil {
		return nil, Errorf(noPos, "pipeline %q has no Analyze function", p.Name)
	}
	if len(p.Renderers) == 0 {
		return nil, Errorf(noPos, "pipeline %q has no Renderers", p.Name)
	}

	// 1. Validate args against the requested kind.
	if errs := ValidateTypes(pkg, args, p.Kind); len(errs) > 0 {
		return nil, errs[0]
	}

	// 2. Analyze.
	data, err := p.Analyze(pkg, args, cfg, opts)
	if err != nil {
		return nil, err
	}

	// Short-circuit when the data type opts into [SkippableData] and
	// reports no content — generators like sentinel emit nothing for
	// packages with no errors to test.
	if skippable, ok := any(data).(SkippableData); ok && !skippable.HasContent() {
		return &Result{}, nil
	}

	// 3. Directive validation (unknown directives are errors).
	if p.DirectiveValidator != nil && p.Methods != nil {
		if errs := p.DirectiveValidator(p.Methods(data)); len(errs) > 0 {
			return nil, errs[0]
		}
	}

	// 4. Enrich.
	if p.Enrich != nil {
		if enrichErr := p.Enrich(data, pkg); enrichErr != nil {
			return nil, enrichErr
		}
	}

	// 5. Composition validation (conflict / required-pair).
	if p.CompositionValidator != nil && p.Methods != nil {
		for _, m := range p.Methods(data) {
			if compErr := p.CompositionValidator(m.Directives, m.Pos); compErr != nil {
				return nil, compErr
			}
		}
	}

	// 6. Parse templates.
	pattern := p.TemplatePattern
	if pattern == "" {
		pattern = "templates/*.tmpl"
	}
	tmpl, err := NewTemplateSet().ParseFS(p.Templates, pattern)
	if err != nil {
		return nil, fmt.Errorf("parse templates for %s: %w", p.Name, err)
	}

	// 7. Source attribution.
	var positions []token.Position
	if p.Positions != nil {
		positions = p.Positions(data)
	}
	sourceFile := SourceAttribution(positions)

	header := Header{
		Subcommand: p.Name,
		Args:       p.Name + " " + strings.Join(args, " "),
		SourceFile: sourceFile,
		BuildTag:   opts.BuildTag,
	}
	if len(args) == 0 {
		header.Args = p.Name
	}
	// Prefer the verbatim CLI invocation when available so the
	// "// Source:" attribution preserves flags (-o, --build-tag, ...)
	// the reconstructed form drops. The CLI runner sets Invocation
	// from os.Args[1:]; direct test callers leave it empty.
	if opts.Invocation != "" {
		header.Args = opts.Invocation
	}

	// 8. Render every Renderer.
	files := make([]OutputFile, 0, len(p.Renderers))
	for _, r := range p.Renderers {
		content, err := renderTemplateOrSet(tmpl, r.TemplateName, data, header)
		if err != nil {
			return nil, fmt.Errorf("render %s/%s: %w", p.Name, r.TemplateName, err)
		}
		files = append(files, OutputFile{
			Path:    r.Path(opts),
			Content: content,
		})
	}

	// 9. PostRender — append auxiliary artifacts (e.g. enum's wire
	//    goldens) computed from the data model.
	if p.PostRender != nil {
		extra, err := p.PostRender(data, opts)
		if err != nil {
			return nil, fmt.Errorf("post-render %s: %w", p.Name, err)
		}
		files = append(files, extra...)
	}
	return &Result{Files: files}, nil
}

// renderTemplateOrSet picks the right ExecuteTemplate path based on
// whether the caller named a sub-template.
func renderTemplateOrSet(tmpl *template.Template, name string, data any, header Header) ([]byte, error) {
	return RenderTemplate(tmpl, name, data, header)
}
