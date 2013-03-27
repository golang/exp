// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Locale identifier table generator.
// Data read from the web.

package main

import (
	"bufio"
	"code.google.com/p/go.exp/locale/cldr"
	"flag"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

var (
	url = flag.String("cldr",
		"http://www.unicode.org/Public/cldr/22/core.zip",
		"URL of CLDR archive.")
	test = flag.Bool("test", false,
		"test existing tables; can be used to compare web data with package data.")
	localFiles = flag.Bool("local", false,
		"data files have been copied to the current directory; for debugging only.")
)

var comment = []string{
	`
lang holds an alphabetically sorted list of bcp47 language identifiers.
All entries are 4 bytes. The index of the identifier (divided by 4) is the language ID.
For 2-byte language identifiers, the two successive bytes have the following meaning:
    - if the first letter of the 2- and 3-letter ISO codes are the same:
      the second and third letter of the 3-letter ISO code.
    - otherwise: a 0 and a by 2 bits right-shifted index into mappedLang.
For 3-byte language identifiers the 4th byte is 0.`,
	`
mappedLang holds an alphabetically sorted list of non-canonical language
identifiers (by definition of BCP47 or CLDR) with a mapping to their cannonical
equivalents. Each entry is 4 bytes.  The first 3 bytes are the language code.
(May be a 2-letter code followed by a space.) The 4th byte is one of the following values:
    - [a-z]:   The canonical code is the first letter of the non-canonical code plus
               this character. The majority of mappings can be expressed this way.
    - [0-'a']: Index into mappedLangID, an array of language ids.`,
	`
mappedLangID holds a list of language IDs, which correspond to the 4-byte index
into lang. A negative index indicates a mapping to a tag.`,
	`
tagAlias holds a mapping from legacy and grandfathered tags to their locale ID.`,
	`
scripts is an alphabetically sorted list of ISO 15924 codes. The index
of the script in the string, divided by 4, is the internal script ID.`,
	`
isoRegionOffset needs to be added to the index of regionISO to obtain the regionID
for 2-letter ISO codes. (The first isoRegionOffset regionIDs are reserved for
the UN.M49 codes used for groups.)`,
	`
regionISO holds a list of alphabetically sorted 2-letter ISO region codes.
Each 2-letter codes is followed by two bytes with the following meaning:
    - [A-Z}{2}: the first letter of the 2-letter code plus these two 
                letters form the 3-letter ISO code.
    - 0, n:     index into altRegionISO3.`,
	`
m49 maps regionIDs to UN.M49 codes. The first isoRegionOffset entries are
codes indicating collections of regions.`,
	`
altRegionISO3 holds a list of 3-letter region codes that cannot be
mapped to 2-letter codes using the default algorithm. This is a short list.`,
	`
altRegionIDs holsd a list of regionIDs the positions of which match those
of the 3-letter ISO codes in altRegionISO3.`,
	`
currency holds an alphabetically sorted list of canonical 3-letter currency identifiers.
Each identifier is followed by a byte of which the 6 most significant bits
indicated the rounding and the least 2 significant bits indicate the
number of decimal positions.`,
}

// TODO: consider changing some of these strutures to tries. This can reduce
// memory, but may increase the need for memory allocations. This could be
// mitigated if we can piggyback on locale strings for common cases.

func failOnError(e error) {
	if e != nil {
		log.Panic(e)
	}
}

type setType int

const (
	Indexed setType = 1 + iota // all elements must be of same size
	Linear
)

type stringSet struct {
	s              []string
	sorted, frozen bool

	// We often need to update values after the creation of an index is completed.
	// We include a convenience map for keeping track of this.
	update map[string]string
	typ    setType // used for checking.
}

func (ss *stringSet) clone() stringSet {
	c := *ss
	c.s = append([]string(nil), c.s...)
	return c
}

func (ss *stringSet) setType(t setType) {
	if ss.typ != t && ss.typ != 0 {
		log.Panicf("type %d cannot be assigned as it was already %d", t, ss.typ)
	}
}

// parse parses a whitespace-separated string and initializes ss with its
// components.
func (ss *stringSet) parse(s string) {
	scan := bufio.NewScanner(strings.NewReader(s))
	scan.Split(bufio.ScanWords)
	for scan.Scan() {
		ss.add(scan.Text())
	}
}

func (ss *stringSet) assertChangeable() {
	if ss.frozen {
		log.Panic("attempt to modify a frozen stringSet")
	}
}

func (ss *stringSet) add(s string) {
	ss.assertChangeable()
	ss.s = append(ss.s, s)
	ss.sorted = ss.frozen
}

func (ss *stringSet) freeze() {
	ss.compact()
	ss.frozen = true
}

func (ss *stringSet) compact() {
	if ss.sorted {
		return
	}
	a := ss.s
	sort.Strings(a)
	k := 0
	for i := 1; i < len(a); i++ {
		if a[k] != a[i] {
			a[k+1] = a[i]
			k++
		}
	}
	ss.s = a[:k+1]
	ss.sorted = ss.frozen
}

func (ss *stringSet) remove(s string) {
	ss.assertChangeable()
	if i, ok := ss.find(s); ok {
		copy(ss.s[i:], ss.s[i+1:])
		ss.s = ss.s[:len(ss.s)-1]
	}
}

func (ss *stringSet) replace(ol, nu string) {
	ss.s[ss.index(ol)] = nu
	ss.sorted = ss.frozen
}

func (ss *stringSet) index(s string) int {
	ss.setType(Indexed)
	i, ok := ss.find(s)
	if !ok {
		log.Println(ss.s)
		if i < len(ss.s) {
			log.Panicf("find: item %q is not in list. Closest match is %q.", s, ss.s[i])
		}
		log.Panicf("find: item %q is not in list", s)

	}
	return i
}

func (ss *stringSet) find(s string) (int, bool) {
	ss.compact()
	i := sort.SearchStrings(ss.s, s)
	return i, i != len(ss.s) && ss.s[i] == s
}

func (ss *stringSet) slice() []string {
	ss.compact()
	return ss.s
}

func (ss *stringSet) updateLater(v, key string) {
	if ss.update == nil {
		ss.update = map[string]string{}
	}
	ss.update[v] = key
}

// join joins the string and ensures that all entries are of the same length.
func (ss *stringSet) join() string {
	ss.setType(Indexed)
	n := len(ss.s[0])
	for _, s := range ss.s {
		if len(s) != n {
			log.Panic("join: not all entries are of the same length")
		}
	}
	ss.s = append(ss.s, strings.Repeat("\xff", n))
	return strings.Join(ss.s, "")
}

type builder struct {
	w      io.Writer   // multi writer
	out    io.Writer   // set to Stdout
	hash32 hash.Hash32 // for checking whether tables have changed.
	size   int
	data   *cldr.CLDR
	supp   *cldr.SupplementalData

	// indices
	locale   stringSet // common locales
	lang     stringSet // canonical language ids (2 or 3 letter ISO codes)
	script   stringSet // 4-letter ISO codes
	region   stringSet // 2-letter ISO or 3-digit UN M49 codes
	currency stringSet // 3-letter ISO currency codes
}

func newBuilder(url *string) *builder {
	if *localFiles {
		pwd, _ := os.Getwd()
		*url = "file://" + path.Join(pwd, path.Base(*url))
	}
	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
	c := &http.Client{Transport: t}
	resp, err := c.Get(*url)
	failOnError(err)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf(`bad GET status for "%s": %s`, *url, resp.Status)
	}
	d := &cldr.Decoder{}
	d.SetDirFilter("supplemental")
	data, err := d.DecodeZip(resp.Body)
	failOnError(err)
	b := builder{
		out:    os.Stdout,
		data:   data,
		supp:   data.Supplemental(),
		hash32: fnv.New32(),
	}
	b.w = io.MultiWriter(b.out, b.hash32)
	return &b
}

