// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fmt_test

import (
	"golang.org/x/exp/errors"
	"golang.org/x/exp/errors/fmt"
)

func fn() error {
	err := errors.New("baz flopped")
	err = fmt.Errorf("bar(nameserver 139): %v", err)
	return fmt.Errorf("foo: %s", err)
}
func Example_formatting() {
	err := fn()
	fmt.Println("Error:")
	fmt.Printf("%v\n", err)
	fmt.Println()
	fmt.Println("Detailed error:")
	fmt.Printf("%+v\n", err)
	// Output:
	// Error:
	// foo: bar(nameserver 139): baz flopped
	//
	// Detailed error:
	// foo
	// --- bar(nameserver 139)
	// --- baz flopped
}
