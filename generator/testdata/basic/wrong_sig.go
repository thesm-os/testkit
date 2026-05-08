// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

// WrongSig has methods that look like Stringer/Parse/marshalers but
// have wrong signatures. The enum generator's signature predicates
// must reject all of them — otherwise the generated tests would
// reference methods that don't satisfy the encoding interfaces.
//
// Deliberately has no `go:generate` directive — this fixture is
// for predicate-rejection coverage in analyze tests, not end-to-end
// generation. (If generated, the resulting test would compile —
// every wrong-shape method is correctly skipped — but the
// generation step is redundant with the predicate-level coverage.)
type WrongSig int

const (
	WrongSigA WrongSig = iota
	WrongSigB
)

// String takes an argument — not the canonical fmt.Stringer shape.
// HasMethod(_, "String", StringerSig) must return false.
func (s WrongSig) String(int) string { return "" }

// ParseWrongSig returns (string, error) — wrong result type.
// HasFunc(_, "ParseWrongSig", ParseSig("WrongSig")) must return false.
func ParseWrongSig(_ string) (string, error) { return "", nil }

// MarshalText returns a single string instead of ([]byte, error).
// MarshalTextSig must reject it.
func (s WrongSig) MarshalText() string { return "" }

// UnmarshalText takes a string instead of []byte.
// UnmarshalTextSig must reject it.
func (s *WrongSig) UnmarshalText(string) error { return nil }
