// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"golang.org/x/exp/slices"
)

// Definitions for Value.
// The Value type itself can be found in value_{safe,unsafe}.go.

// Kind is the kind of a Value.
type Kind int

// Unexported version of Kind, just so we can store Kinds in Values.
// (No user-provided value has this type.)
type kind Kind

// The following list is sorted alphabetically, but it's also important that
// AnyKind is 0 so that a zero Value represents nil.

const (
	AnyKind Kind = iota
	BoolKind
	DurationKind
	Float64Kind
	Int64Kind
	StringKind
	TimeKind
	Uint64Kind
	GroupKind
	LogValuerKind
)

var kindStrings = []string{
	"Any",
	"Bool",
	"Duration",
	"Float64",
	"Int64",
	"String",
	"Time",
	"Uint64",
	"GroupKind",
	"LogValuer",
}

func (k Kind) String() string {
	if k >= 0 && int(k) < len(kindStrings) {
		return kindStrings[k]
	}
	return "<unknown slog.Kind>"
}

//////////////// Constructors

// IntValue returns a Value for an int.
func IntValue(v int) Value {
	return Int64Value(int64(v))
}

// Int64Value returns a Value for an int64.
func Int64Value(v int64) Value {
	return Value{num: uint64(v), any: Int64Kind}
}

// Uint64Value returns a Value for a uint64.
func Uint64Value(v uint64) Value {
	return Value{num: v, any: Uint64Kind}
}

// Float64Value returns a Value for a floating-point number.
func Float64Value(v float64) Value {
	return Value{num: math.Float64bits(v), any: Float64Kind}
}

// BoolValue returns a Value for a bool.
func BoolValue(v bool) Value {
	u := uint64(0)
	if v {
		u = 1
	}
	return Value{num: u, any: BoolKind}
}

// Unexported version of *time.Location, just so we can store *time.Locations in
// Values. (No user-provided value has this type.)
type timeLocation *time.Location

// TimeValue returns a Value for a time.Time.
// It discards the monotonic portion.
func TimeValue(v time.Time) Value {
	return Value{num: uint64(v.UnixNano()), any: timeLocation(v.Location())}
}

// DurationValue returns a Value for a time.Duration.
func DurationValue(v time.Duration) Value {
	return Value{num: uint64(v.Nanoseconds()), any: DurationKind}
}

// GroupValue returns a new Value for a list of Attrs.
// The caller must not subsequently mutate the argument slice.
func GroupValue(as ...Attr) Value {
	return groupValue(as)
}

// AnyValue returns a Value for the supplied value.
//
// Given a value of one of Go's predeclared string, bool, or
// (non-complex) numeric types, AnyValue returns a Value of kind
// String, Bool, Uint64, Int64, or Float64. The width of the
// original numeric type is not preserved.
//
// Given a time.Time or time.Duration value, AnyValue returns a Value of kind
// TimeKind or DurationKind. The monotonic time is not preserved.
//
// For nil, or values of all other types, including named types whose
// underlying type is numeric, AnyValue returns a value of kind AnyKind.
func AnyValue(v any) Value {
	switch v := v.(type) {
	case string:
		return StringValue(v)
	case int:
		return Int64Value(int64(v))
	case int64:
		return Int64Value(v)
	case uint64:
		return Uint64Value(v)
	case bool:
		return BoolValue(v)
	case time.Duration:
		return DurationValue(v)
	case time.Time:
		return TimeValue(v)
	case uint8:
		return Uint64Value(uint64(v))
	case uint16:
		return Uint64Value(uint64(v))
	case uint32:
		return Uint64Value(uint64(v))
	case uintptr:
		return Uint64Value(uint64(v))
	case int8:
		return Int64Value(int64(v))
	case int16:
		return Int64Value(int64(v))
	case int32:
		return Int64Value(int64(v))
	case float64:
		return Float64Value(v)
	case float32:
		return Float64Value(float64(v))
	case []Attr:
		return GroupValue(v...)
	case Kind:
		return Value{any: kind(v)}
	default:
		return Value{any: v}
	}
}

//////////////// Accessors

// Any returns v's value as an any.
func (v Value) Any() any {
	switch v.Kind() {
	case AnyKind, GroupKind, LogValuerKind:
		if k, ok := v.any.(kind); ok {
			return Kind(k)
		}
		return v.any
	case Int64Kind:
		return int64(v.num)
	case Uint64Kind:
		return v.num
	case Float64Kind:
		return v.float()
	case StringKind:
		return v.str()
	case BoolKind:
		return v.bool()
	case DurationKind:
		return v.duration()
	case TimeKind:
		return v.time()
	default:
		panic(fmt.Sprintf("bad kind: %s", v.Kind()))
	}
}

