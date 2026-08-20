// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"cmp"
	"context"
	"maps"
	"regexp"
	"slices"
	"strings"

	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/sdk"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/model"
	"go.thesmos.sh/testkit/generator/suite"
	"go.thesmos.sh/testkit/generator/tiers"
)

// Evidence is one classification and what, if anything, asserts it.
//
// The union census ADR-0018 asks for. [Annotate] measures that a
// classification is *stamped* somewhere in the corpus, which every one of
// these passes — a fixture exists, its directive parses, the annotator
// produces the mark. What that cannot see is whether the stamp bought
// anything: a classification can be registered upstream, fixtured here, and
// asserted by neither tier, and every existing gate stays green because each
// is asking a question this one falls between.
//
// So the two tiers are asked directly, unioned across the whole corpus rather
// than per fixture. Per fixture is the generated header's question and it has
// a different answer — a classification checked on one interface and silent on
// another is a gap in that file. Here the question is whether the *vocabulary*
// has evidence at all, so one fixture asserting it is enough.
type Evidence struct {
	// Axis is `detector`, `mixin` or `contract`, and Name the classification.
	Axis, Name string

	// Refused reports that a derivation rule reached this classification
	// somewhere in the corpus and could not complete — the state between
	// asserted and unlooked-at.
	//
	// A refusal is an argument the generator computed rather than one a
	// reader wrote into [UnevidencedClassifications], and it is a better
	// argument: it names the interface, the reason and the directive that
	// would close it, and it goes stale the moment the rule starts
	// deriving. Counting it as evidence would be wrong — nothing is
	// asserted — but counting it as an unexamined gap is wrong too, and
	// that is what the census did before it could see refusals.
	Refused bool

	// Checked reports that some fixture's suite tier asserts it, and Modeled
	// that some fixture's model tier binds a law for it against a reference
	// the run can actually build.
	//
	// Both, rather than one `covered` bool, because the two are different
	// kinds of evidence and a reader triaging a red row needs to know which
	// one is missing. A classification with laws and no check may be complete;
	// one with neither is the finding.
	Checked, Modeled bool

	// Where names a fixture that supplied the evidence, empty where none did.
	// The corpus address a green row is answerable from.
	Where string
}

// Evidenced reports whether anything at all asserts this classification.
func (e Evidence) Evidenced() bool { return e.Checked || e.Modeled }

// Evidenced runs the real generators over the corpus in memory and reports, per
// registered classification, which tier asserts it.
//
// The same pipeline [Emission] runs and for the same reason: a subset would
// measure a production never runs. This walk reads the suite generator's own
// per-interface coverage rather than re-deriving it, because two derivations of
// "is this checked" are two chances to disagree — and the one that matters is
// the generator's, since that is what the consumer's header prints.
//
// Registered-but-unstamped classifications appear with neither flag set, which
// is correct: [Compare] is what reports them as a corpus gap, and a
// classification nothing stamps is also a classification nothing asserts.
//
// The run is kept while [evidenceFrom] has nothing to read off it, because a
// census pointed at a pattern matching nothing must fail rather than measure an
// empty corpus as fully evidenced — see that function's TRANSITION paragraph.
func Evidenced(ctx context.Context, root string, patterns ...string) ([]Evidence, error) {
	pipe, err := runCorpus(ctx, root, patterns, corpusGenerators())
	if err != nil {
		return nil, err
	}
	return evidenceFrom(pipe), nil
}

