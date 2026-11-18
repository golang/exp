// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sorted_test

import (
	"cmp"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"golang.org/x/exp/collection/internal/fmtsort"
	"golang.org/x/exp/collection/sorted"
)

// TODO(jba): factor out common JSON tests.

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

func genericCmp[T any](x, y T) int {
	return fmtsort.Compare(reflect.ValueOf(x), reflect.ValueOf(y))
}

// TestMapMarshalJSON checks that a sorted.Map marshals as a JSON object
// in the map's key order, matching a built-in map on key-type treatment,
// values, options, and errors.
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

	// Entries are always emitted in the map's key order (not lexicographic
	// JSON-name order), regardless of the Deterministic option.
	m := sorted.NewMap[int, string](cmp.Compare)
	m.Set(10, "ten")
	m.Set(9, "nine")
	m.Set(-1, "minus one")
	m.Set(100, "hundred")
	const wantOrder = `{"-1":"minus one","9":"nine","10":"ten","100":"hundred"}`
	for _, marshal := range []func(any) ([]byte, error){
		jsonv1.Marshal,
		func(x any) ([]byte, error) { return json.Marshal(x, json.Deterministic(false)) },
		func(x any) ([]byte, error) { return json.Marshal(x, json.Deterministic(true)) },
	} {
		got, err := marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != wantOrder {
			t.Errorf("got %s, want %s", got, wantOrder)
		}
	}
}

func testMarshal[K comparable, V any](t *testing.T, gomap map[K]V) {
	t.Helper()
	smap := sorted.NewMap[K, V](genericCmp[K])
	for k, v := range gomap {
		smap.Set(k, v)
	}

	// To check that marshaling works when the map is nested in something else.
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
		{"v1 indent", func(x any) ([]byte, error) { return jsonv1.MarshalIndent(x, " ", "  ") }},
		{"v2 non-deterministic", func(x any) ([]byte, error) {
			return json.Marshal(x, json.Deterministic(false))
		}},
		{"v2 deterministic", func(x any) ([]byte, error) {
			return json.Marshal(x, json.Deterministic(true))
		}},
		{"v2 options", func(x any) ([]byte, error) {
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
		{"v2 marshalers", func(x any) ([]byte, error) {
			return json.Marshal(x,
				json.Deterministic(true),
				json.WithMarshalers(json.MarshalFunc(func(s string) ([]byte, error) {
					return jsontext.AppendQuote(nil, strings.ToUpper(s))
				})))
		}},
	} {
		// Marshal the maps both at top level and nested.
		got, gotErr := test.marshal(smap)
		want, wantErr := test.marshal(gomap)
		checkMapMarshal(t, fmt.Sprintf("%s: %T", test.name, gomap), smap, got, gotErr, want, wantErr)

		got, gotErr = test.marshal(wrapper[*sorted.Map[K, V]]{1, smap, []*sorted.Map[K, V]{smap, smap}})
		want, wantErr = test.marshal(wrapper[map[K]V]{1, gomap, []map[K]V{gomap, gomap}})
		checkMapMarshal(t, fmt.Sprintf("%s: nested %T", test.name, gomap), smap, got, gotErr, want, wantErr)
	}
}

// checkMapMarshal checks that got/gotErr matches want/wantErr:
// error presence matches, unmarshaled JSON values match, and for string-keyed
// maps with default string ordering, the serialized bytes match identically.
func checkMapMarshal[K comparable, V any](t *testing.T, name string, smap *sorted.Map[K, V], got []byte, gotErr error, want []byte, wantErr error) {
	t.Helper()
	if (gotErr != nil) != (wantErr != nil) {
		t.Errorf("%s: sorted.Map error = %v, built-in map error = %v", name, gotErr, wantErr)
		return
	}
	if gotErr != nil {
		return
	}
	// When K has kind string and deterministic/v1 marshaling is used without
	// key-reordering custom marshalers, key order equals JSON-name order,
	// so the serialized bytes must match byte-for-byte (including indentation).
	if reflect.TypeFor[K]().Kind() == reflect.String &&
		!strings.Contains(name, "non-deterministic") &&
		!strings.Contains(name, "marshalers") {
		if string(got) != string(want) {
			t.Errorf("%s:\n sorted.Map: %s\nbuilt-in map: %s", name, got, want)
		}
		return
	}
	// Otherwise key order is m's cmp order rather than JSON-name order;
	// verify that both decode to the same JSON structure.
	uopts := json.JoinOptions(jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true))
	var gotv, wantv any
	if err := json.Unmarshal(got, &gotv, uopts); err != nil {
		t.Fatalf("%s: Unmarshal(%s): %v", name, got, err)
	}
	if err := json.Unmarshal(want, &wantv, uopts); err != nil {
		t.Fatalf("%s: Unmarshal(%s): %v", name, want, err)
	}
	if !reflect.DeepEqual(gotv, wantv) {
		t.Errorf("%s:\n sorted.Map: %s\nbuilt-in map: %s", name, got, want)
	}
}

