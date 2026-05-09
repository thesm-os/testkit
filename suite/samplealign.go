// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

// sampleAlignmentHint returns a diagnostic message used when a
// shape baseline's "returns expected" primitive observes a value
// different from the contract's framework-default sample. Pure,
// Predicate, Aggregator, MultiAggregator, and PoisonAccessor have
// no input parameter — their return is purely state-derived — so
// when the assertion fails the actionable explanation is "your
// factory's impl needs to produce this value." The message names
// the shape, the resolution pattern, and points at the per-method
// docblock's "Sample alignment" note for the longer rationale.
//
// shape is a human-readable shape name ("Pure", "Aggregator", …);
// method is a descriptive label ("the method", "the aggregator",
// "Description", …) that reads naturally in the failure summary.
func sampleAlignmentHint(shape, method string) string {
	return shape + " sample alignment: " + method +
		" must return the framework's default sample value. Configure" +
		" your factory (typically an override field on your in-mem, e.g." +
		" SetCountOverride for int Aggregators) so it returns the" +
		" sampled value — see the per-method docblock's 'Sample" +
		" alignment' note in the generated _spec.gen_test.go for the" +
		" longer rationale and the recommended in-mem pattern."
}
