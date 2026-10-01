// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hashed defines hash-based collections such as [Map].
package hashed

import (
	"fmt"
	"hash/maphash"
	"iter"
	"reflect"
	"slices"
	"strings"
	"sync"

	"golang.org/x/exp/collection/internal/fmtsort"
)

// Map[K, V] is a hash-table based mapping from keys K to values V,
// using the hash function and key-equivalence relation specified
// at construction.
//
// A Map must be created with [NewMap].
// Neither a nil pointer nor the zero value are valid maps.
//
// Map values must not be copied; use [Map.Clone] instead.
//
// Do not use Map with a comparable key type K and its usual ==
// equivalence relation; instead, use Go's built-in map type.
// (If you need set operations such as Union and Intersection,
// use the helpers in the collection/set package.)
type Map[K, V any] struct {
	seed   maphash.Seed
	table  map[uint64][]entry[K, V] // maps hash to entries of that hash
	len    int                      // number of map entries
	hasher maphash.Hasher[K]

	_ noCopy
}

// NewMap returns a new map that uses the specified hash
// function and key-equivalence relation.
func NewMap[K, V any](hasher maphash.Hasher[K]) *Map[K, V] {
	var m Map[K, V]
	m.init(hasher)
	return &m
}

type entry[K, V any] struct {
	used  bool
	key   K
	value V
}

func (m *Map[K, V]) init(hasher maphash.Hasher[K]) {
	m.seed = maphash.MakeSeed()
	m.table = make(map[uint64][]entry[K, V])
	m.hasher = hasher
}

// hash returns the hash of the specified key.
func (m *Map[K, V]) hash(key K) uint64 {
	// Because the m.hasher.Hash(h) call is dynamic,
	// escape analysis is conservative, causing Hash h to escape.
	// To amortize allocation, we recycle Hashes using a sync.Pool.
	//
	// (Even if we had expressed the hasher type as an additional
	// type parameter, it would not have been sufficient to cause
	// escape analysis to stack-allocate h.)
	h := hashPool.Get().(*maphash.Hash)
	h.SetSeed(m.seed)
	m.hasher.Hash(h, key)
	sum := h.Sum64()
	hashPool.Put(h)
	return sum
}

var hashPool = sync.Pool{
	New: func() any { return new(maphash.Hash) },
}

// Len returns the number of map entries.
func (m *Map[K, V]) Len() int {
	return m.len
}

// All returns an iterator over the key/value entries of the map in
// unspecified order.
//
// It is safe to modify the map during iteration. As with the built-in
// map, if a key is removed during iteration, its entry will not be
// yielded by the iterator. If a key is added during iteration, it may
// or may not be yielded by the iterator. The iterator yields the
// value most recently associated with a given key.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	_ = m.len
	return func(yield func(K, V) bool) {
		// Iterate by hash key to dynamically evaluate bucket slice length.
		// This avoids holding a stale slice header if a concurrent or
		// nested Set reallocates the bucket, matching built-in map semantics.
		for hash := range m.table {
			for i := 0; i < len(m.table[hash]); i++ {
				e := &m.table[hash][i]
				if e.used && !yield(e.key, e.value) {
					return
				}
			}
		}
	}
}

// Clone returns a new non-nil map with the same entries as m.
func (m *Map[K, V]) Clone() *Map[K, V] {
	var copy Map[K, V]
	copy.cloneFrom(m)
	return &copy
}

func (m *Map[K, V]) cloneFrom(src *Map[K, V]) {
	*m = Map[K, V]{
		hasher: src.hasher,
		seed:   src.seed,
		table:  make(map[uint64][]entry[K, V]),
	}
	m.setAll(src.All())
}

// Keys returns an iterator over the keys of the map in random order.
//
// It is safe to modify the map during iteration, with the same
// semantics as the [Map.All] iterator.
func (m *Map[K, V]) Keys() iter.Seq[K] {
	_ = m.len
	return func(yield func(K) bool) {
		// Iterate by hash key to dynamically evaluate bucket slice length.
		// This safely handles additions/reallocations during iteration.
		for hash := range m.table {
			for i := 0; i < len(m.table[hash]); i++ {
				e := &m.table[hash][i]
				if e.used && !yield(e.key) {
					return
				}
			}
		}
	}
}

