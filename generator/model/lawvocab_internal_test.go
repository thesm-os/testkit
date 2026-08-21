// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"path"
	"testing"

	"go.thesmos.sh/testkit"
)

// A field's emit kind IS the template that renders it, so a shape with
// no template renders nothing and the law struct comes out short a
// field.
//
// Short, not broken: the composite literal still forms, the file still
// formats, and the missing closure reads as a law that binds in its
// unrefined form. It fails in whichever corpus package arms it, which is
// a long way from the shape table that caused it.
func TestEveryShapeHasATemplate(t *testing.T) {
	t.Parallel()

	for _, s := range lawShapes() {
		testkit.True(t, templateExists(t, s),
			"the closure shape "+string(s)+" has no template under "+LawFieldKindPrefix)
	}
}

// The vocabulary is written out here as well as declared above, so a
// shape added without a template fails at the count before anyone has to
// notice the render came up short.
//
// Written out rather than derived, because deriving it from the consts
// is not possible in Go and deriving it from the template directory
// would assert that the templates match themselves.
func TestShapeVocabularyIsAccountedFor(t *testing.T) {
	t.Parallel()

	testkit.Len(t, lawShapes(), 38,
		"a shape added to lawvocab.go needs a row here and a template beside it; "+
			"both, because either alone renders a law that is quietly missing an arm")
}

// lawShapes is the closure vocabulary, one row per const declared in
// lawvocab.go.
func lawShapes() []lawShape {
	return []lawShape{
		shapeKeyedRead, shapeValueOp, shapeDrainSlice, shapeDrainSeq,
		shapeScalar, shapeScalarLen, shapeBoolCall, shapeResultCall,
		shapeInputCall, shapeCtxOp, shapeErrOp, shapeKeyedOp, shapeKVOp,
		shapeSum, shapeMerge, shapeSave, shapeAppendOff, shapeReplay,

		shapeOkOp, shapeNextOp, shapeDoOp, shapePinnedWrite, shapeCtxOpFixed,
		shapeScheduleAt, shapeCountObs, shapeSubscribe, shapeCtxKeyedOp,

		shapeHandleCall, shapeHandleOp, shapeSagaRun, shapeComputeCall,
		shapeBodyRun, shapePageRead,

		shapeKeyedHandle, shapeKeyedWrite, shapeHandleWrite,

		shapePeerSync, shapeEachSettle,
	}
}

// templateExists reports whether the shape's template is in the embedded
// tree, read from the same filesystem the backend dispatches through.
func templateExists(t *testing.T, s lawShape) bool {
	t.Helper()
	name := LawFieldKindPrefix + string(s) + ".tmpl"
	_, err := goTemplatesFS.ReadFile(path.Join("templates", "golang", "law", name))
	return err == nil
}
