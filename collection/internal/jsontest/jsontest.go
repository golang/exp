// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package jsontest provides JSON test helpers for collection packages.
package jsontest

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"testing"
)

// RoundTrip checks that `in` survives JSON marshaling and unmarshaling,
// both at top level and nested in a struct, under encoding/json (v1) and
// encoding/json/v2 (with Deterministic both true and false).
//
// If wantJSON is non-empty, it must match the deterministic v2 encoding
// of in.
//
// Before each unmarshal, newOut is called to create a fresh target of type T.
// equal reports whether the unmarshaled value matches in.
func RoundTrip[T any](t *testing.T, in T, wantJSON string, newOut func() T, equal func(T, T) bool) {
	t.Helper()
	type wrapper struct {
		A int
		X T
		B string
	}

	for _, test := range []struct {
		name          string
		checkWantJSON bool
		marshal       func(any) ([]byte, error)
		unmarshal     func([]byte, any) error
	}{
		{
			name:          "v2 deterministic",
			checkWantJSON: true,
			marshal:       func(x any) ([]byte, error) { return json.Marshal(x, json.Deterministic(true)) },
			unmarshal:     func(data []byte, x any) error { return json.Unmarshal(data, x) },
		},
		{
			name:      "v2 non-deterministic",
			marshal:   func(x any) ([]byte, error) { return json.Marshal(x, json.Deterministic(false)) },
			unmarshal: func(data []byte, x any) error { return json.Unmarshal(data, x) },
		},
		{
			name:      "v1",
			marshal:   jsonv1.Marshal,
			unmarshal: jsonv1.Unmarshal,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run("TopLevel", func(t *testing.T) {
				data, err := test.marshal(in)
				if err != nil {
					t.Fatalf("Marshal: %v", err)
				}
				if test.checkWantJSON && wantJSON != "" && string(data) != wantJSON {
					t.Errorf("Marshal: got %s, want %s", data, wantJSON)
				}
				out := newOut()
				if err := test.unmarshal(data, &out); err != nil {
					t.Fatalf("Unmarshal(%s): %v", data, err)
				}
				if !equal(out, in) {
					t.Errorf("Unmarshal: got %v, want %v", out, in)
				}
			})
			t.Run("Nested", func(t *testing.T) {
				// Nested in a struct.
				win := wrapper{A: 1, X: in, B: "z"}
				wdata, err := test.marshal(win)
				if err != nil {
					t.Fatalf("nested Marshal: %v", err)
				}
				if test.checkWantJSON && wantJSON != "" {
					wantWrapped := `{"A":1,"X":` + wantJSON + `,"B":"z"}`
					if string(wdata) != wantWrapped {
						t.Errorf("nested Marshal: got %s, want %s", wdata, wantWrapped)
					}
				}
				wout := wrapper{X: newOut()}
				if err := test.unmarshal(wdata, &wout); err != nil {
					t.Fatalf("nested Unmarshal(%s): %v", wdata, err)
				}
				if wout.A != win.A || wout.B != win.B || !equal(wout.X, in) {
					t.Errorf("nested Unmarshal: got %+v, want %+v", wout, win)
				}
			})
		})
	}
}
