// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sorted_test

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"golang.org/x/exp/collection/sorted"
)

// caseInsensitive compares strings, ignoring letter case.
func caseInsensitive(x, y string) int {
	return cmp.Compare(strings.ToLower(x), strings.ToLower(y))
}

func TestSet(t *testing.T) {
	// Adapted from hashed/set_test.go.

	// Helper to create a set from strings.
	setOf := func(items ...string) *sorted.Set[string] {
		s := sorted.NewSet(caseInsensitive)
		for _, item := range items {
			s.Insert(item)
		}
		return s
	}

	// Helper to assert set contents.
	check := func(t *testing.T, s *sorted.Set[string], want ...string) {
		t.Helper()
		if got := s.Len(); got != len(want) {
			t.Errorf("Len: got %d, want %d", got, len(want))
		}
		for _, w := range want {
			if !s.Contains(w) {
				t.Errorf("Contains(%q): got false, want true", w)
			}
		}
		// Also check that All() yields the expected items (ignoring order/case variants stored).
		// Since we use case-insensitive equality, we can't strictly compare strings returned by All()
		// against 'want' unless we normalize them. But here we just want to ensure size and presence.
		count := 0
		for range s.All() {
			count++
		}
		if count != len(want) {
			t.Errorf("All() yielded %d items, want %d", count, len(want))
		}
	}

	t.Run("Basics", func(t *testing.T) {
		s := setOf("Red", "GREEN")
		check(t, s, "RED", "green") // Case insensitive check

		if s.Insert("red") {
			t.Error("Insert(red) reported growth, want false (already present)")
		}

		if !s.Delete("Green") {
			t.Error("Delete(Green) reported missing, want present")
		}
		check(t, s, "RED")

		s.Clear()
		check(t, s)
	})

	t.Run("UnicodeFolding", func(t *testing.T) {
		// 'K' (0x4b), 'k' (0x6b), and Kelvin symbol 'K' (U+212A) should all fold to same.
		s := setOf("K")

		if s.Len() != 1 {
			t.Errorf("Len: got %d, want 1", s.Len())
		}

		// Check presence of equivalent forms.
		for _, w := range []string{"K", "k", "K"} {
			if !s.Contains(w) {
				t.Errorf("Contains(%q): got false, want true", w)
			}
		}

		if s.Insert("\u212a") {
			t.Error("Insert(Kelvin) reported growth, want false (already present as K)")
		}
	})

	t.Run("SetAlgebra", func(t *testing.T) {
		s1 := setOf("a", "b")
		s2 := setOf("B", "c")

		// Union: {a, b, c}
		check(t, s1.Union(s2), "A", "B", "C")

		// Intersection: {b}
		check(t, s1.Intersection(s2), "b")

		// Difference (s1 - s2): {a}
		check(t, s1.Difference(s2), "a")

		// SymmetricDifference: {a, c}
		check(t, s1.SymmetricDifference(s2), "a", "c")
	})

	t.Run("MutationVariants", func(t *testing.T) {
		// UnionWith
		s := setOf("a")
		s.UnionWith(setOf("B"))
		check(t, s, "A", "b")

		// IntersectionWith
		s = setOf("a", "b")
		s.IntersectionWith(setOf("B", "c"))
		check(t, s, "b")

		// DifferenceWith
		s = setOf("a", "b", "c")
		s.DifferenceWith(setOf("b", "d"))
		check(t, s, "a", "c")

		// SymmetricDifferenceWith
		s = setOf("a", "b")
		s.SymmetricDifferenceWith(setOf("b", "c"))
		check(t, s, "a", "c")
	})

	t.Run("Clone", func(t *testing.T) {
		s1 := setOf("a", "b")
		s2 := s1.Clone()
		check(t, s2, "a", "b")

		s2.Insert("c")
		check(t, s1, "a", "b") // s1 unchanged
		check(t, s2, "a", "b", "c")
	})

	t.Run("Intersects", func(t *testing.T) {
		s1 := setOf("a", "b")
		s2 := setOf("B", "c")
		s3 := setOf("d")

		if !s1.Intersects(s2) {
			t.Error("Intersects(s1, s2) got false, want true")
		}
		if s1.Intersects(s3) {
			t.Error("Intersects(s1, s3) got true, want false")
		}
	})

	t.Run("String", func(t *testing.T) {
		// Output order is deterministic (sorted).
		s := setOf("b", "A")
		if got, want := s.String(), "{A, b}"; got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("Constructors", func(t *testing.T) {
		s := sorted.NewSet(caseInsensitive)
		if s.Len() != 0 {
			t.Error("new set should have Len 0")
		}
		if s.Contains("foo") {
			t.Error("new set should not contain anything")
		}
	})
}

func TestNilSetPanics(t *testing.T) {
	panics := func(name string, f func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("want panic for nil Set.%s", name)
			}
		}()
		f()
	}

	var s *sorted.Set[string]
	other := sorted.NewSet(caseInsensitive)

	panics("Len", func() { s.Len() })
	panics("All", func() { s.All() })
	panics("Contains", func() { s.Contains("key") })
	panics("Insert", func() { s.Insert("key") })
	panics("Delete", func() { s.Delete("key") })
	panics("Clear", func() { s.Clear() })
	panics("String", func() { _ = s.String() })
	panics("Clone", func() { s.Clone() })
	panics("Union", func() { s.Union(other) })
	panics("Intersection", func() { s.Intersection(other) })
	panics("IntersectionWith", func() { s.IntersectionWith(other) })
	panics("Difference", func() { s.Difference(other) })
	panics("SymmetricDifference", func() { s.SymmetricDifference(other) })
	panics("Intersects", func() { s.Intersects(other) })
	// UnionWith, DifferenceWith and SymmetricDifferenceWith don't panic on
	// a nil receiver when other is empty, because they only iterate over other.

	// Also test when 'other' is nil on an initialized set
	initialized := sorted.NewSet(caseInsensitive)

	panics("Initialized.Union(nil)", func() { initialized.Union(s) })
	panics("Initialized.Intersection(nil)", func() { initialized.Intersection(s) })
	panics("Initialized.SymmetricDifference(nil)", func() { initialized.SymmetricDifference(s) })
	// Initialized.Difference(nil) doesn't panic when initialized is empty,
	// because it never calls a method on other.
}

