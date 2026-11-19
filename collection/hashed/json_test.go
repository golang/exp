// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashed_test

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"hash/maphash"
	"io"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"golang.org/x/exp/collection/hashed"
	"golang.org/x/exp/collection/internal/jsontest"
)

// textKey has a MarshalText method with a pointer receiver.
type textKey struct{ a, b int }

func (k *textKey) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%d/%d", k.a, k.b), nil
}

func (k *textKey) UnmarshalText(data []byte) error {
	as, bs, ok := strings.Cut(string(data), "/")
	if !ok {
		return fmt.Errorf("invalid textKey %q", data)
	}
	var err error
	if k.a, err = strconv.Atoi(as); err != nil {
		return err
	}
	k.b, err = strconv.Atoi(bs)
	return err
}

// constKey always marshals to the same JSON string.
type constKey int

func (constKey) MarshalText() ([]byte, error) { return []byte("k"), nil }

// jsonNumKey marshals to a JSON number, which is not a valid object name.
type jsonNumKey int

func (jsonNumKey) MarshalJSON() ([]byte, error) { return []byte("1"), nil }

// ptrValue has a MarshalJSON method with a pointer receiver.
type ptrValue struct{ x int }

func (v *ptrValue) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `"v%d"`, v.x), nil
}

type namedString string

// TestMapMarshalJSON checks that a hashed.Map marshals
// exactly like the built-in map with the same entries.
func TestMapMarshalJSON(t *testing.T) {
	testMarshal(t, map[string]int{})
	testMarshal(t, map[string]int{"one": 1})
	testMarshal(t, map[string]int{"b": 2, "a": 1, "": 0, "é": 4, "a\"<\n": 5, "\xff": 6, "c": 3})
	testMarshal(t, map[namedString][]string{"b": {"x", "y"}, "a": nil})
	testMarshal(t, map[int]string{10: "ten", 9: "nine", -1: "minus one", 100: "hundred"})
	testMarshal(t, map[uint8]any{2: nil, 1: map[string]int{"x": 1, "a": 2}})
	testMarshal(t, map[float64]bool{1.5: true, -2: false, 1e30: true})
	testMarshal(t, map[textKey]int{{1, 2}: 3, {0, 5}: 4, {1, 1}: 5})
	testMarshal(t, map[string]ptrValue{"b": {2}, "a": {1}})
	testMarshal(t, map[string]map[string]int{"b": {"z": 1, "y": 2}, "a": {}, "c": nil})
	testMarshal(t, map[string]*int{"a": nil, "b": new(int)})

	// Errors.
	testMarshal(t, map[bool]int{true: 1, false: 0})
	testMarshal(t, map[[2]int]int{{1, 2}: 3, {3, 4}: 5})
	testMarshal(t, map[struct{ X int }]int{{1}: 1, {2}: 2})
	testMarshal(t, map[jsonNumKey]int{1: 1, 2: 2})
	testMarshal(t, map[constKey]int{1: 1})
	testMarshal(t, map[string]chan int{"a": nil, "b": nil})
	var zero float64
	testMarshal(t, map[string]float64{"a": 1, "b": 0, "c": 1 / zero})
}

