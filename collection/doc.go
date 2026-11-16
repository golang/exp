// Packages in the collection/... hierarchy provide various implementation
// of collection types.
//   - [set] provides a convenient wrapper around map[T]struct{}.
//   - [hashed] provides data structures based on hashing.
//   - [sorted] provides data structures based on comparison.
//
// This package is intended to provide general documentation, declarations,
// helper functions, and tests that are independent of any particular
// collection data type.
//
// [set]: https://pkg.go.dev/golang.org/x/exp/collection/set
// [hashed]: https://pkg.go.dev/golang.org/x/exp/collection/hashed
// [sorted]: https://pkg.go.dev/golang.org/x/exp/collection/sorted
package collection
