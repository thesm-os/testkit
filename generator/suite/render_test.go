// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the renderer and
// its view are the unexported seam the emitter drives.
package suite

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

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
