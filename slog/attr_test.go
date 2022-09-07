// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"fmt"
	"testing"
	"time"
	"unsafe"
)

func TestEqual(t *testing.T) {
	var x, y int
	vals := []Attr{
		{},
		Int64("key", 1),
		Int64("key", 2),
		Float64("key", 3.5),
		Float64("key", 3.7),
		Bool("key", true),
		Bool("key", false),
		Any("key", &x),
		Any("key", &y),
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

func TestNilAttr(t *testing.T) {
	n := Any("k", nil)
	if g := n.Value(); g != nil {
		t.Errorf("got %#v, want nil", g)
	}
	if n.HasValue() {
		t.Error("n has value, wanted not")
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

func TestString(t *testing.T) {
	for _, test := range []struct {
		v    Attr
		want string
	}{
		{Int64("key", -3), "-3"},
		{Float64("key", .15), "0.15"},
		{Bool("key", true), "true"},
		{String("key", "foo"), "foo"},
		{Any("key", 3*time.Second), "3s"},
	} {
		if got := test.v.String(); got != test.want {
			t.Errorf("%#v: got %q, want %q", test.v, got, test.want)
		}
	}
}

func TestAttrNoAlloc(t *testing.T) {
	// Assign values just to make sure the compiler doesn't optimize away the statements.
	var (
		i int64
		u uint64
		f float64
		b bool
		s string
		x any
		p = &i
		d time.Duration
	)
	a := int(testing.AllocsPerRun(5, func() {
		i = Int64("key", 1).Int64()
		u = Uint64("key", 1).Uint64()
		f = Float64("key", 1).Float64()
		b = Bool("key", true).Bool()
		s = String("key", "foo").String()
		d = Duration("key", d).Duration()
		x = Any("key", p).Value()
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

func TestAnyLevel(t *testing.T) {
	// Because typical Levels are small integers,
	// they are zero-alloc.
	var a Attr
	x := DebugLevel + 100
	wantAllocs(t, 0, func() { a = Any("k", x) })
	v := a.Value()
	if _, ok := v.(Level); !ok {
		t.Errorf("wanted Level, got %T", v)
	}
}

func wantAllocs(t *testing.T, want int, f func()) {
	t.Helper()
	got := int(testing.AllocsPerRun(5, f))
	if got != want {
		t.Errorf("got %d allocs, want %d", got, want)
	}
}

//////////////// Benchmark for accessing Attr values

// The "As" form is the slowest.
// The switch-panic and visitor times are almost the same.
// BenchmarkDispatch/switch-checked-8         	 8669427	       137.7 ns/op
// BenchmarkDispatch/As-8                     	 8212087	       145.3 ns/op
// BenchmarkDispatch/Visit-8                  	 8926146	       135.3 ns/op
func BenchmarkDispatch(b *testing.B) {
	kvs := []Attr{
		Int64("i", 32768),
		Uint64("u", 0xfacecafe),
		String("s", "anything"),
		Bool("b", true),
		Float64("f", 1.2345),
		Duration("d", time.Second),
		Any("a", b),
	}
	var (
		ii int64
		s  string
		bb bool
		u  uint64
		d  time.Duration
		f  float64
		a  any
	)
	b.Run("switch-checked", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, kv := range kvs {
				switch kv.Kind() {
				case StringKind:
					s = kv.String()
				case Int64Kind:
					ii = kv.Int64()
				case Uint64Kind:
					u = kv.Uint64()
				case Float64Kind:
					f = kv.Float64()
				case BoolKind:
					bb = kv.Bool()
				case DurationKind:
					d = kv.Duration()
				case AnyKind:
					a = kv.Value()
				default:
					panic("bad kind")
				}
			}
		}
		_ = ii
		_ = s
		_ = bb
		_ = u
		_ = d
		_ = f
		_ = a

	})
	b.Run("As", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, kv := range kvs {
				if v, ok := kv.AsString(); ok {
					s = v
				} else if v, ok := kv.AsInt64(); ok {
					ii = v
				} else if v, ok := kv.AsUint64(); ok {
					u = v
				} else if v, ok := kv.AsFloat64(); ok {
					f = v
				} else if v, ok := kv.AsBool(); ok {
					bb = v
				} else if v, ok := kv.AsDuration(); ok {
					d = v
				} else if v, ok := kv.AsAny(); ok {
					a = v
				} else {
					panic("bad kind")
				}
			}
		}
		_ = ii
		_ = s
		_ = bb
		_ = u
		_ = d
		_ = f
		_ = a
	})

	b.Run("Visit", func(b *testing.B) {
		v := &setVisitor{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, kv := range kvs {
				kv.Visit(v)
			}
		}
	})
}

type setVisitor struct {
	i int64
	s string
	b bool
	u uint64
	d time.Duration
	f float64
	a any
}

func (v *setVisitor) String(s string)          { v.s = s }
func (v *setVisitor) Int64(i int64)            { v.i = i }
func (v *setVisitor) Uint64(x uint64)          { v.u = x }
func (v *setVisitor) Float64(x float64)        { v.f = x }
func (v *setVisitor) Bool(x bool)              { v.b = x }
func (v *setVisitor) Duration(x time.Duration) { v.d = x }
func (v *setVisitor) Any(x any)                { v.a = x }

// When dispatching on all types, the "As" functions are slightly slower
// than switching on the kind and then calling a function that checks
// the kind again. See BenchmarkDispatch above.

func (a Attr) AsString() (string, bool) {
	if a.Kind() == StringKind {
		return a.str(), true
	}
	return "", false
}

func (a Attr) AsInt64() (int64, bool) {
	if a.Kind() == Int64Kind {
		return int64(a.num), true
	}
	return 0, false
}

func (a Attr) AsUint64() (uint64, bool) {
	if a.Kind() == Uint64Kind {
		return a.num, true
	}
	return 0, false
}

func (a Attr) AsFloat64() (float64, bool) {
	if a.Kind() == Float64Kind {
		return a.float(), true
	}
	return 0, false
}

func (a Attr) AsBool() (bool, bool) {
	if a.Kind() == BoolKind {
		return a.bool(), true
	}
	return false, false
}

func (a Attr) AsDuration() (time.Duration, bool) {
	if a.Kind() == DurationKind {
		return a.duration(), true
	}
	return 0, false
}

func (a Attr) AsAny() (any, bool) {
	if a.Kind() == AnyKind {
		return a.any, true
	}
	return nil, false
}

// Problem: adding a type means adding a method, which is a breaking change.
// Using an unexported method to force embedding will make programs compile,
// But they will panic at runtime when we call the new method.
type Visitor interface {
	String(string)
	Int64(int64)
	Uint64(uint64)
	Float64(float64)
	Bool(bool)
	Duration(time.Duration)
	Any(any)
}

func (a Attr) Visit(v Visitor) {
	switch a.Kind() {
	case StringKind:
		v.String(a.str())
	case Int64Kind:
		v.Int64(int64(a.num))
	case Uint64Kind:
		v.Uint64(a.num)
	case BoolKind:
		v.Bool(a.bool())
	case Float64Kind:
		v.Float64(a.float())
	case DurationKind:
		v.Duration(a.duration())
	case AnyKind:
		v.Any(a.any)
	default:
		panic("bad kind")
	}
}

// A Attr with "unsafe" strings is significantly faster:
// safe:  1785 ns/op, 0 allocs
// unsafe: 690 ns/op, 0 allocs

// Run this with and without -tags unsafe_kvs to compare.
func BenchmarkUnsafeStrings(b *testing.B) {
	b.ReportAllocs()
	dst := make([]Attr, 100)
	src := make([]Attr, len(dst))
	b.Logf("Attr size = %d", unsafe.Sizeof(Attr{}))
	for i := range src {
		src[i] = String("k", fmt.Sprintf("string#%d", i))
	}
	b.ResetTimer()
	var d string
	for i := 0; i < b.N; i++ {
		copy(dst, src)
		for _, a := range dst {
			d = a.String()
		}
	}
	_ = d
}
