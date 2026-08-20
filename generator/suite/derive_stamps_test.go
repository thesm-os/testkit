// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal on purpose, the suite_internal_test.go precedent: the
// sentinel rule reads mixin params through the unexported mixinParams
// projection, which only [mixinParamsOf] over a stamped bag populates
// — the exported surface reaches it solely through the sdk pipeline.
package suite

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/aggregator"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/reader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/tiers"
)

// stampMethod builds one key-drawing method with a detected shape and
// attached mixins, its stamps set through the real shape keys.
func stampMethod(name, detected string, mixins ...string) Method {
	src := &node.Method{Name: name}
	if detected != "" {
		shape.MetaShape.Set(src.EnsureMeta(), detected, "test")
	}
	return Method{
		Sig: &golang.Sig{
			Name:   name,
			Params: []golang.Param{{Name: "key", Source: storefixture.Named("Key")}},
			Source: src,
		},
		Mixins:    mixins,
		ArgFields: []string{"Key"},
	}
}

// bareMethod is stampMethod without a draw, the teardown and
// aggregator shapes.
func bareMethod(name, detected string, mixins ...string) Method {
	m := stampMethod(name, detected, mixins...)
	m.Params = nil
	m.ArgFields = nil
	return m
}

// sentinelReader is the kv Get shape: a reader declaring its OWN miss
// sentinel, read through the real param key.
func sentinelReader() Method {
	m := stampMethod("Get", reader.Name, MixinNotFound)
	// Stamped on the DECLARATION, which is where the annotator writes
	// it, and the projected map derived from that — the order the
	// pipeline runs in. Setting a detached bag and assigning the map
	// from it left the declaration bare, so a reader consulting the
	// source saw an unstamped method and the fixture pinned a claim no
	// real run produces.
	shape.MixinParamKey(MixinNotFound, MixinNotFoundSentinel).
		Set(m.Source.EnsureMeta(), "kv.ErrNotFound", "test")
	m.mixinParams = mixinParamsOf(m.Source.Meta(), m.Mixins)
	return m
}

// stampIface pairs the methods with a fixture that can deliver the
// key draw.
func stampIface(methods ...Method) Iface {
	return Iface{
		Name: "Store", Token: "store", Qualifier: "store", Methods: methods,
		Fixture: Fixture{Fields: []FixtureField{{
			Name:   "Key",
			Sample: golang.Sample{Text: `"k"`},
			Other:  golang.Sample{Text: `"o"`},
		}}},
	}
}

// seededIface is [stampIface] for a run that zips a corpus from its
// pools, which is what puts something there for a hit to find and
// leaves one key out for a miss to draw.
func seededIface(methods ...Method) Iface {
	f := stampIface(methods...)
	f.Corpus = true
	return f
}

// stampCase is one stamp shape and the ID set its rule licenses,
// beside the number of gaps the rule names rather than emitting.
type stampCase struct {
	name     string
	iface    Iface
	want     []vocab.ID
	refusals int
}

func (c stampCase) Name() string { return c.name }

func TestStampsDeriveTheStampFamilies(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []stampCase{
		{
			"idempotent probes the repeat",
			stampIface(bareMethod("Close", "", MixinIdempotent)),
			[]vocab.ID{"Close/idempotent"},
			0,
		},
		{
			"a stamped sentinel reader derives its miss",
			stampIface(sentinelReader(), stampMethod("Put", writer.Name)),
			[]vocab.ID{"Get/miss"},
			0,
		},
		{
			"a reader beside a writer derives its miss",
			stampIface(stampMethod("Lookup", reader.Name), stampMethod("Put", writer.Name)),
			[]vocab.ID{"Lookup/miss"},
			0,
		},
		{
			"a seeded reader derives miss and hit",
			seededIface(stampMethod("Lookup", reader.Name)),
			[]vocab.ID{"Lookup/miss", "Lookup/hit"},
			0,
		},
		{
			"a seeded aggregator derives its count",
			seededIface(bareMethod("Size", aggregator.Name)),
			[]vocab.ID{"Size/count"},
			0,
		},
		{
			"an aggregator beside a writer licenses nothing",
			stampIface(bareMethod("Len", aggregator.Name), stampMethod("Put", writer.Name)),
			nil,
			0,
		},
		{
			// A transform: reader-shaped, but nothing writes and no
			// corpus seeds, so no draw is one nothing supplied.
			"a reader shape with nothing to supply it refuses",
			stampIface(stampMethod("Encode", reader.Name)),
			nil,
			1,
		},
	}, func(t *testing.T, tc stampCase) {
		plans, refusals := Stamps{}.Derive(tc.iface)
		testkit.Len(t, refusals, tc.refusals, "the rule names exactly these gaps")
		got := make([]vocab.ID, 0, len(plans))
		for _, p := range plans {
			id, err := p.ID.Render()
			testkit.NoError(t, err, "the derived ID is well formed")
			got = append(got, id)
		}
		if len(tc.want) == 0 {
			testkit.Len(t, got, 0, "the rule licenses nothing here")
			return
		}
		testkit.Equal(t, got, tc.want, "the rule licenses exactly these checks")
	})
}

