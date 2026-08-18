// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
)

// TestBundleCollects pins the accumulation rules: a value with a nil error
// lands in its slice, a value with an error becomes an error and nothing
// else, so a misconfigured harness cannot half-join the run.
func TestBundleCollects(t *testing.T) {
	t.Parallel()

	var b suite.Bundle[fake]
	b.AddSubject(suite.Subject[fake]{Name: "ok", New: func(testing.TB) fake { return fake{} }}, nil)
	b.AddSubject(suite.Subject[fake]{}, errors.New("bad harness"))
	b.AddCheck(check("A/one"), nil)
	b.AddCheck(suite.Check[fake]{}, errors.New("bad row"))
	b.AddDrops("A/one")

	if len(b.Subjects) != 1 || b.Subjects[0].Name != "ok" {
		t.Errorf("only the well-formed subject joins, got %d", len(b.Subjects))
	}
	if len(b.Extra) != 1 || b.Extra[0].ID != "A/one" {
		t.Errorf("only the bound row joins, got %d", len(b.Extra))
	}
	if len(b.Drops) != 1 {
		t.Errorf("drops accumulate, got %d", len(b.Drops))
	}

	f := testkit.NewFailableTB()
	b.Fail(f, "RunFake")
	if !f.Failed() {
		t.Fatal("a bundle holding errors must stop the run")
	}
	// Every error is reported through Errorf under the entry point's name;
	// the captive TB keeps only the last message, which is enough to pin
	// the prefix contract.
	if msg := f.Msg(); !strings.Contains(msg, "RunFake: bad row") {
		t.Errorf("errors must carry the entry point's name, got %q", msg)
	}
}

func TestBundleFailPassesWhenClean(t *testing.T) {
	t.Parallel()

	var b suite.Bundle[fake]
	b.AddSubject(suite.Subject[fake]{Name: "ok", New: func(testing.TB) fake { return fake{} }}, nil)

	f := testkit.NewFailableTB()
	b.Fail(f, "RunFake")
	if f.Failed() {
		t.Errorf("a clean bundle must not fail, got %q", f.Msg())
	}
}

func TestConfigOnceRefusesTheSecond(t *testing.T) {
	t.Parallel()

	var b suite.Bundle[int]
	if !b.ConfigOnce("StoreConfig") {
		t.Fatal("the first config must be accepted")
	}
	if b.ConfigOnce("StoreConfig") {
		t.Fatal("the second config must be refused, not last-wins")
	}
	f := testkit.NewFailableTB()
	b.Fail(f, "RunStore")
	if !f.Failed() || !strings.Contains(f.Msg(), "two StoreConfigs passed") {
		t.Errorf("the refusal must name the config and the hazard, got %q", f.Msg())
	}
}

func TestAddErrAccumulates(t *testing.T) {
	t.Parallel()

	var b suite.Bundle[int]
	b.AddErr(errors.New("first wiring mistake"))
	b.AddErr(errors.New("second wiring mistake"))
	f := testkit.NewFailableTB()
	b.Fail(f, "RunX")
	msg := strings.Join(f.Logs(), "\n")
	if !strings.Contains(msg, "first wiring mistake") || !strings.Contains(msg, "second wiring mistake") {
		t.Errorf("every wiring mistake must be reported in one pass, got %q", msg)
	}
}
