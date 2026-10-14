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
	"slices"
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

func TestMinMax(t *testing.T) {
	for N := range 11 {
		m := NewMap[int, int](cmp.Compare)
		_, slice := permute(m, N)
		haveMinKey, haveMinVal, haveMinOK := m.Min()
		haveMaxKey, haveMaxVal, haveMaxOK := m.Max()
		var (
			wantMinKey, wantMinVal int
			wantMaxKey, wantMaxVal int
			wok                    bool
		)
		if N != 0 {
			wantMinKey = 1
			wantMinVal = slice[wantMinKey]
			wantMaxKey = 2*N - 1
			wantMaxVal = slice[wantMaxKey]
			wok = true
		}
		if haveMinKey != wantMinKey || haveMinVal != wantMinVal || haveMinOK != wok {
			t.Errorf("N=%d Min() returned %d, %d, %t want %d, %d, %t", N, haveMinKey, haveMinVal, haveMinOK, wantMinKey, wantMinVal, wok)
		}
		if haveMaxKey != wantMaxKey || haveMaxVal != wantMaxVal || haveMaxOK != wok {
			t.Errorf("N=%d Max() returned %d, %d, %t want %d, %d, %t", N, haveMaxKey, haveMaxVal, haveMaxOK, wantMaxKey, wantMaxVal, wok)
		}
	}
}

func TestMinMaxRange(t *testing.T) {
	for N := range 11 {
		for blo, bhi := range bounds(N) {
			m := NewMap[int, int](cmp.Compare)
			_, slice := permute(m, N)
			r := newMapSpan(m, blo, bhi)
			haveMinKey, haveMinVal, haveMinOK := r.Min()
			haveMaxKey, haveMaxVal, haveMaxOK := r.Max()
			var (
				wantMinKey, wantMinVal int
				wantMaxKey, wantMaxVal int
				wok                    bool
			)
			for k, v := range slice {
				if v != 0 && in(k, blo, bhi) {
					if !wok {
						wantMinKey = k
						wantMinVal = v
						wok = true
					}
					wantMaxKey = k
					wantMaxVal = v
				}
			}
			if haveMinKey != wantMinKey || haveMinVal != wantMinVal || haveMinOK != wok {
				t.Errorf("N=%d, r=%s: Min() returned %d, %d, %t want %d, %d, %t",
					N, rdump(r), haveMinKey, haveMinVal, haveMinOK, wantMinKey, wantMinVal, wok)
			}
			if haveMaxKey != wantMaxKey || haveMaxVal != wantMaxVal || haveMaxOK != wok {
				t.Errorf("N=%d, r=%s: Max() returned %d, %d, %t want %d, %d, %t",
					N, rdump(r), haveMaxKey, haveMaxVal, haveMaxOK, wantMaxKey, wantMaxVal, wok)
			}
		}
	}
}

func TestDeleteFunc(t *testing.T) {
	m := NewMap[int, int](cmp.Compare)
	// keys 0..4, values 1..5
	for k, v := range []int{1, 2, 3, 4, 5} {
		m.Set(k, v)
	}

	// Delete even values
	gotb := m.DeleteFunc(func(k, v int) bool {
		return v%2 == 0
	})
	if !gotb {
		t.Fatal("got false, want true")
	}

	got := maps.Collect(m.All())
	want := map[int]int{0: 1, 2: 3, 4: 5}
	if !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	// Delete remaining where key > 2
	gotb = m.DeleteFunc(func(k, v int) bool {
		return k > 2
	})
	if !gotb {
		t.Fatal("got false, want true")
	}

	got = maps.Collect(m.All())
	want = map[int]int{0: 1, 2: 3}
	if !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	// Delete none
	gotb = m.DeleteFunc(func(k, v int) bool {
		return false
	})
	if gotb {
		t.Fatal("got true, want false")
	}
}

func TestAll(t *testing.T) {
	for N := range 11 {
		m := NewMap[int, int](cmp.Compare)
		_, slice := permute(m, N)
		var have []int
		for k, v := range m.All() {
			if v != slice[k] {
				t.Errorf("All() returned %d, %d want %d, %d", k, v, k, slice[k])
			}
			have = append(have, k)
			if len(have) > N+5 { // too many; looping?
				break
			}
		}
		want := nonzeroIndexes(slice)
		if !slices.Equal(have, want) {
			t.Errorf("All() = %v, want %v", have, want)
		}
	}
}

