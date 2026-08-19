// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// families are the reserved lowercase words that may start an ID instead
// of a method name. Adding one needs an RFC (ADR-0020): these strings
// land in consumers' lock files.
//
// A method name always starts with an uppercase letter, because Go only
// exports capitalised identifiers. That is what makes a collision between
// a method scope and a family scope impossible, rather than merely
// unlikely.
//
// Unexported behind [IsFamily] and [FamilyNames]: an exported map would
// let any package's init mint a reserved word, and a write racing
// ValidateID from a parallel run would be a data race inside the package
// built to catch other people's.
const (
	FamilyChain     = "chain"
	FamilyModel     = "model"
	FamilyPoison    = "poison"
	FamilyCrossRole = "cross-role"
	FamilySim       = "sim"
	FamilyHand      = "own" // the package's own hand-written checks, no method scope of their own
)

var families = map[string]bool{
	FamilyChain:     true,
	FamilyModel:     true,
	FamilyPoison:    true,
	FamilyCrossRole: true,
	FamilySim:       true,
	FamilyHand:      true,
}

// IsFamily reports whether a scope word is a reserved family.
func IsFamily(name string) bool { return families[name] }

// FamilyNames lists the reserved families, sorted.
func FamilyNames() []string { return slices.Sorted(maps.Keys(families)) }

// MethodID composes a method-scoped check ID: MethodID("Put", SegSmoke)
// is "Put/smoke". Generated code composes rather than spelling, so the
// method name has one home per interface — the same constant the engine
// action and the failure message read.
func MethodID(method, segment string) ID { return ID(method + "/" + segment) }

// FamilyID composes a family-scoped check ID.
//
// qualifier is the interface's lowercase name, required in a package
// holding more than one model-bearing interface and empty otherwise
// (RFC-0004 A18): Store and Journal both have a differential leg, and
// "model/differential" can name only one of them.
//
//	FamilyID(FamilyModel, "store", SegLaws)       -> "model/store/laws"
//	FamilyID(FamilyModel, "", lawid.TTLExpiry)    -> "model/AUTO-TTL-EXPIRY"
func FamilyID(family, qualifier, segment string) ID {
	if qualifier == "" {
		return ID(family + "/" + segment)
	}
	return ID(family + "/" + qualifier + "/" + segment)
}

// ValidateID reports why an ID is malformed, or nil when it is well
// formed. The grammar:
//
//	id            = scope-segment 1*( "/" sub-segment )
//	scope-segment = exported Go method name | reserved family word
//	sub-segment   = slug | law-id
//	slug          = 1*( lowercase / DIGIT / "-" )
//	law-id        = "AUTO-" 1*( UPPER / DIGIT / "-" )
//
// A bare scope segment is not an ID. A scope names a group of checks, so
// every check carries at least one segment after it.
//
// # Why a slug rather than the claim
//
// The first grammar let a sub-segment be any printable ASCII, and the
// generated IDs were the claim in prose — `Put/reports a cancelled context`.
// The typed constant in the index protects a drop *site* from regeneration;
// the string is what lands in checks.lock and in the report, and prose gets
// edited. One change in the improvement programme reworded five generated
// check messages at once, and the same generator writes both — so a claim
// rewording and an ID rewording are one edit, with one of them frozen.
//
// [Check.Claim] carries the sentence and is free to improve forever.
func ValidateID(id ID) error {
	if id == "" {
		return errors.New("empty check ID")
	}
	parts := strings.Split(string(id), "/")
	if len(parts) < 2 {
		return fmt.Errorf(
			"check ID %q has only a scope segment; a scope names a group, "+
				"so a check needs at least one more segment", id,
		)
	}

	if err := validateScope(id, parts[0]); err != nil {
		return err
	}
	for _, seg := range parts[1:] {
		if err := validateSegment(id, seg); err != nil {
			return err
		}
	}
	return nil
}

// validateScope holds the first segment to one of its two forms: a
// reserved family word, or an exported Go method name.
//
// The whole segment is checked, not just its first byte. IDs double as
// subtest names and -run patterns, and go test rewrites whitespace in
// subtest names — so a scope like "Put me down" that passed a first-byte
// check would record one ID and match another.
func validateScope(id ID, scope string) error {
	switch {
	case scope == "":
		return fmt.Errorf("check ID %q starts with an empty scope", id)
	case families[scope]:
		return nil
	case scope[0] >= 'A' && scope[0] <= 'Z':
		for _, r := range scope[1:] {
			if !isLower(r) && !isUpper(r) && !isDigit(r) && r != '_' {
				return fmt.Errorf(
					"check ID %q: scope %q is not a Go method name — letters, digits "+
						"and '_' only, starting uppercase", id, scope,
				)
			}
		}
		return nil
	case scope[0] >= 'a' && scope[0] <= 'z':
		return fmt.Errorf(
			"check ID %q starts with %q, which is neither an exported method "+
				"name nor a known family (%s)", id, scope, strings.Join(FamilyNames(), ", "),
		)
	default:
		return fmt.Errorf("check ID %q starts with %q, which is not a name", id, scope)
	}
}

