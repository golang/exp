// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashed_test

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"hash/maphash"
	"io"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"golang.org/x/exp/collection/hashed"
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
	m := hashed.NewMap[string, []int](caseInsensitive{})
	m.Set("a", []int{1})
	m.Set("B", nil)
	m.Set("c", []int{2, 3})
	data, err := json.Marshal(m, json.Deterministic(true))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"B":[],"a":[1],"c":[2,3]}`; got != want {
		t.Errorf("Marshal: got %s, want %s", got, want)
	}
	m2 := hashed.NewMap[string, []int](caseInsensitive{})
	if err := json.Unmarshal(data, m2); err != nil {
		t.Fatal(err)
	}
	if got, want := m2.String(), "{B: [], a: [1], c: [2 3]}"; got != want {
		t.Errorf("Unmarshal: got %s, want %s", got, want)
	}
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