func TestStampsHoldTheCensusPosture(t *testing.T) {
	t.Parallel()

	t.Run("a law-backed stamp is the model tier's, silently", func(t *testing.T) {
		t.Parallel()
		plans, refusals := Stamps{}.Derive(stampIface(stampMethod("Put", "", MixinTTL)))
		testkit.Len(t, plans, 0, "the laws deriver owns it")
		testkit.Len(t, refusals, 0, "tiers recognizes it, so it is not a gap")
	})

	t.Run("an unknown stamp refuses with the census framing", func(t *testing.T) {
		t.Parallel()
		plans, refusals := Stamps{}.Derive(stampIface(stampMethod("Put", "", "brand-new-shape")))
		testkit.Len(t, plans, 0, "nothing derives from an unknown stamp")
		testkit.Len(t, refusals, 1, "the gap is named, never silent")
		testkit.Contains(t, refusals[0].What, "brand-new-shape", "the refusal names the stamp")
		testkit.Contains(t, refusals[0].Remedy, "census", "the remedy points at the coverage mechanism")
	})

	t.Run("an undeliverable draw refuses the stamp checks", func(t *testing.T) {
		t.Parallel()
		iface := Iface{
			Name:      "Store",
			Token:     "store",
			Qualifier: "store",
			Methods:   []Method{stampMethod("Get", reader.Name)},
		}
		plans, refusals := Stamps{}.Derive(iface)
		testkit.Len(t, plans, 0, "no check derives over a draw nothing supplies")
		testkit.Len(t, refusals, 1, "the whole stamp set folds into one refusal")
		testkit.Equal(t, refusals[0].What, "Get's stamp checks", "the refusal names the method's stamp set")
	})

	t.Run("the corpus claims come out verbatim", func(t *testing.T) {
		t.Parallel()
		plans, _ := Stamps{}.Derive(stampIface(sentinelReader(), stampMethod("Put", writer.Name)))
		testkit.Len(t, plans, 1, "one miss check")
		testkit.Equal(t, plans[0].Claim, "Get reports ErrNotFound for a key nothing wrote",
			"the manifest spelling: bare sentinel, writer-fed verb")
	})
}

// The census gate, the tiers/actions pattern reused: every upstream
// classification is tabled here, law-backed in tiers, or recorded
// below with the reason it owes no rule row — an input another
// derivation consumes, a behaviour another tier owns, or a PENDING
// gap naming the work that closes it. Held equal to the live
// registries in both directions, so an eidos addition fails by name
// in the build that makes it stampable, and a recorded entry goes
// stale loudly the moment a row or law covers it. The names here are
// census data, not vocabulary: a misspelled key is an orphan the gate
// rejects.
//
// PENDING entries must be empty by the flip; everything else is a
// permanent placement with its reason.

var recordedMixins = map[string]string{
	"concurrent":        "the laws deriver lowers it to the linearizable leg; PENDING mixin-rows batch: the incumbent's concurrent smoke beside it",
	"hooks":             "PENDING mixin-rows batch: the incumbent's direct check, as a rule row",
	"nilsafe":           "PENDING mixin-rows batch: the incumbent's direct check, as a rule row",
	"orderafter":        "PENDING mixin-rows batch: the incumbent's direct check, as a rule row",
	"partition":         "PENDING mixin-rows batch: the incumbent's direct check, as a rule row",
	"sample":            "PENDING mixin-rows batch: the incumbent's direct check, as a rule row",
	"sideeffect":        "PENDING mixin-rows batch: the incumbent's direct check, as a rule row",
	"validates":         "PENDING mixin-rows batch: the incumbent's direct check, as a rule row",
	"wrappedvia":        "PENDING mixin-rows batch: the incumbent's direct check, as a rule row",
	"timeaware":         "PENDING caps deriver: lowers to the harness clock capability, not a probe",
	"concurrentreaders": "PENDING tiers row: a concurrency claim, the model tier's to bind",
	"indexed":           "PENDING tiers row: positions-into-collection is a law over the sizing method",
	"retrysucceeds":     "PENDING tiers row: convergence under retry is a property, not a probe",
	"deprecated":        "documentation stamp: colours generated prose, owes no check",
	"notfound": "an identity, not a claim: it names WHAT a miss reports, and the check " +
		"that a miss IS reported is the reader shape's own — MissSentinel reads it, " +
		"which is what turns that check's body from the zero arm into the sentinel arm",
	"integrationonly": "run gate: scopes checks behind the integration env, owes none of its own",
	"scope":           "needs a value no run can invent — the incumbent's exclusion, kept",
	"errors":          "declares error returns contractual — a derivation input, owing no probe of its own",
}

