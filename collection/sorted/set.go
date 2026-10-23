// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sorted

import (
	"fmt"
	"iter"
	"strings"

	"golang.org/x/exp/collection/internal/iterfuncs"
)

// A Set is a set of elements of type T, ordered according to
// a comparison function specified at construction.
//
// A Set must be created with [NewSet].
// Neither a nil pointer nor the zero value are valid sets.
//
// Set values must not be copied; use [Set.Clone] instead.
type Set[T any] struct {
	m Map[T, struct{}]
}

// NewSet returns a new [Set] ordered according to cmp.
func NewSet[T any](cmp func(T, T) int) *Set[T] {
	return &Set[T]{m: Map[T, struct{}]{cmp: cmp}}
}

// Len returns the number of elements in s.
func (s *Set[T]) Len() int {
	return s.m.Len()
}

// Span returns an unbounded [SetSpan] for the set.
func (s *Set[T]) Span() SetSpan[T] { return SetSpan[T]{ms: s.m.Span()} }

// All returns an iterator over the elements of s from smallest to largest.
// See [Map.All] for the guarantee provided if s is modified during the iteration.
func (s *Set[T]) All() iter.Seq[T] {
	return s.m.Keys()
}

// Backward returns an iterator over the elements of s from largest to smallest.
// See [Map.All] for the guarantee provided if s is modified during the iteration.
func (s *Set[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.m.Backward()(func(x T, _ struct{}) bool { return yield(x) })
	}
}

// Min returns the minimum element in s.
// The second return value is false if the set is empty, otherwise true.
func (s *Set[T]) Min() (T, bool) {
	k, _, ok := s.m.Min()
	return k, ok
}

// Max returns the maximum element in s.
// The second return value is false if the set is empty, otherwise true.
func (s *Set[T]) Max() (T, bool) {
	k, _, ok := s.m.Max()
	return k, ok
}

// Nth returns the element at index i.
// It panics if i < 0 or i >= s.Len().
// See [Set.Index] for the inverse operation.
func (s *Set[T]) Nth(i int) T {
	k, _ := s.m.Nth(i)
	return k
}

// Index returns the index of e in s, or -1 if e is not present.
// The index of an element is its position in the sorted sequence of elements.
// For example, the smallest element has index 0,
// and the largest has index s.Len()-1.
// See [Set.Nth] for the inverse operation.
func (s *Set[T]) Index(e T) int {
	return s.m.Index(e)
}

// Insert adds the specified element to the set
// (if no equivalent element is present),
// and reports whether the set grew.
func (s *Set[T]) Insert(e T) bool {
	_, present := s.m.Set(e, struct{}{})
	return !present
}

// Delete removes the given element, if present.
// It reports whether the element was present.
func (s *Set[T]) Delete(e T) bool {
	_, ok := s.m.Delete(e)
	return ok
}

// DeleteFunc deletes each element e in the set for which del returns true.
// It reports whether the set changed.
func (s *Set[T]) DeleteFunc(del func(T) bool) bool {
	return s.m.DeleteFunc(func(e T, _ struct{}) bool {
		return del(e)
	})
}

// Contains reports whether the set contains the specified element.
func (s *Set[T]) Contains(e T) bool {
	return s.m.Contains(e)
}

// Clear removes all elements from the set.
func (s *Set[T]) Clear() {
	s.m.Clear()
}

// Clone returns a shallow copy of s.
func (s *Set[T]) Clone() *Set[T] {
	return &Set[T]{m: *s.m.Clone()}
}

// Equal reports whether s and other contain the same elements.
func (s *Set[T]) Equal(other *Set[T]) bool {
	return s == other ||
		s.Len() == other.Len() && iterfuncs.Every(other.All(), s.Contains)
}

// Union returns a new set containing all elements of s and other.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	res := s.Clone()
	if s != other {
		res.UnionWith(other)
	}
	return res
}

// UnionWith adds all elements of other to s.
func (s *Set[T]) UnionWith(other *Set[T]) {
	if s == other {
		return
	}
	for t := range other.All() {
		s.Insert(t)
	}
}

// Intersection returns a new set containing elements that are in both s and other.
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	if s == other {
		return s.Clone()
	}
	res := NewSet(s.m.cmp)
	if s.Len() > other.Len() {
		s, other = other, s
	}
	for e := range s.All() {
		if other.Contains(e) {
			res.Insert(e)
		}
	}
	return res
}

// IntersectionWith removes any elements from s that are not in other.
func (s *Set[T]) IntersectionWith(other *Set[T]) {
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
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	res := NewSet(s.m.cmp)
	if s != other {
		res.diff(s, other)
	}
	return res
}

// DifferenceWith removes any elements from s that are in other.
func (s *Set[T]) DifferenceWith(other *Set[T]) {
	if s == other {
		s.Clear()
		return
	}
	for e := range other.All() {
		s.Delete(e)
	}
}

// SymmetricDifference returns a new set containing elements that are in either s or other, but not both.
func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T] {
	res := NewSet(s.m.cmp)
	if s != other {
		res.diff(s, other)
		res.diff(other, s)
	}
	return res
}