// Values returns an iterator over the values of the map in random order.
//
// It is safe to modify the map during iteration, with the same
// semantics as the [Map.All] iterator.
func (m *Map[K, V]) Values() iter.Seq[V] {
	_ = m.len
	return func(yield func(V) bool) {
		// Iterate by hash key to dynamically evaluate bucket slice length.
		// This safely handles additions/reallocations during iteration.
		for hash := range m.table {
			for i := 0; i < len(m.table[hash]); i++ {
				e := &m.table[hash][i]
				if e.used && !yield(e.value) {
					return
				}
			}
		}
	}
}

// Delete removes the entry with the given key, if present.
// It returns the previous value for key if present, or the zero value if not.
// The boolean (deleted) indicates whether the operation deleted a map entry.
func (m *Map[K, V]) Delete(key K) (prev V, deleted bool) {
	bucket := m.table[m.hash(key)]
	for i, e := range bucket {
		if e.used && m.hasher.Equal(key, e.key) {
			// We can't compact the bucket as it
			// would disturb iterators.
			prev := e.value
			bucket[i] = entry[K, V]{}
			m.len--
			return prev, true
		}
	}
	return *new(V), false
}

// DeleteFunc removes each map entry for which del returns true.
// It reports whether the map changed.
func (m *Map[K, V]) DeleteFunc(del func(K, V) bool) bool {
	changed := false
	for k, v := range m.All() {
		if del(k, v) {
			if _, ok := m.Delete(k); ok {
				changed = true
			}
		}
	}
	return changed
}

// Lookup reports whether the map contains the specified key, and returns
// the corresponding value if found, or the zero value if not.
func (m *Map[K, V]) Lookup(key K) (V, bool) {
	for _, e := range m.table[m.hash(key)] {
		if e.used && m.hasher.Equal(key, e.key) {
			return e.value, true
		}
	}
	return *new(V), false
}

// Get returns the value corresponding the given key,
// or the zero value if the entry is not present.
func (m *Map[K, V]) Get(key K) V {
	v, _ := m.Lookup(key)
	return v
}

// Contains reports whether the map contains an entry with the given key.
func (m *Map[K, V]) Contains(key K) bool {
	_, ok := m.Lookup(key)
	return ok
}

// Set updates the map entry for key to value.
// It returns the previous value for key if present, or the zero value if not.
// The boolean (present) indicates whether the operation updated an existing entry
// for that key.
func (m *Map[K, V]) Set(key K, value V) (prev V, present bool) {
	hash := m.hash(key)
	bucket := m.table[hash]
	var hole *entry[K, V]
	for i := range bucket {
		e := &bucket[i]
		if e.used {
			if m.hasher.Equal(key, e.key) {
				prev = e.value
				e.key = key
				e.value = value
				return prev, true
			}
		} else if hole == nil {
			hole = e
		}
	}
	if hole != nil {
		*hole = entry[K, V]{true, key, value} // overwrite deleted entry
	} else {
		m.table[hash] = append(bucket, entry[K, V]{true, key, value})
	}

	m.len++
	return prev, false
}

// setAll calls Set(k, v) for each key/value pair in the sequence.
// It reports whether the number of map entries increased.
func (m *Map[K, V]) setAll(seq iter.Seq2[K, V]) bool {
	pre := m.len
	for k, v := range seq {
		m.Set(k, v)
	}
	return m.len > pre
}

// Clear removes all entries from the map.
func (m *Map[K, V]) Clear() {
	clear(m.table)
	m.len = 0
}

// String returns a string representation of the map's entries
// in an unspecified but deterministic order.
//
// Keys and values are printed as if by [fmt.Sprint].
func (m *Map[K, V]) String() string {
	var buf strings.Builder
	buf.WriteByte('{')
	// Sort the entries by their shallow representation,
	// using the same logic as fmt.Sprint(map).
	sorted := make([]*entry[K, V], 0, m.Len())
	for _, bucket := range m.table {
		for i, e := range bucket {
			if e.used {
				sorted = append(sorted, &bucket[i])
			}
		}
	}
	slices.SortStableFunc(sorted, func(a, b *entry[K, V]) int {
		return fmtsort.Compare(reflect.ValueOf(a.key), reflect.ValueOf(b.key))
	})

	for i, e := range sorted {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(&buf, "%v: %v", e.key, e.value)
	}
	buf.WriteByte('}')
	return buf.String()
}

// noCopy may be added to structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
