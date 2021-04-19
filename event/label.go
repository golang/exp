// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"reflect"
	"strconv"
	"unsafe"
)

// ValueHandler is used to safely unpack unknown labels.
type ValueHandler interface {
	String(v string)
	Quote(v string)
	Int(v int64)
	Uint(v uint64)
	Float(v float64)
	Value(v interface{})
}

// LabelDispatcher is used as the identity of a Label.
type LabelDispatcher func(h ValueHandler, l Label)

// Label holds a key and value pair.
// It is normally used when passing around lists of labels.
type Label struct {
	key      string
	dispatch LabelDispatcher
	packed   uint64
	untyped  interface{}
}

// OfValue creates a new label from the key and value.
// This method is for implementing new key types, label creation should
// normally be done with the Of method of the key.
func OfValue(k string, d LabelDispatcher, value interface{}) Label {
	return Label{key: k, dispatch: d, untyped: value}
}

// UnpackValue assumes the label was built using LabelOfValue and returns the value
// that was passed to that constructor.
// This method is for implementing new key types, for type safety normal
// access should be done with the From method of the key.
func (l Label) UnpackValue() interface{} { return l.untyped }

// Of64 creates a new label from a key and a uint64. This is often
// used for non uint64 values that can be packed into a uint64.
// This method is for implementing new key types, label creation should
// normally be done with the Of method of the key.
func Of64(k string, d LabelDispatcher, v uint64) Label {
	return Label{key: k, dispatch: d, packed: v}
}

// Unpack64 assumes the label was built using LabelOf64 and returns the value that
// was passed to that constructor.
// This method is for implementing new key types, for type safety normal
// access should be done with the From method of the key.
func (l Label) Unpack64() uint64 { return l.packed }

type stringptr unsafe.Pointer

// OfString creates a new label from a key and a string.
// This method is for implementing new key types, label creation should
// normally be done with the Of method of the key.
func OfString(k string, d LabelDispatcher, v string) Label {
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&v))
	return Label{
		key:      k,
		dispatch: d,
		packed:   uint64(hdr.Len),
		untyped:  stringptr(hdr.Data),
	}
}

// UnpackString assumes the label was built using LabelOfString and returns the
// value that was passed to that constructor.
// This method is for implementing new key types, for type safety normal
// access should be done with the From method of the key.
func (l Label) UnpackString() string {
	var v string
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&v))
	hdr.Data = uintptr(l.untyped.(stringptr))
	hdr.Len = int(l.packed)
	return v
}

// Valid returns true if the Label is a valid one (it has a key).
func (l Label) Valid() bool { return l.key != "" }

// Key returns the key of this Label.
func (l Label) Key() string { return l.key }

// Apply calls the appropriate method of h on the label's value.
func (l Label) Apply(h ValueHandler) {
	if l.dispatch != nil {
		l.dispatch(h, l)
	}
}

//////////////////////////////////////////////////////////////////////

// These are more demos of what Apply can do, rather than things we'd
// necessarily want here.

// Value is an expensive but general way to get a label's value.
func (l Label) Value() interface{} {
	var v interface{}
	l.Apply(vhandler{&v})
	return v
}

type vhandler struct {
	pv *interface{}
}

func (h vhandler) String(v string)     { *h.pv = v }
func (h vhandler) Quote(v string)      { *h.pv = strconv.Quote(v) }
func (h vhandler) Int(v int64)         { *h.pv = v }
func (h vhandler) Uint(v uint64)       { *h.pv = v }
func (h vhandler) Float(v float64)     { *h.pv = v }
func (h vhandler) Value(v interface{}) { *h.pv = v }

// AppendValue appends the value of l to *dst as text.
func (l Label) AppendValue(dst *[]byte) {
	l.Apply(ahandler{dst})
}

type ahandler struct {
	b *[]byte
}

func (h ahandler) String(v string)     { *h.b = append(*h.b, v...) }
func (h ahandler) Quote(v string)      { *h.b = strconv.AppendQuote(*h.b, v) }
func (h ahandler) Int(v int64)         { *h.b = strconv.AppendInt(*h.b, v, 10) }
func (h ahandler) Uint(v uint64)       { *h.b = strconv.AppendUint(*h.b, v, 10) }
func (h ahandler) Float(v float64)     { *h.b = strconv.AppendFloat(*h.b, v, 'E', -1, 32) }
func (h ahandler) Value(v interface{}) { *h.b = append(*h.b, fmt.Sprint(v)...) }
