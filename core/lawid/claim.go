// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lawid

import (
	"fmt"
	"strings"
)

// Claim is one law's human claim: the sentence a lock row, a report,
// and a skipped subtest speak for it. It lives beside the identifier
// for the reason the identifier lives here at all — the generator
// writes the sentence into manifests and the engine reports outcomes
// under it, and a wording spelled in two modules drifts where no
// compiler can see.
//
// Parametric where the wording names something only a declaration
// knows. The placeholders are the closed vocabulary below; a claim
// interpolating anything else fails this package's own census, and
// [Claim.Fill] refuses a sentence left half-filled rather than
// publishing a bracket into a manifest.
type Claim string

// The placeholder vocabulary a claim may interpolate. Each names a
// fact the selecting declaration carries; the consumer filling a
// claim resolves them from its own stamps, and over-supplying is
// free — an absent placeholder ignores its pair.
const (
	// PlaceClose is the close method the selecting declaration names:
	// the after-close teardown, or a produced handle's release.
	PlaceClose = "{close}"
	// PlaceNext is the produced handle's reader.
	PlaceNext = "{next}"
	// PlaceProduced is the contract's own word for the produced
	// handle.
	PlaceProduced = "{produced}"
	// PlaceSubject is the subject interface's token.
	PlaceSubject = "{subject}"
)

// Placeholders enumerates the vocabulary, for the census that holds
// every claim's tokens to it.
func Placeholders() []string {
	return []string{PlaceClose, PlaceNext, PlaceProduced, PlaceSubject}
}

// Fill interpolates placeholder/value pairs and refuses a claim left
// unfilled: a leftover bracket in a manifest row would read as prose
// and diff forever after.
func (c Claim) Fill(pairs ...string) (string, error) {
	if len(pairs)%2 != 0 {
		return "", fmt.Errorf("lawid: Fill takes placeholder/value pairs, got %d values", len(pairs))
	}
	out := string(c)
	for i := 0; i < len(pairs); i += 2 {
		out = strings.ReplaceAll(out, pairs[i], pairs[i+1])
	}
	for _, p := range Placeholders() {
		if strings.Contains(out, p) {
			return "", fmt.Errorf("lawid: claim %q left %s unfilled", string(c), p)
		}
	}
	return out, nil
}

// ClaimOf returns the law's claim, false for an identifier this
// package does not word yet — the consumer's signal to refuse the
// row by name rather than invent a sentence. Wordings accrete toward
// the full registry under the conformance corpus, which surfaces
// every unworded law the day a fixture stamps its classification.
func ClaimOf(id string) (Claim, bool) {
	c, ok := claims()[id]
	return c, ok
}

// claims words the laws the proof-of-concept corpus pinned; the
// spellings are its manifests', verbatim.
func claims() map[string]Claim {
	return map[string]Claim{
		TTLExpiry:                "an entry stops being readable once its lifetime has run out",
		LifecycleAfterClose:      "once {close} has run, every method reports the closed sentinel",
		PoisonConsistent:         "once the {subject} reports it is closed, it keeps reporting it",
		CursorCloseIdempotent:    "a second {close} on a {produced} changes nothing",
		CursorNextAfterClose:     "once a {produced} is closed, {next} reports the closed sentinel",
		AppenderMonotonicOffsets: "offsets of successive appends strictly increase",
		LeaseDoubleAcquireBlocks: "a second acquire of a held key reports the held sentinel",
		LeaseReleasedOnCancel:    "a held lease frees once its acquiring context is cancelled",
	}
}
