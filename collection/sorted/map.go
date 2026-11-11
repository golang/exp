// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sorted defines collections based on key order
// such as [Map].
package sorted

// The implementation is a treap: a tree that is both a binary search tree ordered
// by keys (node.key), and a binary min-heap, using random priorities (node.pri)
// for the heap invariant.
// See:
// https://en.wikipedia.org/wiki/Treap
// https://faculty.washington.edu/aragon/pubs/rst89.pdf

import (
	"fmt"
	"iter"
	"math/rand/v2"
	"strings"

	"golang.org/x/exp/collection/internal"
)

// A Map is a mapping from keys to values, where the keys are ordered
// according to a comparison function specified at construction.
//
// A Map must be created with [NewMap].
// Neither a nil pointer nor the zero value are valid maps.
//
// Map values must not be copied; use [Map.Clone] instead.
type Map[K, V any] struct {
	root *node[K, V]
	cmp  func(K, K) int
	gen  uint64 // generation count, for efficient range deletion
	_    internal.NoCopy
}

// A node is a node in the treap.
type node[K any, V any] struct {
	parent *node[K, V]
	left   *node[K, V]
	right  *node[K, V]
	key    K
	val    V
	pri    uint64 // priority, for the heap invariant
	sz     int    // number of keys in this node's subtree
}

// NewMap returns a new [Map] ordered according to cmp.
func NewMap[K, V any](cmp func(K, K) int) *Map[K, V] {
	return &Map[K, V]{cmp: cmp}
}

// Span returns an unbounded [MapSpan] for the map.
func (m *Map[K, V]) Span() MapSpan[K, V] { return MapSpan[K, V]{m: m} }

// find reports where a node with the key would be: at *pos.
// If *pos != nil, then key is present in the tree;
// otherwise *pos is where a new node with the key should be attached.
//
// If parent != nil, then pos is either &parent.left or &parent.right
// depending on how parent.key compares with key.
// If parent == nil, then pos is m.root.
func (m *Map[K, V]) find(k K) (pos **node[K, V], parent *node[K, V]) {
	pos = &m.root
	for x := *pos; x != nil; x = *pos {
		cmp := m.cmp(x.key, k)
		if cmp == 0 {
			break
		}
		parent = x
		if cmp > 0 {
			pos = &x.left
		} else {
			pos = &x.right
		}
	}
	return pos, parent
}

// Contains reports whether the map contains an entry with the given key.
func (m *Map[K, V]) Contains(key K) bool {
	pos, _ := m.find(key)
	return *pos != nil
}

// Lookup returns the value of m[key] and reports whether it exists.
func (m *Map[K, V]) Lookup(key K) (V, bool) {
	pos, _ := m.find(key)
	if x := *pos; x != nil {
		return x.val, true
	}
	var zero V
	return zero, false
}

// Get returns the value of m[key], or the zero value if key is not present.
func (m *Map[K, V]) Get(key K) V {
	v, _ := m.Lookup(key)
	return v
}

// Set updates the map entry for key to value.
// It returns the previous value for key if present, or the zero value if not.
// The boolean (present) indicates whether the operation updated an existing entry
// for that key.
func (m *Map[K, V]) Set(key K, val V) (old V, present bool) {
	pos, parent := m.find(key)
	if x := *pos; x != nil {
		old = x.val
		x.val = val
		return old, true
	}
	// Choose a random non-zero priority. 0 is the tombstone marker.
	pri := rand.Uint64() | 1
	x := &node[K, V]{key: key, val: val, pri: pri, parent: parent, sz: 1}
	*pos = x
	// Walk up, adjusting size.
	for p := x.parent; p != nil; p = p.parent {
		p.sz++
	}

	m.rotateUp(x)
	return *new(V), false
}

// Len returns the number of keys in m.
func (m *Map[K, V]) Len() int { return m.root.size() }

func (n *node[K, V]) size() int {
	if n == nil {
		return 0
	}
	return n.sz
}

// -- rotation --

// Rotations re-establish the heap property (each node's priority is less than those
// of its children) while maintaining the search tree property (each node's key is
// greater than its left child's and less than its right child's).

