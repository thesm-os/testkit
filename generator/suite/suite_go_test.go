// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"reflect"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	enginesuite "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/engine/suite/prove"
	"go.thesmos.sh/testkit/generator/suite"
)

// Every import path the generated harness names is checked against the
// package it is supposed to name, read off a linked type rather than
// spelled a second time.
//
// A constant here that drifts from the tree produces a file which
// generates, formats and passes drift — and then fails to compile in a
// consumer's repository, naming a package that does not exist. Nothing
// in this repository would catch it: the corpus regenerates from these
// same constants, so it drifts with them.
func TestEmittedImportPathsNameRealPackages(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []pathCase{
		{
			"the suite vocabulary the generated rows construct",
			suite.Vocab, reflect.TypeOf(enginesuite.Strength("")).PkgPath(),
		},
		{
			"the falsifiability harness the companion drives",
			suite.Prove, reflect.TypeOf(prove.Defects[any]{}).PkgPath(),
		},
		{
			"the law identifiers the index accessors name",
			suite.LawIDs, reflect.TypeOf(lawid.Claim("")).PkgPath(),
		},
	}, func(t *testing.T, tc pathCase) {
		testkit.Equal(t, tc.declared, tc.actual,
			"generated code imports this path, and only a consumer's compiler would find it wrong")
	})
}

// The three module-relative paths compose from two roots, and a caller
// assembling a build against generated output has to require both.
func TestEmittedPathsComposeFromTwoModules(t *testing.T) {
	t.Parallel()

	// A go.mod naming only the root fails to resolve the engine module
	// with no clue that there were two.
	testkit.Equal(t, suite.EngineModule, suite.Module+"/engine",
		"the engine is its own module, not a package of the root")
	testkit.HasPrefix(t, suite.Vocab, suite.EngineModule,
		"the vocabulary is the engine's")
	testkit.HasPrefix(t, suite.Prove, suite.Vocab,
		"and prove sits under it")
	testkit.HasPrefix(t, suite.LawIDs, suite.Module,
		"while the law identifiers are the root's, which is the reason both are required")
}

// The two outputs are told apart by tag and by suffix, and the pipeline
// depends on which one is which.
func TestGoOutputs(t *testing.T) {
	t.Parallel()

	outs := suite.GoOutputs()
	testkit.Len(t, outs, 2, "the harness and its companion")

	t.Run("the primary is the empty tag at index zero", func(t *testing.T) {
		t.Parallel()
		// Which the pipeline validates at Build, so getting it wrong fails
		// the run rather than the output.
		testkit.Equal(t, outs[0].Tag, "", "the primary carries no tag")
		testkit.Equal(t, outs[0].Suffix, suite.GoPrimarySuffix, "and the harness suffix")
	})

	t.Run("the companion is addressable by tag", func(t *testing.T) {
		t.Parallel()
		// Routing overrides and CLI selection address an output by tag, so
		// the two files can be routed apart.
		testkit.Equal(t, outs[1].Tag, suite.GoTestOutputTag, "named, so it can be selected")
		testkit.Equal(t, outs[1].Suffix, suite.GoTestSuffix, "with the companion suffix")
	})

	t.Run("the companion suffix earns the external test package", func(t *testing.T) {
		t.Parallel()
		// The shift is what makes the companion reach the harness across a
		// package boundary the way a consumer does; a suffix not ending
		// _test.go would leave it inside, where unexported names are in
		// scope and the proofs stop proving what a consumer can reach.
		testkit.HasSuffix(t, suite.GoTestSuffix, "_test.go",
			"Layout grants the external-test shift on this suffix alone")
		testkit.False(t, strings.HasSuffix(suite.GoPrimarySuffix, "_test.go"),
			"and the harness must not take it, or consumers could not import it")
	})
}

// pathCase is one declared import path and the package it must name.
type pathCase struct {
	name     string
	declared string
	actual   string
}

func (c pathCase) Name() string { return c.name }
