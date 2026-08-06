// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package seed holds values and a companion another package's builders draw
// from, so the cross-package half of default resolution has somewhere to point.
//
// It declares no builder of its own. Its whole purpose is to be referenced from
// outside, which is the case a single-package fixture cannot show.
package seed

// Region is a default a field in another package names.
const Region = "eu-west"

// Retries is a numeric default, present so the cross-package path is exercised
// for more than one literal kind.
const Retries = 3

// Config is the type [ConfigDefaults] seeds. It lives here rather than beside
// the builder so the companion is genuinely cross-package.
type Config struct {
	Region  string
	Retries int
	Label   string
}

// ConfigDefaults is the companion a builder in another package seeds from.
func ConfigDefaults() Config {
	return Config{Region: Region, Retries: Retries, Label: "seeded"}
}
