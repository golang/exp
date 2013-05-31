// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Locale identifier table generator.
// Data read from the web.

package main

import (
	"bufio"
	"code.google.com/p/go.text/cldr"
	"flag"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"log"
	"math"
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
		"http://www.unicode.org/Public/cldr/"+cldr.Version+"/core.zip",
		"URL of CLDR archive.")
	iana = flag.String("iana",
		"http://www.iana.org/assignments/language-subtag-registry",
		"URL of IANA language subtag registry.")
	test = flag.Bool("test", false,
		"test existing tables; can be used to compare web data with package data.")
	localFiles = flag.Bool("local", false,
		"data files have been copied to the current directory; for debugging only.")
)

var comment = []string{
	`
lang holds an alphabetically sorted list of BCP 47 language identifiers.
All entries are 4 bytes. The index of the identifier (divided by 4) is the language ID.
For 2-byte language identifiers, the two successive bytes have the following meaning:
    - if the first letter of the 2- and 3-letter ISO codes are the same:
      the second and third letter of the 3-letter ISO code.
    - otherwise: a 0 and a by 2 bits right-shifted index into altLangISO3.
For 3-byte language identifiers the 4th byte is 0.`,
	`
langNoIndex is a bit vector of all 3-letter language codes that are not used as an index
in lookup tables. The language ids for these language codes are derived directly
from the letters and are not consecutive.`,
	`
altLangISO3 holds an alphabetically sorted list of 3-letter language code alternatives
to 2-letter language codes that cannot be derived using the method described above.
Each 3-letter code is followed by its 1-byte langID.`,
	`
tagAlias holds a mapping from legacy and grandfathered tags to their locale ID.`,
	`
langOldMap maps deprecated langIDs to their suggested replacements.`,
	`
langMacroMap maps languages to their macro language replacement, if applicable.`,
	`
script is an alphabetically sorted list of ISO 15924 codes. The index
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
	`
suppressScript is an index from langID to the dominant script for that language,
if it exists.  If a script is given, it should be suppressed from the language tag.`,
	`
nRegionGroups is the number of region groups.  All regionIDs < nRegionGroups
are groups.`,
	`
regionInclusion maps region identifiers to sets of regions in regionInclusionBits,
where each set holds all groupings that are directly connected in a region
containment graph.`,
	`
regionInclusionBits is an array of bit vectors where every vector represents
a set of region groupings.  These sets are used to compute the distance
between two regions for the purpos of locale matching.`,
	`
regionInclusionNext marks, for each entry in regionInclusionBits, the set of
all groups that are reachable from the groups set in the respective entry.`,
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

type funcSorter struct {
	fn func(a, b string) bool
	sort.StringSlice
}

func (s funcSorter) Less(i, j int) bool {
	return s.fn(s.StringSlice[i], s.StringSlice[j])
}

func (ss *stringSet) sortFunc(f func(a, b string) bool) {
	ss.compact()
	sort.Sort(funcSorter{f, sort.StringSlice(ss.s)})
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

// ianaEntry holds information for an entry in the IANA Language Subtag Repository.
// All types use the same entry.
// See http://tools.ietf.org/html/bcp47#section-5.1 for a description of the various
// fields.
type ianaEntry struct {
	typ            string
	tag            string
	description    []string
	scope          string
	added          string
	preferred      string
	deprecated     string
	suppressScript string
	macro          string
	prefix         []string
}

type builder struct {
	w      io.Writer   // multi writer
	out    io.Writer   // set to Stdout
	hash32 hash.Hash32 // for checking whether tables have changed.
	size   int
	data   *cldr.CLDR
	supp   *cldr.SupplementalData

	// indices
	locale      stringSet // common locales
	lang        stringSet // canonical language ids (2 or 3 letter ISO codes) with data
	langNoIndex stringSet // 3-letter ISO codes with no associated data
	script      stringSet // 4-letter ISO codes
	region      stringSet // 2-letter ISO or 3-digit UN M49 codes
	currency    stringSet // 3-letter ISO currency codes

	// langInfo
	registry map[string]*ianaEntry
}

func openReader(url *string) io.ReadCloser {
	if *localFiles {
		pwd, _ := os.Getwd()
		*url = "file://" + path.Join(pwd, path.Base(*url))
	}
	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
	c := &http.Client{Transport: t}
	resp, err := c.Get(*url)
	failOnError(err)
	if resp.StatusCode != 200 {
		log.Fatalf(`bad GET status for "%s": %s`, *url, resp.Status)
	}
	return resp.Body
}

func newBuilder() *builder {
	r := openReader(url)
	defer r.Close()
	d := &cldr.Decoder{}
	d.SetDirFilter("supplemental")
	data, err := d.DecodeZip(r)
	failOnError(err)
	b := builder{
		out:    os.Stdout,
		data:   data,
		supp:   data.Supplemental(),
		hash32: fnv.New32(),
	}
	b.w = io.MultiWriter(b.out, b.hash32)
	b.parseRegistry()
	return &b
}

func (b *builder) parseRegistry() {
	r := openReader(iana)
	defer r.Close()
	b.registry = make(map[string]*ianaEntry)

	scan := bufio.NewScanner(r)
	scan.Split(bufio.ScanWords)
	var record *ianaEntry
	for more := scan.Scan(); more; {
		key := scan.Text()
		more = scan.Scan()
		value := scan.Text()
		switch key {
		case "Type:":
			record = &ianaEntry{typ: value}
		case "Subtag:", "Tag:":
			record.tag = value
			if info, ok := b.registry[value]; ok {
				if info.typ != "language" || record.typ != "extlang" {
					log.Fatalf("parseRegistry: tag %q already exists", value)
				}
			} else {
				b.registry[value] = record
			}
		case "Suppress-Script:":
			record.suppressScript = value
		case "Added:":
			record.added = value
		case "Deprecated:":
			record.deprecated = value
		case "Macrolanguage:":
			record.macro = value
		case "Preferred-Value:":
			record.preferred = value
		case "Prefix:":
			record.prefix = append(record.prefix, value)
		case "Scope:":
			record.scope = value
		case "Description:":
			buf := []byte(value)
			for more = scan.Scan(); more; more = scan.Scan() {
				b := scan.Bytes()
				if b[0] == '%' || b[len(b)-1] == ':' {
					break
				}
				buf = append(buf, ' ')
				buf = append(buf, b...)
			}
			record.description = append(record.description, string(buf))
			continue
		default:
			continue
		}
		more = scan.Scan()
	}
	if scan.Err() != nil {
		log.Panic(scan.Err())
	}
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
		if t.Kind() == reflect.Struct {
			fmt.Fprintf(b.w, "\n\t%#v, ", v.Index(i).Interface())
		} else {
			if i%12 == 0 {
				fmt.Fprintf(b.w, "\n\t")
			}
			fmt.Fprintf(b.w, "%d, ", v.Index(i).Interface())
		}
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

const base = 'z' - 'a' + 1

func strToInt(s string) uint {
	v := uint(0)
	for i := 0; i < len(s); i++ {
		v *= base
		v += uint(s[i] - 'a')
	}
	return v
}

func (b *builder) writeBitVector(name string, ss []string) {
	vec := make([]uint8, int(math.Ceil(math.Pow(base, float64(len(ss[0])))/8)))
	for _, s := range ss {
		v := strToInt(s)
		vec[v/8] |= 1 << (v % 8)
	}
	b.writeSlice(name, vec)
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

func (b *builder) langIndex(s string) uint16 {
	if i, ok := b.lang.find(s); ok {
		return uint16(i)
	}
	return uint16(strToInt(s)) + uint16(len(b.lang.s))
}

// inc advances the string to its lexicographical successor.
func inc(s string) string {
	i := len(s) - 1
	for ; s[i]+1 > 'z'; i-- {
	}
	return fmt.Sprintf("%s%s%s", s[:i], string(s[i]+1), s[i+1:])
}

func (b *builder) parseIndices() {
	meta := b.supp.Metadata

	for k, v := range b.registry {
		var ss *stringSet
		switch v.typ {
		case "language":
			if len(k) == 2 || v.suppressScript != "" || v.scope == "special" {
				b.lang.add(k)
				continue
			} else {
				ss = &b.langNoIndex
			}
		case "region":
			ss = &b.region
		case "script":
			ss = &b.script
		default:
			continue
		}
		if s := strings.SplitN(k, "..", 2); len(s) > 1 {
			for a := s[0]; a <= s[1]; a = inc(a) {
				ss.add(a)
			}
		} else {
			ss.add(k)
		}
	}

	// currency codes
	for _, reg := range b.supp.CurrencyData.Region {
		for _, cur := range reg.Currency {
			b.currency.add(cur.Iso4217)
		}
	}

	// common locales
	b.locale.parse(meta.DefaultContent.Locales)
}

// writeLanguage generates all tables needed for language canonicalization.
func (b *builder) writeLanguage() {
	meta := b.supp.Metadata

	b.writeConst("unknownLang", b.lang.index("und"))

	// Get language codes that need to be mapped (overlong 3-letter codes, deprecated
	// 2-letter codes and grandfathered tags.
	langOldMap := stringSet{}

	// Mappings for macro languages
	langMacroMap := stringSet{}

	// altLangISO3 get the alternative ISO3 names that need to be mapped.
	altLangISO3 := stringSet{}

	// legacyTag maps from tag to language code.
	legacyTag := make(map[string]string)

	lang := b.lang.clone()
	for _, a := range meta.Alias.LanguageAlias {
		if a.Replacement == "" {
			a.Replacement = "und"
		}
		// TODO: support mapping to tags
		repl := strings.SplitN(a.Replacement, "_", 2)[0]
		if a.Reason == "overlong" {
			if len(a.Replacement) == 2 && len(a.Type) == 3 {
				lang.updateLater(a.Replacement, a.Type)
			}
		} else if len(a.Type) <= 3 {
			if a.Reason != "deprecated" {
				langMacroMap.add(a.Type)
				langMacroMap.updateLater(a.Type, repl)
			}
		} else {
			legacyTag[strings.Replace(a.Type, "_", "-", -1)] = repl
		}
	}
	for k, v := range b.registry {
		// Also add deprecated values for 3-letter ISO codes, which CLDR omits.
		if v.typ == "language" && v.deprecated != "" && v.preferred != "" {
			langOldMap.add(k)
			langOldMap.updateLater(k, v.preferred)
		}
	}
	// Fix CLDR mappings.
	lang.updateLater("tl", "tgl")
	lang.updateLater("sh", "hbs")
	lang.updateLater("mo", "mol")
	lang.updateLater("no", "nor")
	lang.updateLater("tw", "twi")
	lang.updateLater("nb", "nob")
	lang.updateLater("ak", "aka")

	// Ensure that each 2-letter code is matched with a 3-letter code.
	for _, v := range lang.s {
		s, ok := lang.update[v]
		if !ok {
			if s, ok = lang.update[langOldMap.update[v]]; !ok {
				continue
			}
			lang.update[v] = s
		}
		if v[0] != s[0] {
			altLangISO3.add(s)
			altLangISO3.updateLater(s, v)
		}
	}

	// Complete canonialized language tags.
	lang.freeze()
	for i, v := range lang.s {
		// We can avoid these manual entries by using the IANI registry directly.
		// Seems easier to update the list manually, as changes are rare.
		// The panic in this loop will trigger if we miss an entry.
		add := ""
		if s, ok := lang.update[v]; ok {
			if s[0] == v[0] {
				add = s[1:]
			} else {
				add = string([]byte{0, byte(altLangISO3.index(s))})
			}
		} else if len(v) == 3 {
			add = "\x00"
		} else {
			log.Panicf("no data for long form of %q", v)
		}
		lang.s[i] += add
	}
	b.writeString("lang", lang.join())

	b.writeConst("langNoIndexOffset", len(b.lang.s))

	// space of all valid 3-letter language identifiers.
	b.writeBitVector("langNoIndex", b.langNoIndex.slice())

	for i, s := range altLangISO3.slice() {
		idx := b.lang.index(altLangISO3.update[s])
		altLangISO3.s[i] += string([]byte{byte(idx)})
	}
	b.writeString("altLangISO3", altLangISO3.join())

	makeMap := func(name string, ss *stringSet) {
		ss.sortFunc(func(i, j string) bool {
			return b.langIndex(i) < b.langIndex(j)
		})
		m := []struct{ from, to uint16 }{}
		for _, s := range ss.s {
			m = append(m, struct{ from, to uint16 }{
				b.langIndex(s),
				b.langIndex(ss.update[s]),
			})
		}
		b.writeSlice(name, m)
	}
	makeMap("langOldMap", &langOldMap)
	makeMap("langMacroMap", &langMacroMap)

	b.writeMapFunc("tagAlias", legacyTag, func(s string) uint16 {
		return uint16(b.langIndex(s))
	})
}

func (b *builder) writeScript() {
	unknown := uint8(b.script.index("Zzzz"))
	b.writeConst("unknownScript", unknown)
	b.writeString("script", b.script.join())

	supp := make([]uint8, len(b.lang.slice()))
	for i, v := range b.lang.slice() {
		supp[i] = unknown
		if sc := b.registry[v].suppressScript; sc != "" {
			supp[i] = uint8(b.script.index(sc))
		}
	}
	b.writeSlice("suppressScript", supp)
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
	for _, tc := range b.supp.CodeMappings.TerritoryCodes {
		i := regionISO.index(tc.Type)
		if len(tc.Alpha3) == 3 {
			if tc.Alpha3[0] == tc.Type[0] {
				regionISO.s[i] += tc.Alpha3[1:]
			} else {
				regionISO.s[i] += string([]byte{0, byte(len(altRegionISO3))})
				altRegionISO3 += tc.Alpha3
				altRegionIDs = append(altRegionIDs, uint16(isoOffset+i))
			}
		}
		if d := m49map[isoOffset+i]; d != 0 {
			log.Panicf("%s found as a duplicate UN.M49 code of %03d", tc.Numeric, d)
		}
		m49map[isoOffset+i] = parseM49(tc.Numeric)
	}
	for i, s := range regionISO.s {
		if len(s) != 4 {
			regionISO.s[i] = s + "  "
		}
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

func (b *builder) writeRegionInclusionData() {
	type index uint
	groups := make(map[int]index)
	// Create group indices.
	for i := 0; b.region.s[i][0] < 'A'; i++ { // Base M49 indices on regionID.
		groups[i] = index(i)
	}
	for _, g := range b.supp.TerritoryContainment.Group {
		group := b.region.index(g.Type)
		if _, ok := groups[group]; !ok {
			groups[group] = index(len(groups))
		}
	}
	if len(groups) > 32 {
		log.Fatalf("only 32 groups supported, found %d", len(groups))
	}
	b.writeConst("nRegionGroups", len(groups))
	mm := make(map[int][]index)
	for _, g := range b.supp.TerritoryContainment.Group {
		group := b.region.index(g.Type)
		for _, mem := range strings.Split(g.Contains, " ") {
			r := b.region.index(mem)
			mm[r] = append(mm[r], groups[group])
			if g, ok := groups[r]; ok {
				mm[group] = append(mm[group], g)
			}
		}
	}
	regionInclusion := make([]uint8, len(b.region.s))
	bvs := make(map[uint32]index)
	// Make the first bitvector positions correspond with the groups.
	for r, i := range groups {
		bv := uint32(1 << i)
		for _, g := range mm[r] {
			bv |= 1 << g
		}
		bvs[bv] = i
		regionInclusion[r] = uint8(bvs[bv])
	}
	for r := 0; r < len(b.region.s); r++ {
		if _, ok := groups[r]; !ok {
			bv := uint32(0)
			for _, g := range mm[r] {
				bv |= 1 << g
			}
			if bv == 0 {
				// Pick the world for unspecified regions.
				bv = 1 << groups[b.region.index("001")]
			}
			if _, ok := bvs[bv]; !ok {
				bvs[bv] = index(len(bvs))
			}
			regionInclusion[r] = uint8(bvs[bv])
		}
	}
	b.writeSlice("regionInclusion", regionInclusion)
	regionInclusionBits := make([]uint32, len(bvs))
	for k, v := range bvs {
		regionInclusionBits[v] = uint32(k)
	}
	// Add bit vectors for increasingly large distances until a fixed point is reached.
	regionInclusionNext := []uint8{}
	for i := 0; i < len(regionInclusionBits); i++ {
		bits := regionInclusionBits[i]
		next := bits
		for i := uint(0); i < uint(len(groups)); i++ {
			if bits&(1<<i) != 0 {
				next |= regionInclusionBits[i]
			}
		}
		if _, ok := bvs[next]; !ok {
			bvs[next] = index(len(bvs))
			regionInclusionBits = append(regionInclusionBits, next)
		}
		regionInclusionNext = append(regionInclusionNext, uint8(bvs[next]))
	}
	b.writeSlice("regionInclusionBits", regionInclusionBits)
	b.writeSlice("regionInclusionNext", regionInclusionNext)
}

var header = `// Generated by running
//		maketables -url=%s -iana=%s
// DO NOT EDIT

package locale
`

func main() {
	flag.Parse()
	b := newBuilder()
	fmt.Fprintf(b.out, header, *url, *iana)

	b.parseIndices()
	b.writeLanguage()
	b.writeScript()
	b.writeRegion()
	// TODO: b.writeLocale()
	b.writeCurrencies()
	b.writeRegionInclusionData()

	fmt.Fprintf(b.out, "\n// Size: %.1fK (%d bytes); Check: %X\n", float32(b.size)/1024, b.size, b.hash32.Sum32())
}
