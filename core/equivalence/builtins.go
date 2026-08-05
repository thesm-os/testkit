// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package equivalence

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Strict returns a [Relation] that contributes no options. The
// resulting [Chain] falls back to go-cmp's default deep equality
// for any path no other relation covers.
//
// Strict is the default tail of every chain; consumers usually
// don't need to add it explicitly.
func Strict() Relation { return strictRel{} }

type strictRel struct{}

func (strictRel) Name() string          { return "strict" }
func (strictRel) Options() []cmp.Option { return nil }

// IgnoreFields returns a [Relation] that omits the named struct
// fields of typ from comparison. Drops timestamps that differ
// legitimately between impls, generated metadata, audit fields.
func IgnoreFields(typ reflect.Type, fields ...string) Relation {
	return ignoreFieldsRel{typ: typ, fields: append([]string(nil), fields...)}
}

type ignoreFieldsRel struct {
	typ    reflect.Type
	fields []string
}

func (r ignoreFieldsRel) Name() string {
	return "ignore-fields:" + r.typ.String() + ":" + strings.Join(r.fields, ",")
}

func (r ignoreFieldsRel) Options() []cmp.Option {
	zero := reflect.New(r.typ).Elem().Interface()
	return []cmp.Option{cmpopts.IgnoreFields(zero, r.fields...)}
}

// IgnoreMapKeys returns a [Relation] that drops the named keys from
// map fields of typ before comparison. Used when a map carries
// transient or server-assigned keys that differ across impls.
func IgnoreMapKeys(typ reflect.Type, keys ...string) Relation {
	return ignoreMapKeysRel{typ: typ, keys: append([]string(nil), keys...)}
}

type ignoreMapKeysRel struct {
	typ  reflect.Type
	keys []string
}

func (r ignoreMapKeysRel) Name() string {
	return "ignore-map-keys:" + r.typ.String() + ":" + strings.Join(r.keys, ",")
}

func (r ignoreMapKeysRel) Options() []cmp.Option {
	keys := r.keys
	zero := reflect.New(r.typ).Elem().Interface()
	return []cmp.Option{
		cmpopts.IgnoreMapEntries(func(k string, _ any) bool {
			return slices.Contains(keys, k)
		}),
		// IgnoreMapEntries operates on any map[string]any value
		// reachable from the comparison root. The zero shim above
		// is included so cmpopts can introspect the type if it
		// needs to; the matcher does the actual work.
		cmpopts.IgnoreUnexported(zero),
	}
}

// Approximate returns a [Relation] that compares the named float64
// field of typ within the given tolerance. Both values are
// considered equal when |a - b| <= tolerance. Used for floating-
// point fields where bit-exact equality isn't meaningful.
func Approximate(typ reflect.Type, field string, tolerance float64) Relation {
	return approxRel{typ: typ, field: field, tol: tolerance}
}

type approxRel struct {
	typ   reflect.Type
	field string
	tol   float64
}

func (r approxRel) Name() string {
	return fmt.Sprintf("approximate:%s:%s:%v", r.typ.String(), r.field, r.tol)
}

func (r approxRel) Options() []cmp.Option {
	tol := r.tol
	return []cmp.Option{
		fieldFilter(r.typ, r.field, cmp.Comparer(func(a, b float64) bool {
			d := a - b
			if d < 0 {
				d = -d
			}
			return d <= tol
		})),
	}
}

// RegexFields returns a [Relation] that compares the named string
// fields of typ by checking both values match the given regex.
// Used for opaque tokens where the exact byte sequence varies but
// the format is fixed (UUIDs, hex hashes, ulid strings).
func RegexFields(typ reflect.Type, fields []string, pattern string) Relation {
	re := regexp.MustCompile(pattern)
	return regexFieldsRel{typ: typ, fields: append([]string(nil), fields...), re: re, pattern: pattern}
}

type regexFieldsRel struct {
	typ     reflect.Type
	fields  []string
	re      *regexp.Regexp
	pattern string
}

func (r regexFieldsRel) Name() string {
	return "regex-fields:" + r.typ.String() + ":" + strings.Join(r.fields, ",") + ":" + r.pattern
}

func (r regexFieldsRel) Options() []cmp.Option {
	re := r.re
	opts := make([]cmp.Option, 0, len(r.fields))
	for _, f := range r.fields {
		opts = append(opts, fieldFilter(r.typ, f, cmp.Comparer(func(a, b string) bool {
			return re.MatchString(a) && re.MatchString(b)
		})))
	}
	return opts
}

