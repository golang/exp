// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"fmt"
	"time"
)

// An Attr is a key-value pair.
// It can represent most small values without an allocation.
// The zero Attr has a key of "" and a value of nil.
type Attr struct {
	key   string
	value Value
}

// Key returns the Attr's key.
func (a Attr) Key() string { return a.key }

// Value returns the Attr's Value.
func (a Attr) Value() Value { return a.value }

// String returns an Attr for a string value.
func String(key, value string) Attr {
	return Attr{key, StringValue(value)}
}

// Int64 returns an Attr for an int64.
func Int64(key string, value int64) Attr {
	return Attr{key, Int64Value(value)}
}

// Int converts an int to an int64 and returns
// an Attr with that value.
func Int(key string, value int) Attr {
	return Int64(key, int64(value))
}

// Uint64 returns an Attr for a uint64.
func Uint64(key string, v uint64) Attr {
	return Attr{key, Uint64Value(v)}
}

// Float64 returns an Attr for a floating-point number.
func Float64(key string, v float64) Attr {
	return Attr{key, Float64Value(v)}
}

// Bool returns an Attr for a bool.
func Bool(key string, v bool) Attr {
	return Attr{key, BoolValue(v)}
}

// Time returns an Attr for a time.Time.
// It discards the monotonic portion.
func Time(key string, v time.Time) Attr {
	return Attr{key, TimeValue(v)}
}

// Duration returns an Attr for a time.Duration.
func Duration(key string, v time.Duration) Attr {
	return Attr{key, DurationValue(v)}
}

// Any returns an Attr for the supplied value.
// See [Value.AnyValue] for how values are treated.
func Any(key string, value any) Attr {
	return Attr{key, AnyValue(value)}
}

// WithKey returns an attr with the given key and the receiver's value.
func (a Attr) WithKey(key string) Attr {
	a.key = key
	return a
}

// Equal reports whether two Attrs have equal keys and values.
func (a1 Attr) Equal(a2 Attr) bool {
	return a1.key == a2.key && a1.value.Equal(a2.value)
}

func (a Attr) String() string {
	return fmt.Sprintf("%s=%s", a.key, a.value)
}