func testMarshal[K comparable, V any](t *testing.T, gomap map[K]V) {
	t.Helper()
	hmap := hashed.NewMap[K, V](maphash.ComparableHasher[K]{})
	for k, v := range gomap {
		hmap.Set(k, v)
	}

	// To check that marshalling works when the map is nested in something else.
	type wrapper[M any] struct {
		A int
		M M
		B []M
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
				json.FormatNilMapAsNull(true),
				json.FormatNilSliceAsNull(true),
				jsontext.AllowDuplicateNames(true),
				jsontext.AllowInvalidUTF8(true),
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
		// Marshal the maps both at top level and nested.
		got, gotErr := test.marshal(hmap)
		want, wantErr := test.marshal(gomap)
		checkMarshal(t, fmt.Sprintf("%s: %T", test.name, gomap), got, gotErr, want, wantErr)

		got, gotErr = test.marshal(wrapper[*hashed.Map[K, V]]{1, hmap, []*hashed.Map[K, V]{hmap, hmap}})
		want, wantErr = test.marshal(wrapper[map[K]V]{1, gomap, []map[K]V{gomap, gomap}})
		checkMarshal(t, fmt.Sprintf("%s: nested %T", test.name, gomap), got, gotErr, want, wantErr)
	}

	// Without Deterministic, the order of entries is unspecified,
	// so compare the results of unmarshaling.
	got, gotErr := json.Marshal(hmap)
	want, wantErr := json.Marshal(gomap)
	if gotErr == nil && wantErr == nil {
		var gotv, wantv any
		if err := json.Unmarshal(got, &gotv); err != nil {
			t.Errorf("v2: %T: Unmarshal(%s): %v", gomap, got, err)
		}
		if err := json.Unmarshal(want, &wantv); err != nil {
			t.Fatal(err)
		}
		got, _ = json.Marshal(gotv, json.Deterministic(true))
		want, _ = json.Marshal(wantv, json.Deterministic(true))
	}
	checkMarshal(t, fmt.Sprintf("v2: %T", gomap), got, gotErr, want, wantErr)
}

func checkMarshal(t *testing.T, name string, got []byte, gotErr error, want []byte, wantErr error) {
	t.Helper()
	if (gotErr != nil) != (wantErr != nil) {
		t.Errorf("%s: hashed.Map error = %v, built-in map error = %v", name, gotErr, wantErr)
		return
	}
	if gotErr != nil {
		// The error messages differ in detail, since the
		// json package reports the type of the marshaled value.
		return
	}
	if string(got) != string(want) {
		t.Errorf("%s:\n hashed.Map: %s\nbuilt-in map: %s", name, got, want)
	}
}

// TestMapMarshalJSONDuplicateNames checks that distinct keys that
// marshal to the same JSON name are an error unless duplicate
// names are allowed, as for built-in maps. (When they are allowed,
// the order of the duplicates is unspecified even for built-in
// maps, so testMarshal cannot be used.)
func TestMapMarshalJSONDuplicateNames(t *testing.T) {
	hmap := hashed.NewMap[constKey, int](maphash.ComparableHasher[constKey]{})
	hmap.Set(1, 1)
	hmap.Set(2, 2)
	for _, deterministic := range []bool{false, true} {
		_, err := json.Marshal(hmap, json.Deterministic(deterministic))
		if err == nil || !errors.Is(err, jsontext.ErrDuplicateName) {
			t.Errorf("deterministic=%t: got error %v, want ErrDuplicateName", deterministic, err)
		}
		got, err := json.Marshal(hmap, json.Deterministic(deterministic), jsontext.AllowDuplicateNames(true))
		if err != nil {
			t.Fatal(err)
		}
		if g := string(got); g != `{"k":1,"k":2}` && g != `{"k":2,"k":1}` {
			t.Errorf("deterministic=%t: got %s", deterministic, g)
		}
	}
}

