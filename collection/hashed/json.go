// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashed

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
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
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	var err error
	if deterministic, _ := json.GetOption(enc.Options(), json.Deterministic); deterministic && m.len > 1 {
		err = m.marshalSorted(enc)
	} else {
		err = m.marshalUnsorted(enc)
	}
	if err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// marshalUnsorted writes the map entries to enc, which must be
// positioned within a JSON object, in unspecified order.
func (m *Map[K, V]) marshalUnsorted(enc *jsontext.Encoder) error {
	for k, v := range m.All() {
		// The encoder expects an object name, so it rejects
		// anything that doesn't encode as a string. Built-in
		// maps behave the same.
		//
		// Keys and values are passed by value, not by address,
		// so that the json package decides whether to call
		// marshal methods declared on pointer receivers:
		// for map keys and values, v2 does and v1 does not.
		if err := json.MarshalEncode(enc, k); err != nil {
			return keyError[K](err)
		}
		if err := json.MarshalEncode(enc, v); err != nil {
			return err
		}
	}
	return nil
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
			return keyError[K](err)
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

// keyError adjusts an error from marshaling a key of type K
// to match the error reported for the key of a built-in map.
func keyError[K any](err error) error {
	if errors.Is(err, jsontext.ErrNonStringName) {
		// The json package fills in the remaining fields.
		return &json.SemanticError{GoType: reflect.TypeFor[K](), Err: jsontext.ErrNonStringName}
	}
	return err
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
	switch kind := dec.PeekKind(); kind {
	case jsontext.KindNull:
		_, _ = dec.ReadToken() // can't fail
		m.Clear()
		return nil
	case jsontext.KindBeginObject:
		if m.hasher == nil {
			return errors.New("hashed.Map was not created by NewMap")
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
	if !allowDuplicates && m.len > 0 {
		seen = NewSet(m.hasher)
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
			if isFatalUnmarshalError(err, legacyErrors) {
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
			if isFatalUnmarshalError(err, legacyErrors) {
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

// isFatalUnmarshalError reports whether unmarshaling should stop
// after err, which was returned by [json.UnmarshalDecode].
// All errors are fatal, except that encoding/json (v1) carries on
// after an error that is not a syntax or I/O error, such as
// a mismatch between a JSON value and a Go type, or an error
// returned by an unmarshal method.
func isFatalUnmarshalError(err error, legacyErrors bool) bool {
	if !legacyErrors {
		return true
	}
	// The json package's own classification of I/O errors is
	// not available here, so only truncated input is recognized.
	// That is sufficient: after any other I/O error, the next
	// read from the decoder fails too, and the callers of this
	// function treat that failure as fatal.
	var (
		syntaxErr    *jsonv1.SyntaxError
		syntacticErr *jsontext.SyntacticError
	)
	return errors.As(err, &syntaxErr) || errors.As(err, &syntacticErr) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
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
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	var err error
	if deterministic, _ := json.GetOption(enc.Options(), json.Deterministic); deterministic && s.Len() > 1 {
		err = s.marshalSorted(enc)
	} else {
		err = s.marshalUnsorted(enc)
	}
	if err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndArray)
}

// marshalUnsorted writes the set elements to enc, which must be
// positioned within a JSON array, in unspecified order.
func (s *Set[E]) marshalUnsorted(enc *jsontext.Encoder) error {
	var e E // Reuse the same e so it only is copied to the heap once
	// (it escapes json.MarshalEncode)
	for e = range s.All() {
		// Elements are passed by address, so that marshal methods
		// declared on pointer receivers are called even by v1,
		// as they are for the elements of a slice.
		if err := json.MarshalEncode(enc, &e); err != nil {
			return err
		}
	}
	return nil
}

// marshalSorted writes the set elements to enc, which must be
// positioned within a JSON array, in order of their JSON encodings.
func (s *Set[E]) marshalSorted(enc *jsontext.Encoder) error {
	// Marshal each element separately, using the encoder's options,
	// then sort the encodings and write them out.
	// (The encoder re-indents each value as it is written.)
	opts := enc.Options()
	values := make([]jsontext.Value, 0, s.Len())
	var e E // see similar code in marshalUnsorted
	for e = range s.All() {
		b, err := json.Marshal(&e, opts)
		if err != nil {
			return err
		}
		values = append(values, b)
	}
	slices.SortFunc(values, func(x, y jsontext.Value) int {
		return bytes.Compare(x, y)
	})
	for _, v := range values {
		if err := enc.WriteValue(v); err != nil {
			return err
		}
	}
	return nil
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
	kind := dec.PeekKind()
	switch kind {
	case jsontext.KindNull:
		_, _ = dec.ReadToken() // can't fail
		s.Clear()
		return nil
	case jsontext.KindBeginArray, jsontext.KindBeginObject:
		if s.m.hasher == nil {
			return errors.New("hashed.Set was not created by NewSet")
		}
		_, _ = dec.ReadToken() // can't fail
	case jsontext.KindInvalid:
		_, err := dec.ReadToken()
		return err
	default:
		// The json package fills in the remaining fields.
		return &json.SemanticError{JSONKind: kind, GoType: reflect.TypeFor[Set[E]]()}
	}

	// This is true when called from encoding/json (v1).
	legacyErrors, _ := json.GetOption(dec.Options(), jsonv1.ReportErrorsWithLegacySemantics)

	if kind == jsontext.KindBeginObject {
		return s.unmarshalObject(dec, legacyErrors)
	}
	var firstErr error // first non-fatal error
	for dec.PeekKind() != jsontext.KindEndArray {
		start := dec.InputOffset()
		var e E
		if err := json.UnmarshalDecode(dec, &e); err != nil {
			// An error is also fatal if the decoder made no progress,
			// since retrying would fail in the same way forever.
			if isFatalUnmarshalError(err, legacyErrors) || dec.InputOffset() == start {
				return err
			}
			// Omit an element that could not be decoded.
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s.Insert(e)
	}
	_, _ = dec.ReadToken() // ']' by loop condition
	return firstErr
}

// unmarshalObject adds to the set the names of the members of a
// JSON object. The decoder must be positioned after the object's '{'.
func (s *Set[E]) unmarshalObject(dec *jsontext.Decoder, legacyErrors bool) error {
	var firstErr error // first non-fatal error
	for dec.PeekKind() != jsontext.KindEndObject {
		// Unmarshal the key.
		//
		// Since the decoder expects an object name, the json
		// package decodes numeric keys from strings and rejects
		// key types that cannot be decoded from a JSON string,
		// just as it does for built-in maps.
		var e E
		keyErr := json.UnmarshalDecode(dec, &e)
		if keyErr != nil {
			if isFatalUnmarshalError(keyErr, legacyErrors) {
				return keyErr
			}
			if firstErr == nil {
				firstErr = keyErr
			}
		}

		// Check the value.
		val, err := dec.ReadValue()
		if err != nil {
			return err
		}
		if keyErr != nil {
			continue // omit an element that could not be decoded
		}
		if !isSetMemberValue(val) {
			// The json package fills in the remaining fields.
			err := &json.SemanticError{JSONKind: val.Kind(), GoType: reflect.TypeFor[Set[E]](), Err: errSetMemberValue}
			if !legacyErrors {
				return err
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s.Insert(e)
	}
	_, _ = dec.ReadToken() // '}' by loop condition
	return firstErr
}

var errSetMemberValue = errors.New("value of a set element must be a boolean or {}")

// isSetMemberValue reports whether val, a valid JSON value,
// is a boolean or an empty object.
func isSetMemberValue(val jsontext.Value) bool {
	switch val.Kind() {
	case jsontext.KindTrue, jsontext.KindFalse:
		return true
	case jsontext.KindBeginObject:
		// The value is an object with nothing but whitespace inside.
		return len(bytes.TrimSpace(val[1:len(val)-1])) == 0
	}
	return false
}