// Int64 returns v's value as an int64. It panics
// if v is not a signed integer.
func (v Value) Int64() int64 {
	if g, w := v.Kind(), Int64Kind; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return int64(v.num)
}

// Uint64 returns v's value as a uint64. It panics
// if v is not an unsigned integer.
func (v Value) Uint64() uint64 {
	if g, w := v.Kind(), Uint64Kind; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return v.num
}

// Bool returns v's value as a bool. It panics
// if v is not a bool.
func (v Value) Bool() bool {
	if g, w := v.Kind(), BoolKind; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return v.bool()
}

func (a Value) bool() bool {
	return a.num == 1
}

// Duration returns v's value as a time.Duration. It panics
// if v is not a time.Duration.
func (a Value) Duration() time.Duration {
	if g, w := a.Kind(), DurationKind; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}

	return a.duration()
}

func (a Value) duration() time.Duration {
	return time.Duration(int64(a.num))
}

// Float64 returns v's value as a float64. It panics
// if v is not a float64.
func (v Value) Float64() float64 {
	if g, w := v.Kind(), Float64Kind; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}

	return v.float()
}

func (a Value) float() float64 {
	return math.Float64frombits(a.num)
}

// Time returns v's value as a time.Time. It panics
// if v is not a time.Time.
func (v Value) Time() time.Time {
	if g, w := v.Kind(), TimeKind; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return v.time()
}

func (v Value) time() time.Time {
	return time.Unix(0, int64(v.num)).In(v.any.(timeLocation))
}

// LogValuer returns v's value as a LogValuer. It panics
// if v is not a LogValuer.
func (v Value) LogValuer() LogValuer {
	return v.any.(LogValuer)
}

// Group returns v's value as a []Attr.
// It panics if v's Kind is not GroupKind.
func (v Value) Group() []Attr {
	return v.group()
}

//////////////// Other

// Equal reports whether v and w have equal keys and values.
func (v Value) Equal(w Value) bool {
	k1 := v.Kind()
	k2 := w.Kind()
	if k1 != k2 {
		return false
	}
	switch k1 {
	case Int64Kind, Uint64Kind, BoolKind, DurationKind:
		return v.num == w.num
	case StringKind:
		return v.str() == w.str()
	case Float64Kind:
		return v.float() == w.float()
	case TimeKind:
		return v.time().Equal(w.time())
	case AnyKind, LogValuerKind:
		return v.any == w.any // may panic if non-comparable
	case GroupKind:
		return slices.EqualFunc(v.uncheckedGroup(), w.uncheckedGroup(), Attr.Equal)
	default:
		panic(fmt.Sprintf("bad kind: %s", k1))
	}
}

// append appends a text representation of v to dst.
// v is formatted as with fmt.Sprint.
func (v Value) append(dst []byte) []byte {
	switch v.Kind() {
	case StringKind:
		return append(dst, v.str()...)
	case Int64Kind:
		return strconv.AppendInt(dst, int64(v.num), 10)
	case Uint64Kind:
		return strconv.AppendUint(dst, v.num, 10)
	case Float64Kind:
		return strconv.AppendFloat(dst, v.float(), 'g', -1, 64)
	case BoolKind:
		return strconv.AppendBool(dst, v.bool())
	case DurationKind:
		return append(dst, v.duration().String()...)
	case TimeKind:
		return append(dst, v.time().String()...)
	case AnyKind, GroupKind, LogValuerKind:
		return append(dst, fmt.Sprint(v.any)...)
	default:
		panic(fmt.Sprintf("bad kind: %s", v.Kind()))
	}
}

// A LogValuer is any Go value that can convert itself into a Value for logging.
//
// This mechanism may be used to defer expensive operations until they are
// needed, or to expand a single value into a sequence of components.
type LogValuer interface {
	LogValue() Value
}

const maxLogValues = 100

// Resolve repeatedly calls LogValue on v while it implements LogValuer,
// and returns the result.
// If the number of LogValue calls exceeds a threshold, a Value containing an
// error is returned.
// Resolve's return value is guaranteed not to be of Kind LogValuerKind.
func (v Value) Resolve() Value {
	orig := v
	for i := 0; i < maxLogValues; i++ {
		if v.Kind() != LogValuerKind {
			return v
		}
		v = v.LogValuer().LogValue()
	}
	err := fmt.Errorf("LogValue called too many times on Value of type %T", orig.Any())
	return AnyValue(err)
}
