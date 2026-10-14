// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sorted

import (
	"bytes"
	"cmp"
	"fmt"
	"iter"
	"maps"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	for N := range 11 {
		m := NewMap[int, int](cmp.Compare)
		_, slice := permute(m, N)
		checkSize(t, m)
		for k, want := range slice {
			v, ok := m.Lookup(k)
			if v != want || ok != (want > 0) {
				t.Fatalf("Lookup(%d) = %d, %v, want %d, %v\nM: %v", k, v, ok, want, want > 0, dump(m))
			}
		}
	}
}

// permute populates a Map[int, int] with a permutation of
// the integers [0, n-1], as an input to tests.
//
// The actual keys and values are functions of the permutation values, not
// the values themselves.
// In the second return value, slice[k] == a non-zero v iff the map has the
// entry k:v.
func permute(m *Map[int, int], n int) (perm, slice []int) {
	perm = rand.Perm(n)
	slice = make([]int, 2*n+1)
	for i, x := range perm {
		m.Set(2*x+1, i+1)
		slice[2*x+1] = i + 1
	}
	// Overwrite-Set half the entries.
	for i, x := range perm[:len(perm)/2] {
		m.Set(2*x+1, i+100)
		slice[2*x+1] = i + 100
	}
	return perm, slice
}

// dump writes the map to a readable string.
func dump(m *Map[int, int]) string {
	var buf bytes.Buffer
	var walk func(*node[int, int])
	walk = func(x *node[int, int]) {
		if x == nil {
			fmt.Fprintf(&buf, "nil")
			return
		}
		fmt.Fprintf(&buf, "(%d[%d]=%d ", x.key, x.sz, x.val)
		walk(x.left)
		fmt.Fprintf(&buf, " ")
		walk(x.right)
		fmt.Fprintf(&buf, ")")
	}
	walk(m.root)
	return buf.String()
}

func TestGet(t *testing.T) {
	m := NewMap[int, int](cmp.Compare)
	// Missing key returns zero
	if got := m.Get(1); got != 0 {
		t.Errorf("Get(1) on empty map: got %d, want 0", got)
	}
	m.Set(1, 10)
	m.Set(2, 20)
	// Existing key returns value
	if got := m.Get(1); got != 10 {
		t.Errorf("Get(1): got %d, want 10", got)
	}
	if got := m.Get(2); got != 20 {
		t.Errorf("Get(2): got %d, want 20", got)
	}
	// Missing key returns zero
	if got := m.Get(3); got != 0 {
		t.Errorf("Get(3): got %d, want 0", got)
	}
}

func TestSet(t *testing.T) {
	m := NewMap[int, int](cmp.Compare)

	check := func(gotOld int, wasPresent bool) func(int, bool) {
		return func(wantOld int, wantPresent bool) {
			t.Helper()
			checkSize(t, m)
			if gotOld != wantOld || wasPresent != wantPresent {
				t.Errorf("got %d, %t, want %d, %t", gotOld, wasPresent, wantOld, wantPresent)
			}
		}
	}

	check(m.Set(1, 10))(0, false)
	check(m.Set(2, 20))(0, false)
	check(m.Set(1, 5))(10, true)
	check(m.Set(1, 8))(5, true)
}

func checkSize(t *testing.T, m *Map[int, int]) {
	t.Helper()
	var check func(n *node[int, int])
	check = func(n *node[int, int]) {
		t.Helper()
		if n == nil {
			return
		}
		if g, w := n.size(), 1+n.left.size()+n.right.size(); g != w {
			t.Fatalf("bad size: got %d, want %d", g, w)
		}
		check(n.left)
		check(n.right)
	}
	check(m.root)
}

func TestRangeCreation(t *testing.T) {
	m := NewMap[int, int](nil)
	for _, tc := range []struct {
		r    MapSpan[int, int]
		want string
	}{
		{m.From(2), "[2, ∞)"},
		{m.Above(2), "(2, ∞)"},
		{m.To(2), "(-∞, 2]"},
		{m.Below(2), "(-∞, 2)"},
		{m.From(2).To(5), "[2, 5]"},
		{m.From(2).Below(5), "[2, 5)"},
		{m.Above(2).To(5), "(2, 5]"},
		{m.Above(2).Below(5), "(2, 5)"},
	} {
		got := rdump(tc.r)
		if got != tc.want {
			t.Errorf("got %s, want %s", got, tc.want)
		}
	}
}

func rdump[K, V any](r MapSpan[K, V]) string {
	var b strings.Builder
	if !r.lo.present {
		b.WriteString("(-∞")
	} else {
		if r.lo.inclusive {
			b.WriteByte('[')
		} else {
			b.WriteByte('(')
		}
		fmt.Fprintf(&b, "%v", r.lo.key)
	}
	b.WriteString(", ")
	if !r.hi.present {
		b.WriteString("∞)")
	} else {
		fmt.Fprintf(&b, "%v", r.hi.key)
		if r.hi.inclusive {
			b.WriteByte(']')
		} else {
			b.WriteByte(')')
		}
	}
	return b.String()
}

