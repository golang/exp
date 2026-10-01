// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashed_test

import (
	"fmt"
	"hash/maphash"
	"math/big"

	"golang.org/x/exp/collection/hashed"
)

// bigIntHasher is a maphash.Hasher for [*big.Int] values
// that compares them mathematically, not by pointer.
type bigIntHasher struct{}

func (bigIntHasher) Hash(h *maphash.Hash, b *big.Int) {
	maphash.WriteComparable(h, b.Sign())
	for _, word := range b.Bits() {
		maphash.WriteComparable(h, word)
	}
}

func (bigIntHasher) Equal(x, y *big.Int) bool {
	return x.Cmp(y) == 0
}

func ExampleMap_bigInt() {
	m := hashed.NewMap[*big.Int, string](bigIntHasher{})

	// Build a map.
	i100 := big.NewInt(100)
	i200 := big.NewInt(200)
	m.Set(i100, "one hundred")
	m.Set(i200, "two hundred")
	fmt.Println("Map:", m)

	// Look up an entry using a distinct but
	// numerically equivalent big.Int.
	i100b := big.NewInt(100)
	fmt.Println("Same pointer:", i100 == i100b)
	fmt.Println("Get(i100b):", m.Get(i100b))

	// Output:
	//
	// Map: {100: one hundred, 200: two hundred}
	// Same pointer: false
	// Get(i100b): one hundred
}
