// Copyright 2024 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sorted_test

import (
	"cmp"
	"fmt"

	"golang.org/x/exp/collection/sorted"
)

func ExampleMap_All() {
	m := sorted.NewMap[int, string](cmp.Compare)
	m.Set(1, "one")
	m.Set(2, "two")
	m.Set(3, "three")

	for k, v := range m.All() {
		fmt.Println(k, v)
	}

	// Output:
	// 1 one
	// 2 two
	// 3 three
}

func ExampleMap_From() {
	m := sorted.NewMap[int, string](cmp.Compare)
	m.Set(1, "one")
	m.Set(2, "two")
	m.Set(3, "three")

	for k, v := range m.From(2).All() {
		fmt.Println(k, v)
	}

	// Output:
	// 2 two
	// 3 three
}

func ExampleMap_Below() {
	m := sorted.NewMap[int, string](cmp.Compare)
	m.Set(1, "one")
	m.Set(2, "two")
	m.Set(3, "three")

	for k, v := range m.Above(1).Below(3).All() {
		fmt.Println(k, v)
	}

	// Output:
	// 2 two
}