// TestMapUnmarshalJSON checks that a hashed.Map unmarshals
// exactly like the built-in map with the same entries.
func TestMapUnmarshalJSON(t *testing.T) {
	type point struct{ X, Y int }

	none := func() map[string]int { return nil }
	some := func() map[string]int { return map[string]int{"a": 5, "z": 6} }
	testUnmarshal(t, none, `{}`)
	testUnmarshal(t, none, `{"a":1,"b":2}`)
	testUnmarshal(t, none, ` { "a" : 1 , "\u00e9" : 2, "" : 3 } `)
	testUnmarshal(t, none, `null`)
	testUnmarshal(t, some, `{}`)
	testUnmarshal(t, some, `{"a":1,"b":2}`)
	testUnmarshal(t, some, `null`)
	testUnmarshal(t, func() map[namedString][]string { return nil }, `{"a":["x","y"],"b":null}`)
	testUnmarshal(t, func() map[int]string { return nil }, `{"1":"a","-2":"b"}`)
	testUnmarshal(t, func() map[uint8]any { return nil }, `{"1":{"x":[1,2]},"2":null}`)
	testUnmarshal(t, func() map[float64]bool { return nil }, `{"1.5":true,"-2":false,"1e30":true}`)
	testUnmarshal(t, func() map[textKey]int { return nil }, `{"1/2":3,"0/5":4}`)
	testUnmarshal(t, func() map[string]*int { return nil }, `{"a":null,"b":1}`)

	// Merging into existing values.
	testUnmarshal(t, func() map[string]map[string]int {
		return map[string]map[string]int{"a": {"x": 1}, "c": {"w": 0}}
	}, `{"a":{"y":2},"b":{"z":3}}`)
	testUnmarshal(t, func() map[string]point {
		return map[string]point{"a": {1, 2}, "c": {3, 4}}
	}, `{"a":{"Y":5},"b":{"X":6}}`)
	testUnmarshal(t, func() map[string]*point {
		return map[string]*point{"a": {1, 2}, "c": {3, 4}}
	}, `{"a":{"Y":5},"b":{"X":6}}`)
	testUnmarshal(t, func() map[string][]int {
		return map[string][]int{"a": {1, 2, 3}}
	}, `{"a":[4]}`)

	// Errors.
	testUnmarshal(t, some, `[1,2]`)
	testUnmarshal(t, some, `"x"`)
	testUnmarshal(t, some, `7`)
	testUnmarshal(t, some, `true`)
	testUnmarshal(t, none, `{"a":"x","b":2}`)
	testUnmarshal(t, some, `{"a":"x","b":2}`)
	testUnmarshal(t, none, `{"a":[1,{"x":2}],"b":2,"c":{},"d":4}`)
	testUnmarshal(t, func() map[int]int { return nil }, `{"x":1,"2":3}`)
	testUnmarshal(t, func() map[int]int { return nil }, `{"1.5":1,"2":3}`)
	testUnmarshal(t, func() map[uint8]int { return nil }, `{"1":1,"300":3,"2":2}`)
	testUnmarshal(t, func() map[bool]int { return nil }, `{"true":1}`)
	testUnmarshal(t, func() map[point]int { return nil }, `{"a":1}`)
	testUnmarshal(t, func() map[textKey]int { return nil }, `{"1/2":3,"bad":4,"5/6":7}`)
	testUnmarshal(t, none, `{"a":1,"a":2}`)
	testUnmarshal(t, some, `{"a":1,"a":2}`)
	testUnmarshal(t, some, `{"b":1,"c":2,"b":3}`)
	testUnmarshal(t, func() map[int]int { return nil }, `{"1":1,"2":2,"1":3}`)
	testUnmarshal(t, func() map[float64]int { return nil }, `{"0":1,"-0":2,"0.0":3}`)
	testUnmarshal(t, func() map[float64]int { return map[float64]int{0: 9} }, `{"0":1,"1":2,"0.0":3}`)
	testUnmarshal(t, none, `{"a":1`)
	testUnmarshal(t, none, `{"a":1,}`)
	testUnmarshal(t, none, `{"a":1 "b":2}`)
	testUnmarshal(t, none, `{"a"}`)
	testUnmarshal(t, none, `{1:2}`)
	testUnmarshal(t, none, `{"a":1}}`)
	testUnmarshal(t, none, ``)
}

