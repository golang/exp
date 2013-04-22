// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package locale

import (
	"fmt"
	"sort"
	"strconv"
)

// get gets the string of length n for id from the given 4-byte string index.
func get(idx string, id, n int) string {
	return idx[id<<2:][:n]
}

// cmp returns an integer comparing a and b lexicographically.
func cmp(a string, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i, c := range b[:n] {
		switch {
		case a[i] > c:
			return 1
		case a[i] < c:
			return -1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// search searchs for the insertion point of key in smap, which is a
// string with consecutive 4-byte entries. Only the first len(key)
// bytes from the start of the 4-byte entries will be considered.
func search(smap string, key []byte) int {
	n := len(key)
	return sort.Search(len(smap)>>2, func(i int) bool {
		return cmp(get(smap, i, n), key) != -1
	}) << 2
}

func index(smap string, key []byte) int {
	i := search(smap, key)
	if cmp(smap[i:i+len(key)], key) != 0 {
		return -1
	}
	return i
}

func searchUint(imap []uint16, key uint16) int {
	return sort.Search(len(imap), func(i int) bool {
		return imap[i] >= key
	})
}

// fixCase reformats s to the same pattern of cases as pat.
// If returns false if string s is malformed.
func fixCase(pat string, b []byte) bool {
	if len(pat) != len(b) {
		return false
	}
	for i, c := range b {
		r := pat[i]
		if r <= 'Z' {
			if c >= 'a' {
				c -= 'z' - 'Z'
			}
			if c > 'Z' || c < 'A' {
				return false
			}
		} else {
			if c <= 'Z' {
				c += 'z' - 'Z'
			}
			if c > 'z' || c < 'a' {
				return false
			}
		}
		b[i] = c
	}
	return true
}

type langID uint16

// getLangID returns the langID of s if s is a canonical ID
// or langUnknown if s is not a canonical langID.
func getLangID(s []byte) langID {
	if len(s) == 2 {
		return getLangISO2(s)
	}
	return getLangISO3(s)
}

// mapLang returns the mapped langID of id according to mapping m.
func normLang(m []struct{ from, to uint16 }, id langID) langID {
	k := sort.Search(len(m), func(i int) bool {
		return m[i].from >= uint16(id)
	})
	if m[k].from == uint16(id) {
		return langID(m[k].to)
	}
	return id
}

// getLangISO2 returns the langID for the given 2-letter ISO language code
// or unknownLang if this does not exist.
func getLangISO2(s []byte) langID {
	if len(s) == 2 && fixCase("zz", s) {
		if i := index(lang, s); i != -1 && lang[i+3] != 0 {
			return langID(i >> 2)
		}
	}
	return unknownLang
}

const base = 'z' - 'a' + 1

func strToInt(s []byte) uint {
	v := uint(0)
	for i := 0; i < len(s); i++ {
		v *= base
		v += uint(s[i] - 'a')
	}
	return v
}

// converts the given integer to the original ASCII string passed to strToInt.
// len(s) must match the number of characters obtained.
func intToStr(v uint, s []byte) {
	for i := len(s) - 1; i >= 0; i-- {
		s[i] = byte(v%base) + 'a'
		v /= base
	}
}

// getLangISO3 returns the langID for the given 3-letter ISO language code
// or unknownLang if this does not exist.
func getLangISO3(s []byte) langID {
	if fixCase("und", s) {
		// first try to match canonical 3-letter entries
		for i := search(lang, s[:2]); cmp(lang[i:i+2], s[:2]) == 0; i += 4 {
			if lang[i+3] == 0 && lang[i+2] == s[2] {
				return langID(i >> 2)
			}
		}
		if i := index(altLangISO3, s); i != -1 {
			return langID(altLangISO3[i+3])
		}
		n := strToInt(s)
		if langNoIndex[n/8]&(1<<(n%8)) != 0 {
			return langID(n) + langNoIndexOffset
		}
		// Check for non-canonical uses of ISO3.
		for i := search(lang, s[:1]); lang[i] == s[0]; i += 4 {
			if cmp(lang[i+2:][:2], s[1:3]) == 0 {
				return langID(i >> 2)
			}
		}
	}
	return unknownLang
}

// stringToBuf writes the string to b and returns the number of bytes
// written.  cap(b) must be >= 3.
func (id langID) stringToBuf(b []byte) int {
	if id >= langNoIndexOffset {
		intToStr(uint(id)-langNoIndexOffset, b[:3])
		return 3
	}
	l := lang[id<<2:]
	if l[3] == 0 {
		return copy(b, l[:3])
	}
	return copy(b, l[:2])
}

// String returns the BCP 47 representation of the langID.
func (id langID) String() string {
	if id >= langNoIndexOffset {
		id -= langNoIndexOffset
		buf := [3]byte{}
		intToStr(uint(id), buf[:])
		return string(buf[:])
	}
	l := lang[id<<2:]
	if l[3] == 0 {
		return l[:3]
	}
	return l[:2]
}

// ISO3 returns the ISO 639-3 language code.
func (id langID) ISO3() string {
	if id >= langNoIndexOffset {
		return id.String()
	}
	l := lang[id<<2:]
	if l[3] == 0 {
		return l[:3]
	} else if l[2] == 0 {
		return get(altLangISO3, int(l[3]), 3)
	}
	// This allocation will only happen for 3-letter ISO codes
	// that are non-canonical BCP 47 language identifiers.
	return l[0:1] + l[2:4]
}

type regionID uint16

// getRegionID returns the region id for s if s is a valid 2-letter region code
// or unknownRegion.
func getRegionID(s []byte) regionID {
	if len(s) == 3 {
		if isAlpha(s[0]) {
			return getRegionISO3(s)
		}
		if i, err := strconv.ParseUint(string(s), 10, 10); err == nil {
			return getRegionM49(int(i))
		}
	}
	return getRegionISO2(s)
}

// getRegionISO2 returns the regionID for the given 2-letter ISO country code
// or unknownRegion if this does not exist.
func getRegionISO2(s []byte) regionID {
	if fixCase("ZZ", s) {
		if i := index(regionISO, s); i != -1 {
			return regionID(i>>2) + isoRegionOffset
		}
	}
	return unknownRegion
}

// getRegionISO3 returns the regionID for the given 3-letter ISO country code
// or unknownRegion if this does not exist.
func getRegionISO3(s []byte) regionID {
	if fixCase("ZZZ", s) {
		for i := search(regionISO, s[:1]); regionISO[i] == s[0]; i += 4 {
			if cmp(regionISO[i+2:][:2], s[1:3]) == 0 {
				return regionID(i>>2) + isoRegionOffset
			}
		}
		for i := 0; i < len(altRegionISO3); i += 3 {
			if cmp(altRegionISO3[i:i+3], s) == 0 {
				return regionID(altRegionIDs[i/3])
			}
		}
	}
	return unknownRegion
}

func getRegionM49(n int) regionID {
	// These will mostly be group IDs, which are at the start of the list.
	// For other values this may be a bit slow, as there are over 300 entries.
	// TODO: group id is sorted!
	if n == 0 {
		return unknownRegion
	}
	for i, v := range m49 {
		if v == uint16(n) {
			return regionID(i)
		}
	}
	return unknownRegion
}

// String returns the BCP 47 representation for the region.
func (r regionID) String() string {
	if r < isoRegionOffset {
		return fmt.Sprintf("%03d", r.m49())
	}
	r -= isoRegionOffset
	return get(regionISO, int(r), 2)
}

// The use of this is uncommon.
// Note: not all regionIDs have corresponding 3-letter ISO codes!
func (r regionID) iso3() string {
	if r < isoRegionOffset {
		return ""
	}
	r -= isoRegionOffset
	reg := regionISO[r<<2:]
	switch reg[2] {
	case 0:
		return altRegionISO3[reg[3]:][:3]
	case ' ':
		return ""
	}
	return reg[0:1] + reg[2:4]
}

func (r regionID) m49() uint16 {
	return m49[r]
}

type scriptID uint8

// getScriptID returns the script id for string s. It assumes that s
// is of the format [A-Z][a-z]{3}.
func getScriptID(idx string, s []byte) scriptID {
	if fixCase("Zzzz", s) {
		if i := index(idx, s); i != -1 {
			return scriptID(i >> 2)
		}
	}
	return unknownScript
}

// String returns the script code in title case.
func (s scriptID) String() string {
	return get(script, int(s), 4)
}

type currencyID uint16

func getCurrencyID(idx string, s []byte) currencyID {
	if fixCase("XXX", s) {
		if i := index(idx, s); i != -1 {
			return currencyID(i >> 2)
		}
	}
	return unknownCurrency
}

// String returns the upper case representation of the currency.
func (c currencyID) String() string {
	return get(currency, int(c), 3)
}

func round(index string, c currencyID) int {
	return int(index[c<<2+3] >> 2)
}

func decimals(index string, c currencyID) int {
	return int(index[c<<2+3] & 0x03)
}
