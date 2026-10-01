// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sorted defines collections based on key order
// such as [Map].
package sorted

import (
	"iter"
)

// A Map is a mapping from keys to values, where the keys are ordered according to
// a comparison function specified at construction.
//
// A Map must be created with [NewMap].
// Neither a nil pointer nor the zero value are valid maps.
//
// Map values must not be copied; use [Map.Clone] instead.
type Map[K, V any] struct {
	cmp func(K, K) int
}

// NewMap returns a new [Map] ordered according to cmp.
func NewMap[K, V any](cmp func(K, K) int) *Map[K, V] {
	return nil
}

// Contains reports whether the map contains an entry with the given key.
func (m *Map[K, V]) Contains(key K) bool {
	return false
}

// Lookup returns the value of m[key] and reports whether it exists.
func (m *Map[K, V]) Lookup(key K) (V, bool) {
	return *new(V), false
}

// Get returns the value of m[key], or the zero value if key is not present.
func (m *Map[K, V]) Get(key K) V {
	return *new(V)
}

// Set updates the map entry for key to value.
// It returns the previous value for key if present, or the zero value if not.
// The boolean (present) indicates whether the operation updated an existing entry
// for that key.
func (m *Map[K, V]) Set(key K, val V) (old V, added bool) {
	return
}

// Delete removes the entry with the given key, if present.
// It returns the previous value for key if present, or the zero value if not.
// The boolean (deleted) indicates whether the operation deleted a map entry.
func (m *Map[K, V]) Delete(key K) (prev V, deleted bool) {
	return
}

// DeleteFunc removes all the entries for which f returns true.
// It reports whether the map changed.
func (m *Map[K, V]) DeleteFunc(f func(K, V) bool) bool {
	return false
}

// Min returns the minimum key in m along with its value.
// The third return value is false if the map is empty, otherwise true.
func (m *Map[K, V]) Min() (K, V, bool) {
	return *new(K), *new(V), false
}

// Max returns the maximum key in m along with its value.
// The third return value is false if the map is empty, otherwise true.
func (m *Map[K, V]) Max() (K, V, bool) {
	return *new(K), *new(V), false
}

// All returns an iterator over the map m from smallest to largest key.
//
// It is safe to modify the map during iteration.
// All guarantees that when a key is yielded, it is the successor of the key
// that was previously yielded (or the minimum key in the map, if it is the first
// key).
//
// For example, if the map contains keys 10 and 20, the iterator has yielded 10,
// and then 15 is inserted, then the next yielded key will be 15.
//
// Another example: if the map contains keys 10, 20 and 30, the iterator has yielded
// 10, and then 20 is deleted, then the next yielded key will be 30.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return nil
}

// Keys returns an iterator over the keys in m from smallest to largest.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (m *Map[K, V]) Keys() iter.Seq[K] {
	return nil
}

// Values returns an iterator over the values in m from smallest to largest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (m *Map[K, V]) Values() iter.Seq[V] {
	return nil
}

// Backward returns an iterator over the map m from largest to smallest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (m *Map[K, V]) Backward() iter.Seq2[K, V] {
	return nil
}

// Clear removes all entries from the map.
func (m *Map[K, V]) Clear() {
}

// Clone returns a shallow copy of m.
func (m *Map[K, V]) Clone() *Map[K, V] {
	return nil
}

// Len returns the number of keys in m.
func (m *Map[K, V]) Len() int { return 0 }

// String returns a string representation of m in the form "{k1: v1, k2: v2}".
func (m *Map[K, V]) String() string {
	return ""
}

// Nth returns the key and value at index i.
// It panics if i < 0 or i >= m.Len().
// See [Map.Index] for the inverse operation.
func (m *Map[K, V]) Nth(i int) (K, V) { return *new(K), *new(V) }

// Index returns the index of key in m, or -1 if key is not present.
// The index of a key is its position in the sequence of keys.
// For example, the smallest key has index 0,
// and the largest has index m.Len()-1.
// See [Map.Nth] for the inverse operation.
func (m *Map[K, V]) Index(key K) int {
	return 0
}