// testUnmarshal unmarshals data into a hashed.Map and into a built-in
// map, both initialized from init(), and compares the results.
// (init is a function so that the two maps do not share values.)
func testUnmarshal[K comparable, V any](t *testing.T, init func() map[K]V, data string) {
	t.Helper()
	newHashed := func() *hashed.Map[K, V] {
		hmap := hashed.NewMap[K, V](maphash.ComparableHasher[K]{})
		for k, v := range init() {
			hmap.Set(k, v)
		}
		return hmap
	}
	entries := func(hmap *hashed.Map[K, V]) map[K]V {
		if hmap == nil {
			return nil
		}
		return maps.Collect(hmap.All())
	}
	type wrapper[M any] struct {
		A int
		M M
		B int
	}

	for _, test := range []struct {
		name      string
		unmarshal func([]byte, any) error
	}{
		{"v1", jsonv1.Unmarshal},
		{"v2", func(data []byte, x any) error { return json.Unmarshal(data, x) }},
		{"v2 options", func(data []byte, x any) error {
			return json.Unmarshal(data, x,
				jsontext.AllowDuplicateNames(true),
				jsontext.AllowInvalidUTF8(true),
				json.StringifyNumbers(true))
		}},
		{"v2 unmarshalers", func(data []byte, x any) error {
			return json.Unmarshal(data, x,
				json.WithUnmarshalers(json.UnmarshalFunc(func(data []byte, s *string) error {
					*s = "S" + string(data)
					return nil
				})))
		}},
	} {
		name := fmt.Sprintf("%s: %T <- %s", test.name, init(), data)

		// Top level.
		hmap := newHashed()
		gomap := init()
		gotErr := test.unmarshal([]byte(data), hmap)
		wantErr := test.unmarshal([]byte(data), &gomap)
		checkUnmarshal(t, name, entries(hmap), gotErr, gomap, wantErr)

		// Nested in a struct.
		// (Unlike a map, a nil *hashed.Map field cannot be populated.)
		hw := wrapper[*hashed.Map[K, V]]{M: newHashed()}
		gw := wrapper[map[K]V]{M: init()}
		nested := `{"A":1,"M":` + data + `,"B":2}`
		gotErr = test.unmarshal([]byte(nested), &hw)
		wantErr = test.unmarshal([]byte(nested), &gw)
		checkUnmarshal(t, name+" (nested)", entries(hw.M), gotErr, gw.M, wantErr)
		if hw.A != gw.A || hw.B != gw.B {
			t.Errorf("%s (nested): hashed.Map: A=%d, B=%d; built-in map: A=%d, B=%d",
				name, hw.A, hw.B, gw.A, gw.B)
		}
	}
}

func checkUnmarshal[K comparable, V any](t *testing.T, name string, got map[K]V, gotErr error, want map[K]V, wantErr error) {
	t.Helper()
	if (gotErr != nil) != (wantErr != nil) {
		t.Errorf("%s: hashed.Map error = %v, built-in map error = %v", name, gotErr, wantErr)
	}
	// A nil map is equivalent to an empty one.
	if len(got) != len(want) || len(got) > 0 && !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n  hashed.Map: %v (err = %v)\nbuilt-in map: %v (err = %v)", name, got, gotErr, want, wantErr)
	}
}

// TestMapUnmarshalJSONEquivalentKeys checks that names that unmarshal
// to equivalent keys are duplicates.
func TestMapUnmarshalJSONEquivalentKeys(t *testing.T) {
	const data = `{"Hello":1,"world":2,"HELLO":3}`

	m := hashed.NewMap[string, int](caseInsensitive{})
	err := json.Unmarshal([]byte(data), m)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("got error %v, want duplicate name error", err)
	}

	// The last duplicate wins.
	for _, test := range []struct {
		name      string
		unmarshal func([]byte, any) error
	}{
		{"v1", jsonv1.Unmarshal},
		{"v2", func(data []byte, x any) error {
			return json.Unmarshal(data, x, jsontext.AllowDuplicateNames(true))
		}},
	} {
		m := hashed.NewMap[string, int](caseInsensitive{})
		m.Set("WORLD", 0)
		m.Set("other", 4)
		if err := test.unmarshal([]byte(data), m); err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if got, want := m.String(), "{HELLO: 3, other: 4, world: 2}"; got != want {
			t.Errorf("%s: got %s, want %s", test.name, got, want)
		}
	}

	// A name is not a duplicate merely because it is
	// equivalent to a key that is already in the map.
	m = hashed.NewMap[string, int](caseInsensitive{})
	m.Set("HELLO", 0)
	if err := json.Unmarshal([]byte(`{"hello":1,"world":2}`), m); err != nil {
		t.Fatal(err)
	}
	if got, want := m.String(), "{hello: 1, world: 2}"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestMapUnmarshalJSONUninitialized checks that unmarshaling
