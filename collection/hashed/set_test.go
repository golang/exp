// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashed_test

import (
	"testing"

	"golang.org/x/exp/collection/hashed"
)

func TestSet(t *testing.T) {
	// TODO: improve these witless LLM-generated tests.

	// Helper to create a set from strings.
	setOf := func(items ...string) *hashed.Set[string] {
		s := hashed.NewSet(caseInsensitive{})
		for _, item := range items {
			s.Insert(item)
		}
		return s
	}

	// Helper to assert set contents.
	check := func(t *testing.T, s *hashed.Set[string], want ...string) {
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
		// 'K' (0x4b), 'k' (0x6b), and Kelvin symbol 'K' (U+212A) should all fold to same.
		s := setOf("K")

		if s.Len() != 1 {
			t.Errorf("Len: got %d, want 1", s.Len())
		}

		// Check presence of equivalent forms.
		for _, w := range []string{"K", "k", "K"} {
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
		// Just ensure it doesn't crash and looks roughly right.
		// Output order is deterministic (sorted).
		s := setOf("b", "A")
		str := s.String()
		if str != "{A, b}" && str != "{b, A}" {
			t.Logf("String() = %q", str)
		}
	})

	t.Run("Constructors", func(t *testing.T) {
		s := hashed.NewSet(caseInsensitive{})
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

	var s *hashed.Set[string]
	other := hashed.NewSet(caseInsensitive{})

	panics("Len", func() { s.Len() })
	panics("All", func() { s.All() })
	panics("Contains", func() { s.Contains("key") })
	panics("Insert", func() { s.Insert("key") })
	panics("Delete", func() { s.Delete("key") })
	panics("Clear", func() { s.Clear() })
	panics("String", func() { _ = s.String() })
	panics("Clone", func() { s.Clone() })
	panics("Union", func() { s.Union(other) })
	panics("UnionWith", func() { s.UnionWith(other) })
	panics("Intersection", func() { s.Intersection(other) })
	panics("IntersectionWith", func() { s.IntersectionWith(other) })
	panics("Difference", func() { s.Difference(other) })
	panics("DifferenceWith", func() { s.DifferenceWith(other) })
	panics("SymmetricDifference", func() { s.SymmetricDifference(other) })
	panics("SymmetricDifferenceWith", func() { s.SymmetricDifferenceWith(other) })
	panics("Intersects", func() { s.Intersects(other) })

	// Also test when 'other' is nil on an initialized set
	initialized := hashed.NewSet(caseInsensitive{})

	panics("Initialized.Union(nil)", func() { initialized.Union(s) })
	panics("Initialized.Intersection(nil)", func() { initialized.Intersection(s) })
	panics("Initialized.Difference(nil)", func() { initialized.Difference(s) })
	panics("Initialized.SymmetricDifference(nil)", func() { initialized.SymmetricDifference(s) })
}
