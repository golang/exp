// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
	"unsafe"
)

// Value holds any value in an efficient way that avoids allocations for
// most types.
type Value struct {
	packed  uint64
	untyped interface{}
}

// Label is a named value.
type Label struct {
	Name  string
	Value Value
}

// Equal reports whether two labels are equal.
func (l1 Label) Equal(l2 Label) bool {
	return l1.Name == l2.Name && l1.Value.Equal(l2.Value)
}

// stringptr is used in untyped when the Value is a string
type stringptr unsafe.Pointer

// bytesptr is used in untyped when the Value is a byte slice
type bytesptr unsafe.Pointer

// int64Kind is used in untyped when the Value is a signed integer
type int64Kind struct{}

// uint64Kind is used in untyped when the Value is an unsigned integer
type uint64Kind struct{}

// float64Kind is used in untyped when the Value is a floating point number
type float64Kind struct{}

// boolKind is used in untyped when the Value is a boolean
type boolKind struct{}

// durationKind is used in untyped when the Value is a time.Duration
type durationKind struct{}

// HasValue returns true if the value is set to any type.
func (v *Value) HasValue() bool { return v.untyped != nil }

// Equal reports whether two values are equal.
func (v1 *Value) Equal(v2 Value) bool {
	if !v1.HasValue() {
		return !v2.HasValue()
	}
	if !v2.HasValue() {
		return false
	}
	switch {
	case v1.IsString():
		return v2.IsString() && v1.String() == v2.String()
	case v1.IsInt64():
		return v2.IsInt64() && v1.packed == v2.packed
	case v1.IsUint64():
		return v2.IsUint64() && v1.packed == v2.packed
	case v1.IsFloat64():
		return v2.IsFloat64() && v1.Float64() == v2.Float64()
	case v1.IsBool():
		return v2.IsBool() && v1.packed == v2.packed
	case v1.IsDuration():
		return v2.IsDuration() && v1.packed == v2.packed
	default:
		return v1.untyped == v2.untyped
	}
}

// ValueOf returns a Value for the supplied value.
func ValueOf(value interface{}) Value {
	return Value{untyped: value}
}

// Interface returns the value.
// This will never panic, things that were not set using SetInterface will be
// unpacked and returned anyway.
func (v Value) Interface() interface{} {
	switch {
	case v.IsString():
		return v.String()
	case v.IsInt64():
		return v.Int64()
	case v.IsUint64():
		return v.Uint64()
	case v.IsFloat64():
		return v.Float64()
	case v.IsBool():
		return v.Bool()
	case v.IsDuration():
		return v.Duration()
	default:
		return v.untyped
	}
}

// StringOf returns a new Value for a string.
func StringOf(s string) Value {
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	return Value{packed: uint64(hdr.Len), untyped: stringptr(hdr.Data)}
}

// String returns the value as a string.
// This does not panic if v's Kind is not String, instead, it returns a string
// representation of the value in all cases.
func (v Value) String() string {
	if sp, ok := v.untyped.(stringptr); ok {
		var s string
		hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
		hdr.Data = uintptr(sp)
		hdr.Len = int(v.packed)
		return s
	}
	// not a string, convert to one
	switch {
	case v.IsInt64():
		return strconv.FormatInt(v.Int64(), 10)
	case v.IsUint64():
		return strconv.FormatUint(v.Uint64(), 10)
	case v.IsFloat64():
		return strconv.FormatFloat(v.Float64(), 'g', -1, 64)
	case v.IsBool():
		if v.Bool() {
			return "true"
		} else {
			return "false"
		}
	default:
		return fmt.Sprint(v.Interface())
	}
}

// IsString returns true if the value was built with StringOf.
func (v Value) IsString() bool {
	_, ok := v.untyped.(stringptr)
	return ok
}

// BytesOf returns a new Value for a string.
func BytesOf(data []byte) Value {
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	return Value{packed: uint64(hdr.Len), untyped: bytesptr(hdr.Data)}
}

// Bytes returns the value as a bytes array.
func (v Value) Bytes() []byte {
	bp, ok := v.untyped.(bytesptr)
	if !ok {
		panic("Bytes called on non []byte value")
	}
	var buf []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	hdr.Data = uintptr(bp)
	hdr.Len = int(v.packed)
	hdr.Cap = hdr.Len // TODO: is this safe?
	return buf
}

// IsBytes returns true if the value was built with BytesOf.
func (v Value) IsBytes() bool {
	_, ok := v.untyped.(bytesptr)
	return ok
}

// Int64Of returns a new Value for a signed integer.
func Int64Of(u int64) Value {
	return Value{packed: uint64(u), untyped: int64Kind{}}
}

// Int64 returns the int64 from a value that was set with SetInt64.
// It will panic for any value for which IsInt64 is not true.
func (v Value) Int64() int64 {
	if !v.IsInt64() {
		panic("Int64 called on non int64 value")
	}
	return int64(v.packed)
}

// IsInt64 returns true if the value was built with SetInt64.
func (v Value) IsInt64() bool {
	_, ok := v.untyped.(int64Kind)
	return ok
}

// Uint64Of returns a new Value for an unsigned integer.
func Uint64Of(u uint64) Value {
	return Value{packed: u, untyped: uint64Kind{}}
}

// Uint64 returns the uint64 from a value that was set with SetUint64.
// It will panic for any value for which IsUint64 is not true.
func (v Value) Uint64() uint64 {
	if !v.IsUint64() {
		panic("Uint64 called on non uint64 value")
	}
	return v.packed
}

// IsUint64 returns true if the value was built with SetUint64.
func (v Value) IsUint64() bool {
	_, ok := v.untyped.(uint64Kind)
	return ok
}

// Float64Of returns a new Value for a floating point number.
func Float64Of(f float64) Value {
	return Value{packed: math.Float64bits(f), untyped: float64Kind{}}
}

// Float64 returns the float64 from a value that was set with SetFloat64.
// It will panic for any value for which IsFloat64 is not true.
func (v Value) Float64() float64 {
	if !v.IsFloat64() {
		panic("Float64 called on non float64 value")
	}
	return math.Float64frombits(v.packed)
}

// IsFloat64 returns true if the value was built with SetFloat64.
func (v Value) IsFloat64() bool {
	_, ok := v.untyped.(float64Kind)
	return ok
}

// BoolOf returns a new Value for a bool.
func BoolOf(b bool) Value {
	if b {
		return Value{packed: 1, untyped: boolKind{}}
	}
	return Value{packed: 0, untyped: boolKind{}}
}

// Bool returns the bool from a value that was set with SetBool.
// It will panic for any value for which IsBool is not true.
func (v Value) Bool() bool {
	if !v.IsBool() {
		panic("Bool called on non bool value")
	}
	if v.packed != 0 {
		return true
	}
	return false
}

// IsBool returns true if the value was built with SetBool.
func (v Value) IsBool() bool {
	_, ok := v.untyped.(boolKind)
	return ok
}

func DurationOf(d time.Duration) Value {
	return Value{packed: uint64(d), untyped: durationKind{}}
}

func (v Value) Duration() time.Duration {
	if !v.IsDuration() {
		panic("Duration called on non-Duration value")
	}
	return time.Duration(v.packed)
}

func (v Value) IsDuration() bool {
	_, ok := v.untyped.(durationKind)
	return ok
}
