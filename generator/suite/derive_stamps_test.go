// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal on purpose, the suite_internal_test.go precedent: the
// sentinel rule reads mixin params through the unexported mixinParams
// projection, which only [mixinParamsOf] over a stamped bag populates
// — the exported surface reaches it solely through the sdk pipeline.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/aggregator"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/reader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
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

// sentinelReader is the kv Get shape: a reader whose ttl declaration
// names the miss sentinel, read through the real param key.
func sentinelReader() Method {
	bag := sdk.NewBag()
	shape.MixinParamKey(MixinTTL, MixinTTLNotFound).Set(bag, "kv.ErrNotFound", "test")
	m := stampMethod("Get", reader.Name, MixinTTL)
	m.mixinParams = mixinParamsOf(bag)
	return m
}

// stampIface pairs the methods with a fixture that can deliver the
// key draw.
func stampIface(methods ...Method) Iface {
	return Iface{
		Name: "Store", Token: "store", Methods: methods,
		Fixture: Fixture{Fields: []FixtureField{{
			Name:   "Key",
			Sample: golang.Sample{Text: `"k"`},
			Other:  golang.Sample{Text: `"o"`},
		}}},
	}
}

// stampCase is one stamp shape and the ID set its rule licenses.
type stampCase struct {
	name  string
	iface Iface
	want  []vocab.ID
}

func (c stampCase) Name() string { return c.name }

func TestStampsDeriveTheStampFamilies(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []stampCase{
		{
			"idempotent probes the repeat",
			stampIface(bareMethod("Close", "", MixinIdempotent)),
			[]vocab.ID{"Close/idempotent"},
		},
		{
			"a stamped sentinel reader derives its miss",
			stampIface(sentinelReader(), stampMethod("Put", writer.Name)),
			[]vocab.ID{"Get/miss"},
		},
		{
			"a seeded reader derives miss and hit",
			stampIface(stampMethod("Lookup", reader.Name)),
			[]vocab.ID{"Lookup/miss", "Lookup/hit"},
		},
		{
			"a seeded aggregator derives its count",
			stampIface(bareMethod("Size", aggregator.Name)),
			[]vocab.ID{"Size/count"},
		},
		{
			"an aggregator beside a writer licenses nothing",
			stampIface(bareMethod("Len", aggregator.Name), stampMethod("Put", writer.Name)),
			nil,
		},
	}, func(t *testing.T, tc stampCase) {
		plans, refusals := Stamps{}.Derive(tc.iface)
		testkit.Len(t, refusals, 0, "tabled stamps refuse nothing")
		got := make([]vocab.ID, 0, len(plans))
		for _, p := range plans {
			got = append(got, p.ID.Render())
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
		iface := Iface{Name: "Store", Token: "store", Methods: []Method{stampMethod("Get", reader.Name)}}
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
