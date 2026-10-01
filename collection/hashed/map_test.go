// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashed_test

import (
	"hash/maphash"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"golang.org/x/exp/collection/hashed"
)

// caseInsensitive is a string Hasher that ignores letter case.
type caseInsensitive struct{}

var _ maphash.Hasher[string] = caseInsensitive{}

func (caseInsensitive) Hash(h *maphash.Hash, s string) {
	var buf [utf8.UTFMax]byte
	for _, r := range s {
		n := utf8.EncodeRune(buf[:], unicode.ToLower(r))
		h.Write(buf[:n])
	}
}

func (caseInsensitive) Equal(x, y string) bool {
	return strings.EqualFold(x, y)
}

func TestMap(t *testing.T) {
	m := hashed.NewMap[string, int](caseInsensitive{})

	// Length of empty map.
	if l := m.Len(); l != 0 {
		t.Errorf("Len() on empty Map: got %d, want 0", l)
	}
	// Get of missing key.
	if v := m.Get("foo"); v != 0 {
		t.Errorf("Get() on empty Map: got %v, want 0", v)
	}
	// Deletion of missing key.
	if _, removed := m.Delete("foo"); removed {
		t.Errorf("Delete() on empty Map: got true, want false")
	}
	// Contains of missing key.
	if m.Contains("foo") {
		t.Errorf("Contains() on empty Map: got true, want false")
	}

	// Set of new key.
	if prev, present := m.Set("Hello", 1); prev != 0 {
		t.Errorf("Set() on empty Map returned non-zero previous value %d", prev)
	} else if present {
		t.Errorf("Set() on empty Map returned present=true")
	}

	// Now: {"Hello": 1}

	if l := m.Len(); l != 1 {
		t.Errorf("Len(): got %d, want 1", l)
	}
	if v := m.Get("Hello"); v != 1 {
		t.Errorf(`Get("Hello"): got %v, want 1`, v)
	}
	// Case-insensitive get
	if v := m.Get("hello"); v != 1 {
		t.Errorf(`Get("hello"): got %v, want 1`, v)
	}
	if v := m.Get("HELLO"); v != 1 {
		t.Errorf(`Get("HELLO"): got %v, want 1`, v)
	}
	if !m.Contains("hElLo") {
		t.Errorf(`Contains("hElLo") returned false`)
	}

	// Update existing key (also replaces stored key "Hello" with "HELLO")
	if prev, present := m.Set("HELLO", 2); prev != 1 {
		t.Errorf(`Set("HELLO") previous value: got %d, want 1`, prev)
	} else if !present {
		t.Errorf(`Set("HELLO") returned present=false`)
	}

	// Set another key
	if prev, present := m.Set("World", 3); prev != 0 {
		t.Errorf(`Set("World") previous value: got %d, want 0`, prev)
	} else if present {
		t.Errorf(`Set("World") returned present=true`)
	}

	// Now: {"HELLO": 2, "World": 3}

	want1 := "{HELLO: 2, World: 3}"
	want2 := "{World: 3, HELLO: 2}"
	if s := m.String(); s != want1 && s != want2 {
		t.Errorf("String(): got %s, want %s or %s", s, want1, want2)
	}

	// Test iterators
	keys := slices.Collect(m.Keys())
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"HELLO", "World"}) {
		t.Errorf("Keys(): got %v", keys)
	}

	values := slices.Collect(m.Values())
	slices.Sort(values)
	if !slices.Equal(values, []int{2, 3}) {
		t.Errorf("Values(): got %v", values)
	}

	entries := make(map[string]int)
	for k, v := range m.All() {
		entries[k] = v
	}
	if len(entries) != 2 || entries["HELLO"] != 2 || entries["World"] != 3 {
		t.Errorf("All(): got %v", entries)
	}

	// Clone
	m2 := m.Clone()
	if l := m2.Len(); l != 2 {
		t.Errorf("Clone.Len() = %d, want 2", l)
	}
	if m2.Get("hello") != 2 {
		t.Errorf(`Clone.Get("hello") = %d`, m2.Get("hello"))
	}

	// Delete existing
	if prev, removed := m.Delete("hello"); !removed || prev != 2 {
		t.Errorf(`Delete("hello") = %v, %v, want 2, true`, prev, removed)
	}

	// Now: {"World": 3}

	if l := m.Len(); l != 1 {
		t.Errorf("Len(): got %d, want 1", l)
	}
	if m.Contains("Hello") {
		t.Errorf(`Contains("Hello") returned true after Delete`)
	}

	// Re-insert "Hello" (reuses deleted bucket slot)
	if prev, present := m.Set("Hello", 4); prev != 0 || present {
		t.Errorf(`Set("Hello", 4) = %d, %t, want 0, false`, prev, present)
	}
	if v := m.Get("hello"); v != 4 {
		t.Errorf(`Get("hello") = %d, want 4`, v)
	}

	// DeleteFunc
	if !m.DeleteFunc(func(k string, v int) bool { return v == 4 }) {
		t.Errorf("DeleteFunc did not report change")
	}
	if m.DeleteFunc(func(k string, v int) bool { return v == 4 }) {
		t.Errorf("DeleteFunc reported change for no-op")
	}
	if l := m.Len(); l != 1 || m.Contains("Hello") {
		t.Errorf("after DeleteFunc: Len()=%d, Contains(Hello)=%t", l, m.Contains("Hello"))
	}

	// Clear
	m.Clear()
	if l := m.Len(); l != 0 {
		t.Errorf("Len() after Clear: got %d, want 0", l)
	}
	if m.Contains("World") {
		t.Errorf(`Contains("World") returned true after Clear`)
	}

	// Ensure m2 (clone) is unaffected by Clear
	if l := m2.Len(); l != 2 {
		t.Errorf("m2.Len() after m.Clear() = %d, want 2", l)
	}
}

func TestNilMapPanics(t *testing.T) {
	panics := func(name string, f func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for nil Map.%s", name)
			}
		}()
		f()
	}

	var m *hashed.Map[string, int]

	panics("Len", func() { m.Len() })
	panics("All", func() { m.All() })
	panics("Keys", func() { m.Keys() })
	panics("Values", func() { m.Values() })
	panics("Lookup", func() { m.Lookup("key") })
	panics("Get", func() { m.Get("key") })
	panics("Contains", func() { m.Contains("key") })
	panics("Set", func() { m.Set("key", 1) })
	panics("Delete", func() { m.Delete("key") })
	panics("Clear", func() { m.Clear() })
	panics("String", func() { _ = m.String() })
	panics("Clone", func() { m.Clone() })
}
