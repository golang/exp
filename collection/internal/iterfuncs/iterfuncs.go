// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package iterfuncs provides functions that work on iterators.
package iterfuncs

import "iter"

// Every reports whether f(v) is true for every element in seq.
// It stops at the first element where f returns false.
func Every[V any](seq iter.Seq[V], f func(V) bool) bool {
	for v := range seq {
		if !f(v) {
			return false
		}
	}
	return true
}

// Some reports whether f(v) is true for some element in seq.
// It stops at the first element where f returns true.
func Some[V any](seq iter.Seq[V], f func(V) bool) bool {
	for v := range seq {
		if f(v) {
			return true
		}
	}
	return false
}
