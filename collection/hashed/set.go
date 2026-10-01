// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashed

// This file defines a [Set] based on [Map].

import (
	"fmt"
	"hash/maphash"
	"iter"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/exp/collection/internal/fmtsort"
	"golang.org/x/exp/collection/internal/iterfuncs"
)

// Set[E] is a hash-table based set of elements of type E,
// using the hash function and key-equivalence relation specified
// at construction.
//
// A Set must be created with [NewSet].
// Neither a nil pointer nor the zero value are valid sets.
//
// Set values must not be copied; use [Set.Clone] instead.
//
// Do not use Set with a comparable element type E and its usual ==
// equivalence relation; instead, use Go's built-in map type.
// (If you need set operations such as Union and Intersection,
// use the helpers in the [golang.org/x/exp/collection/set] package.)
//
// Set should not be used with elements that are (or contain)
// floating-point numbers whose value may be NaN, as it may take
// shortcuts based on algebraic identities that assume e == e for any
// element e. For example, Intersects(s, s) returns true for any
// non-empty set, without inspecting the elements.
//
// For all binary operations on sets, like the set algebra methods
// [Set.Union], [Set.Intersection] and so on, the two sets must
// use the same key-equivalence relation.
type Set[E any] struct {
	m Map[E, unit]
}

type unit = struct{}

// NewSet returns a new empty set that uses the specified hash
// function and key-equivalence relation.
func NewSet[E any](hasher maphash.Hasher[E]) *Set[E] {
	var s Set[E]
	s.m.init(hasher)
	return &s
}

// Len returns the number of set elements.
func (s *Set[E]) Len() int {
	return s.m.Len()
}

// All returns an iterator over the elements of the set in unspecified order.
//
// It is safe to modify the set during iteration. As with the built-in
// set, if an element is removed during iteration, it will not be
// yielded by the iterator. If an element is added during iteration,
// it may or may not be yielded by the iterator.
func (s *Set[E]) All() iter.Seq[E] {
	return s.m.Keys()
}

// insertAll calls Insert for each element in the sequence.
// It reports whether the size changed.
func (s *Set[E]) insertAll(seq iter.Seq[E]) bool {
	pre := s.m.len
	for e := range seq {
		s.m.Set(e, unit{})
	}
	return s.m.len > pre
}

// DeleteFunc deletes each element e in the set for which del returns true.
// It reports whether the set changed.
func (s *Set[E]) DeleteFunc(del func(E) bool) bool {
	pre := s.m.len
	for e := range s.All() {
		if del(e) {
			s.m.Delete(e)
		}
	}
	return s.m.len < pre
}

// Clone returns a new set with the same elements as s.
func (s *Set[E]) Clone() *Set[E] {
	var copy Set[E]
	copy.m.cloneFrom(&s.m)
	return &copy
}

// Delete removes the given element, if present.
// It reports whether the element was present.
func (s *Set[E]) Delete(e E) bool {
	_, ok := s.m.Delete(e)
	return ok
}

// Contains reports whether the set contains the specified element.
func (s *Set[E]) Contains(e E) bool {
	return s.m.Contains(e)
}

// Insert adds the specified element to the set
// (if no equal element is present),
// and reports whether the set grew.
func (s *Set[E]) Insert(e E) bool {
	pre := s.Len()
	s.m.Set(e, unit{})
	return s.Len() > pre
}

// Clear removes all entries from the set.
func (s *Set[E]) Clear() {
	s.m.Clear()
}

// containsAll reports whether set s contains all element of the sequence.
func (s *Set[E]) containsAll(seq iter.Seq[E]) bool {
	_ = s.m.len
	return iterfuncs.Every(seq, s.Contains)
}

// Equal reports whether s and other contain the same elements.
func (s *Set[E]) Equal(other *Set[E]) bool {
	return s == other ||
		s.Len() == other.Len() && s.containsAll(other.All())
}

// -- set algebra --

// Union returns a new set containing all elements of s and other.
func (s *Set[E]) Union(other *Set[E]) *Set[E] {
	res := s.Clone() // panics if s is nil
	if s != other {
		res.UnionWith(other)
	}
	return res
}

// UnionWith adds all elements of other to s.
func (s *Set[E]) UnionWith(other *Set[E]) {
	if s == other {
		return
	}
	s.insertAll(other.All())
}

// Intersection returns a new set containing elements that are in both s and other.
func (s *Set[E]) Intersection(other *Set[E]) *Set[E] {
	if s == other {
		return s.Clone() // s ∩ s = s
	}
	// Iterate over the smaller set for efficiency.
	if s.Len() > other.Len() {
		s, other = other, s
	}
	res := NewSet(s.m.hasher)
	for e := range s.All() {
		if other.Contains(e) {
			res.Insert(e)
		}
	}
	return res
}

// IntersectionWith removes any elements from s that are not in other.
func (s *Set[E]) IntersectionWith(other *Set[E]) {
	if s == other {
		return
	}
	for e := range s.All() {
		if !other.Contains(e) {
			s.Delete(e)
		}
	}
}

// Difference returns a new set containing elements that are in s but not in other.
func (s *Set[E]) Difference(other *Set[E]) *Set[E] {
	_ = other.m.len // panic if other is nil
	res := NewSet(s.m.hasher)
	if s != other {
		res.diff(s, other)
	}
	return res
}

// DifferenceWith removes any elements from s that are in other.
func (s *Set[E]) DifferenceWith(other *Set[E]) {
	_ = s.m.len     // panic if s is nil
	_ = other.m.len // panic if other is nil
	if s == other {
		s.Clear()
		return
	}
	for e := range other.All() {
		s.Delete(e)
	}
}

// SymmetricDifference returns a new set containing elements that are in either s or other, but not both.
func (s *Set[E]) SymmetricDifference(other *Set[E]) *Set[E] {
	_ = other.m.len // panic if other is nil
	res := NewSet(s.m.hasher)
	if s != other {
		res.diff(s, other)
		res.diff(other, s)
	}
	return res
}

// SymmetricDifferenceWith sets s to the symmetric difference of s and other.
func (s *Set[E]) SymmetricDifferenceWith(other *Set[E]) {
	_ = s.m.len     // panic if s is nil
	_ = other.m.len // panic if other is nil
	if s == other {
		s.Clear()
		return
	}
	for e := range other.All() {
		if s.Contains(e) {
			s.Delete(e)
		} else {
			s.Insert(e)
		}
	}
}

// diff adds to s the elements of from that are not in exclude.
func (s *Set[E]) diff(from, exclude *Set[E]) {
	for e := range from.All() {
		if !exclude.Contains(e) {
			s.Insert(e)
		}
	}
}

// Intersects reports whether s and other have any elements in common.
func (s *Set[E]) Intersects(other *Set[E]) bool {
	if s == other {
		return s.Len() > 0
	}
	if other.Len() > s.Len() {
		s, other = other, s
	}
	// Iterate over the smaller set (other).
	return iterfuncs.Some(other.All(), s.Contains)
}

// --

// String returns a string representation of the set's elements
// in an unspecified but deterministic order.
//
// Elements are printed as if by [fmt.Sprint].
func (s *Set[E]) String() string {
	var buf strings.Builder
	buf.WriteByte('{')
	// Sort the entries by their shallow representation,
	// using the same logic as fmt.Sprint(map).
	sorted := slices.Collect(s.All())
	slices.SortStableFunc(sorted, func(a, b E) int {
		return fmtsort.Compare(reflect.ValueOf(a), reflect.ValueOf(b))
	})

	for i, e := range sorted {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprint(&buf, e)
	}
	buf.WriteByte('}')
	return buf.String()
}
