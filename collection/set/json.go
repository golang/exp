// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package set

import (
	"encoding/json/jsontext"

	"golang.org/x/exp/collection/internal"
)

// MarshalJSONTo implements [encoding/json/v2.MarshalerTo].
//
// A Set marshals as a JSON array of its elements,
// each encoded as an element of a slice []E would be.
// A nil Set marshals as an empty array.
//
// If the [encoding/json/v2.Deterministic] option is true, the elements are
// sorted by their JSON encoding. (This is always the case when
// marshaling with encoding/json.) Otherwise their order is unspecified.
//
// The encoding is the same as that of a
// [golang.org/x/exp/collection/hashed.Set] with the same elements.
func (x Set[E]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return internal.MarshalSet(enc, x.All(), len(x), true)
}

// UnmarshalJSONFrom implements [encoding/json/v2.UnmarshalerFrom].
//
// Unmarshaling the Set *x from a JSON array or object causes each array
// element or object key to be unmarshalled and inserted into the Set in
// order. All object values must be true, false or {}.
//
// If the Set is nil it is allocated first.
// Unmarshaling a Set from a JSON 'null' sets it to nil.
func (x *Set[E]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return internal.UnmarshalSet[Set[E]](
		dec,
		func() { *x = nil },
		func() error {
			if *x == nil {
				*x = make(Set[E])
			}
			return nil
		},
		func(e E) { (*x)[e] = struct{}{} },
	)
}