func newMapSpan[K cmp.Ordered, V any](m *Map[K, V], lo, hi bound[K]) MapSpan[K, V] {
	return MapSpan[K, V]{m: m, lo: lo, hi: hi}
}

func inf() bound[int] { return bound[int]{} }

// bounds returns a sequence of all pairs of bound[int] for numbers in [0, n).
func bounds(n int) iter.Seq2[bound[int], bound[int]] {
	return func(yield func(_, _ bound[int]) bool) {
		for hi := range n {
			for _, bhi := range []bound[int]{including(hi), excluding(hi), inf()} {
				for lo := range hi + 1 {
					for _, blo := range []bound[int]{including(lo), excluding(lo), inf()} {
						if blo == inf() && bhi == inf() {
							continue
						}
						if !yield(blo, bhi) {
							return
						}
					}
				}
			}
		}
		// Produce an interval past the end. We already generate one before
		// the beginning, (-∞, 0).
		if !yield(excluding(n), inf()) {
			return
		}
		// Yield the infinite interval even if n == 0.
		yield(inf(), inf())
	}
}

func TestBounds(t *testing.T) {
	got := map[string]bool{}
	for blo, bhi := range bounds(2) {
		got[rdump(newMapSpan(NewMap[int, int](cmp.Compare), blo, bhi))] = true
	}
	wants := []string{
		"(0, 0)",
		"(0, 1)",
		"(0, ∞)",
		"[0, 0)",
		"[0, 1)",
		"[0, ∞)",
		"(0, 0]",
		"(0, 1]",
		"[0, 0]",
		"[0, 1]",

		"(1, 1)",
		"(1, ∞)",
		"[1, 1)",
		"[1, ∞)",
		"(1, 1]",
		"[1, 1]",

		"(-∞, 0)",
		"(-∞, 0]",
		"(-∞, 1)",
		"(-∞, 1]",

		"(2, ∞)",
		"(-∞, ∞)",
	}

	want := map[string]bool{}
	for _, w := range wants {
		want[w] = true
	}
	if !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRdump(t *testing.T) {
	m := NewMap[int, int](nil)
	for _, test := range []struct {
		r    MapSpan[int, int]
		want string
	}{
		{newMapSpan(m, inf(), inf()), "(-∞, ∞)"},
		{newMapSpan(m, including(1), inf()), "[1, ∞)"},
		{newMapSpan(m, inf(), excluding(3)), "(-∞, 3)"},
		{newMapSpan(m, including(1), including(3)), "[1, 3]"},
		{newMapSpan(m, including(1), excluding(3)), "[1, 3)"},
		{newMapSpan(m, excluding(1), including(3)), "(1, 3]"},
		{newMapSpan(m, excluding(1), excluding(3)), "(1, 3)"},
	} {
		got := rdump(test.r)
		if got != test.want {
			t.Errorf("%v: got %q, want %q", test.r, got, test.want)
		}
	}
}

func TestRangeBounds(t *testing.T) {
	m := NewMap[int, int](cmp.Compare[int])
	for _, test := range []struct {
		name string
		got  MapSpan[int, int]
		want string
	}{
		{
			"To1",
			m.span().To(10),
			"(-∞, 10]",
		},
		{
			"To2",
			m.span().To(10).To(10),
			"(-∞, 10]",
		},
		{
			"To3",
			m.span().Below(10).To(10),
			"(-∞, 10)",
		},
		{
			"To4",
			m.span().To(9).To(10),
			"(-∞, 9]",
		},
		{
			"Below1",
			m.span().Below(10),
			"(-∞, 10)",
		},
		{
			"Below2",
			m.span().To(10).Below(10),
			"(-∞, 10)",
		},
		{
			"Below3",
			m.span().Below(10).Below(10),
			"(-∞, 10)",
		},
		{
			"Below4",
			m.span().To(9).Below(10),
			"(-∞, 9]",
		},
		{
			"From1",
			m.span().From(10),
			"[10, ∞)",
		},
		{
			"From2",
			m.span().From(10).From(10),
			"[10, ∞)",
		},
		{
			"From3",
			m.span().Above(10).From(10),
			"(10, ∞)",
		},
		{
			"From4",
			m.span().From(11).From(10),
			"[11, ∞)",
		},
		{
			"Above1",
			m.span().Above(10),
			"(10, ∞)",
		},
		{
			"Above2",
			m.span().From(10).Above(10),
			"(10, ∞)",
		},
		{
			"Above3",
			m.span().Above(10).Above(10),
			"(10, ∞)",
		},
		{
			"Above4",
			m.span().From(11).Above(10),
			"[11, ∞)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := rdump(test.got)
			if got != test.want {
				t.Errorf("got %s, want %s", got, test.want)
			}
		})
	}
}
