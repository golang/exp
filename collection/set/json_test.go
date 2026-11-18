// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package set_test

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"hash/maphash"
	"slices"
	"strings"
	"testing"

	"golang.org/x/exp/collection/hashed"
	"golang.org/x/exp/collection/set"
)

// ptrElem has a MarshalJSON method with a pointer receiver.
type ptrElem struct{ x int }

func (e *ptrElem) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `"e%d"`, e.x), nil
}

// TestMarshalJSON checks that a set.Set marshals to the same JSON as
// a hashed.Set with the same elements and the natural hasher, and as a
// slice of the elements sorted by their encodings.
func TestMarshalJSON(t *testing.T) {
	testMarshal(t, []string{})
	testMarshal(t, []string{"one"})
	testMarshal(t, []string{"banana", "apple", "cherry"})
	testMarshal(t, []string{"b", "a", "", "é", "a\"<\n", "c"})
	testMarshal(t, []int{10, 9, -1, 100})
	testMarshal(t, []float64{1.5, -2, 1e30})
	testMarshal(t, []bool{true, false})
	testMarshal(t, [][2]int{{3, 4}, {1, 2}})
	testMarshal(t, []struct{ X, Y int }{{1, 2}, {0, 5}})
	testMarshal(t, []ptrElem{{2}, {1}})
	testMarshal(t, []any{"x", 1.5, nil, true})

	// Errors.
	testMarshal(t, []chan int{make(chan int), make(chan int)})
	var zero float64
	testMarshal(t, []float64{1, 1 / zero})

	for _, test := range []struct {
		set  any
		want string
	}{
		{set.Of("banana", "apple", "cherry"), `["apple","banana","cherry"]`},
		// The order is that of the encodings, not of the elements.
		{set.Of(9, 100, 10), `[10,100,9]`},
		{set.Of[int](), `[]`},
		{set.Set[int](nil), `[]`},
		{struct{ S set.Set[int] }{}, `{"S":[]}`},
		{&struct{ S set.Set[int] }{}, `{"S":[]}`},
	} {
		for _, deterministic := range []bool{false, true} {
			got, err := json.Marshal(test.set, json.Deterministic(deterministic))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want && (deterministic || len(test.want) < 10) {
				t.Errorf("v2 deterministic=%t: %v: got %s, want %s", deterministic, test.set, got, test.want)
			}
		}
		got, err := jsonv1.Marshal(test.set)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != test.want {
			t.Errorf("v1: %v: got %s, want %s", test.set, got, test.want)
		}
	}
}

