// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"sort"
	"sync"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
)

// Registry holds an ordered set of [Detector]s and dispatches
// [Classify] requests through them in descending priority. The
// zero value is empty; use [NewRegistry] for the populated default
// or build a custom one for tests.
type Registry struct {
	mu        sync.RWMutex
	detectors []Detector // sorted by priority desc, registration order on ties
}

// Register adds a detector to the registry. Detectors are sorted
// by [Detector.Priority] (descending) on each insert, with a stable
// sort so ties resolve in registration order.
func (r *Registry) Register(d Detector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detectors = append(r.detectors, d)
	sort.SliceStable(r.detectors, func(i, j int) bool {
		return r.detectors[i].Priority() > r.detectors[j].Priority()
	})
}

// Detectors returns the registered detectors in priority order.
// The slice is a copy; mutating it does not affect the registry.
func (r *Registry) Detectors() []Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Detector, len(r.detectors))
	copy(out, r.detectors)
	return out
}

// Classify walks the registered detectors in priority order. The
// first matching detector wins; if none matches, the result has
// Shape == [Unknown].
func (r *Registry) Classify(m generator.MethodInfo, tracker *generator.ImportTracker, dirs []directive.Directive) Info {
	sig := ParseSignature(m, tracker, dirs)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, d := range r.detectors {
		if info, ok := d.Detect(sig); ok {
			return info
		}
	}
	return Info{Shape: Unknown}
}

// NewRegistry returns a [Registry] populated with every shipped
// detector. The detector set covers the elite catalog described
// in the package doc.
//
// The detectors register in declaration order; priorities determine
// dispatch order, not the order in this constructor.
func NewRegistry() *Registry {
	r := &Registry{}
	for _, d := range defaultDetectors() {
		r.Register(d)
	}
	return r
}

// defaultDetectors returns the canonical detector slice. Listed in
// descending priority for documentation purposes; the registry
// re-sorts on Register.
//
// Priority ranges (audit reference):
//
//	1000   StreamReader      (iter.Seq* return — wins outright)
//	 950   BatchReader       (variadic key)
//	 900   StreamConsumer    (interface-typed non-ctx param)
//	 850   Lookup            (3 results, last bool)
//	 840   ReaderWithBool    (2 results, last bool)
//	 830   PoisonAccessor    (() error — exact match)
//	 820   Predicate         (() bool)
//	 810   VoidLifecycle     (() — exact match)
//	 800   Pure              (() T — single non-error return)
//	 750   MultiArgWriter    (ctx + 3+ non-ctx + error)
//	 700   CompositeWriter   (ctx?, K1, V) error
//	 650   MultiReader       (ctx?, K) (V1, V2, error)
//	 600   MultiAggregator   (ctx?) (V1, V2, error)
//	 550   Deleter           (ctx?, K) error + //testkit:deleter
//	 500   Writer            (ctx?, V) error | (V, error) | (R, error)
//	 450   PointerReader     (ctx?, K) *V
//	 420   Reader            (ctx?, K) (V, error)
//	 400   ReaderNoError     (ctx?, K) V
//	 350   Aggregator        (ctx?) (T, error) | (T)
//	 300   Mutator           (ctx?, V) void — signature-detected
//	 200   Lifecycle         (ctx) error
func defaultDetectors() []Detector {
	return []Detector{
		streamReaderDetector{},
		batchReaderDetector{},
		streamConsumerDetector{},
		lookupDetector{},
		readerWithBoolDetector{},
		poisonAccessorDetector{},
		predicateDetector{},
		voidLifecycleDetector{},
		pureDetector{},
		multiArgWriterDetector{},
		compositeWriterDetector{},
		multiReaderDetector{},
		multiAggregatorDetector{},
		deleterDetector{},
		writerDetector{},
		pointerReaderDetector{},
		readerDetector{},
		readerNoErrorDetector{},
		aggregatorDetector{},
		mutatorDetector{},
		lifecycleDetector{},
	}
}

// defaultRegistry is initialized lazily on first use.
var (
	defaultOnce     sync.Once
	defaultRegistry *Registry
)

// DefaultRegistry returns the package-level [Registry] populated
// with every shipped detector. The registry is built once on first
// access; subsequent calls return the same instance.
func DefaultRegistry() *Registry {
	defaultOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// Classify is a convenience wrapper around DefaultRegistry().Classify.
func Classify(m generator.MethodInfo, tracker *generator.ImportTracker, dirs []directive.Directive) Info {
	return DefaultRegistry().Classify(m, tracker, dirs)
}
