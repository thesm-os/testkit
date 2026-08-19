// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the renderer and
// its view are the unexported seam the emitter drives.
package suite

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"text/template"

	langgo "go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// bodyTemplates parses the body subtree exactly as the backend will:
// the prefixed function map read back from the plugin itself, so this
// parse and the run's cannot diverge. Test scaffolding on purpose —
// production rendering is the backend's, and a parse path with no
// production consumer does not belong in production code.
func bodyTemplates() (*template.Template, error) {
	t, err := template.New("body").
		Funcs(New().TemplateFuncs(langgo.Language)).
		Funcs(backendPlaceholders()).
		ParseFS(goTemplatesFS, "templates/golang/body/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("suite: parse the body templates: %w", err)
	}
	return t, nil
}

// renderBody executes one body variant's template — the dispatch is
// the variant's own kind, which IS the template's name, so an
// unregistered variant fails by name rather than rendering nothing.
func renderBody(v bodyView) (string, error) {
	t, err := bodyTemplates()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.ExecuteTemplate(&b, string(v.Body.BodyKind()), v); err != nil {
		return "", fmt.Errorf("suite: render %s: %w", v.Body.BodyKind(), err)
	}
	return b.String(), nil
}

// renderCase is one body variant and the text its template emits.
type renderCase struct {
	name string
	view bodyView
	want string
}

func (c renderCase) Name() string { return c.name }

func TestSmokeBodiesRenderTheirArms(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []renderCase{
		{
			"the plain arm calls and discards by arity",
			bodyView{
				Recv: "l", Check: "logAppend", Discard: "_ =",
				Body: projection.SmokeSurvives{Call: projection.CallPlan{
					Method: "Append",
					Args:   []projection.Expr{projection.ExprCtx, "logEntry()"},
				}},
			},
			"suite.Survives(tb, logAppend, func(ctx context.Context) {\n" +
				"\t_ = l.Append(ctx, logEntry())\n" +
				"})",
		},
		{
			"the opener arm closes what it opens",
			bodyView{
				Recv: "l", Check: "logScan", Discard: "_, _ =",
				Body: projection.SmokeSurvives{
					Call:          projection.CallPlan{Method: "Scan", Args: []projection.Expr{projection.ExprCtx}},
					CloseProduced: "Close",
				},
			},
			"suite.Survives(tb, logScan, func(ctx context.Context) {\n" +
				"\tproduced, err := l.Scan(ctx)\n" +
				"\tif err != nil {\n" +
				"\t\treturn\n" +
				"\t}\n" +
				"\t_ = produced.Close(ctx)\n" +
				"})",
		},
		{
			"the borrow arm borrows first and guards the failed borrow",
			bodyView{
				Recv: "p", Check: "poolPut", Discard: "_ =",
				Body: projection.SmokeSurvives{
					Call: projection.CallPlan{
						Method: "Put",
						Args:   []projection.Expr{projection.ExprCtx, projection.ExprBorrowed},
					},
					Borrow: projection.CallPlan{Method: "Get", Args: []projection.Expr{projection.ExprCtx}},
				},
			},
			"suite.Survives(tb, poolPut, func(ctx context.Context) {\n" +
				"\tborrowed, err := p.Get(ctx)\n" +
				"\tif err != nil {\n" +
				"\t\t// Nothing borrowed, nothing to return; the producer's own\n" +
				"\t\t// smoke judges this path.\n" +
				"\t\treturn\n" +
				"\t}\n" +
				"\t_ = p.Put(ctx, borrowed)\n" +
				"})",
		},
	}, func(t *testing.T, tc renderCase) {
		got, err := renderBody(tc.view)
		testkit.NoError(t, err, "a registered variant renders")
		testkit.Equal(t, got, tc.want, "the emitted body, byte for byte")
	})
}

func TestBodyTemplateCensus(t *testing.T) {
	t.Parallel()

	parsed, err := bodyTemplates()
	testkit.NoError(t, err, "the tree parses under its own function map")

	kinds := map[string]bool{}
	for _, k := range projection.BodyKinds() {
		kinds[string(k)] = true
	}
	defined := 0
	for _, tmpl := range parsed.Templates() {
		name := tmpl.Name()
		if !strings.HasPrefix(name, projection.BodyKindPrefix) {
			continue
		}
		defined++
		testkit.True(t, kinds[name], name+" names a registered body variant — an orphan template renders nothing")
	}
	// One direction only until the emission set completes: every
	// defined template names a variant. The equality gate — every
	// variant owns a template — arms when the last body lands, and
	// the count below is what will flip it.
	testkit.True(t, defined >= 1, "the smoke family is defined")
}

// backendPlaceholders stands in for the helpers the backend owns, so
// this harness can PARSE every body template.
//
// It cannot execute the ones that use them, and deliberately does not
// try: renderExpr and external are how an emitted reference registers
// its import, and a harness that reimplemented them would be pinning
// bytes against a second import machinery rather than against the one
// that runs. The bodies that reach for them are pinned by the pipeline
// golden instead; what this harness still answers is the census —
// every defined template names a registered variant.
func backendPlaceholders() template.FuncMap {
	invoked := func(...any) (string, error) {
		return "", errors.New("suite: a backend helper cannot run in the parse-only harness")
	}
	return template.FuncMap{"renderExpr": invoked, "external": invoked}
}