// TestMapMarshalJSONDuplicateNames checks that distinct keys that
// marshal to the same JSON name are an error unless duplicate
// names are allowed, as for built-in maps.
func TestMapMarshalJSONDuplicateNames(t *testing.T) {
	smap := sorted.NewMap[constKey, int](cmp.Compare)
	smap.Set(2, 2)
	smap.Set(1, 1)
	for _, deterministic := range []bool{false, true} {
		_, err := json.Marshal(smap, json.Deterministic(deterministic))
		if err == nil || !errors.Is(err, jsontext.ErrDuplicateName) {
			t.Errorf("deterministic=%t: got error %v, want ErrDuplicateName", deterministic, err)
		}
		got, err := json.Marshal(smap, json.Deterministic(deterministic), jsontext.AllowDuplicateNames(true))
		if err != nil {
			t.Fatal(err)
		}
		// Keys 1 then 2 in key order -> values 1 then 2.
		if g := string(got); g != `{"k":1,"k":2}` {
			t.Errorf("deterministic=%t: got %s, want {\"k\":1,\"k\":2}", deterministic, g)
		}
	}
}

// TestMapUnmarshalJSON checks that a sorted.Map unmarshals
// like the built-in map with the same entries.
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

// testUnmarshal unmarshals data into a sorted.Map and into a built-in
// map, both initialized from init(), and compares the results.
// (init is a function so that the two maps do not share values.)
func testUnmarshal[K comparable, V any](t *testing.T, init func() map[K]V, data string) {
	t.Helper()
	newSorted := func() *sorted.Map[K, V] {
		smap := sorted.NewMap[K, V](genericCmp[K])
		for k, v := range init() {
			smap.Set(k, v)
		}
		return smap
	}
	entries := func(smap *sorted.Map[K, V]) map[K]V {
		if smap == nil {
			return nil
		}
		return maps.Collect(smap.All())
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
		smap := newSorted()
		gomap := init()
		gotErr := test.unmarshal([]byte(data), smap)
		wantErr := test.unmarshal([]byte(data), &gomap)
		checkUnmarshal(t, name, entries(smap), gotErr, gomap, wantErr)

		// Nested in a struct.
		// (Unlike a map, a nil *sorted.Map field cannot be populated.)
		sw := wrapper[*sorted.Map[K, V]]{M: newSorted()}
		gw := wrapper[map[K]V]{M: init()}
		nested := `{"A":1,"M":` + data + `,"B":2}`
		gotErr = test.unmarshal([]byte(nested), &sw)
		wantErr = test.unmarshal([]byte(nested), &gw)
		checkUnmarshal(t, name+" (nested)", entries(sw.M), gotErr, gw.M, wantErr)
		if sw.A != gw.A || sw.B != gw.B {
			t.Errorf("%s (nested): sorted.Map: A=%d, B=%d; built-in map: A=%d, B=%d",
				name, sw.A, sw.B, gw.A, gw.B)
		}
	}
}

func checkUnmarshal[K comparable, V any](t *testing.T, name string, got map[K]V, gotErr error, want map[K]V, wantErr error) {
	t.Helper()
	if (gotErr != nil) != (wantErr != nil) {
		t.Errorf("%s: sorted.Map error = %v, built-in map error = %v", name, gotErr, wantErr)
	}
	// A nil map is equivalent to an empty one.
	if len(got) != len(want) || len(got) > 0 && !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n  sorted.Map: %v (err = %v)\nbuilt-in map: %v (err = %v)", name, got, gotErr, want, wantErr)
	}
}