// evidenceFrom reads the per-classification verdicts the corpus run produced.
//
// Split from [Evidenced] for [Measure]'s sake: both answer the same question
// and only one of them should decide how many times the corpus is loaded.
//
// Both tiers are read off the same store, and each names a fixture rather than
// a bool, so a green row is answerable — [TestEvidencedRowsNameTheirFixture] is
// what holds it to that.
func evidenceFrom(pipe *pipeline.Pipeline) []Evidence {
	checked, modeled := suiteEvidence(pipe), modelEvidence(pipe)
	refused := suiteRefused(pipe)

	var out []Evidence
	for axis, names := range Registered() {
		for _, name := range names {
			e := Evidence{
				Axis:    axis,
				Name:    name,
				Checked: checked[name] != "",
				Modeled: modeled[name] != "",
				Refused: refused[name] != "",
			}
			switch {
			case checked[name] != "":
				e.Where = checked[name]
			case modeled[name] != "":
				e.Where = modeled[name]
			default:
				e.Where = refused[name]
			}
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b Evidence) int {
		if a.Axis != b.Axis {
			return cmp.Compare(a.Axis, b.Axis)
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return out
}

// suiteEvidence reads which classification each derived check was licensed by,
// mapping it to a fixture that carries one.
//
// The generator's own answer rather than a re-derivation: [projection.Licence]
// is stamped where the rules table dispatched, so this reads the classification
// the deriver actually read. Deriving it here from the check's class would be a
// second opinion, and the wrong one — the class families and the registry axes
// do not line up.
func suiteEvidence(pipe *pipeline.Pipeline) map[string]string {
	out := map[string]string{}
	for origin, c := range sdk.PendingByOrigin[*suite.Contract](pipe.Store().Emit()) {
		where := c.Inventory.Iface
		if iface, ok := origin.(*sdk.Interface); ok {
			where = iface.Package + "." + iface.Name
		}
		for _, check := range c.Inventory.Checks {
			if !check.Licensed.Licensed() {
				// The shape earned it, not a classification: every
				// signature-family check is here, and none of them is
				// evidence that any vocabulary bought anything.
				continue
			}
			if _, seen := out[check.Licensed.Name]; !seen {
				out[check.Licensed.Name] = where
			}
		}
	}
	return out
}

// suiteRefused reads which classifications a derivation rule reached and
// declined FOR A STATED REASON, mapping each to a fixture that carries the
// refusal.
//
// The same store and the same attribution as [suiteEvidence], off the field
// the deriver stamps at its dispatch. A refusal with no classification is
// skipped rather than guessed at: those are the shape-reached ones — an
// undeliverable argument, a missing seed — and they belong to no vocabulary
// row.
//
// [suite.Refusal.Unaccounted] is skipped for the opposite reason: it is the
// deriver reporting that nothing decided what the classification owes, which
// is the gap itself. Reading it as an argument was this census's own
// silent-green bug — moving a stamp out of the accounting tables turned it
// from covered into refused, and both counted, so the gate could not fail.
func suiteRefused(pipe *pipeline.Pipeline) map[string]string {
	out := map[string]string{}
	for origin, c := range sdk.PendingByOrigin[*suite.Contract](pipe.Store().Emit()) {
		where := c.Inventory.Iface
		if iface, ok := origin.(*sdk.Interface); ok {
			where = iface.Package + "." + iface.Name
		}
		for _, r := range c.Refusals {
			// Elsewhere is skipped with Unaccounted, for the opposite
			// reason: it says another tier owns the claim, which is an
			// argument about ownership rather than about evidence. Whether
			// that tier asserts anything is what [Evidence.Modeled]
			// answers, and reading this as evidence would let a dark tier
			// vouch for itself.
			if r.Licensed.Name == "" || r.Unaccounted || r.Elsewhere {
				continue
			}
			if _, seen := out[r.Licensed.Name]; !seen {
				out[r.Licensed.Name] = where
			}
		}
	}
	return out
}

// modelEvidence reads which classifications the bound laws were earned by.
//
// A law is bound per fixture and reaches its classifications through the rules
// catalogue, which is where "this stamp earns this law" is written down. A rule
// needing two classifications evidences both, for the reason [tiers.LawsFor]
// reports under both: from either one's point of view the law was reachable,
// and the binding is what makes it more than reachable.
func modelEvidence(pipe *pipeline.Pipeline) map[string]string {
	earnedBy := map[string][]string{}
	for _, r := range tiers.Rules() {
		earnedBy[r.Law] = append(earnedBy[r.Law], r.Needs...)
	}

	out := map[string]string{}
	for origin, b := range sdk.PendingByOrigin[*model.Bindings](pipe.Store().Emit()) {
		where := b.IfaceName
		if iface, ok := origin.(*sdk.Interface); ok {
			where = iface.Package + "." + iface.Name
		}
		for _, l := range b.Laws {
			for _, name := range earnedBy[l.ID] {
				if _, seen := out[name]; !seen {
					out[name] = where
				}
			}
		}
	}
	return out
}

// Unevidenced returns the classifications no tier asserts and no row argues,
// each as `<axis>/<name>`.
//
// The register's other direction is [UnevidencedClassifications] itself: a row
// for a classification some tier now asserts is a stale excuse, and
// [TestEveryUnevidencedRowIsStillTrue] deletes it by failing.
func Unevidenced(all []Evidence) []string {
	var out []string
	for _, e := range all {
		if e.Evidenced() {
			continue
		}
		if _, argued := UnevidencedClassifications[e.Name]; argued {
			continue
		}
		out = append(out, e.Axis+"/"+e.Name)
	}
	slices.Sort(out)
	return out
}

// modelOwned is every classification some law in the catalogue reaches.
//
// Read off [tiers.Rules] rather than listed here, for the reason [Registered]
// reads the live registries: a rule added upstream moves a classification out
// of the suite tier's account on the next build, and a table copied into this
// file would keep gating it here long after the law arrived.
func modelOwned() map[string]bool {
	out := map[string]bool{}
	for _, r := range tiers.Rules() {
		for _, name := range r.Needs {
			out[name] = true
		}
	}
	return out
}

// UnevidencedBySuite returns the classifications the suite tier owns outright
// and does not assert — each as `<axis>/<name>`.
//
// The half of [Unevidenced] that is answerable while the model tier is
// unregistered, and the reason that skip does not have to take the whole
// question with it. A classification no rule in the catalogue reaches can
// never be the model tier's evidence, however the tier is wired: if the suite
// tier does not assert it and no row argues it, the gap is real today and will
// still be real on the morning the tier comes back.
//
// What this deliberately does not report is a classification some law reaches.
// That one's evidence is dark rather than absent, and reporting it here would
// be the relaxed-threshold mistake [skipUntilModelRelinked] argues against,
// pointed the other way: a gate that reddens on work that is deferred teaches
// the reader to switch it off.
func UnevidencedBySuite(all []Evidence) []string {
	owned := modelOwned()
	var out []string
	for _, e := range all {
		if e.Checked || e.Refused || owned[e.Name] {
			continue
		}
		if _, accounted := suite.Accounting(e.Name); accounted {
			continue
		}
		if _, argued := UnevidencedClassifications[e.Name]; argued {
			continue
		}
		out = append(out, e.Axis+"/"+e.Name)
	}
	slices.Sort(out)
	return out
}

// ArguedButEvidenced returns the register rows a tier has since covered.
func ArguedButEvidenced(all []Evidence) []string {
	byName := make(map[string]Evidence, len(all))
	for _, e := range all {
		byName[e.Name] = e
	}
	var out []string
	for name := range maps.Keys(UnevidencedClassifications) {
		if byName[name].Evidenced() {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

// Census is everything one corpus run measures.
type Census struct {
	// Emitted is what each armed interface bound, and Evidence what each
	// registered classification is asserted by.
	Emitted  []Emitted
	Evidence []Evidence

	// SuccessOnly is every distinct claim the corpus puts on a check that
	// judges nothing but whether the call succeeded, with the method name
	// normalized. Both directions of the wording register read it.
	SuccessOnly []string
}

// The two reasons that cover most of [SuccessOnlyClaims], named because
// they are the argument rather than a label: a sentence qualifies there
// when what it promises is exactly what the error channel can answer.
const (
	reasonSurvives = "says only that the call returned, which is what the body checks; " +
		"the suffix names where the input came from and promises nothing about it"
	reasonErrChannel = "a claim about the error channel, judged on the error channel"
)

// SuccessOnlyClaims is every sentence the suite tier is allowed to put on
// a check that judges nothing but whether the call succeeded, and why
// that sentence is honest about it.
//
// An allowlist rather than a list of forbidden words, and the direction
// is the whole point. A blocklist reddens on the phrasings somebody
// thought of; this reddens on every sentence nobody has vouched for,
// which includes the one a future rule invents. Adding a row is the
// decision — read the body, and if it reads state back the check is not
// success-only and this is the wrong list.
//
// The class it exists to stop has been through here twice. `idempotent`
// promised a second call "changes nothing" over a body that reads
// nothing back, and `validates` promised a method refuses EXACTLY what
// its validator refuses over a body that draws one value and compares
// two verdicts — which catches refusing too much and cannot catch
// refusing too little. Both passed their own planted defects, because a
// defect derived from the check agrees with the check.
//
// Keys have the method name replaced by <method>, so one row covers the
// sentence wherever it is emitted.
//
//nolint:gochecknoglobals // a wording register, read-only, test-facing.
var SuccessOnlyClaims = map[string]string{
	"<method> survives a call":                              reasonSurvives,
	"<method> survives a call with derived inputs":          reasonSurvives,
	"<method> survives a call with seeded inputs":           reasonSurvives,
	"<method> survives a call with a derived <input>":       reasonSurvives,
	"<method> survives a call with a seeded <input>":        reasonSurvives,
	"<method> survives a call with an input <builder> made": reasonSurvives,
	"<method> survives returning a borrowed resource":       "the resource came back; what it is worth is not claimed",

	"<method> reports a cancelled context as cancelled":                reasonErrChannel,
	"<method> reports an expired deadline as exceeded":                 reasonErrChannel,
	"<method> returns an error rather than panicking on a nil context": reasonErrChannel,
	"<method> reports a nil <input> rather than panicking":             reasonErrChannel,
	"<method> reports <sentinel> for a key nothing wrote":              reasonErrChannel,
	"<method> reports <sentinel> until <partner> has run":              reasonErrChannel,

	"a second <method> after a clean one is accepted":               "accepted, not unchanged — the body calls twice and reads nothing back",
	"a second <method> is accepted rather than refused as a repeat": "the same narrowing, worded for the accumulates mixin",
	"a second <method> of what is already there reports <sentinel>": reasonErrChannel,

	"<method> agrees with <partner> about the values this run draws": "agrees, not refuses-exactly: the body compares two error " +
		"channels on one drawn value, and on a value the partner accepts " +
		"it cannot see the subject accepting too much",
}

// wordingFrom returns every distinct claim the corpus puts on a
// success-only check, method name normalized.
//
// Read off the same run as the rest of the census, and off the plan
// rather than the emitted text: [projection.Body] carries the strength,
// so this compares what a check SAYS against what its body was written
// to do rather than against a reading of the generated file.
func wordingFrom(pipe *pipeline.Pipeline) []string {
	var out []string
	for _, c := range sdk.PendingByOrigin[*suite.Contract](pipe.Store().Emit()) {
		for _, check := range c.Inventory.Checks {
			if check.Body == nil || check.Body.Strength() != vocab.StrengthErrorOnly {
				continue
			}
			out = append(out, normalizeClaim(check.Claim, check.ID.Method))
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// UnvouchedClaims returns the success-only claims no row vouches for.
func UnvouchedClaims(successOnly []string) []string {
	var out []string
	for _, claim := range successOnly {
		if _, vouched := SuccessOnlyClaims[claim]; !vouched {
			out = append(out, claim)
		}
	}
	return out
}

// StaleClaimRows returns the register rows the corpus no longer emits.
func StaleClaimRows(successOnly []string) []string {
	emitted := make(map[string]bool, len(successOnly))
	for _, claim := range successOnly {
		emitted[claim] = true
	}
	var out []string
	for row := range maps.Keys(SuccessOnlyClaims) {
		if !emitted[row] {
			out = append(out, row)
		}
	}
	slices.Sort(out)
	return out
}

// The names a claim carries that vary per interface without changing what
// the sentence promises: the parameter it drew, the sentinel it names, the
// partner it waits on.
//
// Collapsed so the register holds sentence SHAPES. Left alone, "Get
// survives a call with a derived key" and "Put survives a call with a
// derived value" are two rows saying one thing, and a register with a row
// per parameter name in the corpus is a register nobody reads — which is
// how the row that matters gets waved through.
//
//nolint:gochecknoglobals // compiled once, read-only.
var claimNouns = []struct {
	pat  *regexp.Regexp
	with string
}{
	{regexp.MustCompile(`\bErr[A-Z]\w*`), "<sentinel>"},
	{regexp.MustCompile(`\ba (derived|seeded) \w+`), "a $1 <input>"},
	{regexp.MustCompile(`\ban (derived|seeded) \w+`), "an $1 <input>"},
	{regexp.MustCompile(`\ban input \w+ made`), "an input <builder> made"},
	{regexp.MustCompile(`\ba nil \w+ rather than panicking`), "a nil <input> rather than panicking"},
	{regexp.MustCompile(`\buntil \w+ has run`), "until <partner> has run"},
	{regexp.MustCompile(`\bagrees with \w+ about`), "agrees with <partner> about"},
}

// normalizeClaim reduces a claim to its shape: the method's own name, the
// parameters it drew and the sentinels it names all become placeholders,
// so one row covers the sentence wherever it is emitted.
func normalizeClaim(claim, method string) string {
	if method != "" {
		claim = strings.ReplaceAll(claim, method, "<method>")
	}
	for _, n := range claimNouns {
		claim = n.pat.ReplaceAllString(claim, n.with)
	}
	return claim
}

// Measure runs the corpus once and reads every census off the same store.
//
// One run rather than one per question, and the reason is not only speed.
// `go/types` resolves an Alias through an unsynchronized memoization, and the
// package loader is concurrent — so every additional full corpus load is
// another chance for the race detector to pair two accesses to it. The gate's
// TestMain pins GOMAXPROCS to 1 to narrow that window; narrowing a window is
// not the same as closing it, and the exposure scales with how many times the
// corpus is loaded.
//
// It had grown to six independent loads. This is the one seam where that count
// is decided, so it is decided here.
func Measure(ctx context.Context, root string, patterns ...string) (Census, error) {
	pipe, err := runCorpus(ctx, root, patterns, corpusGenerators())
	if err != nil {
		return Census{}, err
	}
	return Census{
		Emitted:     emittedFrom(pipe),
		Evidence:    evidenceFrom(pipe),
		SuccessOnly: wordingFrom(pipe),
	}, nil
}