// rotateUp rotates x upward in the tree to correct any priority inversions.
func (m *Map[K, V]) rotateUp(x *node[K, V]) {
	// Rotate up into tree according to priority.
	for x.parent != nil && x.parent.pri > x.pri {
		if x.parent.left == x {
			m.rotateRight(x.parent)
		} else {
			m.rotateLeft(x.parent)
		}
	}
}

// rotateLeft rotates the subtree rooted at node x,
// turning (x a (y b c)) into (y (x a b) c).
func (m *Map[K, V]) rotateLeft(x *node[K, V]) {
	// p -> (x a (y b c))
	p := x.parent
	y := x.right
	b := y.left

	y.left = x
	x.parent = y
	x.right = b
	if b != nil {
		b.parent = x
	}

	y.parent = p
	if p == nil {
		m.root = y
	} else if p.left == x {
		p.left = y
	} else if p.right == x {
		p.right = y
	} else {
		// unreachable
		panic("corrupt treap")
	}

	// Recompute sizes.
	// x used to include a, b, c and the single node y; it has lost
	// y and c.
	x.sz -= 1 + y.right.size()
	// y used to include just b and c; it now also has the node x
	// and the subtree a.
	y.sz += 1 + x.left.size()
	// parent size doesn't change.
}

// rotateRight rotates the subtree rooted at node y,
// turning (y (x a b) c) into (x a (y b c)).
func (m *Map[K, V]) rotateRight(y *node[K, V]) {
	// p -> (y (x a b) c)
	p := y.parent
	x := y.left
	b := x.right

	x.right = y
	y.parent = x
	y.left = b
	if b != nil {
		b.parent = y
	}

	x.parent = p
	if p == nil {
		m.root = x
	} else if p.left == y {
		p.left = x
	} else if p.right == y {
		p.right = x
	} else {
		// unreachable
		panic("corrupt treap")
	}

	// Recompute sizes.
	// x used to include just a and b; now it also includes the node y and
	// the subtree at c.
	x.sz += 1 + y.right.size()
	// y included a, b, c and the node x; it has lost the node x and the
	// subtree at a.
	y.sz -= 1 + x.left.size()
	// parent size doesn't change.
}

// -- deletion --

// Delete removes the entry with the given key, if present.
// It returns the previous value for key if present, or the zero value if not.
// The boolean (deleted) indicates whether the operation deleted a map entry.
func (m *Map[K, V]) Delete(key K) (prev V, deleted bool) {
	pos, _ := m.find(key)
	x := *pos
	if x == nil {
		return prev, false
	}
	m.deleteNode(x)
	return x.val, true
}

// deleteNode removes x from the treap. x must be in m.
func (m *Map[K, V]) deleteNode(x *node[K, V]) {
	// Rotate x down to be a leaf of the tree for removal, respecting priorities.
	for x.right != nil || x.left != nil {
		if x.right == nil || x.left != nil && x.left.pri < x.right.pri {
			m.rotateRight(x)
		} else {
			m.rotateLeft(x)
		}
	}

	// Remove x, now a leaf.
	switch p := x.parent; {
	case p == nil:
		m.root = nil
	case p.left == x:
		p.left = nil
	default:
		p.right = nil
	}
	x.pri = 0 // mark deleted

	// Walk up, adjusting size.
	for p := x.parent; p != nil; p = p.parent {
		p.sz--
	}
}

// DeleteFunc removes all the entries for which del returns true.
// It reports whether the map changed.
func (m *Map[K, V]) DeleteFunc(del func(K, V) bool) bool {
	changed := false
	for x := m.Span().minNode(); x != nil; {
		gen := m.gen
		doDel := del(x.key, x.val)
		next := x.next(m, gen != m.gen)
		gen = m.gen
		if doDel && x.pri != 0 {
			m.deleteNode(x)
			changed = true
		}
		x = next
	}
	return changed
}