// TestMapUnmarshalJSONEquivalentKeys checks that names that unmarshal
// to equivalent keys are duplicates.
func TestMapUnmarshalJSONEquivalentKeys(t *testing.T) {
	const data = `{"Hello":1,"world":2,"HELLO":3}`

	m := sorted.NewMap[string, int](caseInsensitive)
	err := json.Unmarshal([]byte(data), m)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("got error %v, want duplicate name error", err)
	}

	// The last duplicate value wins; sorted.Map.Set keeps the existing key when
	// updating an equivalent key.
	for _, test := range []struct {
		name      string
		unmarshal func([]byte, any) error
	}{
		{"v1", jsonv1.Unmarshal},
		{"v2", func(data []byte, x any) error {
			return json.Unmarshal(data, x, jsontext.AllowDuplicateNames(true))
		}},
	} {
		m := sorted.NewMap[string, int](caseInsensitive)
		m.Set("WORLD", 0)
		m.Set("other", 4)
		if err := test.unmarshal([]byte(data), m); err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if got, want := m.String(), "{Hello: 3, other: 4, WORLD: 2}"; got != want {
			t.Errorf("%s: got %s, want %s", test.name, got, want)
		}
	}

	// A name is not a duplicate merely because it is
	// equivalent to a key that is already in the map.
	m = sorted.NewMap[string, int](caseInsensitive)
	m.Set("HELLO", 0)
	if err := json.Unmarshal([]byte(`{"hello":1,"world":2}`), m); err != nil {
		t.Fatal(err)
	}
	if got, want := m.String(), "{HELLO: 1, world: 2}"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestMapUnmarshalJSONUninitialized checks that unmarshaling
