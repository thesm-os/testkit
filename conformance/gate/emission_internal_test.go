// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"testing"

	"go.thesmos.sh/eidos/plugin"

	"go.thesmos.sh/testkit"
)

// A pipeline that will not build is reported, never measured.
//
// The other half of [TestEmissionSurfacesARunFailure], and the more dangerous
// half: a run that fails is loud, while a *build* that fails leaves a census
// with no store to read. Every census in this package then reports its subject
// as covered by nothing — which reads exactly like a corpus that regressed, and
// would send a reader to fix fixtures that are fine.
//
// Driven by handing the builder one generator twice, which is the malformed
// set eidos rejects by name. Reachable only from here: the exported entry
// points supply [corpusGenerators], and that is the point of them.
func TestRunCorpusReportsAMalformedPluginSet(t *testing.T) {
	t.Parallel()

	gens := corpusGenerators()
	testkit.True(t, len(gens) > 0, "the corpus registers generators to duplicate")

	doubled := append([]plugin.Generator{gens[0]}, gens...)
	_, err := runCorpus(t.Context(), "..", []string{"./corpus/..."}, doubled)
	testkit.True(t, err != nil, "a plugin registered twice fails the build rather than the census")
}