func TestBackward(t *testing.T) {
	for N := range 11 {
		m := NewMap[int, int](cmp.Compare)
		_, slice := permute(m, N)
		var have []int
		for k, v := range m.Backward() {
			if v != slice[k] {
				t.Errorf("All() returned %d, %d want %d, %d", k, v, k, slice[k])
			}
			have = append(have, k)
			if len(have) > N+5 { // too many; looping?
				break
			}
		}
		want := nonzeroIndexes(slice)
		slices.Reverse(want)
		if !slices.Equal(have, want) {
			t.Errorf("All() = %v, want %v", have, want)
		}
	}
}

func TestKeys(t *testing.T) {
	t.Run("Map", func(t *testing.T) {
		for N := range 11 {
			m := NewMap[int, int](cmp.Compare)
			_, slice := permute(m, N)
			var have []int
			for k := range m.Keys() {
				have = append(have, k)
				if len(have) > N+5 {
					break
				}
			}
			want := nonzeroIndexes(slice)
			if !slices.Equal(have, want) {
				t.Errorf("N=%d: Keys() = %v, want %v", N, have, want)
			}
		}
	})
}

func TestValues(t *testing.T) {
	t.Run("Map", func(t *testing.T) {
		for N := range 11 {
			m := NewMap[int, int](cmp.Compare)
			_, slice := permute(m, N)
			var have []int
			for v := range m.Values() {
				have = append(have, v)
				if len(have) > N+5 {
					break
				}
			}
			var want []int
			for _, k := range nonzeroIndexes(slice) {
				want = append(want, slice[k])
			}
			if !slices.Equal(have, want) {
				t.Errorf("N=%d: Values() = %v, want %v", N, have, want)
			}
		}
	})
}

func TestAllRange(t *testing.T) {
	check := func(m *Map[int, int], slice []int, blo, bhi bound[int]) {
		var have []int
		r := newMapSpan(m, blo, bhi)
		for k, v := range r.All() {
			if v != slice[k] {
				t.Errorf("All() returned %d, %d want %d, %d", k, v, k, slice[k])
			}
			have = append(have, k)
			if len(have) > len(slice)+5 { // too many; looping?
				break
			}
		}
		var want []int
		for k, v := range slice {
			if v != 0 && in(k, blo, bhi) {
				want = append(want, k)
			}
		}
		if !slices.Equal(have, want) {
			t.Errorf("All() = %v, want %v", have, want)
		}
	}

	for N := range 11 {
		m := NewMap[int, int](cmp.Compare)
		_, slice := permute(m, N)
		for blo, bhi := range bounds(len(slice)) {
			check(m, slice, blo, bhi)
		}
	}
}

func TestBackwardRange(t *testing.T) {
	check := func(m *Map[int, int], slice []int, blo, bhi bound[int]) {
		var have []int
		r := newMapSpan(m, blo, bhi)
		for k, v := range r.Backward() {
			if v != slice[k] {
				t.Errorf("Backward(%s) returned %d, %d want %d, %d", r, k, v, k, slice[k])
			}
			have = append(have, k)
			if len(have) > len(slice)+5 { // too many; looping?
				break
			}
		}
		var want []int
		for k, v := range slice {
			if v != 0 && in(k, blo, bhi) {
				want = append(want, k)
			}
		}
		slices.Reverse(want)
		if !slices.Equal(have, want) {
			t.Errorf("Backward(%s) = %v, want %v", rdump(r), have, want)
		}
	}

	for N := range 11 {
		m := NewMap[int, int](cmp.Compare)
		_, slice := permute(m, N)
		for blo, bhi := range bounds(len(slice)) {
			check(m, slice, blo, bhi)
		}
	}
}

func TestKeysRange(t *testing.T) {
	t.Run("Range", func(t *testing.T) {
		for N := range 11 {
			m := NewMap[int, int](cmp.Compare)
			_, slice := permute(m, N)
			for blo, bhi := range bounds(len(slice)) {
				r := MapSpan[int, int]{m: m, lo: blo, hi: bhi}
				var have []int
				for k := range r.Keys() {
					have = append(have, k)
					if len(have) > len(slice)+5 {
						break
					}
				}
				var want []int
				for k, v := range slice {
					if v != 0 && in(k, blo, bhi) {
						want = append(want, k)
					}
				}
				if !slices.Equal(have, want) {
					t.Errorf("N=%d, r=%s: Keys() = %v, want %v", N, rdump(r), have, want)
				}
			}
		}
	})
}