// TestSetAlgebra checks the set operations against a reference implementation
// on randomly generated sets of ints.
func TestSetAlgebra(t *testing.T) {
	const (
		maxElem = 20 // elements are in [0, maxElem)
		maxLen  = 12
		trials  = 200
	)

	// A reference set is a sorted slice of distinct ints.
	type ref = []int

	refOf := func(s *sorted.Set[int]) ref {
		return slices.Collect(s.All())
	}
	randomSet := func() (*sorted.Set[int], ref) {
		s := sorted.NewSet(cmp.Compare[int])
		for range rand.IntN(maxLen + 1) {
			s.Insert(rand.IntN(maxElem))
		}
		return s, refOf(s)
	}
	setOf := func(r ref) *sorted.Set[int] {
		s := sorted.NewSet(cmp.Compare[int])
		for _, e := range r {
			s.Insert(e)
		}
		return s
	}
	filter := func(r ref, keep func(int) bool) ref {
		var res ref
		for _, e := range r {
			if keep(e) {
				res = append(res, e)
			}
		}
		return res
	}
	// Reference set operations. Results are sorted, since inputs are.
	union := func(a, b ref) ref {
		res := slices.Concat(a, b)
		slices.Sort(res)
		return slices.Compact(res)
	}
	intersection := func(a, b ref) ref {
		return filter(a, func(e int) bool { return slices.Contains(b, e) })
	}
	difference := func(a, b ref) ref {
		return filter(a, func(e int) bool { return !slices.Contains(b, e) })
	}
	symmetricDifference := func(a, b ref) ref {
		return union(difference(a, b), difference(b, a))
	}

	checkEqual := func(t *testing.T, op string, a, b ref, got *sorted.Set[int], want ref) {
		t.Helper()
		if g := refOf(got); !slices.Equal(g, want) {
			t.Errorf("%v %s %v = %v, want %v", a, op, b, g, want)
		}
		if got.Len() != len(want) {
			t.Errorf("%v %s %v: Len() = %d, want %d", a, op, b, got.Len(), len(want))
		}
	}

	// Each operation comes in a functional form, which must not modify its
	// arguments, and a mutating form, which modifies its receiver.
	type op struct {
		name string
		ref  func(a, b ref) ref
		fn   func(a, b *sorted.Set[int]) *sorted.Set[int]
		with func(a, b *sorted.Set[int])
	}
	ops := []op{
		{"∪", union, (*sorted.Set[int]).Union, (*sorted.Set[int]).UnionWith},
		{"∩", intersection, (*sorted.Set[int]).Intersection, (*sorted.Set[int]).IntersectionWith},
		{"−", difference, (*sorted.Set[int]).Difference, (*sorted.Set[int]).DifferenceWith},
		{"△", symmetricDifference, (*sorted.Set[int]).SymmetricDifference, (*sorted.Set[int]).SymmetricDifferenceWith},
	}

	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			for range trials {
				a, ar := randomSet()
				b, br := randomSet()
				want := op.ref(ar, br)

				// Functional form: result is correct and arguments are unchanged.
				checkEqual(t, op.name, ar, br, op.fn(a, b), want)
				if g := refOf(a); !slices.Equal(g, ar) {
					t.Errorf("%v %s %v modified receiver to %v", ar, op.name, br, g)
				}
				if g := refOf(b); !slices.Equal(g, br) {
					t.Errorf("%v %s %v modified argument to %v", ar, op.name, br, g)
				}

				// Mutating form: receiver becomes the result, argument is unchanged.
				op.with(a, b)
				checkEqual(t, op.name+"=", ar, br, a, want)
				if g := refOf(b); !slices.Equal(g, br) {
					t.Errorf("%v %s= %v modified argument to %v", ar, op.name, br, g)
				}

				// Aliasing: both operands are the same set.
				c := setOf(ar)
				checkEqual(t, op.name, ar, ar, op.fn(c, c), op.ref(ar, ar))
				op.with(c, c)
				checkEqual(t, op.name+"=", ar, ar, c, op.ref(ar, ar))
			}
		})
	}

	t.Run("Equal", func(t *testing.T) {
		for range trials {
			a, ar := randomSet()
			b, br := randomSet()
			if got, want := a.Equal(b), slices.Equal(ar, br); got != want {
				t.Errorf("%v.Equal(%v) = %t, want %t", ar, br, got, want)
			}
			if !a.Equal(a) {
				t.Errorf("%v.Equal(itself) = false", ar)
			}
			if !a.Equal(setOf(ar)) {
				t.Errorf("%v.Equal(copy) = false", ar)
			}
		}
	})

	t.Run("Intersects", func(t *testing.T) {
		for range trials {
			a, ar := randomSet()
			b, br := randomSet()
			want := len(intersection(ar, br)) > 0
			if got := a.Intersects(b); got != want {
				t.Errorf("%v.Intersects(%v) = %t, want %t", ar, br, got, want)
			}
			if got := b.Intersects(a); got != want {
				t.Errorf("%v.Intersects(%v) = %t, want %t", br, ar, got, want)
			}
			if got, want := a.Intersects(a), len(ar) > 0; got != want {
				t.Errorf("%v.Intersects(itself) = %t, want %t", ar, got, want)
			}
		}
	})
}
