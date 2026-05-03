// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package stub provides the runtime types for generated test stubs.
//
// [MethodStub] is the per-method dispatch engine with fault injection,
// latency simulation, and call recording. [Recorder] captures
// [RecordedCall] values for after-the-fact inspection. [OrderTracker]
// enforces cross-method call ordering from the order-after directive.
package stub
