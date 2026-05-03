// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fielddefaults

//go:generate testkit builder -o fielddefaultstest/builders.gen.go Config

// Config uses per-field //testkit:default directives.
type Config struct {
	Host    string //testkit:default "localhost"
	Port    int    //testkit:default 8080
	Verbose bool   //testkit:default true
	Name    string // no default — uses zero value
}