var recordedDetectors = map[string]string{
	"answeringwriter": "PENDING detector-rows batch: the incumbent's answer round-trip, as a rule row; a seed input meanwhile",
	"batchreader":     "PENDING detector-rows batch: the incumbent's batch-size check, as a rule row",
	"multireader":     "PENDING detector-rows batch: the miss family over N value slots",
	"streamconsumer":  "PENDING contracts deriver: a stream arm, not a per-method probe",
	"closer":          "teardown shape: the signature families cover it, the after-close laws bind the rest",
	"voidlifecycle":   "teardown shape: the signature families cover it, the after-close laws bind the rest",
	"mutator":         "a writer answering nothing: excluded from seeding on purpose, and the signature families cover it",
}

var recordedContracts = map[string]string{
	"if-absent":       "PENDING contracts deriver: the incumbent's direct check, as a contract rule",
	"if-match":        "PENDING contracts deriver: the incumbent's direct check, as a contract rule",
	"outbox":          "PENDING contracts deriver: the incumbent's direct check, as a contract rule",
	"circuit-breaker": "PENDING tiers row: a protocol needing induced failure, the model tier's to bind",
	"leader-election": "PENDING tiers row: a multi-node protocol, the model tier's to bind",
	"rate-limit":      "PENDING tiers row: a clocked budget protocol, the model tier's to bind",
}

// assertCensus holds one axis's registry to the three-way partition.
func assertCensus(t *testing.T, registry []string, tabled map[string]stampRule, recorded map[string]string) {
	t.Helper()

	var uncovered []string
	for _, name := range registry {
		if _, ok := tabled[name]; ok {
			continue
		}
		if len(tiers.LawsFor(name)) > 0 {
			continue
		}
		if _, ok := recorded[name]; ok {
			continue
		}
		uncovered = append(uncovered, name)
	}
	slices.Sort(uncovered)
	testkit.Len(t, uncovered, 0, "every classification is tabled, law-backed, or recorded with a reason — uncovered: "+
		strings.Join(uncovered, ", "))

	var stale, orphaned []string
	for name := range recorded {
		if _, ok := tabled[name]; ok || len(tiers.LawsFor(name)) > 0 {
			stale = append(stale, name)
		}
		if !slices.Contains(registry, name) {
			orphaned = append(orphaned, name)
		}
	}
	slices.Sort(stale)
	slices.Sort(orphaned)
	testkit.Len(t, stale, 0, "a recorded entry a row or law now covers must be deleted: "+
		strings.Join(stale, ", "))
	testkit.Len(t, orphaned, 0, "a recorded entry the registry no longer carries is a typo or a removal: "+
		strings.Join(orphaned, ", "))
}

func TestStampCensusCoversTheMixinRegistry(t *testing.T) {
	t.Parallel()
	names := make([]string, 0, len(mixins.All()))
	for _, m := range mixins.All() {
		names = append(names, m.Name)
	}
	assertCensus(t, names, mixinRules(), recordedMixins)
}

func TestStampCensusCoversTheDetectorRegistry(t *testing.T) {
	t.Parallel()
	names := make([]string, 0, len(detectors.All()))
	for _, d := range detectors.All() {
		names = append(names, d.Name)
	}
	assertCensus(t, names, detectorRules(), recordedDetectors)
}

func TestStampCensusCoversTheContractRegistry(t *testing.T) {
	t.Parallel()
	names := make([]string, 0, len(contracts.All()))
	for _, c := range contracts.All() {
		names = append(names, c.Name)
	}
	// No contract rules table exists yet — the contracts deriver is
	// the next batch; until it lands every contract is law-backed or
	// recorded pending.
	assertCensus(t, names, nil, recordedContracts)
}