var commentIndex = make(map[string]string)

func init() {
	for _, s := range comment {
		key := strings.TrimSpace(strings.SplitN(s, " ", 2)[0])
		commentIndex[key] = strings.Replace(s, "\n", "\n// ", -1)
	}
}

func (b *builder) comment(name string) {
	fmt.Fprintln(b.out, commentIndex[name])
}

func (b *builder) pf(f string, x ...interface{}) {
	fmt.Fprintf(b.w, f, x...)
	fmt.Fprint(b.w, "\n")
}

func (b *builder) p(x ...interface{}) {
	fmt.Fprintln(b.w, x...)
}

func (b *builder) addSize(s int) {
	b.size += s
	b.pf("// Size: %d bytes", s)
}

func (b *builder) addArraySize(s, n int) {
	b.size += s
	b.pf("// Size: %d bytes, %d elements", s, n)
}

func (b *builder) writeConst(name string, x interface{}) {
	b.comment(name)
	b.pf("const %s = %v", name, x)
}

func (b *builder) writeSlice(name string, ss interface{}) {
	b.comment(name)
	v := reflect.ValueOf(ss)
	t := v.Type().Elem()
	b.addArraySize(v.Len()*int(t.Size()), v.Len())
	fmt.Fprintf(b.w, `var %s = [%d]%s{`, name, v.Len(), t)
	for i := 0; i < v.Len(); i++ {
		if i%12 == 0 {
			fmt.Fprintf(b.w, "\n\t")
		}
		fmt.Fprintf(b.w, "%+v, ", v.Index(i).Interface())
	}
	b.p("\n}")
}