// into a Map that was not created by NewMap is an error.
func TestMapUnmarshalJSONUninitialized(t *testing.T) {
	type wrapper struct {
		M *hashed.Map[string, int]
	}
	for _, test := range []struct {
		name      string
		unmarshal func([]byte, any) error
	}{
		{"v1", jsonv1.Unmarshal},
		{"v2", func(data []byte, x any) error { return json.Unmarshal(data, x) }},
	} {
		var w wrapper
		err := test.unmarshal([]byte(`{"M":{"a":1}}`), &w)
		if err == nil || !strings.Contains(err.Error(), "NewMap") {
			t.Errorf("%s: got error %v, want one mentioning NewMap", test.name, err)
		}

		// A null is fine.
		w = wrapper{}
		if err := test.unmarshal([]byte(`{"M":null}`), &w); err != nil {
			t.Errorf("%s: null: %v", test.name, err)
		}
		if w.M != nil {
			t.Errorf("%s: null: got non-nil Map", test.name)
		}
	}
}

func TestMapJSONRoundTrip(t *testing.T) {
	newMap := func() *hashed.Map[string, []int] {
		return hashed.NewMap[string, []int](caseInsensitive{})
	}
	m := newMap()
	m.Set("a", []int{1})
	m.Set("B", nil)
	m.Set("c", []int{2, 3})
	jsontest.RoundTrip(t, m, `{"B":[],"a":[1],"c":[2,3]}`, newMap,
		func(a, b *hashed.Map[string, []int]) bool { return a.String() == b.String() })
}

// TestMapUnmarshalJSONReadError checks that an I/O error
// while decoding a map is reported, as for built-in maps.
func TestMapUnmarshalJSONReadError(t *testing.T) {
	errRead := errors.New("read error")
	for _, prefix := range []string{`{`, `{"a"`, `{"a":`, `{"a":1,`, `{"a":1,"b":[2,`} {
		newReader := func() io.Reader {
			return io.MultiReader(strings.NewReader(prefix), iotest.ErrReader(errRead))
		}
		for _, test := range []struct {
			name   string
			decode func(io.Reader, any) error
		}{
			{"v1", func(r io.Reader, x any) error { return jsonv1.NewDecoder(r).Decode(x) }},
			{"v2", func(r io.Reader, x any) error { return json.UnmarshalRead(r, x) }},
		} {
			hmap := hashed.NewMap[string, any](maphash.ComparableHasher[string]{})
			var gomap map[string]any
			gotErr := test.decode(newReader(), hmap)
			wantErr := test.decode(newReader(), &gomap)
			if !errors.Is(gotErr, errRead) || !errors.Is(wantErr, errRead) {
				t.Errorf("%s: %s: hashed.Map error = %v, built-in map error = %v; want %v",
					test.name, prefix, gotErr, wantErr, errRead)
			}
		}
	}
}