func testMarshal[E comparable](t *testing.T, elems []E) {
	t.Helper()
	s := set.Of(elems...)
	hs := hashed.NewSet[E](maphash.ComparableHasher[E]{})
	for _, e := range elems {
		hs.Insert(e)
	}

	// To check that marshaling works when the set is nested in something else.
	type wrapper[S any] struct {
		A int
		S S
		B []S
	}

	check := func(name string, got []byte, gotErr error, other string, want []byte, wantErr error) {
		t.Helper()
		if (gotErr != nil) != (wantErr != nil) {
			t.Errorf("%s: set.Set error = %v, %s error = %v", name, gotErr, other, wantErr)
		} else if gotErr == nil && !bytes.Equal(got, want) {
			t.Errorf("%s:\n set.Set: %s\n%s: %s", name, got, other, want)
		}
	}

	for _, test := range []struct {
		name    string
		marshal func(any) ([]byte, error)
	}{
		{"v1", jsonv1.Marshal},
		{"v1 indent", func(x any) ([]byte, error) { return jsonv1.MarshalIndent(x, ">", "  ") }},
		{"v2 deterministic", func(x any) ([]byte, error) {
			return json.Marshal(x, json.Deterministic(true))
		}},
		{"v2 deterministic options", func(x any) ([]byte, error) {
			return json.Marshal(x,
				json.Deterministic(true),
				json.StringifyNumbers(true),
				json.FormatNilSliceAsNull(true),
				jsontext.EscapeForHTML(true),
				jsontext.WithIndent("\t"))
		}},
		{"v2 deterministic marshalers", func(x any) ([]byte, error) {
			return json.Marshal(x,
				json.Deterministic(true),
				json.WithMarshalers(json.MarshalFunc(func(s string) ([]byte, error) {
					return jsontext.AppendQuote(nil, strings.ToUpper(s))
				})))
		}},
	} {
		name := fmt.Sprintf("%s: %T", test.name, elems)

		// A slice of the elements sorted by their individual encodings.
		sorted := slices.Clone(elems)
		if sorted == nil {
			sorted = []E{}
		}
		slices.SortFunc(sorted, func(x, y E) int {
			xb, _ := test.marshal(&x)
			yb, _ := test.marshal(&y)
			return bytes.Compare(xb, yb)
		})

		got, gotErr := test.marshal(s)
		want, wantErr := test.marshal(hs)
		check(name, got, gotErr, "hashed.Set", want, wantErr)
		want, wantErr = test.marshal(sorted)
		check(name, got, gotErr, "slice", want, wantErr)

		got, gotErr = test.marshal(wrapper[set.Set[E]]{1, s, []set.Set[E]{s, s}})
		want, wantErr = test.marshal(wrapper[*hashed.Set[E]]{1, hs, []*hashed.Set[E]{hs, hs}})
		check(name+" (nested)", got, gotErr, "hashed.Set", want, wantErr)
		want, wantErr = test.marshal(wrapper[[]E]{1, sorted, [][]E{sorted, sorted}})
		check(name+" (nested)", got, gotErr, "slice", want, wantErr)
	}

	// Without Deterministic, the order of elements is unspecified,
	// so compare the elements as multisets.
	got, gotErr := json.Marshal(s)
	want, wantErr := json.Marshal(hs)
	if gotErr == nil && wantErr == nil {
		canonical := func(data []byte) []byte {
			var vals []jsontext.Value
			if err := json.Unmarshal(data, &vals); err != nil {
				t.Fatalf("v2: %T: Unmarshal(%s): %v", elems, data, err)
			}
			slices.SortFunc(vals, func(x, y jsontext.Value) int { return bytes.Compare(x, y) })
			var out []byte
			for _, v := range vals {
				out = append(append(out, v...), ',')
			}
			return out
		}
		got = canonical(got)
		want = canonical(want)
	}
	check(fmt.Sprintf("v2: %T", elems), got, gotErr, "hashed.Set", want, wantErr)
}

