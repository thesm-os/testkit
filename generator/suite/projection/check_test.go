// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

func TestIDPlanRenders(t *testing.T) {
	t.Parallel()

	t.Run("method scope through the vocabulary", func(t *testing.T) {
		t.Parallel()
		got := projection.IDPlan{Method: "Append", Seg: suite.SegSmoke}.Render()
		testkit.Equal(t, got, suite.ID("Append/smoke"), "method IDs render Method/seg")
	})

	t.Run("family scope carries its qualifier", func(t *testing.T) {
		t.Parallel()
		got := projection.IDPlan{Family: suite.FamilyModel, Qualifier: "log", Seg: suite.SegLaws}.Render()
		testkit.Equal(t, got, suite.ID("model/log/laws"), "family IDs qualify unconditionally")
	})
}

func TestIDPlanRefusesMalformedPlans(t *testing.T) {
	t.Parallel()

	t.Run("both scopes", func(t *testing.T) {
		t.Parallel()
		testkit.Panics(t, func() {
			projection.IDPlan{Method: "Append", Family: suite.FamilyModel, Qualifier: "log", Seg: "x"}.Render()
		}, "a plan naming both scopes is a deriver bug")
	})

	t.Run("unqualified family", func(t *testing.T) {
		t.Parallel()
		testkit.Panics(t, func() {
			projection.IDPlan{Family: suite.FamilyModel, Seg: suite.SegLaws}.Render()
		}, "qualification is unconditional; an unqualified family plan is a deriver bug")
	})

	t.Run("empty plan", func(t *testing.T) {
		t.Parallel()
		testkit.Panics(t, func() { projection.IDPlan{}.Render() }, "an empty plan renders nothing")
	})
}