func TestValuesRange(t *testing.T) {
	t.Run("Range", func(t *testing.T) {
		for N := range 11 {
			m := NewMap[int, int](cmp.Compare)
			_, slice := permute(m, N)
			for blo, bhi := range bounds(len(slice)) {
				r := MapSpan[int, int]{m: m, lo: blo, hi: bhi}
				var have []int
				for v := range r.Values() {
					have = append(have, v)
					if len(have) > len(slice)+5 {
						break
					}
				}
				var want []int
				for k, v := range slice {
					if v != 0 && in(k, blo, bhi) {
						want = append(want, v)
					}
				}
				if !slices.Equal(have, want) {
					t.Errorf("N=%d, r=%s: Values() = %v, want %v", N, rdump(r), have, want)
				}
			}
		}
	})
}

func TestNth(t *testing.T) {
	for N := range 11 {
		m := NewMap[int, int](cmp.Compare)
		_, slice := permute(m, N)
		var haveKeys, haveVals []int
		for i := range N {
			k, v := m.Nth(i)
			haveKeys = append(haveKeys, k)
			haveVals = append(haveVals, v)
		}
		var wantKeys, wantVals []int
		for k, v := range slice {
			if v != 0 {
				wantKeys = append(wantKeys, k)
				wantVals = append(wantVals, v)
			}
		}
		if !slices.Equal(haveKeys, wantKeys) {
			t.Errorf("keys: have %v, want %v", haveKeys, wantKeys)
		}
		if !slices.Equal(haveVals, wantVals) {
			t.Errorf("values: have %v, want %v", haveVals, wantVals)
		}
	}
}

func TestIndex(t *testing.T) {
	t.Run("Map", func(t *testing.T) {
		m := NewMap[int, int](cmp.Compare)
		// Empty map
		if got := m.Index(1); got != -1 {
			t.Errorf("empty map: got %d, want -1", got)
		}

		// Insert keys 2, 4, 6, 8, 10
		for i := 1; i <= 5; i++ {
			m.Set(2*i, i)
		}

		// Test existing keys
		for i := range 5 {
			key := 2 * (i + 1)
			if got := m.Index(key); got != i {
				t.Errorf("Index(%d): got %d, want %d", key, got, i)
			}
		}

		// Test non-existing keys
		for _, key := range []int{1, 3, 5, 100} {
			if got := m.Index(key); got != -1 {
				t.Errorf("Index(%d): got %d, want -1", key, got)
			}
		}
	})

	t.Run("Range", func(t *testing.T) {
		m := NewMap[int, int](cmp.Compare)
		// Insert keys 2, 4, 6, 8, 10
		for i := 1; i <= 5; i++ {
			m.Set(2*i, i)
		}

		// Range [4, 8] contains 4, 6, 8 at indices 0, 1, 2
		r := m.From(4).To(8)
		for i, key := range []int{4, 6, 8} {
			if got := r.Index(key); got != i {
				t.Errorf("Range.Index(%d): got %d, want %d", key, got, i)
			}
		}

		// Keys in map but outside range
		for _, key := range []int{2, 10} {
			if got := r.Index(key); got != -1 {
				t.Errorf("Range.Index(%d): got %d, want -1", key, got)
			}
		}

		// Keys not in map
		for _, key := range []int{1, 5, 100} {
			if got := r.Index(key); got != -1 {
				t.Errorf("Range.Index(%d): got %d, want -1", key, got)
			}
		}
	})
}

func TestRangeLen(t *testing.T) {
	t.Run("Range", func(t *testing.T) {
		m := NewMap[int, int](cmp.Compare)
		// Empty range on empty map
		if got := m.From(1).To(10).Len(); got != 0 {
			t.Errorf("empty map: got %d, want 0", got)
		}

		// Insert keys 2, 4, 6, 8, 10
		for i := 1; i <= 5; i++ {
			m.Set(2*i, i)
		}

		// Full map
		if got := m.From(0).To(100).Len(); got != 5 {
			t.Errorf("full range: got %d, want 5", got)
		}

		// Range [4, 8] contains 4, 6, 8
		if got := m.From(4).To(8).Len(); got != 3 {
			t.Errorf("From(4).To(8): got %d, want 3", got)
		}

		// Range (4, 8) contains only 6
		if got := m.Above(4).Below(8).Len(); got != 1 {
			t.Errorf("Above(4).Below(8): got %d, want 1", got)
		}

		// Empty range
		if got := m.From(100).To(200).Len(); got != 0 {
			t.Errorf("empty range: got %d, want 0", got)
		}
	})
}

