// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collection_test

import (
	"iter"
	"testing"

	"golang.org/x/exp/collection/hashed"
	"golang.org/x/exp/collection/set"
)

// The following interfaces define the abstract data types for
// collections in Go. They are expressed using F-bounded polymorphic
// interfaces to achieve covariant parameter/result specialization,
// and may be used as constraint types in generic functions.
//
// These interfaces are not yet published, but may be included in a
// future Go release once we have experience of whether these methods
// are necessary and sufficient.
//
// To keep the interfaces simple, many other useful operations are not
// provided as methods on the collection itself. They can be
// implemented externally, as generic library functions, albeit
// sometimes at a cost in performance.
//
// See below for these examples:
//
// - [Arbitrary], though useful methods for sets, can be expressed in
//   terms of Set.All; similarly, [Arbitrary2] using Map.All.
//
// - [ContainsAny], though a useful method for sets, can be
//   expressed in terms of Set.Contains and Set.All.
//
// - [Take] and [Take2], though convenient on both sets and maps and
//   useful for worklists, can be implemented generically in terms of
//   the All and Delete methods. However, it must look up the taken
//   element twice: once for iteration and once for deletion.
//
// - [Subset] and [Superset] for maps are provided in generic forms
//   below. This does forego opportunities to do a fast-path check for
//   identity, and to avoid overheads associated with iterators.
//   However, they would be essentially redundant w.r.t.
//   x.ContainsAll(y.All()).
//
// There will always be useful operations that could be expressed
// either as methods of these interfaces, allowing efficient
// specialization to a particular data structure, or as generic
// functions using iterators and Contains, perhaps at some performance
// cost. Drawing a line between operations that belong in the
// interface and those that don't is inherently a pragmatic judgment.
//
// Until such time as these interfaces are published, when defining
// your own generic functions, we suggest you make your own copy of
// these constraint types. For maximum generality, you may wish to
// delete any methods that are not required by a particular function.

// _AbstractCollection models a collection C of elements E,
// such as *hash.Map, *hash.Set, *ordered.Map, or set.Set.
type _AbstractCollection[E any, C _AbstractCollection[E, C]] interface {
	Clear()
	Clone() C
	Contains(E) bool
	Len() int
	String() string
}

// _AbstractMap models a mapping M from keys K to values V,
// such as *hash.Map or *ordered.Map.
type _AbstractMap[K, V any, M _AbstractMap[K, V, M]] interface {
	_AbstractCollection[K, M]

	All() iter.Seq2[K, V]
	Delete(K) (V, bool)
	DeleteFunc(func(K, V) bool) bool
	Get(K) V
	Keys() iter.Seq[K]
	Lookup(K) (V, bool)
	Set(K, V) (V, bool)
	Values() iter.Seq[V]
}

// _AbstractSet models a set S of elements E,
// such as *hash.Set, or set.Set.
type _AbstractSet[E any, S _AbstractSet[E, S]] interface {
	_AbstractCollection[E, S]

	All() iter.Seq[E]
	Delete(E) bool
	DeleteFunc(func(E) bool) bool
	Difference(S) S
	DifferenceWith(S)
	Equal(S) bool
	Insert(E) bool
	Intersection(S) S
	IntersectionWith(S)
	Intersects(S) bool
	SymmetricDifference(S) S
	SymmetricDifferenceWith(S)
	Union(S) S
	UnionWith(S)
}

// NOTES
//
// These types define the fundamental operations that need to be
// implemented by all set and map types. Their naming, signature, and
// semantic conventions should be followed wherever possible when
// defining new collection types.
//
// AbstractMap cannot have Equal since V values may be non-comparable.
//
// The iteration order of All is not specified by the interface. For
// some collections (e.g. ordered.Map) it is ascending key order; for
// others (e.g. set.Set) it is random; and for still others it may be
// unspecified.
//
// Map.Set should replace an existing entry with an equivalent key.
// This follows the built-in map: https://go.dev/play/p/pkH8kkFTuEg.

// -- conformance --

// This is a static compilation test of various symmetries,
// expressed using F-bounded polymorphic interfaces to
// achieve covariant parameter/result specialization.

var _ _AbstractSet[int, set.Set[int]] = make(set.Set[int])

// var _ _AbstractSet[int, *hash.Set[int]] = new(hash.Set[int])

var _ _AbstractMap[string, int, *hashed.Map[string, int]] = new(hashed.Map[string, int])

// var _ AbstractMap[string, int, *ordered.Map[string, int]] = new(ordered.Map[string, int]) // currently at github.com/jba/omap@v0.4.0/ordered.Map

func Test(*testing.T) {} // placeholder

// The functions below provide of standalone generic
// operators over abstract collections.

// ContainsAny reports whether set x contains any element of sequence y.
func ContainsAny[E any, S _AbstractSet[E, S]](x S, y iter.Seq[E]) bool {
	for elem := range y {
		if x.Contains(elem) {
			return true
		}
	}
	return false
}

// ContainsAll reports whether set x contains all elements of sequence y.
func ContainsAll[E any, S _AbstractSet[E, S]](x S, y iter.Seq[E]) bool {
	for elem := range y {
		if !x.Contains(elem) {
			return false
		}
	}
	return true
}

// Take removes and returns an arbitrary element from a set.
// It returns zero if the set was empty.
func Take[S _AbstractSet[E, S], E any](set S) (e E, found bool) {
	for e = range set.All() {
		found = true
		set.Delete(e) // may fail for NaN
		break
	}
	return
}

// Take2 removes and returns an arbitrary key/value entry from a map.
// It returns zero if the map was entry.
func Take2[M _AbstractMap[K, V, M], K, V any](m M) (k K, v V, found bool) {
	for k, v = range m.All() {
		found = true
		m.Delete(k) // may fail for NaN
		break
	}
	return
}

// Arbitrary returns an arbitrary element from a set.
// It returns zero if the set was empty.
func Arbitrary[S _AbstractSet[E, S], E any](set S) (e E, found bool) {
	for e = range set.All() {
		found = true
		break
	}
	return
}

// Arbitrary2 removes and returns an arbitrary key/value entry from a map.
// It returns zero if the map was entry.
func Arbitrary2[M _AbstractMap[K, V, M], K, V any](m M) (k K, v V, found bool) {
	for k, v = range m.All() {
		break
	}
	return
}

// Subset reports whether set x is a subset of set y.
func Subset[S _AbstractSet[E, S], E any](x, y S) bool {
	// We cannot shortcut if x == y here because
	// it may panic for some types (e.g. maps) or give
	// the wrong answer for others (e.g. strings
	// considered as unordered sets of bytes).
	// Secondarily, pointer identity also doesn't
	// repect NaN != NaN.

	return x.Len() <= y.Len() && ContainsAll(y, x.All())
}

// Superset reports whether x is a superset of y.
func Superset[S _AbstractSet[E, S], E any](x, y S) bool {
	return Subset(y, x)
}
