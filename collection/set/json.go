// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package set

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"slices"
)

// MarshalJSONTo implements [json.MarshalerTo].
//
// A Set marshals as a JSON array of its elements,
// each encoded as an element of a slice []E would be.
// A nil Set marshals as an empty array.
//
// If the [json.Deterministic] option is true, the elements are
// sorted by their JSON encoding. (This is always the case when
// marshaling with encoding/json.) Otherwise their order is unspecified.
//
// The encoding is the same as that of a
// [golang.org/x/exp/collection/hashed.Set] with the same elements.
func (x Set[E]) MarshalJSONTo(enc *jsontext.Encoder) error {
	// TODO: consider sharing under collection/internal.
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	var err error
	if deterministic, _ := json.GetOption(enc.Options(), json.Deterministic); deterministic && len(x) > 1 {
		err = x.marshalSorted(enc)
	} else {
		err = x.marshalUnsorted(enc)
	}
	if err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndArray)
}

// marshalUnsorted writes the set elements to enc, which must be
// positioned within a JSON array, in unspecified order.
func (x Set[E]) marshalUnsorted(enc *jsontext.Encoder) error {
	// TODO: consider sharing under collection/internal.
	var e E // reduce allocation: &e only escapes once
	for e = range x {
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
func (x Set[E]) marshalSorted(enc *jsontext.Encoder) error {
	// TODO: consider sharing under collection/internal.
	// Marshal each element separately, using the encoder's options,
	// then sort the encodings and write them out.
	// (The encoder re-indents each value as it is written.)
	opts := enc.Options()
	values := make([]jsontext.Value, 0, len(x))
	var e E // reduce allocation: &e only escapes once
	for e = range x {
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
// Unmarshaling the Set *x from a JSON array causes each array
// element to be unmarshalled and inserted into the Set in order.
// If the Set is nil it is allocated first.
// Unmarshaling a Set from a JSON 'null' sets it to nil.
func (x *Set[E]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	// TODO: consider sharing under collection/internal.
	switch kind := dec.PeekKind(); kind {
	case jsontext.KindNull:
		_, _ = dec.ReadToken() // can't fail
		*x = nil
		return nil
	case jsontext.KindBeginArray:
		_, _ = dec.ReadToken() // can't fail
	case jsontext.KindInvalid:
		_, err := dec.ReadToken()
		return err
	default:
		// The json package fills in the remaining fields.
		return &json.SemanticError{JSONKind: kind}
	}

	// This is true when called from encoding/json (v1).
	legacyErrors, _ := json.GetOption(dec.Options(), jsonv1.ReportErrorsWithLegacySemantics)

	if *x == nil {
		*x = make(Set[E])
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
		(*x)[e] = struct{}{}
	}
	_, _ = dec.ReadToken() // ']' by loop condition
	return firstErr
}

// isFatalUnmarshalError reports whether unmarshaling should stop
// after err, which was returned by [json.UnmarshalDecode].
// All errors are fatal, except that encoding/json (v1) carries on
// after an error that is not a syntax or I/O error, such as
// a mismatch between a JSON value and a Go type, or an error
// returned by an unmarshal method.
func isFatalUnmarshalError(err error, legacyErrors bool) bool {
	// TODO: consider sharing under collection/internal.
	if !legacyErrors {
		return true
	}
	// The json package's own classification of I/O errors is
	// not available here, so only truncated input is recognized.
	var (
		syntaxErr    *jsonv1.SyntaxError
		syntacticErr *jsontext.SyntacticError
	)
	return errors.As(err, &syntaxErr) || errors.As(err, &syntacticErr) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}