func TestRangeNth(t *testing.T) {
	t.Run("Range", func(t *testing.T) {
		m := NewMap[int, int](cmp.Compare)
		// Insert keys 2, 4, 6, 8, 10 with values 1, 2, 3, 4, 5
		for i := 1; i <= 5; i++ {
			m.Set(2*i, i)
		}

		// Range [4, 8] contains 4, 6, 8
		r := m.From(4).To(8)
		for i, wantKey := range []int{4, 6, 8} {
			k, v := r.Nth(i)
			if k != wantKey {
				t.Errorf("Nth(%d) key: got %d, want %d", i, k, wantKey)
			}
			if v != wantKey/2 {
				t.Errorf("Nth(%d) value: got %d, want %d", i, v, wantKey/2)
			}
		}
	})
}

func TestDelete(t *testing.T) {
	checkLen := func(m *Map[int, int], n int) {
		t.Helper()
		if m.Len() != n {
			t.Errorf("m.Len() = %d, want %d", m.Len(), n)
		}
	}

	for N := range 11 {
		m := NewMap[int, int](cmp.Compare)
		checkLen(m, 0)
		_, slice := permute(m, N)
		checkLen(m, N)
		wantLen := N
		for _, x := range rand.Perm(len(slice)) {
			prev, ok := m.Delete(x)
			if prev != slice[x] || ok != (slice[x] > 0) {
				t.Errorf("Delete(%v) = %d, %t, want %d, %t", x, prev, ok, slice[x], slice[x] > 0)
			}
			if ok {
				wantLen--
			}
			checkLen(m, wantLen)
			slice[x] = 0
			var have []int
			for k := range m.All() {
				have = append(have, k)
			}
			want := nonzeroIndexes(slice)
			if !slices.Equal(have, want) {
				t.Errorf("after Delete(%v), All() = %v, want %v", x, have, want)
			}
		}
	}
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

// nonzeroIndexes returns the indexes i such that s[i] != 0.
func nonzeroIndexes(s []int) []int {
	var r []int
	for k, v := range s {
		if v != 0 {
			r = append(r, k)
		}
	}
	return r
}

// in reports whether n is within the interval specified by lo and hi.
func in(n int, lo, hi bound[int]) bool {
	if lo.present && ((lo.inclusive && n < lo.key) || (!lo.inclusive && n <= lo.key)) {
		return false
	}
	if hi.present && ((hi.inclusive && n > hi.key) || (!hi.inclusive && n >= hi.key)) {
		return false
	}
	return true
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
			m.Span().To(10),
			"(-∞, 10]",
		},
		{
			"To2",
			m.Span().To(10).To(10),
			"(-∞, 10]",
		},
		{
			"To3",
			m.Span().Below(10).To(10),
			"(-∞, 10)",
		},
		{
			"To4",
			m.Span().To(9).To(10),
			"(-∞, 9]",
		},
		{
			"Below1",
			m.Span().Below(10),
			"(-∞, 10)",
		},
		{
			"Below2",
			m.Span().To(10).Below(10),
			"(-∞, 10)",
		},
		{
			"Below3",
			m.Span().Below(10).Below(10),
			"(-∞, 10)",
		},
		{
			"Below4",
			m.Span().To(9).Below(10),
			"(-∞, 9]",
		},
		{
			"From1",
			m.Span().From(10),
			"[10, ∞)",
		},
		{
			"From2",
			m.Span().From(10).From(10),
			"[10, ∞)",
		},
		{
			"From3",
			m.Span().Above(10).From(10),
			"(10, ∞)",
		},
		{
			"From4",
			m.Span().From(11).From(10),
			"[11, ∞)",
		},
		{
			"Above1",
			m.Span().Above(10),
			"(10, ∞)",
		},
		{
			"Above2",
			m.Span().From(10).Above(10),
			"(10, ∞)",
		},
		{
			"Above3",
			m.Span().Above(10).Above(10),
			"(10, ∞)",
		},
		{
			"Above4",
			m.Span().From(11).Above(10),
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
