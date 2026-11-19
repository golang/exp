// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashed

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"slices"
	"strings"

	"golang.org/x/exp/collection/internal"
)

// MarshalJSONTo implements [json.MarshalerTo].
//
// A Map marshals exactly like a built-in map[K]V with the same
// entries, including the treatment of key types: the map is encoded
// as a JSON object, and each key must encode as a JSON string
// (or be of a numeric type, which is encoded as a string).
//
// If the [json.Deterministic] option is true, the entries are
// sorted by their encoded key. (This is always the case when
// marshaling with encoding/json.)
func (m *Map[K, V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if deterministic, _ := json.GetOption(enc.Options(), json.Deterministic); deterministic && m.len > 1 {
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		if err := m.marshalSorted(enc); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.EndObject)
	}
	return internal.MarshalMapInOrder(enc, m.All())
}

// marshalSorted writes the map entries to enc, which must be
// positioned within a JSON object, in order of their JSON names.
func (m *Map[K, V]) marshalSorted(enc *jsontext.Encoder) error {
	// The JSON name of a key depends on the key type, its marshal
	// methods, and the encoder options. To find out what it is,
	// marshal all the keys, as the names of a JSON object, to a
	// temporary encoder that has the same options, then read
	// the names back.
	//
	// It is necessary to marshal the keys in object-name position.
	// json.MarshalEncode behaves differently depending on the context;
	// to get the result of marshaling a key, we must marshal the value
	// as a key and not on its own or as an array element.
	//
	// Duplicate names are permitted here so that, if they are
	// an error, it is reported by (and in the context of) enc.
	var buf bytes.Buffer
	tmp := jsontext.NewEncoder(&buf, enc.Options(), jsontext.AllowDuplicateNames(true))
	if err := tmp.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	type member struct {
		name  string // unquoted name
		value V
	}
	members := make([]member, 0, m.len)
	for k, v := range m.All() {
		if err := json.MarshalEncode(tmp, k); err != nil {
			return internal.KeyError[K](err)
		}
		// Don't encode the value here, just record it.
		// Encode it later in the final output.
		// That way we don't have to read the encoded value back in,
		// and we don't have both the original and encoded values in
		// memory at once.
		if err := tmp.WriteToken(jsontext.Null); err != nil {
			return err
		}
		members = append(members, member{value: v})
	}
	if err := tmp.WriteToken(jsontext.EndObject); err != nil {
		return err
	}

	dec := jsontext.NewDecoder(&buf,
		jsontext.AllowDuplicateNames(true),
		jsontext.AllowInvalidUTF8(true))
	if _, err := dec.ReadToken(); err != nil { // {
		return err
	}
	for i := range members {
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		members[i].name = tok.String()
		if err := dec.SkipValue(); err != nil { // null
			return err
		}
	}

	// Sort by unquoted name, as the json package does for built-in maps.
	// This can be nondeterministic if two keys encode to the same JSON key,
	// but there is no way to prevent that, and the json package has the same problem.
	slices.SortFunc(members, func(x, y member) int {
		return strings.Compare(x.name, y.name)
	})
	for _, mem := range members {
		if err := enc.WriteToken(jsontext.String(mem.name)); err != nil {
			return err
		}
		if err := json.MarshalEncode(enc, mem.value); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSONFrom implements [json.UnmarshalerFrom].
//
// A Map unmarshals exactly like a built-in map[K]V, including the
// treatment of key types: a JSON object is merged into the map,
// and a JSON null removes all its entries.
// Two object names are duplicates if they unmarshal to keys that
// are equivalent according to the map's hasher.
//
// The map must have been created by [NewMap], since unmarshaling
// cannot supply a hasher. In particular, it is an error to unmarshal
// a JSON object into a nil *Map variable or struct field.
func (m *Map[K, V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if m.hasher == nil {
		return errors.New("hashed.Map was not created by NewMap")
	}
	return internal.UnmarshalMap[Map[K, V]](
		dec,
		m,
		func() internal.Set[K] { return NewSet(m.hasher) },
	)
}

// MarshalJSONTo implements [json.MarshalerTo].
//
// A Set marshals as a JSON array of its elements,
// each encoded as an element of a slice []E would be.
//
// If the [json.Deterministic] option is true, the elements are
// sorted by their JSON encoding. (This is always the case when
// marshaling with encoding/json.) Otherwise their order is unspecified.
func (s *Set[E]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return internal.MarshalSet(enc, s.All(), s.Len(), true)
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
// cannot supply a hasher. In particular, it is an error to unmarshal
// a JSON array or object into a nil *Set variable or struct field.
func (s *Set[E]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return internal.UnmarshalSet[Set[E]](
		dec,
		s.Clear,
		func() error {
			if s.m.hasher == nil {
				return errors.New("hashed.Set was not created by NewSet")
			}
			return nil
		},
		func(e E) { s.Insert(e) },
	)
}