// deleteRange deletes m's entries between the bounds.
func deleteRange[K, V any](m *Map[K, V], lo, hi bound[K]) {
	// TODO: rewrite to avoid reinsertions.
	switch {
	case !lo.present && !hi.present:
		// This range includes the whole map.
		m.Clear()
	case lo.present && hi.present:
		if c := m.cmp(lo.key, hi.key); c > 0 || (c == 0 && (!lo.inclusive || !hi.inclusive)) {
			return
		}
		// Both bounds are present.
		// Delete all entries between them, inclusive; then reinsert
		// any entries at the exclusive endpoints.
		loVal, loInMap := m.Lookup(lo.key)
		hiVal, hiInMap := m.Lookup(hi.key)
		deleteBetweenInclusive(m, lo.key, hi.key)
		if !lo.inclusive && loInMap {
			m.Set(lo.key, loVal)
		}
		if !hi.inclusive && hiInMap {
			m.Set(hi.key, hiVal)
		}
	case lo.present:
		val, ok := m.Lookup(lo.key)
		// Split m at lo.key. m itself retains only entries
		// whose key is < lo.key.
		_ = splitExclusive(m, lo.key)
		if !lo.inclusive && ok {
			// lo.key is in the map, and the deletion range excludes it,
			// so re-insert it.
			m.Set(lo.key, val)
		}
	case hi.present:
		val, ok := m.Lookup(hi.key)
		// Split m at hi.key. The return value consists of
		// entries whose key is > hi.key.
		after := splitExclusive(m, hi.key)
		// Keep after, discard m.
		m.root = after
		if after != nil {
			after.parent = nil
		}
		if !hi.inclusive && ok {
			m.Set(hi.key, val)
		}
	default:
		panic("unreachable")
	}
}

// deleteBetweenInclusive deletes items from m in [lo, hi].
// Called deleteRange in rsc.io/omap.
func deleteBetweenInclusive[K, V any](m *Map[K, V], lo, hi K) {
	after := splitExclusive(m, hi)
	_ = splitExclusive(m, lo)
	if after != nil {
		// Add after to m.
		// Both lo and all of after's keys are greater than any key in m.
		pos, parent := m.find(lo)
		*pos = after
		after.parent = parent
		// Adjust sizes.
		for p := parent; p != nil; p = p.parent {
			p.sz = 1 + p.left.size() + p.right.size()
		}
		// after is now in the right place by key, but perhaps not by priority.
		m.rotateUp(after)
	}
}

// splitExclusive splits m into two subtrees.
// The returned node contains all keys > key.
// m itself contains all keys < key.
// key itself is not part of either tree.
// Note that after's parent is not changed; it must be set by the caller.
func splitExclusive[K, V any](m *Map[K, V], key K) (after *node[K, V]) {
	pos, parent := m.find(key)
	if *pos == nil {
		*pos = &node[K, V]{parent: parent}
		for p := parent; p != nil; p = p.parent {
			p.sz++
		}
	}
	x := *pos
	x.pri = 0
	x.sz = 0
	m.rotateUp(x)

	m.root = x.left
	if x.left != nil {
		x.left.parent = nil
	}
	return x.right
}

// Min returns the minimum key in m along with its value.
// The third return value is false if the map is empty, otherwise true.
func (m *Map[K, V]) Min() (K, V, bool) { return m.Span().Min() }

// Max returns the maximum key in m along with its value.
// The third return value is false if the map is empty, otherwise true.
func (m *Map[K, V]) Max() (K, V, bool) { return m.Span().Max() }

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
func (m *Map[K, V]) All() iter.Seq2[K, V] { return m.Span().All() }

// Keys returns an iterator over the keys in m from smallest to largest.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (m *Map[K, V]) Keys() iter.Seq[K] { return m.Span().Keys() }

// Values returns an iterator over the values in m from smallest to largest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (m *Map[K, V]) Values() iter.Seq[V] { return m.Span().Values() }

// Backward returns an iterator over the map m from largest to smallest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (m *Map[K, V]) Backward() iter.Seq2[K, V] { return m.Span().Backward() }

// Clear removes all entries from the map.
func (m *Map[K, V]) Clear() {
	m.root = nil
	// Each iterator will check its stored generation number against m.gen
	// before its next yield, detecting that it is invalid.
	m.gen++
}

