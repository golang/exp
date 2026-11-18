// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sorted

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"reflect"

	"golang.org/x/exp/collection/internal"
)

// MarshalJSONTo implements [json.MarshalerTo].
//
// A Map marshals as a JSON object whose entries are in the map's key order,
// regardless of the [json.Deterministic] option. Key types are treated just
// like built-in map[K]V keys: each key must encode as a JSON string
// (or be of a numeric type, which is encoded as a string).
func (m *Map[K, V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return internal.MarshalMapInOrder(enc, m.All())
}

// UnmarshalJSONFrom implements [json.UnmarshalerFrom].
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
	// TODO(jba): factor out commonalities with hashed.Map.
	switch kind := dec.PeekKind(); kind {
	case jsontext.KindNull:
		_, _ = dec.ReadToken() // can't fail
		m.Clear()
		return nil
	case jsontext.KindBeginObject:
		if m.cmp == nil {
			return errors.New("sorted.Map was not created by NewMap")
		}
		_, _ = dec.ReadToken() // can't fail
	case jsontext.KindInvalid:
		_, err := dec.ReadToken()
		return err
	default:
		// The json package fills in the remaining fields.
		return &json.SemanticError{JSONKind: kind, GoType: reflect.TypeFor[Map[K, V]]()}
	}

	opts := dec.Options()
	allowDuplicates, _ := json.GetOption(opts, jsontext.AllowDuplicateNames)
	// These are true when called from encoding/json (v1).
	legacyMerge, _ := json.GetOption(opts, jsonv1.MergeWithLegacySemantics)
	legacyErrors, _ := json.GetOption(opts, jsonv1.ReportErrorsWithLegacySemantics)

	// If allowDuplicates is false, we need to detect duplicate keys in the input.
	// We do this with a set of keys.
	// Optimization: if the map is empty, all of its keys are keys from the input,
	// so we can check for duplicates against its keys instead of creating a separate
	// set.
	var seen *Set[K]
	if !allowDuplicates && m.Len() > 0 {
		seen = NewSet(m.cmp)
	}

	var firstErr error // first non-fatal error
	for dec.PeekKind() != jsontext.KindEndObject {
		// Unmarshal the key.
		//
		// Since the decoder expects an object name, the json
		// package decodes numeric keys from strings and rejects
		// key types that cannot be decoded from a JSON string,
		// just as it does for built-in maps.
		keyOffset := dec.InputOffset()
		var k K
		if err := json.UnmarshalDecode(dec, &k); err != nil {
			if internal.IsFatalUnmarshalError(err, legacyErrors) {
				return err
			}
			if err := dec.SkipValue(); err != nil {
				return err
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		v, present := m.Lookup(k)
		if present {
			// `seen == nil` means m was initially empty, so both
			// duplicates must have come from `dec`.
			if !allowDuplicates && (seen == nil || seen.Contains(k)) {
				return &jsontext.SyntacticError{
					ByteOffset:  keyOffset,
					JSONPointer: dec.StackPointer(),
					Err:         jsontext.ErrDuplicateName,
				}
			}
			// By default, a JSON value is merged (that is, unmarshaled) into the
			// existing map value, if any. In v1, the value is replaced.
			if legacyMerge {
				v = *new(V)
			}
		}
		if seen != nil {
			seen.Insert(k)
		}

		// Unmarshal the value.
		err := json.UnmarshalDecode(dec, &v)
		// Check the error after setting the value, to match existing behavior.
		m.Set(k, v)
		if err != nil {
			if internal.IsFatalUnmarshalError(err, legacyErrors) {
				return err
			}
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	_, _ = dec.ReadToken() // '}' by loop condition
	return firstErr
}

// MarshalJSONTo implements [json.MarshalerTo].
//
// A Set marshals as a JSON array of its elements in the set's order,
// regardless of the [json.Deterministic] option, each encoded as an
// element of a slice []T would be.
func (s *Set[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return internal.MarshalSet(enc, s.All(), s.Len(), false)
}

// UnmarshalJSONFrom implements [json.UnmarshalerFrom].
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