// writeStringSlice writes a slice of strings. This produces a lot
// of overhead. It should typically only be used for debugging.
// TODO: remove
func (b *builder) writeStringSlice(name string, ss []string) {
	b.comment(name)
	t := reflect.TypeOf(ss).Elem()
	sz := len(ss) * int(t.Size())
	for _, s := range ss {
		sz += len(s)
	}
	b.addArraySize(sz, len(ss))
	b.pf(`var %s = [%d]%s{`, name, len(ss), t)
	for i := 0; i < len(ss); i++ {
		b.pf("\t%q,", ss[i])
	}
	b.p("}")
}

func (b *builder) writeString(name, s string) {
	b.comment(name)
	b.addSize(len(s) + int(reflect.TypeOf(s).Size()))
	if len(s) < 40 {
		b.pf(`var %s string = %q`, name, s)
		return
	}
	const cpl = 60
	b.pf(`var %s string = "" +`, name)
	for {
		n := cpl
		if n > len(s) {
			n = len(s)
		}
		var q string
		for {
			q = strconv.Quote(s[:n])
			if len(q) <= cpl+2 {
				break
			}
			n--
		}
		if n < len(s) {
			b.pf(`	%s +`, q)
			s = s[n:]
		} else {
			b.pf(`	%s`, q)
			break
		}
	}
}

// TODO: convert this type into a list or two-stage trie.
func (b *builder) writeMapFunc(name string, m map[string]string, f func(string) uint16) {
	b.comment(name)
	v := reflect.ValueOf(m)
	sz := v.Len() * (2 + int(v.Type().Key().Size()))
	for _, k := range m {
		sz += len(k)
	}
	b.addSize(sz)
	keys := []string{}
	b.pf(`var %s = map[string]uint16{`, name)
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.pf("\t%q: %v,", k, f(m[k]))
	}
	b.p("}")
}

func (ss *stringSet) parseKeyed(slice interface{}, key, value string) {
	v := reflect.ValueOf(slice)
	for i := 0; i < v.Len(); i++ {
		if v.Index(i).Elem().FieldByName(key).String() == value {
			ss.parse(v.Index(i).Interface().(cldr.Elem).GetCommon().Data())
			break
		}
	}
}

