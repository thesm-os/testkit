// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package region is the string-valued half of the enum corpus.
//
// A string enum's declared value *is* its textual form, and it is already
// written down. A generator deriving the identifier instead — `US` rather than
// `us-east` — still round-trips String against Parse, so its own checks pass
// while every value arriving from JSON, a database column or a query parameter
// fails to parse. Only a fixture whose values differ from its identifiers can
// tell the two apart, which is why none of these matches its name.
//
// The fallback differs too: a numeric conversion does not compile for a string
// type, and rendering the value itself is the more useful diagnostic anyway.
package region

// Region is the enumerated type.
//
//testkit:enum
type Region string

// The declared values. Each differs from its identifier, and none is a prefix
// of another — a prefix would make the fallback indistinguishable from a
// declared value under a naive match.
const (
	US Region = "us-east"
	EU Region = "eu-west"
	AP Region = "ap-south"
)
