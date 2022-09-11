// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !safe_attrs

package slog

// This file defines the most compact representation of Attr.

import (
	"reflect"
	"time"
	"unsafe"
)

// An Attr is a key-value pair.
// It can represent most small values without an allocation.
// The zero Attr has a key of "" and a value of nil.
type Attr struct {
	key string
	// num holds the value for Kinds Int64, Uint64, Float64, Bool and Duration,
	// the string length for StringKind, and nanoseconds since the epoch for TimeKind.
	num uint64
	// If any is of type Kind, then the value is in num as described above.
	// If any is of type *time.Location, then the Kind is Time and time.Time value
	// can be constructed from the Unix nanos in num and the location (monotonic time
	// is not preserved).
	// If any is of type stringptr, then the Kind is String and the string value
	// consists of the length in num and the pointer in any.
	// Otherwise, the Kind is Any and any is the value.
	// (This implies that Attrs cannot store values of type Kind, *time.Location
	// or stringptr.)
	any any
}

// stringptr is used in field `a` when the Value is a string.
type stringptr unsafe.Pointer

// Kind returns the Attr's Kind.
func (a Attr) Kind() Kind {
	switch x := a.any.(type) {
	case Kind:
		return x
	case stringptr:
		return StringKind
	case *time.Location:
		return TimeKind
	default:
		return AnyKind
	}
}

// String returns a new Attr for a string.
func String(key, value string) Attr {
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&value))
	return Attr{key: key, num: uint64(hdr.Len), any: stringptr(hdr.Data)}
}

func (a Attr) str() string {
	var s string
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	hdr.Data = uintptr(a.any.(stringptr))
	hdr.Len = int(a.num)
	return s
}

// String returns Attr's value as a string, formatted like fmt.Sprint. Unlike
// the methods Int64, Float64, and so on, which panic if the Attr is of the
// wrong kind, String never panics.
func (a Attr) String() string {
	if sp, ok := a.any.(stringptr); ok {
		// Inlining this code makes a huge difference.
		var s string
		hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
		hdr.Data = uintptr(sp)
		hdr.Len = int(a.num)
		return s
	}
	var buf []byte
	return string(a.appendValue(buf))
}
