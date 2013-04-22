// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package locale

import (
	"fmt"
	"strings"
	"testing"
)

var strdata = []string{
	"aa  ",
	"aaa ",
	"aaaa",
	"aaab",
	"aab ",
	"ab  ",
	"ba  ",
	"xxxx",
}

func strtests() map[string]int {
	return map[string]int{
		"    ": 0,
		"a":    0,
		"aa":   0,
		"aaa":  4,
		"aa ":  0,
		"aaaa": 8,
		"aaab": 12,
		"aaax": 16,
		"b":    24,
		"ba":   24,
		"bbbb": 28,
	}
}

func TestSearch(t *testing.T) {
	for k, v := range strtests() {
		if i := search(strings.Join(strdata, ""), []byte(k)); i != v {
			t.Errorf("%s: found %d; want %d", k, i, v)
		}
	}
}

func TestIndex(t *testing.T) {
	strtests := strtests()
	strtests["    "] = -1
	strtests["aaax"] = -1
	strtests["bbbb"] = -1
	for k, v := range strtests {
		if i := index(strings.Join(strdata, ""), []byte(k)); i != v {
			t.Errorf("%s: found %d; want %d", k, i, v)
		}
	}
}

func b(s string) []byte {
	return []byte(s)
}

func TestFixCase(t *testing.T) {
	tests := []string{
		"aaaa", "AbCD", "abcd",
		"Zzzz", "AbCD", "Abcd",
		"Zzzz", "AbC", "Zzzz",
		"XXX", "ab ", "XXX",
		"XXX", "usd", "USD",
		"cmn", "AB ", "cmn",
		"gsw", "CMN", "cmn",
	}
	for i := 0; i+3 < len(tests); i += 3 {
		tt := tests[i:]
		buf := [4]byte{}
		b := buf[:copy(buf[:], tt[1])]
		res := fixCase(tt[0], b)
		if res && cmp(tt[2], b) != 0 || !res && tt[0] != tt[2] {
			t.Errorf("%s+%s: found %q; want %q", tt[0], tt[1], res, tt[2])
		}
	}
}

func TestLangID(t *testing.T) {
	tests := []struct{ id, bcp47, iso3, norm string }{
		{id: "", bcp47: "und", iso3: "und"},
		{id: "  ", bcp47: "und", iso3: "und"},
		{id: "   ", bcp47: "und", iso3: "und"},
		{id: "    ", bcp47: "und", iso3: "und"},
		{id: "und", bcp47: "und", iso3: "und"},
		{id: "aju", bcp47: "aju", iso3: "aju", norm: "jrb"},
		{id: "jrb", bcp47: "jrb", iso3: "jrb"},
		{id: "es", bcp47: "es", iso3: "spa"},
		{id: "spa", bcp47: "es", iso3: "spa"},
		{id: "ji", bcp47: "ji", iso3: "yid-", norm: "yi"},
		{id: "jw", bcp47: "jw", iso3: "jav-", norm: "jv"},
		{id: "ar", bcp47: "ar", iso3: "ara"},
		{id: "arb", bcp47: "arb", iso3: "arb", norm: "ar"},
		{id: "ar", bcp47: "ar", iso3: "ara"},
		{id: "kur", bcp47: "ku", iso3: "kur"},
		{id: "nl", bcp47: "nl", iso3: "nld"},
		{id: "NL", bcp47: "nl", iso3: "nld"},
		{id: "gsw", bcp47: "gsw", iso3: "gsw"},
		{id: "gSW", bcp47: "gsw", iso3: "gsw"},
		{id: "und", bcp47: "und", iso3: "und"},
		{id: "sh", bcp47: "sh", iso3: "hbs", norm: "sr"},
		{id: "hbs", bcp47: "sh", iso3: "hbs", norm: "sr"},
		{id: "no", bcp47: "no", iso3: "nor", norm: "nb"},
		{id: "nor", bcp47: "no", iso3: "nor", norm: "nb"},
		{id: "cmn", bcp47: "cmn", iso3: "cmn", norm: "zh"},
	}
	for i, tt := range tests {
		want := getLangID(b(tt.id))
		if id := getLangISO2(b(tt.bcp47)); len(tt.bcp47) == 2 && want != id {
			t.Errorf("%d:getISO2(%s): found %v; want %v", i, tt.bcp47, id, want)
		}
		if len(tt.iso3) == 3 {
			if id := getLangISO3(b(tt.iso3)); want != id {
				t.Errorf("%d:getISO3(%s): found %q; want %q", i, tt.iso3, id, want)
			}
			if id := getLangID(b(tt.iso3)); want != id {
				t.Errorf("%d:getID3(%s): found %v; want %v", i, tt.iso3, id, want)
			}
		}
		norm := want
		if tt.norm != "" {
			norm = getLangID(b(tt.norm))
		}
		id := normLang(langOldMap[:], want)
		id = normLang(langMacroMap[:], id)
		if id != norm {
			t.Errorf("%d:norm(%s): found %v; want %v", i, tt.id, id, norm)
		}
		if id := want.String(); tt.bcp47 != id {
			t.Errorf("%d:String(): found %s; want %s", i, id, tt.bcp47)
		}
		if id := want.ISO3(); tt.iso3[:3] != id {
			t.Errorf("%d:iso3(): found %s; want %s", i, id, tt.iso3[:3])
		}
	}
}