// SymmetricDifferenceWith sets s to the symmetric difference of s and other.
func (s *Set[T]) SymmetricDifferenceWith(other *Set[T]) {
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
func (s *Set[T]) diff(from, exclude *Set[T]) {
	for e := range from.All() {
		if !exclude.Contains(e) {
			s.Insert(e)
		}
	}
}

// Intersects reports whether s and other have any elements in common.
func (s *Set[T]) Intersects(other *Set[T]) bool {
	if s == other {
		return s.Len() > 0
	}
	if other.Len() > s.Len() {
		s, other = other, s
	}
	for e := range other.All() {
		if s.Contains(e) {
			return true
		}
	}
	return false
}

// String returns a string representation of s in the form "{e1, e2}".
func (s *Set[T]) String() string {
	return setString(s.All())
}

func setString[T any](seq iter.Seq[T]) string {
	var b strings.Builder
	b.WriteByte('{')
	first := true
	for e := range seq {
		if !first {
			b.WriteString(", ")
		}
		first = false
		fmt.Fprint(&b, e)
	}
	b.WriteByte('}')
	return b.String()
}

// From returns a [SetSpan] with lower bound lo, inclusive, and no upper bound.
func (s *Set[T]) From(lo T) SetSpan[T] { return SetSpan[T]{s.m.From(lo)} }

// Above returns a [SetSpan] with lower bound lo, exclusive, and no upper bound.
func (s *Set[T]) Above(lo T) SetSpan[T] { return SetSpan[T]{s.m.Above(lo)} }

// To returns a [SetSpan] with upper bound hi, inclusive, and no lower bound.
func (s *Set[T]) To(hi T) SetSpan[T] { return SetSpan[T]{s.m.To(hi)} }

// Below returns a [SetSpan] with upper bound hi, exclusive, and no lower bound.
func (s *Set[T]) Below(hi T) SetSpan[T] { return SetSpan[T]{s.m.Below(hi)} }

// A SetSpan is a subsequence of elements in a [Set].
// SetSpans created from the same set share the set, as slices
// created from the same array share the array.
// Modifications made via one SetSpan change the underlying set,
// so are visible to all SetSpans that share the set.
type SetSpan[T any] struct {
	ms MapSpan[T, struct{}]
}

// To returns a [SetSpan] with upper bound hi, inclusive and the same lower bound as s,
// if hi is within s's current upper bound. Otherwise, it returns a SetSpan with the same
// bounds as s.
func (s SetSpan[T]) To(hi T) SetSpan[T] {
	return SetSpan[T]{s.ms.To(hi)}
}

// Below returns a [SetSpan] with upper bound hi, exclusive and the same lower bound as s,
// if hi is within s's current upper bound. Otherwise, it returns a SetSpan with the same
// bounds as s.
func (s SetSpan[T]) Below(hi T) SetSpan[T] {
	return SetSpan[T]{s.ms.Below(hi)}
}

// From returns a [SetSpan] with lower bound lo, inclusive and the same upper bound as s,
// if lo is within s's current lower bound. Otherwise, it returns a SetSpan with the same
// bounds as s.
func (s SetSpan[T]) From(lo T) SetSpan[T] {
	return SetSpan[T]{s.ms.From(lo)}
}

// Above returns a [SetSpan] with lower bound lo, exclusive and the same upper bound as s,
// if lo is within s's current lower bound. Otherwise, it returns a SetSpan with the same
// bounds as s.
func (s SetSpan[T]) Above(lo T) SetSpan[T] {
	return SetSpan[T]{s.ms.Above(lo)}
}

// Min returns the minimum element in s.
// The second return value is false if s is empty, otherwise true.
func (s SetSpan[T]) Min() (T, bool) {
	k, _, ok := s.ms.Min()
	return k, ok
}

// Max returns the maximum element in s.
// The second return value is false if s is empty, otherwise true.
func (s SetSpan[T]) Max() (T, bool) {
	k, _, ok := s.ms.Max()
	return k, ok
}

// Index returns the index of e within s, or -1 if e is not present or not in bounds.
func (s SetSpan[T]) Index(e T) int {
	return s.ms.Index(e)
}

// Len returns the number of elements in s.
func (s SetSpan[T]) Len() int {
	return s.ms.Len()
}

// String returns a string representation of s.
func (s SetSpan[T]) String() string {
	return setString(s.All())
}

// Nth returns the element at index i within s.
// It panics if i < 0 or i >= s.Len().
func (s SetSpan[T]) Nth(i int) T {
	k, _ := s.ms.Nth(i)
	return k
}

// Clone returns a new [Set] containing only the elements in s.
func (s SetSpan[T]) Clone() *Set[T] {
	return &Set[T]{m: *s.ms.Clone()}
}

// Clear deletes all the elements in s from s's underlying set.
func (s SetSpan[T]) Clear() {
	s.ms.Clear()
}

// All returns an iterator over s from smallest to largest element.
// See [Map.All] for the guarantee provided if the underlying set is modified during the iteration.
func (s SetSpan[T]) All() iter.Seq[T] {
	return s.ms.Keys()
}

// Backward returns an iterator over s from largest to smallest element.
// See [Map.All] for the guarantee provided if the underlying set is modified during the iteration.
func (s SetSpan[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.ms.Backward()(func(x T, _ struct{}) bool { return yield(x) })
	}
}
