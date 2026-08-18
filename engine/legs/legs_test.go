// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package legs_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/legs"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/suite"
)

func intSubject() suite.Subject[int] {
	return suite.Subject[int]{Name: "int", New: func(testing.TB) int { return 0 }}
}

// noop is the cheapest action that keeps a leg's stream non-empty.
func noop() model.Action[int] {
	return model.Action[int]{
		Name: "Noop",
		Run:  func(*rapid.T, int, int) model.ActionResult { return model.ActionResult{} },
	}
}

// TestReferencePicksTheDerivedFallback pins the derived arm and its
// tier. The differential arm needs the runner to install the oracle's
// constructor on the subject, which only suite.Run does — the bus
// pack's hard-mode run is that arm's evidence.
func TestReferencePicksTheDerivedFallback(t *testing.T) {
	t.Parallel()

	build, tier := legs.Reference(t, intSubject(), func() int { return 7 })
	if tier != suite.TierDerived {
		t.Errorf("with no oracle the tier is derived, got %q", tier)
	}
	if got := build(); got != 7 {
		t.Errorf("the derived constructor must be the one given, built %d", got)
	}
}

// TestBlendRespectsProvenance pins the gate: a restricted pool reaches
// draws verbatim, a derived pool grows the hostile arm.
func TestBlendRespectsProvenance(t *testing.T) {
	t.Parallel()

	pool := []string{"a", "b"}
	inPool := func(v string) bool { return v == "a" || v == "b" }
	hostile := func(s string) string { return "H:" + s }

	restricted := legs.Blend(false, model.SampledFrom(pool), hostile)
	rapid.Check(t, func(rt *rapid.T) {
		if v := restricted.Draw(rt, "v"); !inPool(v) {
			rt.Fatalf("a restricted pool must draw only its own values, got %q", v)
		}
	})

	widened := legs.Blend(true, model.SampledFrom(pool), hostile)
	sawHostile := false
	rapid.Check(t, func(rt *rapid.T) {
		if v := widened.Draw(rt, "v"); !inPool(v) {
			sawHostile = true
		}
	})
	if !sawHostile {
		t.Error("a derived pool must blend the hostile arm; every draw stayed in the pool")
	}
}

// vacuousLaw declines every draw.
type vacuousLaw struct{}

func (vacuousLaw) ID() string    { return "TEST-VACUOUS" }
func (vacuousLaw) REQID() string { return "" }
func (vacuousLaw) Check(*rapid.T, int, int) error {
	return law.Vacuous
}

// holdingLaw engages and holds.
type holdingLaw struct{}

func (holdingLaw) ID() string                     { return "TEST-HOLDS" }
func (holdingLaw) REQID() string                  { return "" }
func (holdingLaw) Check(*rapid.T, int, int) error { return nil }

// TestLawNotesVacuity pins the leg's census handoff: a law that never
// engages logs it, a law that engages does not.
func TestLawNotesVacuity(t *testing.T) {
	t.Parallel()

	sut := func() int { return 0 }
	ref := func() int { return 0 }

	quiet := testkit.NewFailableTB()
	legs.Law[int](quiet, intSubject(), sut, ref,
		[]model.Action[int]{noop()}, []law.Law[int]{vacuousLaw{}})
	if quiet.Failed() {
		t.Fatalf("a vacuous law is not a failure: %s", quiet.Msg())
	}
	if logs := strings.Join(quiet.Logs(), "\n"); !strings.Contains(logs, "no law engaged") {
		t.Errorf("a leg whose law never engaged must say so, logs:\n%s", logs)
	}

	engaged := testkit.NewFailableTB()
	legs.Law[int](engaged, intSubject(), sut, ref,
		[]model.Action[int]{noop()}, []law.Law[int]{holdingLaw{}})
	if engaged.Failed() {
		t.Fatalf("a holding law must pass: %s", engaged.Msg())
	}
	if logs := strings.Join(engaged.Logs(), "\n"); strings.Contains(logs, "no law engaged") {
		t.Errorf("an engaged leg must not claim vacuity, logs:\n%s", logs)
	}
}