// Timestamp returns a [Relation] that compares the named time.Time
// field of typ within the given tolerance. Used for fields where
// the impl produces a timestamp and the reference produces another
// — both within the tolerance window are considered equivalent.
func Timestamp(typ reflect.Type, field string, tolerance time.Duration) Relation {
	return timestampRel{typ: typ, field: field, tol: tolerance}
}

type timestampRel struct {
	typ   reflect.Type
	field string
	tol   time.Duration
}

func (r timestampRel) Name() string {
	return fmt.Sprintf("timestamp:%s:%s:%v", r.typ.String(), r.field, r.tol)
}

func (r timestampRel) Options() []cmp.Option {
	tol := r.tol
	return []cmp.Option{
		fieldFilter(r.typ, r.field, cmp.Comparer(func(a, b time.Time) bool {
			d := a.Sub(b)
			if d < 0 {
				d = -d
			}
			return d <= tol
		})),
	}
}

// IDField returns a [Relation] that treats the named string field
// of typ as opaque: any non-empty value equals any other non-empty
// value. Used for server-assigned IDs that vary between impls but
// must be present.
func IDField(typ reflect.Type, field string) Relation {
	return idFieldRel{typ: typ, field: field}
}

type idFieldRel struct {
	typ   reflect.Type
	field string
}

func (r idFieldRel) Name() string { return "id-field:" + r.typ.String() + ":" + r.field }

func (r idFieldRel) Options() []cmp.Option {
	return []cmp.Option{
		fieldFilter(r.typ, r.field, cmp.Comparer(func(a, b string) bool {
			return a != "" && b != ""
		})),
	}
}

// RetryCount returns a [Relation] that compares the named integer
// field of typ as either zero (no retry) or any positive value
// (retried). Used when the impl's retry counter varies but
// "either zero or positive" is what matters semantically.
func RetryCount(typ reflect.Type, field string) Relation {
	return retryCountRel{typ: typ, field: field}
}

type retryCountRel struct {
	typ   reflect.Type
	field string
}

func (r retryCountRel) Name() string { return "retry-count:" + r.typ.String() + ":" + r.field }

func (r retryCountRel) Options() []cmp.Option {
	return []cmp.Option{
		fieldFilter(r.typ, r.field, cmp.Comparer(func(a, b int) bool {
			return (a == 0 && b == 0) || (a > 0 && b > 0)
		})),
	}
}

// OrderInvariant returns a [Relation] that compares the named slice
// field of typ as a multiset — same elements, any order. Used for
// fields where the impl's iteration order is unspecified.
//
// The slice element type must be comparable via go-cmp's default
// equality. For element types requiring custom comparison, compose
// further relations.
func OrderInvariant(typ reflect.Type, field string) Relation {
	return orderInvariantRel{typ: typ, field: field}
}

type orderInvariantRel struct {
	typ   reflect.Type
	field string
}

func (r orderInvariantRel) Name() string {
	return "order-invariant:" + r.typ.String() + ":" + r.field
}

func (r orderInvariantRel) Options() []cmp.Option {
	// FilterPath restricts the transformer to the named slice
	// field; reflect.ValueOf at runtime is therefore always a
	// slice. No defensive non-slice branch needed.
	return []cmp.Option{
		cmp.FilterPath(func(p cmp.Path) bool {
			return matchesField(p, r.typ, r.field)
		}, cmp.Transformer("sortSlice", func(v any) any {
			rv := reflect.ValueOf(v)
			out := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
			reflect.Copy(out, rv)
			sortReflectSlice(out)
			return out.Interface()
		})),
	}
}

// Cardinality returns a [Relation] that compares the named
// slice/map field of typ by element count alone — both sides must
// have a length in [lower, upper]. Element identity is not checked.
func Cardinality(typ reflect.Type, field string, lower, upper int64) Relation {
	return cardinalityRel{typ: typ, field: field, lower: lower, upper: upper}
}

type cardinalityRel struct {
	typ          reflect.Type
	field        string
	lower, upper int64
}

func (r cardinalityRel) Name() string {
	return fmt.Sprintf("cardinality:%s:%s:[%d,%d]", r.typ.String(), r.field, r.lower, r.upper)
}

func (r cardinalityRel) Options() []cmp.Option {
	lower, upper := r.lower, r.upper
	return []cmp.Option{
		cmp.FilterPath(func(p cmp.Path) bool {
			return matchesField(p, r.typ, r.field)
		}, cmp.Comparer(func(a, b any) bool {
			// FilterPath gates this to a slice/map field whose
			// element type lenOf can measure; non-length-bearing
			// kinds are not reachable from blackbox usage.
			la := lenOf(a)
			lb := lenOf(b)
			return la >= lower && la <= upper && lb >= lower && lb <= upper
		})),
	}
}

