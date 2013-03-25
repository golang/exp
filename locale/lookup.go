// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package locale

import (
	"fmt"
	"sort"
	"strconv"
)

// search searchs for the insertion point of key in smap, which is a
// string with consecutive 4-byte entries. Only the first len(key)
// bytes from the start of the 4-byte entries will be considered.
func search(smap, key string) int {
	n := len(key)
	return sort.Search(len(smap)>>2, func(i int) bool {
		return smap[i<<2:][:n] >= key
	}) << 2
}

func index(smap, key string) int {
	i := search(smap, key)
	if smap[i:i+len(key)] != key {
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
// If string s is malformed, pat is returned.
func fixCase(pat, s string) string {
	if len(pat) != len(s) {
		return pat
	}
	var b []byte
	for i, r := range pat {
		c := s[i]
		if r <= 'Z' {
			if s[i] >= 'a' {
				c -= 'z' - 'Z'
			}
			if c > 'Z' || c < 'A' {
				return pat
			}
		} else {
			if s[i] <= 'Z' {
				c += 'z' - 'Z'
			}
			if c > 'z' || c < 'a' {
				return pat
			}
		}
		if b == nil {
			if s[i] == c {
				continue
			}
			b = make([]byte, len(pat))
			copy(b, s[:i])
		}
		b[i] = c
	}
	if b == nil {
		return s
	}
	return string(b)
}

type langID uint16

// getLangID returns the langID of s if s is a canonical ID
// or langUnknown if s is not a canonical langID.
func getLangID(s string) langID {
	if len(s) == 2 {
		return getLangISO2(s)
	}
	return getLangISO3(s)
}

func getMappedID(index int) langID {
	m := mappedLang[index:]
	if m[3] >= 'a' {
		return getLangISO2(m[:1] + m[3:4])
	}
	i := mappedLangID[m[3]]
	if i < 0 {
		i = -(i + 1)
		// TODO: this code can be a locale.
		return getLangISO2(altTag[i:][:2])
	}
	return langID(i)
}

// normLang returns the langID of s, canonicalizing the language
// according to BCP47 and CLDR rules.
func normLang(s string) langID {
	s = getLangID(s).String()
	if len(s) < 2 || len(s) > 3 {
		return unknownLang
	}
	if i := index(mappedLang, s); i != -1 {
		if len(s) == 3 || mappedLang[i+2] == ' ' {
			return getMappedID(i)
		}
	}
	return getLangID(s)
}

// getLangISO2 returns the langID for the given 2-letter ISO language code
// or unknownLang if this does not exist.
func getLangISO2(s string) langID {
	if len(s) == 2 {
		s = fixCase("zz", s)
		if i := index(lang, s); i != -1 && lang[i+3] != 0 {
			return langID(i >> 2)
		}
		if i := index(mappedLang, s); i != -1 && mappedLang[i+2] == ' ' {
			return getMappedID(i)
		}
	}
	return unknownLang
}

// getLangISO3 returns the langID for the given 3-letter ISO language code
// or unknownLang if this does not exist.
func getLangISO3(s string) langID {
	s = fixCase("und", s)
	// first try to match canonical 3-letter entries
	for i := search(lang, s[:2]); lang[i:i+2] == s[:2]; i += 4 {
		if lang[i+3] == 0 && lang[i+2] == s[2] {
			return langID(i >> 2)
		}
	}
	if i := index(mappedLang, s); i != -1 {
		return getMappedID(i)
	}
	// Check for non-canonical uses of ISO3.
	for i := search(lang, s[:1]); lang[i] == s[0]; i += 4 {
		if lang[i+2:][:2] == s[1:3] {
			return langID(i >> 2)
		}
	}
	return unknownLang
}

// String returns the BCP47 representation of the langID.
func (id langID) String() string {
	l := lang[id<<2:]
	if l[3] == 0 {
		return l[:3]
	}
	return l[:2]
}

func (id langID) iso3() string {
	l := lang[id<<2:]
	if l[3] == 0 {
		return l[:3]
	} else if l[2] == 0 {
		return mappedLang[l[3]<<2:][:3]
	}
	// This allocation will only happen for 3-letter ISO codes
	// that are non-canonical BCP47 language identifiers.
	return l[0:1] + l[2:4]
}

type regionID uint16

// getRegionID returns the region id for s if s is a valid 2-letter region code
// or unknownRegion.
func getRegionID(s string) regionID {
	if len(s) == 3 {
		if s[0] >= 'A' {
			return getRegionISO3(s)
		}
		if i, err := strconv.ParseUint(s, 10, 10); err == nil {
			return getRegionM49(int(i))
		}
	}
	return getRegionISO2(s)
}

// getRegionISO2 returns the regionID for the given 2-letter ISO country code
// or unknownRegion if this does not exist.
func getRegionISO2(s string) regionID {
	if i := index(regionISO, fixCase("ZZ", s)); i != -1 {
		return regionID(i>>2) + isoRegionOffset
	}
	return unknownRegion
}

// getRegionISO3 returns the regionID for the given 3-letter ISO country code
// or unknownRegion if this does not exist.
func getRegionISO3(s string) regionID {
	s = fixCase("ZZZ", s)
	for i := search(regionISO, s[:1]); regionISO[i] == s[0]; i += 4 {
		if regionISO[i+2:][:2] == s[1:3] {
			return regionID(i>>2) + isoRegionOffset
		}
	}
	for i := 0; i < len(altRegionISO3); i += 3 {
		if altRegionISO3[i:i+3] == s {
			return regionID(altRegionIDs[i/3])
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

// String returns the BCP47 representation for the region.
func (r regionID) String() string {
	if r < isoRegionOffset {
		return fmt.Sprintf("%03d", r.m49())
	}
	r -= isoRegionOffset
	return regionISO[r<<2:][:2]
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
func getScriptID(s string) scriptID {
	if i := index(script, fixCase("Zzzz", s)); i != -1 {
		return scriptID(i >> 2)
	}
	return unknownScript
}

type currencyID uint16

func getCurrencyID(s string) currencyID {
	if i := index(currency, fixCase("XXX", s)); i != -1 {
		return currencyID(i >> 2)
	}
	return unknownCurrency
}

func (c currencyID) round() int {
	return int(currency[c<<2+3] >> 2)
}

func (c currencyID) decimals() int {
	return int(currency[c<<2+3] & 0x03)
}