// TestDifferentialDrivesTheReference pins the leg end to end: an action
// that diverges reds the leg, one that agrees does not.
func TestDifferentialDrivesTheReference(t *testing.T) {
	t.Parallel()

	agree := testkit.NewFailableTB()
	legs.Differential[int](agree, intSubject(), func() int { return 0 },
		[]model.Action[int]{noop()})
	if agree.Failed() {
		t.Fatalf("an agreeing subject must pass: %s", agree.Msg())
	}

	diverge := testkit.NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		legs.Differential[int](diverge, intSubject(), func() int { return 1 },
			[]model.Action[int]{{
				Name: "Compare",
				Run: func(_ *rapid.T, sut, ref int) model.ActionResult {
					if sut != ref {
						return model.ActionResult{Err: errors.New("sut and ref disagree")}
					}
					return model.ActionResult{}
				},
			}})
	}()
	<-done
	if !diverge.Failed() {
		t.Fatal("a diverging action must red the differential leg")
	}

	// The red above is EXPECTED, so the reproduction files rapid persisted
	// for it are noise plus a stale-replay hazard — the same contract
	// prove keeps: after a clean pass, anything left under testdata/rapid
	// IS a finding.
	dir := filepath.Join("testdata", "rapid", diverge.Name())
	if err := os.RemoveAll(dir); err != nil {
		t.Logf("could not remove %s, the expected red's reproduction files: %v", dir, err)
	}
}

// TestLawNotesPartialVacuity pins the bundle half of the census: one
// engaged law beside one that never engaged keeps the leg green but
// names the quiet law — a bundle where one law fires must not read as
// five laws held.
func TestLawNotesPartialVacuity(t *testing.T) {
	t.Parallel()

	sut := func() int { return 0 }
	ref := func() int { return 0 }

	partial := testkit.NewFailableTB()
	legs.Law[int](partial, intSubject(), sut, ref,
		[]model.Action[int]{noop()}, []law.Law[int]{holdingLaw{}, vacuousLaw{}})
	if partial.Failed() {
		t.Fatalf("one engaged law keeps the leg green: %s", partial.Msg())
	}
	if logs := strings.Join(partial.Logs(), "\n"); !strings.Contains(logs, "TEST-VACUOUS") {
		t.Errorf("the law that never engaged must be named, logs:\n%s", logs)
	}
}

// TestReferencePicksTheDeclaredOracle drives the differential arm end
// to end through suite.Run: the non-oracle subject compares against the
// oracle's constructor, while the oracle itself — which cannot compare
// against itself — rides the derived fallback.
//
//nolint:tparallel // the runner owns leg parallelism inside Run; the wrapper subtest must complete before the map is read.
func TestReferencePicksTheDeclaredOracle(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	got := map[string]struct {
		tier  suite.Tier
		built int
	}{}
	check := suite.Check[int]{
		ID:    "model/probe",
		Class: suite.ClassDifferential,
		Claim: "the oracle arm is reachable",
		RunWith: func(tb testing.TB, sub suite.Subject[int]) {
			tb.Helper()
			build, tier := legs.Reference(tb, sub, func() int { return -1 })
			mu.Lock()
			got[sub.Name] = struct {
				tier  suite.Tier
				built int
			}{tier, build()}
			mu.Unlock()
		},
	}
	t.Run("run", func(t *testing.T) {
		suite.Run(t, suite.Suite[int]{Name: "s", Checks: []suite.Check[int]{check}},
			suite.Subject[int]{Name: "oracle", Oracle: true, New: func(testing.TB) int { return 42 }},
			suite.Subject[int]{Name: "subject", New: func(testing.TB) int { return 0 }},
		)
	})

	mu.Lock()
	defer mu.Unlock()
	if r := got["subject"]; r.tier != suite.TierDifferential || r.built != 42 {
		t.Errorf("the subject must compare against the oracle's constructor: %+v", r)
	}
	if r := got["oracle"]; r.tier != suite.TierDerived || r.built != -1 {
		t.Errorf("the oracle cannot compare against itself and rides the derived fallback: %+v", r)
	}
}