// validateSegment holds one sub-segment to the slug grammar, or to the law-id
// form a bound law reports under.
//
// Law identifiers keep their upper case because they are the engine's own
// names and appear verbatim in its output; matching them to a slug would make
// the report and the ID disagree about what a law is called.
func validateSegment(id ID, seg string) error {
	if seg == "" {
		return fmt.Errorf("check ID %q has an empty segment", id)
	}
	if strings.HasPrefix(seg, "AUTO-") {
		if len(seg) == len("AUTO-") {
			return fmt.Errorf(
				"check ID %q: law segment %q names no law — the grammar requires "+
					"at least one character after the prefix", id, seg,
			)
		}
		for _, r := range seg[len("AUTO-"):] {
			if !isUpper(r) && !isDigit(r) && r != '-' {
				return fmt.Errorf(
					"check ID %q: law segment %q may hold only A-Z, 0-9 and '-'", id, seg,
				)
			}
		}
		return nil
	}
	for _, r := range seg {
		if !isLower(r) && !isDigit(r) && r != '-' {
			return fmt.Errorf(
				"check ID %q: segment %q is not a slug — a-z, 0-9 and '-' only. "+
					"The sentence belongs in Claim, which nothing freezes; the ID is "+
					"what a lock file and a Without() call are written against", id, seg,
			)
		}
	}
	return nil
}

func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// SegConst returns the identifier this package declares the segment
// under, for emitted code that must name a segment rather than repeat
// its slug — `suite.SegSmoke`, not "smoke".
//
// Carried as data because Go cannot ask a constant for its own name,
// and a generated index spelling the slug would be the one place the
// grammar is not single-homed. The segments a generated index can
// reach are the ones listed here; a segment the emitter reaches and
// this map does not is a refusal at generation time rather than a
// literal in somebody's output.
func SegConst(seg string) (string, bool) {
	name, ok := segConsts()[seg]
	return name, ok
}

// segConsts pairs each segment with its declaration.
//
// Hand-written beside the constants rather than reflected: this module
// may not import the `go` toolchain packages, and the census in the
// test file is what holds the two lists equal.
func segConsts() map[string]string {
	return map[string]string{
		SegSmoke:        "SegSmoke",
		SegCancel:       "SegCancel",
		SegDeadline:     "SegDeadline",
		SegNilContext:   "SegNilContext",
		SegZeroValue:    "SegZeroValue",
		SegReader:       "SegReader",
		SegIdempotent:   "SegIdempotent",
		SegMiss:         "SegMiss",
		SegHit:          "SegHit",
		SegCount:        "SegCount",
		SegAccumulates:  "SegAccumulates",
		SegDifferential: "SegDifferential",
		SegLaws:         "SegLaws",
		SegConcurrent:   "SegConcurrent",
		SegLinearizable: "SegLinearizable",
		SegClocked:      "SegClocked",
		SegPoison:       "SegPoison",
		SegLifecycle:    "SegLifecycle",
		SegAppender:     "SegAppender",
		SegRecovery:     "SegRecovery",
		SegCrash:        "SegCrash",
		SegFault:        "SegFault",
		SegHandWritten:  "SegHandWritten",
	}
}

// ClassConst returns the identifier this package declares the class
// under, for emitted code that must name a class rather than repeat its
// slug — `suite.ClassSmoke`, not "signature/smoke".
//
// Carried as data for the reason [SegConst] is, and held to the same
// census: a class the emitter reaches and this map does not is a
// refusal at generation time rather than a literal in somebody's
// output.
func ClassConst(c Class) (string, bool) {
	name, ok := classConsts()[c]
	return name, ok
}

// classConsts pairs each class with its declaration.
func classConsts() map[Class]string {
	return map[Class]string{
		ClassSmoke:        "ClassSmoke",
		ClassCancel:       "ClassCancel",
		ClassDeadline:     "ClassDeadline",
		ClassNilContext:   "ClassNilContext",
		ClassZeroValue:    "ClassZeroValue",
		ClassReader:       "ClassReader",
		ClassIdempotent:   "ClassIdempotent",
		ClassDifferential: "ClassDifferential",
		ClassLaws:         "ClassLaws",
		ClassConcurrent:   "ClassConcurrent",
		ClassClocked:      "ClassClocked",
		ClassPoison:       "ClassPoison",
		ClassLifecycle:    "ClassLifecycle",
		ClassAppender:     "ClassAppender",
		ClassSimRecovery:  "ClassSimRecovery",
		ClassSimCrash:     "ClassSimCrash",
		ClassSimFault:     "ClassSimFault",
		ClassHandWritten:  "ClassHandWritten",
	}
}