func TestRegionID(t *testing.T) {
	tests := []struct {
		id, iso2, iso3 string
		m49            int
	}{
		{"AA", "AA", "AAA", 958},
		{"IC", "IC", "", 0},
		{"ZZ", "ZZ", "ZZZ", 999},
		{"EU", "EU", "QUU", 967},
		{"419", "", "", 419},
	}
	for i, tt := range tests {
		want := getRegionID(b(tt.id))
		if id := getRegionISO2(b(tt.iso2)); len(tt.iso2) == 2 && want != id {
			t.Errorf("%d:getISO2(%s): found %d; want %d", i, tt.iso2, id, want)
		}
		if id := getRegionISO3(b(tt.iso3)); len(tt.iso3) == 3 && want != id {
			t.Errorf("%d:getISO3(%s): found %d; want %d", i, tt.iso3, id, want)
		}
		if id := getRegionID(b(tt.iso3)); len(tt.iso3) == 3 && want != id {
			t.Errorf("%d:getID3(%s): found %d; want %d", i, tt.iso3, id, want)
		}
		if id := getRegionM49(tt.m49); tt.m49 != 0 && want != id {
			t.Errorf("%d:getM49(%d): found %d; want %d", i, tt.m49, id, want)
		}
		if len(tt.iso2) == 2 {
			if id := want.String(); tt.iso2 != id {
				t.Errorf("%d:String(): found %s; want %s", i, id, tt.iso2)
			}
		} else {
			if id := want.String(); fmt.Sprintf("%03d", tt.m49) != id {
				t.Errorf("%d:String(): found %s; want %03d", i, id, tt.m49)
			}
		}
		if id := want.iso3(); tt.iso3 != id {
			t.Errorf("%d:iso3(): found %s; want %s", i, id, tt.iso3)
		}
		if id := int(want.m49()); tt.m49 != id {
			t.Errorf("%d:m49(): found %d; want %d", i, id, tt.m49)
		}
	}
}

func TestScript(t *testing.T) {
	idx := "BbbbDdddEeeeZzzz\xff\xff\xff\xff"
	const und = unknownScript
	tests := []struct {
		in  string
		out scriptID
	}{
		{"    ", und},
		{"      ", und},
		{"  ", und},
		{"", und},
		{"Bbbb", 0},
		{"Dddd", 1},
		{"dddd", 1},
		{"dDDD", 1},
		{"Eeee", 2},
		{"Zzzz", 3},
	}
	for i, tt := range tests {
		if id := getScriptID(idx, b(tt.in)); id != tt.out {
			t.Errorf("%d:%s: found %d; want %d", i, tt.in, id, tt.out)
		}
	}
}

func TestCurrency(t *testing.T) {
	curInfo := func(round, dec int) string {
		return string(round<<2 + dec)
	}
	idx := strings.Join([]string{
		"BBB" + curInfo(5, 2),
		"DDD\x00",
		"XXX\x00",
		"ZZZ\x00",
		"\xff\xff\xff\xff",
	}, "")
	const und = unknownCurrency
	tests := []struct {
		in         string
		out        currencyID
		round, dec int
	}{
		{"   ", und, 0, 0},
		{"     ", und, 0, 0},
		{" ", und, 0, 0},
		{"", und, 0, 0},
		{"BBB", 0, 5, 2},
		{"DDD", 1, 0, 0},
		{"dDd", 1, 0, 0},
		{"ddd", 1, 0, 0},
		{"XXX", 2, 0, 0},
		{"Zzz", 3, 0, 0},
	}
	for i, tt := range tests {
		id := getCurrencyID(idx, b(tt.in))
		if id != tt.out {
			t.Errorf("%d:%s: found %d; want %d", i, tt.in, id, tt.out)
		}
		if id <= 3 {
			if d := decimals(idx, id); d != tt.dec {
				t.Errorf("%d:dec(%s): found %d; want %d", i, tt.in, d, tt.dec)
			}
			if d := round(idx, id); d != tt.round {
				t.Errorf("%d:round(%s): found %d; want %d", i, tt.in, d, tt.round)
			}
		}
	}
}
