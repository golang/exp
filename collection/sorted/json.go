// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sorted

import (
	"encoding/json/jsontext"
	"errors"

	"golang.org/x/exp/collection/internal"
)

// MarshalJSONTo implements [encoding/json/v2.MarshalerTo].
//
// A Map marshals as a JSON object whose entries are in the map's key order,
// regardless of the [encoding/json/v2.Deterministic] option. Key types are treated just
// like built-in map[K]V keys: each key must encode as a JSON string
// (or be of a numeric type, which is encoded as a string).
func (m *Map[K, V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return internal.MarshalMapInOrder(enc, m.All())
}

// UnmarshalJSONFrom implements [encoding/json/v2.UnmarshalerFrom].
//
// A Map unmarshals like a built-in map[K]V, including the
// treatment of key types: a JSON object is merged into the map,
// and a JSON null removes all its entries.
// Two object names are duplicates if they unmarshal to keys that
// are equivalent according to the map's comparison function.
//
// The map must have been created by [NewMap], since unmarshaling
// cannot supply a comparison function. In particular, it is an error to
// unmarshal a JSON object into a nil *Map variable or struct field.
func (m *Map[K, V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if m.cmp == nil {
		return errors.New("sorted.Map was not created by NewMap")
	}
	return internal.UnmarshalMap[Map[K, V]](
		dec,
		m,
		func() internal.Set[K] { return NewSet(m.cmp) },
	)
}

// MarshalJSONTo implements [encoding/json/v2.MarshalerTo].
//
// A Set marshals as a JSON array of its elements in the set's order,
// regardless of the [encoding/json/v2.Deterministic] option, each encoded as an
// element of a slice []T would be.
func (s *Set[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return internal.MarshalSet(enc, s.All(), s.Len(), false)
}

// UnmarshalJSONFrom implements [encoding/json/v2.UnmarshalerFrom].
//
// Unmarshaling the Set *s from a JSON array or object causes each array
// element or object key to be unmarshalled and inserted into the Set in
// order. All object values must be true, false or {}.
//
// Unmarshaling a Set from a JSON 'null' clears it.
//
// The set must have been created by [NewSet], since unmarshaling
// cannot supply a comparison function. In particular, it is an error to
// unmarshal a JSON array or object into a nil *Set variable or struct field.
func (s *Set[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return internal.UnmarshalSet[Set[T]](
		dec,
		s.Clear,
		func() error {
			if s.m.cmp == nil {
				return errors.New("sorted.Set was not created by NewSet")
			}
			return nil
		},
		func(e T) { s.Insert(e) },
	)
}
