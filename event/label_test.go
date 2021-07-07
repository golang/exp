// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event_test

import (
	"testing"
	"time"

	"golang.org/x/exp/event"
)

func TestOfAs(t *testing.T) {
	const i = 3
	var v event.Label
	v = event.Int64("key", i)
	if got := v.Int64(); got != i {
		t.Errorf("got %v, want %v", got, i)
	}
	v = event.Uint64("key", i)
	if got := v.Uint64(); got != i {
		t.Errorf("got %v, want %v", got, i)
	}
	v = event.Float64("key", i)
	if got := v.Float64(); got != i {
		t.Errorf("got %v, want %v", got, i)
	}
	v = event.Bool("key", true)
	if got := v.Bool(); got != true {
		t.Errorf("got %v, want %v", got, true)
	}
	const s = "foo"
	v = event.String("key", s)
	if got := v.String(); got != s {
		t.Errorf("got %v, want %v", got, s)
	}
	v = event.Bytes("key", []byte(s))
	if got := v.Bytes(); string(got) != s {
		t.Errorf("got %v, want %v", got, s)
	}
	tm := time.Now()
	v = event.Value("key", tm)
	if got := v.Interface(); got != tm {
		t.Errorf("got %v, want %v", got, tm)
	}
	var vnil event.Label
	if got := vnil.Interface(); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestEqual(t *testing.T) {
	var x, y int
	vals := []event.Label{
		{},
		event.Int64("key", 1),
		event.Int64("key", 2),
		event.Uint64("key", 3),
		event.Uint64("key", 4),
		event.Float64("key", 3.5),
		event.Float64("key", 3.7),
		event.Bool("key", true),
		event.Bool("key", false),
		event.Value("key", &x),
		event.Value("key", &y),
	}
	for i, v1 := range vals {
		for j, v2 := range vals {
			got := v1.Equal(v2)
			want := i == j
			if got != want {
				t.Errorf("%v.Equal(%v): got %t, want %t", v1, v2, got, want)
			}
		}
	}
}

func panics(f func()) (b bool) {
	defer func() {
		if x := recover(); x != nil {
			b = true
		}
	}()
	f()
	return false
}

func TestPanics(t *testing.T) {
	for _, test := range []struct {
		name string
		f    func()
	}{
		{"int64", func() { event.Float64("key", 3).Int64() }},
		{"uint64", func() { event.Int64("key", 3).Uint64() }},
		{"float64", func() { event.Uint64("key", 3).Float64() }},
		{"bool", func() { event.Int64("key", 3).Bool() }},
		{"duration", func() { event.Value("key", "value").Duration() }},
		{"bytes", func() { event.String("key", "value").Bytes() }},
	} {
		if !panics(test.f) {
			t.Errorf("%s: got no panic, want panic", test.name)
		}
	}
}

func TestString(t *testing.T) {
	for _, test := range []struct {
		v    event.Label
		want string
	}{
		{event.Int64("key", -3), "-3"},
		{event.Uint64("key", 3), "3"},
		{event.Float64("key", .15), "0.15"},
		{event.Bool("key", true), "true"},
		{event.String("key", "foo"), "foo"},
		{event.Value("key", time.Duration(3*time.Second)), "3s"},
	} {
		if got := test.v.String(); got != test.want {
			t.Errorf("%#v: got %q, want %q", test.v, got, test.want)
		}
	}
}

func TestNoAlloc(t *testing.T) {
	// Assign values just to make sure the compiler doesn't optimize away the statements.
	var (
		i int64
		u uint64
		f float64
		b bool
		s string
		x interface{}
		p = &i
	)
	a := int(testing.AllocsPerRun(5, func() {
		i = event.Int64("key", 1).Int64()
		u = event.Uint64("key", 1).Uint64()
		f = event.Float64("key", 1).Float64()
		b = event.Bool("key", true).Bool()
		s = event.String("key", "foo").String()
		x = event.Value("key", p).Interface()
	}))
	if a != 0 {
		t.Errorf("got %d allocs, want zero", a)
	}
	_ = u
	_ = f
	_ = b
	_ = s
	_ = x
}