// into a Map that was not created by NewMap is an error.
func TestMapUnmarshalJSONUninitialized(t *testing.T) {
	type wrapper struct {
		M *sorted.Map[string, int]
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
	m := sorted.NewMap[string, []int](caseInsensitive)
	m.Set("c", []int{2, 3})
	m.Set("a", []int{1})
	m.Set("B", nil)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"a":[1],"B":[],"c":[2,3]}`; got != want {
		t.Errorf("Marshal: got %s, want %s", got, want)
	}
	m2 := sorted.NewMap[string, []int](caseInsensitive)
	if err := json.Unmarshal(data, m2); err != nil {
		t.Fatal(err)
	}
	if got, want := m2.String(), "{a: [1], B: [], c: [2 3]}"; got != want {
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
			smap := sorted.NewMap[string, any](cmp.Compare)
			var gomap map[string]any
			gotErr := test.decode(newReader(), smap)
			wantErr := test.decode(newReader(), &gomap)
			if !errors.Is(gotErr, errRead) || !errors.Is(wantErr, errRead) {
				t.Errorf("%s: %s: sorted.Map error = %v, built-in map error = %v; want %v",
					test.name, prefix, gotErr, wantErr, errRead)
			}
		}
	}
}

// TestSetMarshalJSON checks that a sorted.Set marshals like a slice
// of its elements in the set's order, regardless of Deterministic.
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
	testSetMarshal(t, []any{"x", 1.5, true})

	// Errors.
	testSetMarshal(t, []chan int{make(chan int), make(chan int)})
	var zero float64
	testSetMarshal(t, []float64{1, 1 / zero})

	// The order is that of the set's elements, not of their JSON encodings,
	// regardless of Deterministic.
	s := sorted.NewSet[int](cmp.Compare)
	for _, e := range []int{9, 100, 10} {
		s.Insert(e)
	}
	for _, deterministic := range []bool{false, true} {
		got, err := json.Marshal(s, json.Deterministic(deterministic))
		if err != nil {
			t.Fatal(err)
		}
		if want := `[9,10,100]`; string(got) != want {
			t.Errorf("deterministic=%t: got %s, want %s", deterministic, got, want)
		}
	}
}

func testSetMarshal[T comparable](t *testing.T, elems []T) {
	t.Helper()
	set := sorted.NewSet[T](genericCmp[T])
	for _, e := range elems {
		set.Insert(e)
	}
	// The expected result is that of marshaling a slice of the set's elements
	// in the set's order.
	inSetOrder := slices.Collect(set.All())
	if inSetOrder == nil {
		inSetOrder = []T{}
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
		{"v1 indent", func(x any) ([]byte, error) { return jsonv1.MarshalIndent(x, " ", "  ") }},
		{"v2 non-deterministic", func(x any) ([]byte, error) {
			return json.Marshal(x, json.Deterministic(false))
		}},
		{"v2 deterministic", func(x any) ([]byte, error) {
			return json.Marshal(x, json.Deterministic(true))
		}},
		{"v2 options", func(x any) ([]byte, error) {
			return json.Marshal(x,
				json.StringifyNumbers(true),
				json.FormatNilSliceAsNull(true),
				jsontext.EscapeForHTML(true),
				jsontext.WithIndent("\t"))
		}},
		{"v2 marshalers", func(x any) ([]byte, error) {
			return json.Marshal(x,
				json.WithMarshalers(json.MarshalFunc(func(s string) ([]byte, error) {
					return jsontext.AppendQuote(nil, strings.ToUpper(s))
				})))
		}},
	} {
		got, gotErr := test.marshal(set)
		want, wantErr := test.marshal(inSetOrder)
		checkBytesMarshal(t, fmt.Sprintf("%s: %T", test.name, elems), got, gotErr, want, wantErr)

		got, gotErr = test.marshal(wrapper[*sorted.Set[T]]{1, set, []*sorted.Set[T]{set, set}})
		want, wantErr = test.marshal(wrapper[[]T]{1, inSetOrder, [][]T{inSetOrder, inSetOrder}})
		checkBytesMarshal(t, fmt.Sprintf("%s: nested %T", test.name, elems), got, gotErr, want, wantErr)
	}
}

func checkBytesMarshal(t *testing.T, name string, got []byte, gotErr error, want []byte, wantErr error) {
	t.Helper()
	if (gotErr != nil) != (wantErr != nil) {
		t.Errorf("%s: sorted error = %v, slice error = %v", name, gotErr, wantErr)
		return
	}
	if gotErr != nil {
		return
	}
	if string(got) != string(want) {
		t.Errorf("%s:\n sorted: %s\n  slice: %s", name, got, want)
	}
}

func TestSetUnmarshalJSON(t *testing.T) {
	type wrapper struct {
		A int
		S *sorted.Set[int]
		B int
	}
	for _, test := range []struct {
		init    []int // initial elements
		data    string
		want    string // result of String
		wantV1  string // result under v1, if different
		wantErr bool
	}{
		// Arrays.
		{data: `[]`, want: "{}"},
		{data: `[3,1,2]`, want: "{1, 2, 3}"},
		{data: ` [ 3 , 1 ] `, want: "{1, 3}"},
		{data: `null`, want: "{}"},
		{init: []int{7, 8}, data: `[]`, want: "{7, 8}"},
		{init: []int{7, 8}, data: `[1,8]`, want: "{1, 7, 8}"},
		{init: []int{7, 8}, data: `null`, want: "{}"},
		{data: `[1,2,1,1]`, want: "{1, 2}"},

		// Objects.
		{data: `{}`, want: "{}"},
		{data: ` { } `, want: "{}"},
		{data: `{"3":true,"1":true,"2":{},"4":false}`, want: "{1, 2, 3, 4}"},
		{data: ` { "3" : true , "1" : { } , "-2" : false } `, want: "{-2, 1, 3}"},
		{init: []int{7, 8}, data: `{}`, want: "{7, 8}"},
		{init: []int{7, 8}, data: `{"1":true,"8":true}`, want: "{1, 7, 8}"},

		// Errors.
		{init: []int{7}, data: `"x"`, want: "{7}", wantErr: true},
		{init: []int{7}, data: `5`, want: "{7}", wantErr: true},
		{init: []int{7}, data: `[1,"x",3,[4],5]`, want: "{1, 7}", wantV1: "{1, 3, 5, 7}", wantErr: true},
		{data: `[1,2`, want: "{1, 2}", wantErr: true},
		{data: `[1,,2]`, want: "{1}", wantErr: true},
		{data: `[1,]`, want: "{1}", wantErr: true},
		{data: `[,1]`, want: "{}", wantErr: true},
		{data: `[1 2]`, want: "{1}", wantErr: true},
		{data: `[1}`, want: "{1}", wantErr: true},
		{init: []int{7}, data: `{"1":true,"2":5,"3":true}`, want: "{1, 7}", wantV1: "{1, 3, 7}", wantErr: true},
		{data: `{"1":"x","3":true}`, want: "{}", wantV1: "{3}", wantErr: true},
		{data: `{"1":[],"3":true}`, want: "{}", wantV1: "{3}", wantErr: true},
		{data: `{"1":{"a":1},"3":true}`, want: "{}", wantV1: "{3}", wantErr: true},
		{data: `{"1":{"a":{}},"3":true}`, want: "{}", wantV1: "{3}", wantErr: true},
		{data: `{"1":null,"3":true}`, want: "{}", wantV1: "{3}", wantErr: true},
		{data: `{"1":true,"x":true,"3":true}`, want: "{1}", wantV1: "{1, 3}", wantErr: true},
		{data: `{"1.5":true,"3":true}`, want: "{}", wantV1: "{3}", wantErr: true},
		{data: `{"1":true`, want: "{1}", wantErr: true},
		{data: `{"1":true,}`, want: "{1}", wantErr: true},
		{data: `{"1"}`, want: "{}", wantErr: true},
		{data: `{"1":}`, want: "{}", wantErr: true},
		{data: `{1:true}`, want: "{}", wantErr: true},
		{data: `{"1":true]`, want: "{1}", wantErr: true},
		{data: `{"1":true,"1":true}`, want: "{1}", wantV1: "{1}", wantErr: true}, // duplicate name (v2 only)
	} {
		newSet := func() *sorted.Set[int] {
			s := sorted.NewSet[int](cmp.Compare)
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
			wantErr := test.wantErr
			if u.name == "v1" && test.wantV1 != "" {
				want = test.wantV1
			}
			if u.name == "v1" && strings.Contains(test.data, `"1":true,"1":true`) {
				wantErr = false // v1 allows duplicate names
			}
			name := fmt.Sprintf("%s: %v <- %s", u.name, test.init, test.data)

			// Top level.
			s := newSet()
			err := u.unmarshal([]byte(test.data), s)
			if (err != nil) != wantErr {
				t.Errorf("%s: got error %v, want error = %t", name, err, wantErr)
			}
			// v1 checks the syntax of the whole input before decoding anything.
			if got := s.String(); got != want && !(u.name == "v1" && err != nil && got == newSet().String()) {
				t.Errorf("%s: got %s, want %s", name, got, want)
			}

			// Nested in a struct.
			w := wrapper{S: newSet()}
			err = u.unmarshal([]byte(`{"A":1,"S":`+test.data+`,"B":2}`), &w)
			if (err != nil) != wantErr {
				t.Errorf("%s (nested): got error %v, want error = %t", name, err, wantErr)
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

// TestSetUnmarshalJSONEquivalent checks that array elements or object keys
// that are equivalent result in a single set element.
func TestSetUnmarshalJSONEquivalent(t *testing.T) {
	s := sorted.NewSet[string](caseInsensitive)
	if err := json.Unmarshal([]byte(`["Hello","world","HELLO"]`), s); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 2 || !s.Contains("hello") || !s.Contains("WORLD") {
		t.Errorf("got %s, want two elements equivalent to hello and world", s)
	}

	s2 := sorted.NewSet[string](caseInsensitive)
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
		S *sorted.Set[string]
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
	s := sorted.NewSet[string](caseInsensitive)
	for _, e := range []string{"c", "B", "a"} {
		s.Insert(e)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `["a","B","c"]`; got != want {
		t.Errorf("Marshal: got %s, want %s", got, want)
	}
	s2 := sorted.NewSet[string](caseInsensitive)
	if err := json.Unmarshal(data, s2); err != nil {
		t.Fatal(err)
	}
	if !s2.Equal(s) {
		t.Errorf("Unmarshal: got %s, want %s", s2, s)
	}
}

// TestUnmarshalJSONKindError checks that unmarshaling a JSON value of
// the wrong kind returns an error whose Error method does not panic.
func TestUnmarshalJSONKindError(t *testing.T) {
	for _, test := range []struct {
		name      string
		unmarshal func([]byte, any) error
	}{
		{"v1", jsonv1.Unmarshal},
		{"v2", func(data []byte, x any) error { return json.Unmarshal(data, x) }},
	} {
		for _, data := range []string{`[1]`, `{"a":1}`, `"x"`, `5`, `true`} {
			m := sorted.NewMap[string, int](cmp.Compare)
			s := sorted.NewSet[int](cmp.Compare)
			for _, x := range []any{m, s} {
				_, isMap := x.(*sorted.Map[string, int])
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
				if !strings.Contains(msg, "sorted.") {
					t.Errorf("%s: %T <- %s: error %q does not mention the Go type", test.name, x, data, msg)
				}
			}
		}
	}
}