// Clone returns a new non-nil map with the same entries as m.
func (m *Map[K, V]) Clone() *Map[K, V] {
	copy := NewMap[K, V](m.cmp)
	copy.root = m.root.clone(nil)
	return copy
}

func (n *node[K, V]) clone(parent *node[K, V]) *node[K, V] {
	if n == nil {
		return nil
	}
	x2 := new(*n)
	x2.left = n.left.clone(x2)
	x2.right = n.right.clone(x2)
	x2.parent = parent
	return x2
}

// String returns a string representation of m in the form "{k1: v1, k2: v2}".
func (m *Map[K, V]) String() string {
	return mapString(m.All())
}

func mapString[K, V any](items iter.Seq2[K, V]) string {
	var b strings.Builder
	b.WriteByte('{')
	for k, v := range items {
		if b.Len() > len("{") {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%v: %v", k, v)
	}
	b.WriteByte('}')
	return b.String()
}

// Nth returns the key and value at index i,
// which must be nonnegative and less than m.Len().
// See [Map.Index] for the inverse operation.
func (m *Map[K, V]) Nth(i int) (K, V) { return m.root.nth(i) }

func (n *node[K, V]) nth(i int) (K, V) {
	if n == nil {
		panic("index out of range")
	}
	lsz := n.left.size()
	if i == lsz {
		return n.key, n.val
	}
	if i < lsz {
		return n.left.nth(i)
	}
	return n.right.nth(i - lsz - 1)
}

// Index returns the index of key in m, or -1 if key is not present.
// The index of a key is its position in the sequence of keys.
// For example, the smallest key has index 0,
// and the largest has index m.Len()-1.
// See [Map.Nth] for the inverse operation.
func (m *Map[K, V]) Index(key K) int {
	if m.root == nil {
		return -1
	}
	pos, _ := m.find(key)
	if *pos == nil {
		return -1
	}
	return (*pos).rank()
}

// rank returns the 0-based index of n in the sorted sequence that the tree
// rooted at n represents.
// n must not be nil.
func (n *node[K, V]) rank() int {
	if n == nil {
		panic("nil node")
	}
	r := n.left.size()
	for p := n.parent; p != nil; n, p = p, p.parent {
		if n == p.right {
			r += 1 + p.left.size()
		}
	}
	return r
}

// A MapSpan is a restricted view of a [Map] consisting only of the items
// whose keys are between (optional) lower and upper bounds.
//
// A MapSpan's bounds do not change, but the map it refers to may change over
// time, so the items it encompasses may also change.
type MapSpan[K, V any] struct {
	m      *Map[K, V]
	lo, hi bound[K]
}

type bound[K any] struct {
	key       K
	present   bool // if false, there is no bound (it is +/-infinity)
	inclusive bool // does the bound include key?
}

func including[K any](k K) bound[K] {
	return bound[K]{k, true, true}
}

func excluding[K any](k K) bound[K] {
	return bound[K]{k, true, false}
}

// inLo reports whether k is above s's lower bound.
func (s MapSpan[K, V]) inLo(k K) bool {
	if !s.lo.present {
		return true
	}
	if s.lo.inclusive {
		return s.m.cmp(k, s.lo.key) >= 0
	}
	return s.m.cmp(k, s.lo.key) > 0
}

// inHi reports whether k is below s's upper bound.
func (s MapSpan[K, V]) inHi(k K) bool {
	if !s.hi.present {
		return true
	}
	if s.hi.inclusive {
		return s.m.cmp(k, s.hi.key) <= 0
	}
	return s.m.cmp(k, s.hi.key) < 0
}

// From returns a [MapSpan] with lower bound lo, inclusive, and no upper bound.
func (m *Map[K, V]) From(lo K) MapSpan[K, V] { return MapSpan[K, V]{m: m, lo: including(lo)} }

// Above returns a [MapSpan] with lower bound lo, exclusive, and no upper bound.
func (m *Map[K, V]) Above(lo K) MapSpan[K, V] { return MapSpan[K, V]{m: m, lo: excluding(lo)} }

// To returns a [MapSpan] with upper bound hi, inclusive, and no lower bound.
func (m *Map[K, V]) To(hi K) MapSpan[K, V] { return MapSpan[K, V]{m: m, hi: including(hi)} }

// Below returns a [MapSpan] with upper bound hi, exclusive, and no lower bound.
func (m *Map[K, V]) Below(hi K) MapSpan[K, V] { return MapSpan[K, V]{m: m, hi: excluding(hi)} }

// From returns a [MapSpan] with lower bound lo, inclusive, and the same upper bound as s,
// if lo is within s's current lower bound. Otherwise, it returns a MapSpan with the same
// bounds as s.
func (s MapSpan[K, V]) From(lo K) MapSpan[K, V] {
	if s.inLo(lo) {
		s.lo = including(lo)
	}
	return s
}

// Above returns a [MapSpan] with lower bound lo, exclusive, and the same upper bound as s,
// if lo is within s's current lower bound. Otherwise, it returns a MapSpan with the same
// bounds as s.
func (s MapSpan[K, V]) Above(lo K) MapSpan[K, V] {
	if s.inLo(lo) {
		s.lo = excluding(lo)
	}
	return s
}

// To returns a [MapSpan] with upper bound hi, inclusive, and the same lower bound as s,
// if hi is within s's current upper bound. Otherwise, it returns a MapSpan with the same
// bounds as s.
func (s MapSpan[K, V]) To(hi K) MapSpan[K, V] {
	if s.inHi(hi) {
		s.hi = including(hi)
	}
	return s
}

// Below returns a [MapSpan] with upper bound hi, exclusive, and the same lower bound as s,
// if hi is within s's current upper bound. Otherwise, it returns a MapSpan with the same
// bounds as s.
func (s MapSpan[K, V]) Below(hi K) MapSpan[K, V] {
	if s.inHi(hi) {
		s.hi = excluding(hi)
	}
	return s
}

// Min returns the minimum key in s along with its value.
// The third return value is false if s is empty, otherwise true.
func (s MapSpan[K, V]) Min() (K, V, bool) {
	n := s.minNode()
	if n == nil {
		return *new(K), *new(V), false
	}
	return n.key, n.val, true
}

// minNode returns the node with the smallest key in s,
// or nil if there is none.
func (s MapSpan[K, V]) minNode() *node[K, V] {
	var minn func(*node[K, V]) *node[K, V]
	minn = func(n *node[K, V]) *node[K, V] {
		if n == nil {
			return nil
		}
		if !s.inLo(n.key) {
			return minn(n.right)
		}
		if !s.inHi(n.key) {
			return minn(n.left)
		}
		// n.key is in range.
		// If there is a smaller key, it is in the left subtree.
		if l := minn(n.left); l != nil {
			return l
		}
		return n
	}

	return minn(s.m.root)
}

// Max returns the maximum key in s along with its value.
// The third return value is false if s is empty, otherwise true.
func (s MapSpan[K, V]) Max() (K, V, bool) {
	n := s.maxNode()
	if n == nil {
		return *new(K), *new(V), false
	}
	return n.key, n.val, true
}

// maxNode returns the node with the largest key in s,
// or nil if there is none (because the span is empty).
func (s MapSpan[K, V]) maxNode() *node[K, V] {
	var maxn func(*node[K, V]) *node[K, V]
	maxn = func(n *node[K, V]) *node[K, V] {
		if n == nil {
			return nil
		}
		if !s.inLo(n.key) {
			return maxn(n.right)
		}
		if !s.inHi(n.key) {
			return maxn(n.left)
		}
		// n.key is in range.
		// If there is a larger key, it is in the right subtree.
		if r := maxn(n.right); r != nil {
			return r
		}
		return n
	}

	return maxn(s.m.root)
}

// Index returns the index of key within s, or -1 if key is not present or not in bounds.
func (s MapSpan[K, V]) Index(key K) int {
	if !s.inLo(key) || !s.inHi(key) {
		return -1
	}
	pos, _ := s.m.find(key)
	if *pos == nil {
		return -1
	}
	min := s.minNode()
	if min == nil {
		return -1
	}
	return (*pos).rank() - min.rank()
}

// Len returns the number of keys in s.
func (s MapSpan[K, V]) Len() int {
	// Compute in logarithmic time by finding the difference in ranks
	// of the extremal keys.
	min := s.minNode()
	if min == nil {
		return 0
	}
	max := s.maxNode()
	if max == nil {
		return 0
	}
	return max.rank() - min.rank() + 1
}

// String returns a string representation of s.
func (s MapSpan[K, V]) String() string {
	return mapString(s.All())
}

// Nth returns the key and value at index i within s.
// It panics if i < 0 or i >= s.Len().
func (s MapSpan[K, V]) Nth(i int) (K, V) {
	return s.m.root.nth(s.minNode().rank() + i)
}

// Clone returns a new non-nil map with the same entries as s.
func (s MapSpan[K, V]) Clone() *Map[K, V] {
	m := NewMap[K, V](s.m.cmp)
	for k, v := range s.All() {
		m.Set(k, v)
	}
	return m
}

// Clear deletes all the entries in s from s's underlying map.
func (s MapSpan[K, V]) Clear() {
	deleteRange(s.m, s.lo, s.hi)
	s.m.gen++
}

// All returns an iterator over s from smallest to largest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (s MapSpan[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		x := s.minNode()
		gen := s.m.gen
		for x != nil && s.inHi(x.key) && yield(x.key, x.val) {
			x = x.next(s.m, gen != s.m.gen)
			gen = s.m.gen
		}
	}
}

