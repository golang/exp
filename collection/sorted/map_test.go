// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sorted

import (
	"bytes"
	"cmp"
	"fmt"
	"math/rand/v2"
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