// TestSetMarshalJSON checks that a hashed.Set marshals like a slice
// of its elements, sorted by their encodings if marshaling is deterministic.
func TestSetMarshalJSON(t *testing.T) {
	testSetMarshal(t, []string{})
	testSetMarshal(t, []string{"one"})
	testSetMarshal(t, []string{"b", "a", "", "é", "a\"<\n", "c"})
	testSetMarshal(t, []int{10, 9, -1, 100})
	testSetMarshal(t, []float64{1.5, -2, 1e30})
	testSetMarshal(t, []bool{true, false})
	testSetMarshal(t, [][2]int{{3, 4}, {1, 2}})
	testSetMarshal(t, []struct{ X, Y int }{{1, 2}, {0, 5}})
	testSetMarshal(t, []textKey{{1, 2}, {0, 5}, {1, 1}}) // pointer-receiver method
	testSetMarshal(t, []ptrValue{{2}, {1}})              // pointer-receiver method
	testSetMarshal(t, []any{"x", 1.5, nil, true})

	// Errors.
	testSetMarshal(t, []chan int{make(chan int), make(chan int)})
	var zero float64
	testSetMarshal(t, []float64{1, 1 / zero})

	// The order is that of the encodings, not of the elements.
	s := hashed.NewSet[int](maphash.ComparableHasher[int]{})
	for _, e := range []int{9, 100, 10} {
		s.Insert(e)
	}
	got, err := json.Marshal(s, json.Deterministic(true))
	if err != nil {
		t.Fatal(err)
	}
	if want := `[10,100,9]`; string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func testSetMarshal[E comparable](t *testing.T, elems []E) {
	t.Helper()
	set := hashed.NewSet[E](maphash.ComparableHasher[E]{})
	for _, e := range elems {
		set.Insert(e)
	}

	// To check that marshaling works when the set is nested in something else.
	type wrapper[S any] struct {
		A int
		S S
		B []S
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
		// The expected result is that of marshaling a slice of the
		// elements sorted by their individual encodings.
		sorted := slices.Clone(elems)
		if sorted == nil {
			sorted = []E{}
		}
		slices.SortFunc(sorted, func(x, y E) int {
			xb, _ := test.marshal(&x)
			yb, _ := test.marshal(&y)
			return bytes.Compare(xb, yb)
		})

		got, gotErr := test.marshal(set)
		want, wantErr := test.marshal(sorted)
		checkMarshal(t, fmt.Sprintf("%s: %T", test.name, elems), got, gotErr, want, wantErr)

		got, gotErr = test.marshal(wrapper[*hashed.Set[E]]{1, set, []*hashed.Set[E]{set, set}})
		want, wantErr = test.marshal(wrapper[[]E]{1, sorted, [][]E{sorted, sorted}})
		checkMarshal(t, fmt.Sprintf("%s: nested %T", test.name, elems), got, gotErr, want, wantErr)
	}

	// Without Deterministic, the order of elements is unspecified,
	// so compare the results of unmarshaling, as multisets.
	got, gotErr := json.Marshal(set)
	want, wantErr := json.Marshal(elems)
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
	checkMarshal(t, fmt.Sprintf("v2: %T", elems), got, gotErr, want, wantErr)
}

func TestSetUnmarshalJSON(t *testing.T) {
	type wrapper struct {
		A int
		S *hashed.Set[int]
		B int
	}
	for _, test := range []struct {
		init    []int // initial elements
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
		newSet := func() *hashed.Set[int] {
			s := hashed.NewSet[int](maphash.ComparableHasher[int]{})
			for _, e := range test.init {
				s.Insert(e)
			}
			return s
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
			err := u.unmarshal([]byte(test.data), s)
			if (err != nil) != test.wantErr {
				t.Errorf("%s: got error %v, want error = %t", name, err, test.wantErr)
			}
			// v1 checks the syntax of the whole input before decoding anything.
			if got := s.String(); got != want && !(u.name == "v1" && err != nil && got == newSet().String()) {
				t.Errorf("%s: got %s, want %s", name, got, want)
			}

			// Nested in a struct.
			w := wrapper{S: newSet()}
			err = u.unmarshal([]byte(`{"A":1,"S":`+test.data+`,"B":2}`), &w)
			if (err != nil) != test.wantErr {
				t.Errorf("%s (nested): got error %v, want error = %t", name, err, test.wantErr)
			}
			if err == nil || u.name == "v1" && test.wantV1 != "" {
				// A null sets a pointer field to nil without calling the method.
				if w.S == nil {
					if test.data != "null" {
						t.Errorf("%s (nested): got nil Set", name)
					}
				} else if got := w.S.String(); got != want {
					t.Errorf("%s (nested): got %s, want %s", name, got, want)
				}
				if w.A != 1 || w.B != 2 {
					t.Errorf("%s (nested): got A=%d, B=%d; want 1, 2", name, w.A, w.B)
				}
			}
		}
	}
}

// TestSetUnmarshalJSONEquivalent checks that array elements
// that are equivalent result in a single set element.
func TestSetUnmarshalJSONEquivalent(t *testing.T) {
	s := hashed.NewSet[string](caseInsensitive{})
	if err := json.Unmarshal([]byte(`["Hello","world","HELLO"]`), s); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 2 || !s.Contains("hello") || !s.Contains("WORLD") {
		t.Errorf("got %s, want two elements equivalent to hello and world", s)
	}

	// Object keys that are equivalent also collapse without error
	// (distinct JSON names, equivalent according to the hasher).
	s2 := hashed.NewSet[string](caseInsensitive{})
	if err := json.Unmarshal([]byte(`{"Hello":true,"world":{},"HELLO":false}`), s2); err != nil {
		t.Fatal(err)
	}
	if s2.Len() != 2 || !s2.Contains("hello") || !s2.Contains("WORLD") {
		t.Errorf("got %s, want two elements equivalent to hello and world", s2)
	}
}

// TestSetUnmarshalJSONUninitialized checks that unmarshaling
// into a Set that was not created by NewSet is an error.
func TestSetUnmarshalJSONUninitialized(t *testing.T) {
	type wrapper struct {
		S *hashed.Set[string]
	}
	for _, test := range []struct {
		name      string
		unmarshal func([]byte, any) error
	}{
		{"v1", jsonv1.Unmarshal},
		{"v2", func(data []byte, x any) error { return json.Unmarshal(data, x) }},
	} {
		for _, input := range []string{`{"S":["a"]}`, `{"S":{"a":true}}`} {
			var w wrapper
			err := test.unmarshal([]byte(input), &w)
			if err == nil || !strings.Contains(err.Error(), "NewSet") {
				t.Errorf("%s (%s): got error %v, want one mentioning NewSet", test.name, input, err)
			}
		}

		// A null is fine.
		var w wrapper
		if err := test.unmarshal([]byte(`{"S":null}`), &w); err != nil {
			t.Errorf("%s: null: %v", test.name, err)
		}
		if w.S != nil {
			t.Errorf("%s: null: got non-nil Set", test.name)
		}
	}
}

func TestSetJSONRoundTrip(t *testing.T) {
	newSet := func() *hashed.Set[string] {
		return hashed.NewSet[string](caseInsensitive{})
	}
	s := newSet()
	for _, e := range []string{"c", "B", "a"} {
		s.Insert(e)
	}
	jsontest.RoundTrip(t, s, `["B","a","c"]`, newSet, (*hashed.Set[string]).Equal)
}

// TestUnmarshalJSONKindError checks that unmarshaling a JSON value of
// the wrong kind returns an error whose Error method does not panic.
// (With encoding/json, it used to dereference a nil reflect.Type.)
func TestUnmarshalJSONKindError(t *testing.T) {
	for _, test := range []struct {
		name      string
		unmarshal func([]byte, any) error
	}{
		{"v1", jsonv1.Unmarshal},
		{"v2", func(data []byte, x any) error { return json.Unmarshal(data, x) }},
	} {
		for _, data := range []string{`[1]`, `{"a":1}`, `"x"`, `5`, `true`} {
			m := hashed.NewMap[string, int](maphash.ComparableHasher[string]{})
			s := hashed.NewSet[int](maphash.ComparableHasher[int]{})
			for _, x := range []any{m, s} {
				_, isMap := x.(*hashed.Map[string, int])
				if isMap && data[0] == '{' || !isMap && (data[0] == '[' || data[0] == '{') {
					continue // the right kind
				}
				err := test.unmarshal([]byte(data), x)
				if err == nil {
					t.Errorf("%s: %T <- %s: got nil error", test.name, x, data)
					continue
				}
				// Error must not panic, and should mention the Go type.
				msg := err.Error()
				if !strings.Contains(msg, "hashed.") {
					t.Errorf("%s: %T <- %s: error %q does not mention the Go type", test.name, x, data, msg)
				}
			}
		}
	}
}
