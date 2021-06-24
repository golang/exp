// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keys

import (
	"golang.org/x/exp/event"
)

// Value represents a key for untyped values.
type Value string

// From can be used to get a value from a Label.
func (k Value) From(l event.Label) interface{} { return l.Interface() }

// Of creates a new Label with this key and the supplied value.
func (k Value) Of(v interface{}) event.Label {
	return event.Value(string(k), v)
}

// Tag represents a key for tagging labels that have no value.
// These are used when the existence of the label is the entire information it
// carries, such as marking events to be of a specific kind, or from a specific
// package.
type Tag string

// New creates a new Label with this key.
func (k Tag) New() event.Label {
	return event.Label{Name: string(k)}
}

// Int represents a key
type Int string

// Of creates a new Label with this key and the supplied value.
func (k Int) Of(v int) event.Label {
	return event.Int64(string(k), int64(v))
}

// From can be used to get a value from a Label.
func (k Int) From(l event.Label) int { return int(l.Int64()) }

// Int8 represents a key
type Int8 string

// Of creates a new Label with this key and the supplied value.
func (k Int8) Of(v int8) event.Label {
	return event.Int64(string(k), int64(v))
}

// From can be used to get a value from a Label.
func (k Int8) From(l event.Label) int8 { return int8(l.Int64()) }

// Int16 represents a key
type Int16 string

// Of creates a new Label with this key and the supplied value.
func (k Int16) Of(v int16) event.Label {
	return event.Int64(string(k), int64(v))
}

// From can be used to get a value from a Label.
func (k Int16) From(l event.Label) int16 { return int16(l.Int64()) }

// Int32 represents a key
type Int32 string

// Of creates a new Label with this key and the supplied value.
func (k Int32) Of(v int32) event.Label {
	return event.Int64(string(k), int64(v))
}

// From can be used to get a value from a Label.
func (k Int32) From(l event.Label) int32 { return int32(l.Int64()) }

// Int64 represents a key
type Int64 string

// Of creates a new Label with this key and the supplied value.
func (k Int64) Of(v int64) event.Label {
	return event.Int64(string(k), v)
}

// From can be used to get a value from a Label.
func (k Int64) From(l event.Label) int64 { return l.Int64() }

// UInt represents a key
type UInt string

// Of creates a new Label with this key and the supplied value.
func (k UInt) Of(v uint) event.Label {
	return event.Uint64(string(k), uint64(v))
}

// From can be used to get a value from a Label.
func (k UInt) From(l event.Label) uint { return uint(l.Uint64()) }

// UInt8 represents a key
type UInt8 string

// Of creates a new Label with this key and the supplied value.
func (k UInt8) Of(v uint8) event.Label {
	return event.Uint64(string(k), uint64(v))
}

// From can be used to get a value from a Label.
func (k UInt8) From(l event.Label) uint8 { return uint8(l.Uint64()) }

// UInt16 represents a key
type UInt16 string

// Of creates a new Label with this key and the supplied value.
func (k UInt16) Of(v uint16) event.Label {
	return event.Uint64(string(k), uint64(v))
}

// From can be used to get a value from a Label.
func (k UInt16) From(l event.Label) uint16 { return uint16(l.Uint64()) }

// UInt32 represents a key
type UInt32 string

// Of creates a new Label with this key and the supplied value.
func (k UInt32) Of(v uint32) event.Label {
	return event.Uint64(string(k), uint64(v))
}

// From can be used to get a value from a Label.
func (k UInt32) From(l event.Label) uint32 { return uint32(l.Uint64()) }

// UInt64 represents a key
type UInt64 string

// Of creates a new Label with this key and the supplied value.
func (k UInt64) Of(v uint64) event.Label {
	return event.Uint64(string(k), v)
}

// From can be used to get a value from a Label.
func (k UInt64) From(l event.Label) uint64 { return l.Uint64() }

// Float32 represents a key
type Float32 string

// Of creates a new Label with this key and the supplied value.
func (k Float32) Of(v float32) event.Label {
	return event.Float64(string(k), float64(v))
}

// From can be used to get a value from a Label.
func (k Float32) From(l event.Label) float32 { return float32(l.Float64()) }

// Float64 represents a key
type Float64 string

// Of creates a new Label with this key and the supplied value.
func (k Float64) Of(v float64) event.Label {
	return event.Float64(string(k), v)
}

// From can be used to get a value from a Label.
func (k Float64) From(l event.Label) float64 {
	return l.Float64()
}

// String represents a key
type String string

// Of creates a new Label with this key and the supplied value.
func (k String) Of(v string) event.Label {
	return event.String(string(k), v)
}

// From can be used to get a value from a Label.
func (k String) From(l event.Label) string { return l.String() }

// Bool represents a key
type Bool string

// Of creates a new Label with this key and the supplied value.
func (k Bool) Of(v bool) event.Label {
	return event.Bool(string(k), v)
}

// From can be used to get a value from a Label.
func (k Bool) From(l event.Label) bool { return l.Bool() }

// Error represents a key
type Error string

// Of creates a new Label with this key and the supplied value.
func (k Error) Of(v error) event.Label {
	return event.Value(string(k), v)
}

// From can be used to get a value from a Label.
func (k Error) From(l event.Label) error { return l.Interface().(error) }