func (b *builder) parseIndices() {
	meta := b.supp.Metadata

	// canonical language codes
	b.lang.parseKeyed(meta.Validity.Variable, "Id", "$language")
	for _, a := range meta.Alias.LanguageAlias {
		if r := a.Replacement; len(r) >= 2 && len(r) <= 3 {
			b.lang.add(r)
		}
		if a.Reason == "macrolanguage" {
			b.lang.add(a.Type)
		}
		remove := a.Reason == "overlong" || a.Reason == "deprecated"
		if remove {
			b.lang.remove(a.Type)
		}
	}
	b.lang.remove("root")

	// script codes
	b.script.parseKeyed(meta.Validity.Variable, "Id", "$script")

	// canonical regions codes
	for _, g := range b.supp.TerritoryContainment.Group {
		if len(g.Type) == 3 { // UN M49 code
			b.region.add(g.Type)
		}
	}
	for _, tc := range b.supp.CodeMappings.TerritoryCodes {
		b.region.add(tc.Type)
	}

	// currency codes
	b.currency.parseKeyed(meta.Validity.Variable, "Id", "$currency")

	// common locales
	b.locale.parse(meta.DefaultContent.Locales)
}

// writeLanguage generates all tables needed for language canonicalization.
func (b *builder) writeLanguage() {
	meta := b.supp.Metadata

	b.writeConst("unknownLang", b.lang.index("und"))

	// Get language codes that need to be mapped (overlong 3-letter codes, deprecated
	// 2-letter codes and grandfathered tags.
	mappedLang := stringSet{}

	// langSpecial maps from non-canonical to canonical ISO language codes.
	// TODO: Map to Locale id, instead of language.  This allows sh and bhs to be
	// mapped to sr_Latn.
	langSpecial := stringSet{}

	// legacyTag maps from tag to language code.
	legacyTag := make(map[string]string)

	lang := b.lang.clone()
	for _, a := range meta.Alias.LanguageAlias {
		if a.Replacement == "" {
			a.Replacement = "und"
		}
		if len(a.Type) <= 3 {
			code := fmt.Sprintf("%-3s", a.Type)
			if len(a.Replacement) != 2 || a.Type[0] != a.Replacement[0] {
				langSpecial.add(a.Replacement)
				mappedLang.updateLater(code, a.Replacement)
				mappedLang.add(code)
			} else if a.Reason != "overlong" || len(a.Type) != 3 {
				code += a.Replacement[1:]
				mappedLang.add(code)
			}
			if a.Reason == "overlong" && len(a.Type) == 3 && len(a.Replacement) == 2 {
				lang.updateLater(a.Replacement, a.Type)
			}
		} else {
			legacyTag[strings.Replace(a.Type, "_", "-", -1)] = a.Replacement
		}
	}

	// Complete canonialized language tags.
	lang.freeze()
	for i, v := range lang.s {
		// We can avoid these manual entries by using the IANI registry directly.
		// Seems easier to update the list manually, as changes are rare.
		// The panic in this loop will trigger if we miss an entry.
		lang.update["no"] = "nor"
		lang.update["sh"] = "scr"
		lang.update["tl"] = "tgl"
		lang.update["tw"] = "twi"
		// Fix CLDR ambiguities.
		lang.update["nb"] = "nob"
		lang.update["ak"] = "aka"
		add := ""
		if s, ok := lang.update[v]; ok {
			if s[0] == v[0] {
				add = s[1:]
			} else {
				add = string([]byte{0, byte(mappedLang.index(s))})
			}
		} else if len(v) == 3 {
			add = "\x00"
		} else {
			log.Panicf("no data for long form of %q", v)
		}
		lang.s[i] += add
	}
	b.writeString("lang", lang.join())

	// Generate tables for non-canonicalized tags.
	mappedLang.freeze()
	mappedLangID := []int16{}
	altTag := ""
	for _, v := range langSpecial.slice() {
		i := 0
		if len(v) <= 3 {
			i = b.lang.index(v)
		} else {
			i = -1 - len(altTag)
			altTag += v
		}
		mappedLangID = append(mappedLangID, int16(i))
	}

	for k, v := range mappedLang.update {
		i := mappedLang.index(k)
		mappedLang.s[i] += string(langSpecial.index(v))
	}
	b.writeString("mappedLang", mappedLang.join())
	b.writeSlice("mappedLangID", mappedLangID)
	b.writeString("altTag", altTag)
	b.writeMapFunc("tagAlias", legacyTag, func(s string) uint16 {
		return uint16(b.lang.index(s))
	})
}

