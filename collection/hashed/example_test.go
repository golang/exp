// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashed_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"hash/maphash"
	"log"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/exp/collection/hashed"
)

// bigIntHasher is a maphash.Hasher for [*big.Int] values
// that compares them mathematically, not by pointer.
type bigIntHasher struct{}

func (bigIntHasher) Hash(h *maphash.Hash, b *big.Int) {
	maphash.WriteComparable(h, b.Sign())
	for _, word := range b.Bits() {
		maphash.WriteComparable(h, word)
	}
}

func (bigIntHasher) Equal(x, y *big.Int) bool {
	return x.Cmp(y) == 0
}

func ExampleMap_bigInt() {
	m := hashed.NewMap[*big.Int, string](bigIntHasher{})

	// Build a map.
	i100 := big.NewInt(100)
	i200 := big.NewInt(200)
	m.Set(i100, "one hundred")
	m.Set(i200, "two hundred")
	fmt.Println("Map:", m)

	// Look up an entry using a distinct but
	// numerically equivalent big.Int.
	i100b := big.NewInt(100)
	fmt.Println("Same pointer:", i100 == i100b)
	fmt.Println("Get(i100b):", m.Get(i100b))

	// Output:
	//
	// Map: {100: one hundred, 200: two hundred}
	// Same pointer: false
	// Get(i100b): one hundred
}

func ExampleSet_bigInt() {
	s := hashed.NewSet[*big.Int](bigIntHasher{})

	// Build a set.
	i100 := big.NewInt(100)
	i200 := big.NewInt(200)
	s.Insert(i100)
	s.Insert(i200)
	fmt.Println("Set:", s)

	// Look up an entry using a distinct but
	// numerically equivalent big.Int.
	i100b := big.NewInt(100)
	fmt.Println("Same pointer:", i100 == i100b)
	fmt.Println("Contains(i100b):", s.Contains(i100b))

	// Output:
	//
	// Set: {100, 200}
	// Same pointer: false
	// Contains(i100b): true
}

// caseInsensitive is a string [maphash.Hasher] that ignores letter case.
type caseInsensitive struct{}

var _ maphash.Hasher[string] = caseInsensitive{}

func (caseInsensitive) Hash(h *maphash.Hash, s string) {
	var buf [utf8.UTFMax]byte
	for _, r := range s {
		n := utf8.EncodeRune(buf[:], unicode.ToLower(r))
		h.Write(buf[:n])
	}
}

func (caseInsensitive) Equal(x, y string) bool {
	return strings.EqualFold(x, y)
}

// To unmarshal into a nil *hashed.Map (or *hashed.Set, *sorted.Map or *sorted.Set)
// when you know the hasher or comparison function, use the feature of [encoding/json/v2]
// that allows you to provide a custom unmarshaler in the Unmarshal call.
func ExampleMap_UnmarshalJSONFrom() {
	type request struct {
		Method  string
		Headers *hashed.Map[string, string]
	}

	// UnmarshalJSONFrom cannot initialize a nil *hashed.Map because it does not
	// know which hasher to use. Install a custom unmarshaler with
	// json.WithUnmarshalers that allocates the map with the desired hasher
	// before delegating to (*hashed.Map).UnmarshalJSONFrom.
	unmarshalers := json.UnmarshalFromFunc(func(dec *jsontext.Decoder, m **hashed.Map[string, string]) error {
		if dec.PeekKind() == jsontext.KindNull {
			if _, err := dec.ReadToken(); err != nil {
				return err
			}
			*m = nil
			return nil
		}
		if *m == nil {
			*m = hashed.NewMap[string, string](caseInsensitive{})
		}
		return (*m).UnmarshalJSONFrom(dec)
	})

	data := []byte(`{"Method":"GET","Headers":{"Content-Type":"application/json","Accept":"*/*"}}`)
	var req request // req.Headers is nil
	if err := json.Unmarshal(data, &req, json.WithUnmarshalers(unmarshalers)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Method:", req.Method)
	fmt.Println("Headers:", req.Headers)
	fmt.Println("content-type:", req.Headers.Get("content-type"))

	// Output:
	// Method: GET
	// Headers: {Accept: */*, Content-Type: application/json}
	// content-type: application/json
}