// ErrorClass returns a [Relation] that treats any two errors of the
// declared error type (or implementing the declared interface) as
// equivalent. Used when the impl's error type differs between
// backends but both classify the same way (e.g., *pq.Error vs
// *pgx.PgError, both representing a SQL constraint violation).
//
// errType is the reflect.Type of the error type or interface; both
// values must satisfy it.
func ErrorClass(errType reflect.Type) Relation {
	return errorClassRel{errType: errType}
}

type errorClassRel struct {
	errType reflect.Type
}

func (r errorClassRel) Name() string { return "error-class:" + r.errType.String() }

func (r errorClassRel) Options() []cmp.Option {
	want := r.errType
	// Filter over `any` and unwrap to `error` inside, so the
	// comparator fires even when the two arguments have differing
	// concrete error types (the common migration case: *pq.Error
	// vs *pgx.PgError both classified as the same SQL error).
	// go-cmp's interface-typed Comparer doesn't bridge mismatched
	// concrete types at the top level; FilterValues over any does.
	//
	// Nil-handling is intentionally left to go-cmp's top-level
	// fast-path: typed-nil-vs-typed-nil and typed-nil-vs-non-nil
	// pairs short-circuit before this option fires, so the
	// FilterValues body only sees pairs of non-nil errors.
	return []cmp.Option{
		cmp.FilterValues(func(a, b any) bool {
			ea, oka := a.(error)
			eb, okb := b.(error)
			if !oka || !okb {
				return false
			}
			return implements(ea, want) && implements(eb, want)
		}, cmp.Comparer(func(_, _ any) bool {
			// FilterValues established both values implement the
			// named class; same-class is equal regardless of
			// message.
			return true
		})),
	}
}

// Custom returns a [Relation] backed by a consumer-supplied
// comparator. The function is invoked for every value pair the
// chain encounters; consumers wrap with type-specific filtering
// when narrower scope is desired (build a Relation directly,
// implementing the Relation interface).
func Custom(name string, fn func(a, b any) bool) Relation {
	return customRel{name: name, fn: fn}
}

type customRel struct {
	name string
	fn   func(a, b any) bool
}

func (r customRel) Name() string { return "custom:" + r.name }

func (r customRel) Options() []cmp.Option {
	fn := r.fn
	// go-cmp rejects a naked Comparer over `any`; gate it with a
	// FilterValues that always matches so the comparator applies
	// to every value pair.
	return []cmp.Option{
		cmp.FilterValues(func(_, _ any) bool { return true }, cmp.Comparer(fn)),
	}
}

// fieldFilter returns a cmp.Option that applies inner only to
// comparisons of the named field of the given struct type. Other
// paths fall through; if no other relation covers them, go-cmp's
// default deep equality applies.
func fieldFilter(typ reflect.Type, field string, inner cmp.Option) cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		return matchesField(p, typ, field)
	}, inner)
}

// matchesField reports whether p ends in a struct-field step naming
// field on the given struct type.
func matchesField(p cmp.Path, typ reflect.Type, field string) bool {
	if len(p) < 2 {
		return false
	}
	last := p.Last()
	sf, ok := last.(cmp.StructField)
	if !ok || sf.Name() != field {
		return false
	}
	parent := p.Index(len(p) - 2)
	return parent.Type() == typ
}

// implements reports whether a satisfies the given error interface
// or concrete type. Used by ErrorClass; the nil-a path is filtered
// out by errorClassRel.Options before reaching here.
func implements(a error, want reflect.Type) bool {
	at := reflect.TypeOf(a)
	if at == want {
		return true
	}
	if want.Kind() == reflect.Interface {
		return at.Implements(want)
	}
	return false
}

// lenOf returns the length of a slice/map/array/string/chan. The
// caller (cardinalityRel.Options) gates this through FilterPath on
// a slice or map field, so non-length-bearing kinds never reach
// here from blackbox usage.
func lenOf(v any) int64 {
	return int64(reflect.ValueOf(v).Len())
}

// sortReflectSlice sorts a reflect.Value slice using the natural
// ordering produced by go-cmp's diff for the element type. Used by
// OrderInvariant; falls back to comparing element string forms when
// the element type is not comparable.
func sortReflectSlice(rv reflect.Value) {
	sort.SliceStable(rv.Interface(), func(i, j int) bool {
		ai := rv.Index(i).Interface()
		aj := rv.Index(j).Interface()
		return fmt.Sprintf("%v", ai) < fmt.Sprintf("%v", aj)
	})
}