// TestUnmarshalJSON checks that set.Set unmarshals as expected,
// and in the same way as a hashed.Set with the natural hasher.
func TestUnmarshalJSON(t *testing.T) {
	type wrapper struct {
		A int
		S set.Set[int]
		B int
	}
	for _, test := range []struct {
		init    []int // initial elements; nil means a nil Set
		data    string
		want    string // result of String
		wantV1  string // result under v1, if different
		wantErr bool
	}{
		{data: `[]`, want: "{}"},
		{data: `[3,1,2]`, want: "{1, 2, 3}"},
		{data: ` [ 3 , 1 ] `, want: "{1, 3}"},
		{data: `null`, want: "{}"},
		// An array is merged into the existing elements.
		{init: []int{7, 8}, data: `[]`, want: "{7, 8}"},
		{init: []int{7, 8}, data: `[1,8]`, want: "{1, 7, 8}"},
		{init: []int{7, 8}, data: `null`, want: "{}"},
		// Duplicates are not an error.
		{data: `[1,2,1,1]`, want: "{1, 2}"},
		// Errors.
		{init: []int{7}, data: `{}`, want: "{7}", wantErr: true},
		{init: []int{7}, data: `"x"`, want: "{7}", wantErr: true},
		{init: []int{7}, data: `5`, want: "{7}", wantErr: true},
		// v1 carries on after a type mismatch.
		{init: []int{7}, data: `[1,"x",3,[4],5]`, want: "{1, 7}", wantV1: "{1, 3, 5, 7}", wantErr: true},
		{data: `[1,2`, want: "{1, 2}", wantErr: true},
		{data: `[1,,2]`, want: "{1}", wantErr: true},
		{data: `[1,]`, want: "{1}", wantErr: true},
		{data: `[,1]`, want: "{}", wantErr: true},
		{data: `[1 2]`, want: "{1}", wantErr: true},
		{data: `[1}`, want: "{1}", wantErr: true},
	} {
		newSet := func() set.Set[int] {
			if test.init == nil {
				return nil
			}
			return set.Of(test.init...)
		}
		newHashed := func() *hashed.Set[int] {
			hs := hashed.NewSet[int](maphash.ComparableHasher[int]{})
			for _, e := range test.init {
				hs.Insert(e)
			}
			return hs
		}
		for _, u := range []struct {
			name      string
			unmarshal func([]byte, any) error
		}{
			{"v1", jsonv1.Unmarshal},
			{"v2", func(data []byte, x any) error { return json.Unmarshal(data, x) }},
		} {
			want := test.want
			if u.name == "v1" && test.wantV1 != "" {
				want = test.wantV1
			}
			name := fmt.Sprintf("%s: %v <- %s", u.name, test.init, test.data)

			// Top level.
			s := newSet()
			err := u.unmarshal([]byte(test.data), &s)
			if (err != nil) != test.wantErr {
				t.Errorf("%s: got error %v, want error = %t", name, err, test.wantErr)
			}
			// v1 checks the syntax of the whole input before decoding anything.
			if got := s.String(); got != want && !(u.name == "v1" && err != nil && got == newSet().String()) {
				t.Errorf("%s: got %s, want %s", name, got, want)
			}
			if test.data == "null" && s != nil {
				t.Errorf("%s: got non-nil Set", name)
			}
			if err == nil && test.data != "null" && s == nil {
				t.Errorf("%s: got nil Set", name)
			}

			// The same, for a hashed.Set. (A hashed.Set replaces its
			// contents, so the results agree only if the set was empty.)
			hs := newHashed()
			herr := u.unmarshal([]byte(test.data), hs)
			if (herr != nil) != (err != nil) {
				t.Errorf("%s: set.Set error = %v, hashed.Set error = %v", name, err, herr)
			}
			if got, want := s.String(), hs.String(); len(test.init) == 0 && got != want {
				t.Errorf("%s: set.Set = %s, hashed.Set = %s", name, got, want)
			}

			// Nested in a struct.
			w := wrapper{S: newSet()}
			err = u.unmarshal([]byte(`{"A":1,"S":`+test.data+`,"B":2}`), &w)
			if (err != nil) != test.wantErr {
				t.Errorf("%s (nested): got error %v, want error = %t", name, err, test.wantErr)
			}
			if err == nil || u.name == "v1" && test.wantV1 != "" {
				if got := w.S.String(); got != want {
					t.Errorf("%s (nested): got %s, want %s", name, got, want)
				}
				if w.A != 1 || w.B != 2 {
					t.Errorf("%s (nested): got A=%d, B=%d; want 1, 2", name, w.A, w.B)
				}
			}
		}
	}
}

// TestUnmarshalJSONInPlace checks that unmarshaling
// into a non-nil Set modifies it in place.
func TestUnmarshalJSONInPlace(t *testing.T) {
	s := set.Of(1, 2)
	alias := s
	if err := json.Unmarshal([]byte(`[3,4]`), &s); err != nil {
		t.Fatal(err)
	}
	if got, want := alias.String(), "{1, 2, 3, 4}"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	type T struct {
		Name string
		Tags set.Set[string]
	}
	in := T{"x", set.Of("c", "B", "a")}
	data, err := json.Marshal(in, json.Deterministic(true))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"Name":"x","Tags":["B","a","c"]}`; got != want {
		t.Errorf("Marshal: got %s, want %s", got, want)
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != in.Name || !out.Tags.Equal(in.Tags) {
		t.Errorf("Unmarshal: got %v, want %v", out, in)
	}
}
