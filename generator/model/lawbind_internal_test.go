// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/suite"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestClockedLawBinding pins the timeaware instantiation: the ctor spelled
// from the timeaware package, the offsets pool composed bounded, and the
// Advance handle marking the binding clocked.
func TestClockedLawBinding(t *testing.T) {
	t.Parallel()

	t.Run("an Advance handle marks the binding clocked in the timeaware package", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		r := tiers.Rule{Law: lawid.ScheduledFiresAfterAdvance, Fields: []tiers.Field{
			{Name: "Advance", Kind: tiers.KindHandle, From: "clock"},
		}}
		lb, ok := lawOf(b, nil, r, nil, nil, nil)
		testkit.True(t, ok, "the handle binds")
		testkit.True(t, lb.Clocked, "and marks the law clocked")
		testkit.True(t, b.UsesClock, "so the property declares the clock")
	})

	t.Run("the offsets pool composes bounded durations", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		r := tiers.Rule{Law: lawid.ScheduledFiresAfterAdvance, Fields: []tiers.Field{
			{Name: "Offsets", Kind: tiers.KindGenerator, From: "offsets"},
		}}
		field, reason := lawFieldOf(b, nil, r, r.Fields[0], nil, nil)
		testkit.True(t, reason == "", "the pool composes: "+reason)
		testkit.Equal(t, field.Pool, "offsets", "under its own name")
		testkit.True(t, len(b.LawPools) == 1 && b.LawPools[0].Offsets,
			"and the declared pool carries the bounded-duration form")
	})
}

// TestLawSubsumption pins the dedup the optional roles forced: the same law
// re-selected from a partner carrier binds without the refinement only the
// directive's host resolves, and the richer binding must subsume the poorer
// in either arrival order — while genuinely distinct same-ID bindings, one
// per method, both stay.
func TestLawSubsumption(t *testing.T) {
	t.Parallel()

	rich := &LawBinding{ID: "AUTO-X", Fields: []*LawField{
		{Name: "Publish", Method: "Publish"},
		{Name: "Redeliver", Method: "Republish"},
	}}
	poor := &LawBinding{ID: "AUTO-X", Fields: []*LawField{
		{Name: "Publish", Method: "Publish"},
	}}
	other := &LawBinding{ID: "AUTO-X", Fields: []*LawField{
		{Name: "Publish", Method: "Broadcast"},
	}}

	testkit.True(t, lawSubsumes(rich, poor), "the refinement covers its own omission")
	testkit.False(t, lawSubsumes(poor, rich), "and never the other way around")
	testkit.False(t, lawSubsumes(rich, other), "a different resolved method is a different law instance")
	testkit.False(t, lawSubsumes(&LawBinding{ID: "AUTO-Y"}, poor), "as is a different identifier")
}
