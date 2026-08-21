// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// handleFieldOf fills a handle the generated file constructs and shares.
func handleFieldOf(
	b *Bindings, harness *subject.Projection, r tiers.Rule, f tiers.Field,
	field *LawField, m, keyed *subject.Method,
) (*LawField, string) {
	switch f.From {
	case handleKeyProjection:
		if b.Reference.KeyField != "" {
			field.KeyOfName = b.KeyOfName()
			field.KindName = sdk.Kind(LawFieldKindPrefix + "KeyOf")
			return field, ""
		}
		if r.Law == lawid.PaginatorNoDuplicates {
			// Identity over the page element: no projection derives where the
			// only reader is the walk itself, and the element is comparable —
			// the binding row instantiates K at the element for the same
			// reason.
			role, reason := ruleFieldRole(b, harness, r, fPage, m, keyed)
			if reason != "" {
				return nil, f.Name + " " + reason
			}
			elem, why := drainedElem(b, role)
			if why != "" {
				return nil, f.Name + " " + why
			}
			field.Value = elem
			field.KindName = sdk.Kind(LawFieldKindPrefix + "KeyOfIdentity")
			return field, ""
		}
		return nil, f.Name + " needs the key projection, which was not derivable here"

	case "identity-hash":
		// Identity over the drained element: the hash argument is the value
		// itself, so the closure needs only the element's type.
		if elem, why := hashElem(b, harness, r, m, keyed); why == "" {
			field.Value = elem
		}
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Hash")
		return field, ""

	case "subject-factory":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Factory")
		return field, ""

	case handleClassifier:
		spec, why := sessionSpecOf(b, harness, r, m, keyed)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Classify")
		field.KeyOfName = spec.ClassifyName
		return field, ""

	case "natural-order":
		role, reason := roleMethod(b, harness, fromSelf, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		if !orderedScalar(role) {
			return nil, f.Name + " orders " + role.Name + "'s result, which the language does not"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Out = out
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Less")
		return field, ""

	case "observation":
		obs, reason := observationOf(b, harness, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Method = obs.Method.Name
		field.TakesCtx = obs.TakesCtx
		field.Out = obs.Out
		if obs.Keyed {
			field.KeyField = b.Keys.Field
			field.KindName = sdk.Kind(LawFieldKindPrefix + "ObserveKeyed")
			b.LawsUseFixture = true
		} else {
			field.KindName = sdk.Kind(LawFieldKindPrefix + "ObserveCall")
		}
		return field, ""

	case "partitions":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Partitions")
		return field, ""

	case "clock":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Advance")
		return field, ""

	case handleCoalesce:
		// The law's own instrumentation: the compute it hands every caller,
		// counting how often the subject actually ran it.
		call, reason := ruleFieldRole(b, harness, r, fCall, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		out, _, why := resultType(call)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Out = out
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Compute")
		return field, ""

	case "coalesce-counter":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Counter")
		return field, ""

	case handleVersionStamp:
		// The version-coherent draw: read the cell, copy its version member
		// into the drawn attempt. Both types carry the member — the stamp
		// key names one field of one payload — and a fixture where they
		// drift fails to compile in the package that armed it.
		cell, reason := ruleFieldRole(b, harness, r, fRead, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		attempt, reason := ruleFieldRole(b, harness, r, fCAS, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		if len(attempt.CallArgs()) == 0 {
			return nil, f.Name + " stamps " + attempt.Name + "'s attempt, and it takes none"
		}
		member, stamped := stampValue(harness, m, paramCASVersion)
		if !stamped {
			return nil, f.Name + " reads the version member, and the cas directive names none"
		}
		field.Method = cell.Name
		field.TakesCtx = cell.TakesContext()
		field.In = attempt.CallArgs()[0].Type
		field.KeyField = golang.LocalName(member)
		field.KindName = sdk.Kind(LawFieldKindPrefix + "VersionStamp")
		return field, ""

	case handleHistoryLog:
		// The append-recording history: a property-level log of every
		// successful append the sequences drove, cleared by the runner each
		// iteration. The field rides the append role so the inert check
		// catches a derived reference answering it inertly.
		appendRole, reason := roleMethod(b, harness, "chain.append", m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		replayRole, reason := ruleFieldRole(b, harness, r, fReplay, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		elem, why := drainedElem(b, replayRole)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Method = appendRole.Name
		field.Value = elem
		field.KindName = sdk.Kind(LawFieldKindPrefix + "HistoryRef")
		return field, ""
	}
	return nil, f.Name + " needs the " + f.From + " handle, which this build does not construct"
}

// hashElem resolves the identity hash's element: the drained element of the
// same rule's Drain field where one exists, the values pool otherwise.
func hashElem(
	b *Bindings, harness *subject.Projection, r tiers.Rule, m, keyed *subject.Method,
) (sdk.Ref, string) {
	for _, f := range r.Fields {
		if f.Kind != tiers.KindRole || (f.Name != fDrain && f.Name != "Collect") {
			continue
		}
		role, reason := roleMethod(b, harness, f.From, m, keyed)
		if reason != "" {
			return nil, reason
		}
		return drainedElem(b, role)
	}
	if b.Values.Type != nil {
		return b.Values.Type, ""
	}
	return nil, "hashes a value type no method here draws"
}
