// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//go:build race

package model

// RaceEnabled is true when the race detector is active.
// Used to skip negative concurrent tests that would abort on
// data races before Porcupine can check linearizability.
const RaceEnabled = true
