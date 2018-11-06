// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fmt_test

import (
	"path/filepath"
	"regexp"

	"golang.org/x/exp/errors"
	"golang.org/x/exp/errors/fmt"
)

func baz() error { return errors.New("baz flopped") }
func bar() error { return fmt.Errorf("bar(nameserver 139): %v", baz()) }
func foo() error { return fmt.Errorf("foo: %s", bar()) }

func Example_formatting() {
	err := foo()
	fmt.Println("Error:")
	fmt.Printf("%v\n", err)
	fmt.Println()
	fmt.Println("Detailed error:")
	fmt.Println(stripPath(fmt.Sprintf("%+v\n", err)))
	// Output:
	// Error:
	// foo: bar(nameserver 139): baz flopped
	//
	// Detailed error:
	// foo:
	//     golang.org/x/exp/errors/fmt_test.foo
	//         golang.org/x/exp/errors/fmt/format_example_test.go:17
	//   - bar(nameserver 139):
	//     golang.org/x/exp/errors/fmt_test.bar
	//         golang.org/x/exp/errors/fmt/format_example_test.go:16
	//   - baz flopped:
	//     golang.org/x/exp/errors/fmt_test.baz
	//         golang.org/x/exp/errors/fmt/format_example_test.go:15
}

func stripPath(s string) string {
	rePath := regexp.MustCompile(`( [^ ]*)golang.org`)
	s = rePath.ReplaceAllString(s, " golang.org")
	s = filepath.ToSlash(s)
	return s
}