// From returns a [MapSpan] with lower bound lo, inclusive, and no upper bound.
func (m *Map[K, V]) From(lo K) MapSpan[K, V] { return MapSpan[K, V]{} }

// Above returns a [MapSpan] with lower bound lo, exclusive, and no upper bound.
func (m *Map[K, V]) Above(lo K) MapSpan[K, V] { return MapSpan[K, V]{} }

// To returns a [MapSpan] with upper bound hi, inclusive, and no lower bound.
func (m *Map[K, V]) To(hi K) MapSpan[K, V] { return MapSpan[K, V]{} }

// Below returns a [MapSpan] with upper bound hi, exclusive, and no lower bound.
func (m *Map[K, V]) Below(hi K) MapSpan[K, V] { return MapSpan[K, V]{} }

// A MapSpan is a subsequence of keys in a [Map].
// MapSpans created from the same map share the map, as slices
// created from the same array share the array.
// Modifications made via one MapSpan change the underlying map,
// so are visible to all MapSpans that share the map.
type MapSpan[K, V any] struct {
}

// To returns a [MapSpan] with upper bound hi, inclusive and the same lower bound as s,
// if hi is within s's current upper bound. Otherwise, it returns a MapSpan with the same
// bounds as s.
func (s MapSpan[K, V]) To(hi K) MapSpan[K, V] {
	return s
}

// Below returns a [MapSpan] with upper bound hi, exclusive and the same lower bound as s,
// if hi is within s's current upper bound. Otherwise, it returns a MapSpan with the same
// bounds as s.
func (s MapSpan[K, V]) Below(hi K) MapSpan[K, V] {
	return s
}

// From returns a [MapSpan] with lower bound lo, inclusive and the same upper bound as s,
// if lo is within s's current lower bound. Otherwise, it returns a MapSpan with the same
// bounds as s.
func (s MapSpan[K, V]) From(lo K) MapSpan[K, V] {
	return s
}

// Above returns a [MapSpan] with lower bound lo, exclusive and the same upper bound as s,
// if lo is within s's current lower bound. Otherwise, it returns a MapSpan with the same
// bounds as s.
func (s MapSpan[K, V]) Above(lo K) MapSpan[K, V] {
	return s
}

// Min returns the minimum key in s along with its value.
// The third return value is false if s is empty, otherwise true.
func (s MapSpan[K, V]) Min() (K, V, bool) { return *new(K), *new(V), false }

// Max returns the maximum key in s along with its value.
// The third return value is false if s is empty, otherwise true.
func (s MapSpan[K, V]) Max() (K, V, bool) { return *new(K), *new(V), false }

// Index returns the index of key within s, or -1 if key is not present or not in bounds.
func (s MapSpan[K, V]) Index(key K) int { return 0 }

// Len returns the number of keys in s.
func (s MapSpan[K, V]) Len() int { return 0 }

// String returns a string representation of s.
func (s MapSpan[K, V]) String() string {
	return ""
}

// Nth returns the key and value at index i within s.
// It panics if i < 0 or i >= s.Len().
func (s MapSpan[K, V]) Nth(i int) (K, V) { return *new(K), *new(V) }

// Clone returns a new [Map] containing only the keys in s.
func (s MapSpan[K, V]) Clone() *Map[K, V] { return nil }

// Clear deletes all the entries in s from s's underlying map.
func (s MapSpan[K, V]) Clear() {
}

// All returns an iterator over s from smallest to largest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (s MapSpan[K, V]) All() iter.Seq2[K, V] { return nil }

// Keys returns an iterator over the keys in s from smallest to largest.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (s MapSpan[K, V]) Keys() iter.Seq[K] { return nil }

// Values returns an iterator over the values in s from smallest to largest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (s MapSpan[K, V]) Values() iter.Seq[V] { return nil }

// Backward returns an iterator over s from largest to smallest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (s MapSpan[K, V]) Backward() iter.Seq2[K, V] { return nil }
