// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaults

//go:generate testkit builder -o defaultstest/inline.gen.go Profile

// Profile demonstrates the same-package factory variant: when the
// `<Type>Defaults()` function lives next to the type itself (rather
// than in the sibling test package), the builder generator finds it
// via the source-package lookup and seeds New<Profile>() through
// it.
type Profile struct {
	Name  string
	Email string
}

// ProfileDefaults returns the canonical seed value for Profile.
// The builder generator detects it via the `<Type>Defaults() <Type>`
// shape — same as the sibling-pkg variant, but resolved against the
// source package directly.
func ProfileDefaults() Profile {
	return Profile{Name: "test-profile", Email: "test@example.com"}
}
