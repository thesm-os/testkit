// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit/conformance/gate"
)

// The registries are the gate's source of truth, so an empty one would make
// every corpus trivially complete — the failure mode that matters most and the
// one least likely to announce itself.
func TestRegisteredIsNotEmpty(t *testing.T) {
	t.Parallel()

	reg := gate.Registered()
	for _, axis := range []string{gate.AxisDetector, gate.AxisContract, gate.AxisMixin} {
		if len(reg[axis]) == 0 {
			t.Errorf("axis %q registers nothing; the gate would pass vacuously", axis)
		}
	}
}

// Names come straight from eidos and are compared as strings, so a change in
// case or separator upstream silently stops matching. Pinning a few known
// members catches that without pinning the whole vocabulary, which is expected
// to grow.
func TestRegisteredCarriesKnownClassifications(t *testing.T) {
	t.Parallel()

	reg := gate.Registered()
	want := map[string]string{
		gate.AxisDetector: "reader",
		gate.AxisContract: "lease",
		gate.AxisMixin:    "idempotent",
	}
	for axis, name := range want {
		if !slices.Contains(reg[axis], name) {
			t.Errorf("axis %q should register %q, got: %v", axis, name, reg[axis])
		}
	}
}

// An empty corpus must report every classification as missing. This is the
// state the corpus starts in, and a gate that called it complete would never
// ask for a single fixture.
func TestCompareOnEmptyCorpusReportsEverythingMissing(t *testing.T) {
	t.Parallel()

	cov := gate.Compare(nil)

	if cov.Complete() {
		t.Fatal("an empty corpus cannot be complete")
	}
	reg := gate.Registered()
	for axis, names := range reg {
		if len(cov.Missing[axis]) != len(names) {
			t.Errorf("axis %q: expected all %d missing, got %d",
				axis, len(names), len(cov.Missing[axis]))
		}
	}
}

func TestCompareOnFullCorpusIsComplete(t *testing.T) {
	t.Parallel()

	cov := gate.Compare(gate.Registered())

	if !cov.Complete() {
		t.Fatalf("stamping every registered classification must be complete:\n%s", cov)
	}
}

// A name the corpus stamps that eidos does not register is an upstream
// inconsistency. Reporting it here would blame this repository for something
// it cannot fix, so it is dropped rather than surfaced as a gap.
func TestCompareIgnoresUnregisteredStamps(t *testing.T) {
	t.Parallel()

	stamped := gate.Registered()
	stamped[gate.AxisMixin] = append(stamped[gate.AxisMixin], "not-a-real-mixin")

	if cov := gate.Compare(stamped); !cov.Complete() {
		t.Fatalf("an unregistered stamp must not create a gap:\n%s", cov)
	}
}

// A gap has two causes — no fixture, or a fixture whose directive the
// annotator declined to read — and only the stamped set distinguishes them.
// Without it every failure reads as "write more fixtures".
func TestStringReportsBothSides(t *testing.T) {
	t.Parallel()

	reg := gate.Registered()
	partial := map[string][]string{gate.AxisDetector: {reg[gate.AxisDetector][0]}}

	got := gate.Compare(partial).String()

	if !strings.Contains(got, "missing:") {
		t.Errorf("the report must name the gap:\n%s", got)
	}
	if !strings.Contains(got, "stamped:") {
		t.Errorf("the report must name what was stamped, or a gap is unattributable:\n%s", got)
	}
	if !strings.Contains(got, reg[gate.AxisDetector][0]) {
		t.Errorf("the stamped classification must appear:\n%s", got)
	}
}
