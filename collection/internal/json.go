// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal

// This file contains common functions for JSON serialization.
// It is tested by the client tests, for example hashed/json_test.go.

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"iter"
	"reflect"
	"slices"
)

// MarshalMapInOrder marshals the key/value pairs of seq as a JSON object
// in the order yielded by seq.
func MarshalMapInOrder[K, V any](enc *jsontext.Encoder, seq iter.Seq2[K, V]) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := MarshalMapEntries(enc, seq); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// MarshalMapEntries writes the key/value pairs of seq to enc, which must
// be positioned after the opening brace of a JSON object, in the order yielded
// by seq.
func MarshalMapEntries[K, V any](enc *jsontext.Encoder, seq iter.Seq2[K, V]) error {
	for k, v := range seq {
		// The encoder expects an object name, so it rejects
		// anything that doesn't encode as a string. Built-in
		// maps behave the same.
		//
		// Keys and values are passed by value, not by address,
		// so that the json package decides whether to call
		// marshal methods declared on pointer receivers:
		// for map keys and values, v2 does and v1 does not.
		if err := json.MarshalEncode(enc, k); err != nil {
			return KeyError[K](err)
		}
		if err := json.MarshalEncode(enc, v); err != nil {
			return err
		}
	}
	return nil
}

// KeyError adjusts an error from marshaling a key of type K
// to match the error reported for the key of a built-in map.
func KeyError[K any](err error) error {
	if errors.Is(err, jsontext.ErrNonStringName) {
		// The json package fills in the remaining fields.
		return &json.SemanticError{GoType: reflect.TypeFor[K](), Err: jsontext.ErrNonStringName}
	}
	return err
}

// Map is the interface to a map type for [UnmarshalMap].
type Map[K, V any] interface {
	Clear()
	Len() int
	Lookup(K) (V, bool)
	Set(K, V) (V, bool)
}

// Set is the interface to a set of keys for [UnmarshalMap].
type Set[K any] interface {
	Contains(K) bool
	Insert(K) bool
}

// UnmarshalMap unmarshals a JSON object or null into m, which represents a map
// of type M.
//   - On JSON null, it calls m.Clear() and returns nil.
//   - On a JSON object, if duplicate-key detection requires a separate set,
//     it calls newSet.
func UnmarshalMap[M, K, V any](dec *jsontext.Decoder, m Map[K, V], newSet func() Set[K]) error {
	switch kind := dec.PeekKind(); kind {
	case jsontext.KindNull:
		_, _ = dec.ReadToken() // can't fail
		m.Clear()
		return nil
	case jsontext.KindBeginObject:
		_, _ = dec.ReadToken() // can't fail
	case jsontext.KindInvalid:
		_, err := dec.ReadToken()
		return err
	default:
		// The json package fills in the remaining fields.
		return &json.SemanticError{JSONKind: kind, GoType: reflect.TypeFor[M]()}
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
	var seen Set[K]
	if !allowDuplicates && m.Len() > 0 {
		seen = newSet()
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

// MarshalSet marshals elems as a JSON array.
// If sort is true, elements are sorted by their JSON encodings
// when [json.Deterministic] is set and len > 1; otherwise they are written
// in the order yielded by elems.
func MarshalSet[E any](enc *jsontext.Encoder, elems iter.Seq[E], len int, sort bool) error {
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	var err error
	if deterministic, _ := json.GetOption(enc.Options(), json.Deterministic); sort && deterministic && len > 1 {
		err = marshalSetSorted(enc, elems, len)
	} else {
		err = marshalSetUnsorted(enc, elems)
	}
	if err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndArray)
}

// marshalSetUnsorted writes elems to enc, which must be
// positioned within a JSON array, in the order yielded by elems.
func marshalSetUnsorted[E any](enc *jsontext.Encoder, elems iter.Seq[E]) error {
	var e E // Reuse the same e so it only is copied to the heap once
	// (it escapes json.MarshalEncode)
	for e = range elems {
		// Elements are passed by address, so that marshal methods
		// declared on pointer receivers are called even by v1,
		// as they are for the elements of a slice.
		if err := json.MarshalEncode(enc, &e); err != nil {
			return err
		}
	}
	return nil
}

// marshalSetSorted writes elems to enc, which must be
// positioned within a JSON array, in order of their JSON encodings.
func marshalSetSorted[E any](enc *jsontext.Encoder, elems iter.Seq[E], len int) error {
	// Marshal each element separately, using the encoder's options,
	// then sort the encodings and write them out.
	// (The encoder re-indents each value as it is written.)
	opts := enc.Options()
	values := make([]jsontext.Value, 0, len)
	var e E // see similar code in marshalSetUnsorted
	for e = range elems {
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

// UnmarshalSet unmarshals a set of type S with element type E from dec.
//   - On JSON null, it calls onNull and returns nil.
//   - On a JSON array or object, it calls begin (returning its error if
//     non-nil) and calls insert for each decoded element.
func UnmarshalSet[S, E any](dec *jsontext.Decoder, onNull func(), begin func() error, insert func(E)) error {
	kind := dec.PeekKind()
	switch kind {
	case jsontext.KindNull:
		_, _ = dec.ReadToken() // can't fail
		onNull()
		return nil
	case jsontext.KindBeginArray, jsontext.KindBeginObject:
		if err := begin(); err != nil {
			return err
		}
		_, _ = dec.ReadToken() // can't fail
	case jsontext.KindInvalid:
		_, err := dec.ReadToken()
		return err
	default:
		// The json package fills in the remaining fields.
		return &json.SemanticError{JSONKind: kind, GoType: reflect.TypeFor[S]()}
	}

	// This is true when called from encoding/json (v1).
	legacyErrors, _ := json.GetOption(dec.Options(), jsonv1.ReportErrorsWithLegacySemantics)

	if kind == jsontext.KindBeginObject {
		return unmarshalSetObject[S](dec, legacyErrors, insert)
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
		insert(e)
	}
	_, _ = dec.ReadToken() // ']' by loop condition
	return firstErr
}

// unmarshalSetObject calls insert for each member name of a JSON object.
// The decoder must be positioned after the object's '{'.
func unmarshalSetObject[S, E any](dec *jsontext.Decoder, legacyErrors bool, insert func(E)) error {
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
			err := &json.SemanticError{JSONKind: val.Kind(), GoType: reflect.TypeFor[S](), Err: errSetMemberValue}
			if !legacyErrors {
				return err
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		insert(e)
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
