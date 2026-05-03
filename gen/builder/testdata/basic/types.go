// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

//go:generate testkit builder -o basictest/builders.gen.go Item

// Item is a stored value with various field types.
type Item struct {
	ID       string
	Name     string
	Count    int
	Active   bool
	Tags     []string
	Data     []byte
	Metadata map[string]string
	hidden   int // unexported — should not get a setter
}
