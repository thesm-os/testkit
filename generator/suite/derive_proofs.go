// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// The planted-defect rules for the model-family rows — the corpus's
// own audit ("the defect-emitter experiment"): most law proofs reduce
// to mechanical rules expressible from shape and directives alone,
// and the residue honestly ships Argued rather than wearing a proof
// nobody derived. The row constructors consult these tables; a rule
// answering flips the row to Proven with its defect, silence leaves
// it Argued with the pending-proofs reason.
//
// The ttl proof (F1 strip-role-field: zero the field the mixin names)
// waits on its defect variant joining the closed set — a contract
// amendment, recorded in the design doc.

// lawDefectRule plants one law's defect from the interface's own
// stamps, false where the law's carriers do not supply the target.
type lawDefectRule func(f Iface) (projection.Defect, bool)

// lawDefects is the rule table, keyed by law identifier. A law
// without a row is the honest residue — the domain composites (the
// cursor's hand types, the lease's lying accounting) that no
// mechanical rule reaches.
func lawDefects() map[string]lawDefectRule {
	return map[string]lawDefectRule{
		lawid.LifecycleAfterClose:      afterCloseOutlives,
		lawid.PoisonConsistent:         poisonHeals,
		lawid.AppenderMonotonicOffsets: appenderFreezes,
	}
}

// lawDefect resolves one law row's planted defect, false when the
// law has no rule or the rule's target is not on this interface.
func lawDefect(f Iface, law string) (projection.Defect, bool) {
	rule, ruled := lawDefects()[law]
	if !ruled {
		return nil, false
	}
	return rule(f)
}

// observationDefect is W1, the discard-write rule: acknowledge and
// drop on the writer. It proves every agreement-shaped row — the
// differential, the linearizable leg, the observational bundle —
// because a dropped write diverges from any honest reference. False
// where nothing writes.
func observationDefect(f Iface) (projection.Defect, bool) {
	writer := firstWriter(f.Methods)
	if writer == nil {
		return nil, false
	}
	return projection.DiscardWrite{Option: projection.OptionName(f.Name, writer.Name)}, true
}

// afterCloseOutlives plants the after-close defect: one stamped
// method keeps working past Close — "a store whose Put outlives
// Close".
func afterCloseOutlives(f Iface) (projection.Defect, bool) {
	for _, m := range f.Methods {
		if m.HasMixin(MixinAfterClose) {
			return projection.PartialOutlive{Option: projection.OptionName(f.Name, m.Name)}, true
		}
	}
	return nil, false
}

// poisonHeals plants P1, sentinel-once: the subject reports the
// stamped sentinel once and then heals — precisely the un-sticky
// poison the law forbids. The sentinel is the same declaration that
// licensed the law, so the defect cannot name a different one.
func poisonHeals(f Iface) (projection.Defect, bool) {
	for _, m := range f.Methods {
		if v, ok := m.MixinParam(MixinAfterClose, MixinAfterCloseSentinel); ok && v != "" {
			return projection.SentinelOnce{Sentinel: projection.Expr(v)}, true
		}
	}
	return nil, false
}

// appenderFreezes plants R1, freeze-return: the monotonic return
// position pinned to a constant.
func appenderFreezes(f Iface) (projection.Defect, bool) {
	writer := firstWriter(f.Methods)
	if writer == nil {
		return nil, false
	}
	return projection.FreezeReturn{Option: projection.OptionName(f.Name, writer.Name)}, true
}
