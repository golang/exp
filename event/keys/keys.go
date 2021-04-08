// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keys

import (
	"math"

	"golang.org/x/exp/event"
)

// Value represents a key for untyped values.
type Value string

func (k Value) Name() string { return string(k) }

// From can be used to get a value from a Label.
func (k Value) From(t event.Label) interface{} { return t.UnpackValue() }

// Of creates a new Label with this key and the supplied value.
func (k Value) Of(value interface{}) event.Label {
	return event.OfValue(string(k), dispatchValue, value)
}

// Tag represents a key for tagging labels that have no value.
// These are used when the existence of the label is the entire information it
// carries, such as marking events to be of a specific kind, or from a specific
// package.
type Tag string

func (k Tag) Name() string { return string(k) }

// New creates a new Label with this key.
func (k Tag) New() event.Label { return event.OfValue(string(k), nil, nil) }

// Int represents a key
type Int string

func (k Int) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k Int) Of(v int) event.Label { return event.Of64(string(k), dispatchInt, uint64(v)) }

// From can be used to get a value from a Label.
func (k Int) From(l event.Label) int { return int(l.Unpack64()) }

// Int8 represents a key
type Int8 string

func (k Int8) Name() string                         { return string(k) }
func (k Int8) Print(p event.Printer, l event.Label) { p.Int(int64(k.From(l))) }

// Of creates a new Label with this key and the supplied value.
func (k Int8) Of(v int8) event.Label { return event.Of64(string(k), dispatchInt, uint64(v)) }

// From can be used to get a value from a Label.
func (k Int8) From(t event.Label) int8 { return int8(t.Unpack64()) }

// Int16 represents a key
type Int16 string

func (k Int16) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k Int16) Of(v int16) event.Label { return event.Of64(string(k), dispatchInt, uint64(v)) }

// From can be used to get a value from a Label.
func (k Int16) From(t event.Label) int16 { return int16(t.Unpack64()) }

// Int32 represents a key
type Int32 string

func (k Int32) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k Int32) Of(v int32) event.Label { return event.Of64(string(k), dispatchInt, uint64(v)) }

// From can be used to get a value from a Label.
func (k Int32) From(t event.Label) int32 { return int32(t.Unpack64()) }

// Int64 represents a key
type Int64 string

func (k Int64) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k Int64) Of(v int64) event.Label { return event.Of64(string(k), dispatchInt, uint64(v)) }

// From can be used to get a value from a Label.
func (k Int64) From(t event.Label) int64 { return int64(t.Unpack64()) }

// UInt represents a key
type UInt string

func (k UInt) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k UInt) Of(v uint) event.Label { return event.Of64(string(k), dispatchUint, uint64(v)) }

// From can be used to get a value from a Label.
func (k UInt) From(t event.Label) uint { return uint(t.Unpack64()) }

// UInt8 represents a key
type UInt8 string

func (k UInt8) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k UInt8) Of(v uint8) event.Label { return event.Of64(string(k), dispatchUint, uint64(v)) }

// From can be used to get a value from a Label.
func (k UInt8) From(t event.Label) uint8 { return uint8(t.Unpack64()) }

// UInt16 represents a key
type UInt16 string

func (k UInt16) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k UInt16) Of(v uint16) event.Label { return event.Of64(string(k), dispatchUint, uint64(v)) }

// From can be used to get a value from a Label.
func (k UInt16) From(t event.Label) uint16 { return uint16(t.Unpack64()) }

// UInt32 represents a key
type UInt32 string

func (k UInt32) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k UInt32) Of(v uint32) event.Label { return event.Of64(string(k), dispatchUint, uint64(v)) }

// From can be used to get a value from a Label.
func (k UInt32) From(t event.Label) uint32 { return uint32(t.Unpack64()) }

// UInt64 represents a key
type UInt64 string

func (k UInt64) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k UInt64) Of(v uint64) event.Label { return event.Of64(string(k), dispatchUint, v) }

// From can be used to get a value from a Label.
func (k UInt64) From(t event.Label) uint64 { return t.Unpack64() }

// Float32 represents a key
type Float32 string

func (k Float32) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k Float32) Of(v float32) event.Label {
	return event.Of64(string(k), dispatchFloat, uint64(math.Float32bits(v)))
}

// From can be used to get a value from a Label.
func (k Float32) From(t event.Label) float32 {
	return math.Float32frombits(uint32(t.Unpack64()))
}

// Float64 represents a key
type Float64 string

func (k Float64) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k Float64) Of(v float64) event.Label {
	return event.Of64(string(k), dispatchFloat, math.Float64bits(v))
}

// From can be used to get a value from a Label.
func (k Float64) From(t event.Label) float64 {
	return math.Float64frombits(t.Unpack64())
}

// String represents a key
type String string

func (k String) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k String) Of(v string) event.Label { return event.OfString(string(k), dispatchString, v) }

// From can be used to get a value from a Label.
func (k String) From(t event.Label) string { return t.UnpackString() }

// Boolean represents a key
type Boolean string

func (k Boolean) Name() string { return string(k) }

// Of creates a new Label with this key and the supplied value.
func (k Boolean) Of(v bool) event.Label {
	if v {
		return event.Of64(string(k), dispatchBoolean, 1)
	}
	return event.Of64(string(k), dispatchBoolean, 0)
}

// From can be used to get a value from a Label.
func (k Boolean) From(t event.Label) bool { return t.Unpack64() > 0 }

func dispatchValue(h event.ValueHandler, l event.Label)  { h.Value(l.UnpackValue()) }
func dispatchInt(h event.ValueHandler, l event.Label)    { h.Int(int64(l.Unpack64())) }
func dispatchUint(h event.ValueHandler, l event.Label)   { h.Uint(l.Unpack64()) }
func dispatchString(h event.ValueHandler, l event.Label) { h.Quote(l.UnpackString()) }
func dispatchFloat(h event.ValueHandler, l event.Label)  { h.Float(math.Float64frombits(l.Unpack64())) }

func dispatchBoolean(h event.ValueHandler, l event.Label) {
	if l.Unpack64() > 0 {
		h.String("true")
	} else {
		h.String("false")
	}
}
