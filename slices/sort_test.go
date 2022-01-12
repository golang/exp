// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slices

import (
	"math"
	"math/rand"
	"testing"
)

var ints = [...]int{74, 59, 238, -784, 9845, 959, 905, 0, 0, 42, 7586, -5467984, 7586}
var float64s = [...]float64{74.3, 59.0, math.Inf(1), 238.2, -784.0, 2.3, math.NaN(), math.NaN(), math.Inf(-1), 9845.768, -959.7485, 905, 7.8, 7.8}
var strs = [...]string{"", "Hello", "foo", "bar", "foo", "f00", "%*&^*&^&", "***"}

func TestSortIntSlice(t *testing.T) {
	data := ints[:]
	Sort(data)
	if !IsSorted(data) {
		t.Errorf("sorted %v", ints)
		t.Errorf("   got %v", data)
	}
}

func TestSortFuncIntSlice(t *testing.T) {
	data := ints[:]
	SortFunc(data, func(a, b int) bool { return a < b })
	if !IsSorted(data) {
		t.Errorf("sorted %v", ints)
		t.Errorf("   got %v", data)
	}
}

func TestSortFloat64Slice(t *testing.T) {
	data := float64s[:]
	Sort(data)
	if !IsSorted(data) {
		t.Errorf("sorted %v", float64s)
		t.Errorf("   got %v", data)
	}
}

func TestSortStringSlice(t *testing.T) {
	data := strs[:]
	Sort(data)
	if !IsSorted(data) {
		t.Errorf("sorted %v", strs)
		t.Errorf("   got %v", data)
	}
}

func TestSortLarge_Random(t *testing.T) {
	n := 1000000
	if testing.Short() {
		n /= 100
	}
	data := make([]int, n)
	for i := 0; i < len(data); i++ {
		data[i] = rand.Intn(100)
	}
	if IsSorted(data) {
		t.Fatalf("terrible rand.rand")
	}
	Sort(data)
	if !IsSorted(data) {
		t.Errorf("sort didn't sort - 1M ints")
	}
}

type intPair struct {
	a, b int
}

type intPairs []intPair

// Pairs compare on a only.
func intPairLess(x, y intPair) bool {
	return x.a < y.a
}

// Record initial order in B.
func (d intPairs) initB() {
	for i := range d {
		d[i].b = i
	}
}

// InOrder checks if a-equal elements were not reordered.
func (d intPairs) inOrder() bool {
	lastA, lastB := -1, 0
	for i := 0; i < len(d); i++ {
		if lastA != d[i].a {
			lastA = d[i].a
			lastB = d[i].b
			continue
		}
		if d[i].b <= lastB {
			return false
		}
		lastB = d[i].b
	}
	return true
}

func TestStability(t *testing.T) {
	n, m := 100000, 1000
	if testing.Short() {
		n, m = 1000, 100
	}
	data := make(intPairs, n)

	// random distribution
	for i := 0; i < len(data); i++ {
		data[i].a = rand.Intn(m)
	}
	if IsSortedFunc(data, intPairLess) {
		t.Fatalf("terrible rand.rand")
	}
	data.initB()
	SortStableFunc(data, intPairLess)
	if !IsSortedFunc(data, intPairLess) {
		t.Errorf("Stable didn't sort %d ints", n)
	}
	if !data.inOrder() {
		t.Errorf("Stable wasn't stable on %d ints", n)
	}

	// already sorted
	data.initB()
	SortStableFunc(data, intPairLess)
	if !IsSortedFunc(data, intPairLess) {
		t.Errorf("Stable shuffled sorted %d ints (order)", n)
	}
	if !data.inOrder() {
		t.Errorf("Stable shuffled sorted %d ints (stability)", n)
	}

	// sorted reversed
	for i := 0; i < len(data); i++ {
		data[i].a = len(data) - i
	}
	data.initB()
	SortStableFunc(data, intPairLess)
	if !IsSortedFunc(data, intPairLess) {
		t.Errorf("Stable didn't sort %d ints", n)
	}
	if !data.inOrder() {
		t.Errorf("Stable wasn't stable on %d ints", n)
	}
}

func TestBinarySearch(t *testing.T) {
	data := []string{"aa", "ad", "ca", "xy"}
	tests := []struct {
		target string
		want   int
	}{
		{"aa", 0},
		{"ab", 1},
		{"ad", 1},
		{"ax", 2},
		{"ca", 2},
		{"cc", 3},
		{"dd", 3},
		{"xy", 3},
		{"zz", 4},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			i := BinarySearch(data, tt.target)
			if i != tt.want {
				t.Errorf("BinarySearch want %d, got %d", tt.want, i)
			}

			j := BinarySearchFunc(data, func(s string) bool { return s >= tt.target })
			if j != tt.want {
				t.Errorf("BinarySearchFunc want %d, got %d", tt.want, j)
			}
		})
	}
}