func (b *builder) writeScript() {
	b.writeConst("unknownScript", b.script.index("Zzzz"))
	b.writeString("script", b.script.join())
}

func parseM49(s string) uint16 {
	if len(s) == 0 {
		return 0
	}
	v, err := strconv.ParseUint(s, 10, 10)
	failOnError(err)
	return uint16(v)
}

func (b *builder) writeRegion() {
	b.writeConst("unknownRegion", b.region.index("ZZ"))

	isoOffset := b.region.index("AA")
	m49map := make([]uint16, len(b.region.slice()))
	altRegionISO3 := ""
	altRegionIDs := []uint16{}

	b.writeConst("isoRegionOffset", isoOffset)

	// 2-letter region lookup and mapping to numeric codes.
	regionISO := b.region.clone()
	regionISO.s = regionISO.s[isoOffset:]
	regionISO.sorted = false
	for i, tc := range b.supp.CodeMappings.TerritoryCodes {
		if tc.Type != regionISO.s[i] {
			log.Panicf("writeRegion: found %q; want %q", regionISO.s[i], tc.Type)
		}
		if len(tc.Alpha3) == 3 {
			if tc.Alpha3[0] == tc.Type[0] {
				regionISO.s[i] += tc.Alpha3[1:]
			} else {
				regionISO.s[i] += string([]byte{0, byte(len(altRegionISO3))})
				altRegionISO3 += tc.Alpha3
				altRegionIDs = append(altRegionIDs, uint16(isoOffset+i))
			}
		} else {
			regionISO.s[i] += "  "
		}
		if d := m49map[isoOffset+i]; d != 0 {
			log.Panicf("%s found as a duplicate UN.M49 code of %03d", tc.Numeric, d)
		}
		m49map[isoOffset+i] = parseM49(tc.Numeric)
	}
	b.writeString("regionISO", regionISO.join())
	b.writeString("altRegionISO3", altRegionISO3)
	b.writeSlice("altRegionIDs", altRegionIDs)

	// 3-digit region lookup, groupings.
	for i := 0; i < isoOffset; i++ {
		m49map[i] = parseM49(b.region.s[i])
	}
	b.writeSlice("m49", m49map)
}

func (b *builder) writeLocale() {
	b.writeStringSlice("locale", b.locale.slice())
}

func (b *builder) writeLanguageInfo() {
}

func (b *builder) writeCurrencies() {
	unknown := b.currency.index("XXX")
	digits := map[string]uint64{}
	rounding := map[string]uint64{}
	for _, info := range b.supp.CurrencyData.Fractions[0].Info {
		var err error
		digits[info.Iso4217], err = strconv.ParseUint(info.Digits, 10, 2)
		failOnError(err)
		rounding[info.Iso4217], err = strconv.ParseUint(info.Rounding, 10, 6)
		failOnError(err)
	}
	for i, cur := range b.currency.slice() {
		d := uint64(2) // default number of decimal positions
		if dd, ok := digits[cur]; ok {
			d = dd
		}
		var r uint64
		if r = rounding[cur]; r == 0 {
			r = 1 // default rounding increment in units 10^{-digits)
		}
		b.currency.s[i] += string([]byte{byte(r<<2 + d)})
	}
	b.writeString("currency", b.currency.join())
	// Hack alert: gofmt indents a trailing comment after an indented string.
	// Write this constant after currency to force a proper indentation of
	// the final comment.
	b.writeConst("unknownCurrency", unknown)
}

var header = `// Generated by running
//		maketables -url=%s
// DO NOT EDIT

package locale
`

func main() {
	flag.Parse()
	b := newBuilder(url)
	fmt.Fprintf(b.out, header, *url)

	b.parseIndices()
	b.writeLanguage()
	b.writeScript()
	b.writeRegion()
	// TODO: b.writeLocale()
	b.writeCurrencies()

	fmt.Fprintf(b.out, "\n// Size: %.1fK (%d bytes); Check: %X\n", float32(b.size)/1024, b.size, b.hash32.Sum32())
}