// Keys returns an iterator over the keys in s from smallest to largest.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (s MapSpan[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		s.All()(func(k K, _ V) bool { return yield(k) })
	}
}

// Values returns an iterator over the values in s from smallest to largest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (s MapSpan[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		s.All()(func(_ K, v V) bool { return yield(v) })
	}
}

// Backward returns an iterator over s from largest to smallest key.
// See [Map.All] for the guarantee provided if m is modified during the iteration.
func (s MapSpan[K, V]) Backward() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		x := s.maxNode()
		gen := s.m.gen
		for x != nil && s.inLo(x.key) && yield(x.key, x.val) {
			x = x.prev(s.m, gen != s.m.gen)
			gen = s.m.gen
		}
	}
}

// next returns the successor node of n in the treap,
// even if n has been removed from the treap.
// n must not be nil.
func (n *node[K, V]) next(m *Map[K, V], reset bool) *node[K, V] {
	if n.pri == 0 || reset {
		// n has been deleted, or the iterator calling next has been invalidated.
		// Find where n.key would be in the current tree: the node with the smallest
		// key >= n.key.
		nk := n.key
		n = m.Span().From(nk).minNode()
		if n == nil || m.cmp(n.key, nk) != 0 {
			// The new node's key (if any) is greater than the old n.key.
			return n
		}
	}
	// If n has a right child, the next node is its leftmost node.
	if r := n.right; r != nil {
		for r.left != nil {
			r = r.left
		}
		return r
	}
	// Otherwise, the next node is the closest ancestor that contains
	// n in its left subtree.
	for n.parent != nil && n.parent.right == n {
		n = n.parent
	}
	return n.parent
}

// prev returns the predecessor node of n in the treap,
// even if n has been removed from the treap.
// n must not be nil.
func (n *node[K, V]) prev(m *Map[K, V], reset bool) *node[K, V] {
	if n.pri == 0 || reset {
		// n has been deleted, or the iterator calling prev has been invalidated.
		// Find where n.key would be in the current tree: the node with the largest
		// key <= n.key.
		nk := n.key
		n = m.Span().To(nk).maxNode()
		if n == nil || m.cmp(n.key, nk) != 0 {
			// The new node's key (if any) is less than the old n.key.
			return n
		}
	}
	// If n has a left child, the previous node is its rightmost node.
	if l := n.left; l != nil {
		for l.right != nil {
			l = l.right
		}
		return l
	}
	// Otherwise, the previous node is the closest ancestor that contains
	// n in its right subtree.
	for n.parent != nil && n.parent.left == n {
		n = n.parent
	}
	return n.parent
}
