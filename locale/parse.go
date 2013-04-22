// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package locale

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// isAlpha returns true if the byte is not a digit.
// b must be an ASCII letter or digit.
func isAlpha(b byte) bool {
	return b > '9'
}

// isAlphaNum returns true if the string contains ASCII letters or digits.
func isAlphaNum(s []byte) bool {
	for _, c := range s {
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9') {
			return false
		}
	}
	return true
}

var (
	errEmpty    = errors.New("locale: empty locale identifier")
	errInvalid  = errors.New("locale: invalid")
	errTrailSep = errors.New("locale: trailing separator")
)

// scanner is used to scan BCP 47 tokens, which are separated by _ or -.
type scanner struct {
	b     []byte
	bytes [64]byte // small buffer to cover most common cases
	token []byte
	start int // start position of the current token
	end   int // end position of the current token
	next  int // next point for scan
	err   error
	done  bool
}

func makeScannerString(s string) scanner {
	scan := scanner{}
	if len(s) <= len(scan.bytes) {
		scan.b = scan.bytes[:copy(scan.bytes[:], s)]
	} else {
		scan.b = []byte(s)
	}
	scan.init()
	return scan
}

func (s *scanner) init() {
	for i, c := range s.b {
		if c == '_' {
			s.b[i] = '-'
		}
	}
	s.scan()
}

// restToLower converts the string between start and end to lower case.
func (s *scanner) toLower(start, end int) {
	for i := start; i < end; i++ {
		c := s.b[i]
		if 'A' <= c && c <= 'Z' {
			s.b[i] += 'a' - 'A'
		}
	}
}

func (s *scanner) setError(e error) {
	if s.err == nil {
		s.err = e
	}
}

func (s *scanner) setErrorf(f string, x ...interface{}) {
	s.setError(fmt.Errorf(f, x...))
}

// replace replaces the current token with repl.
func (s *scanner) replace(repl string) {
	if end := s.start + len(repl); end != s.end {
		diff := end - s.end
		if end < cap(s.b) {
			b := make([]byte, len(s.b)+diff)
			copy(b, s.b[:s.start])
			copy(b[end:], s.b[s.end:])
			s.b = b
		} else {
			s.b = append(s.b[end:], s.b[s.end:]...)
		}
		s.next += diff
		s.end = end
	}
	copy(s.b[s.start:], repl)
}

// gobble removes the current token from the input.
// Caller must call scan after calling gobble.
func (s *scanner) gobble() {
	if s.start == 0 {
		s.b = s.b[:+copy(s.b, s.b[s.next:])]
		s.end = 0
	} else {
		s.b = s.b[:s.start-1+copy(s.b[s.start-1:], s.b[s.end:])]
		s.end = s.start - 1
	}
	s.next = s.start
}

// scan parses the next token of a BCP 47 string.  Tokens that are larger
// than 8 characters or include non-alphanumeric characters result in an error
// and are gobbled and removed from the output.
// It returns the end position of the last token consumed.
func (s *scanner) scan() (end int) {
	end = s.end
	s.token = nil
	for s.start = s.next; s.next < len(s.b); {
		i := bytes.IndexByte(s.b[s.next:], '-')
		if i == -1 {
			s.end = len(s.b)
			s.next = len(s.b)
			i = s.end - s.start
		} else {
			s.end = s.next + i
			s.next = s.end + 1
		}
		token := s.b[s.start:s.end]
		if i < 1 || i > 8 || !isAlphaNum(token) {
			s.setErrorf("locale: invalid token %q", token)
			s.gobble()
			continue
		}
		s.token = token
		return end
	}
	if n := len(s.b); n > 0 && s.b[n-1] == '-' {
		s.setError(errTrailSep)
		s.b = s.b[:len(s.b)-1]
	}
	s.done = true
	return end
}

// acceptMinSize parses multiple tokens of the given size or greater.
// It returns the end position of the last token consumed.
func (s *scanner) acceptMinSize(min int) (end int) {
	end = s.end
	s.scan()
	for ; len(s.token) >= min; s.scan() {
		end = s.end
	}
	return end
}

// Parse parses the given BCP 47 string and returns a valid ID.
// If parsing failed it returns an error and any part of the identifier
// that could be parsed.
// If parsing succeeded but an unknown option was found, it
// returns the valid Locale and an error.
// It accepts identifiers in the BCP 47 format and extensions to this standard
// defined in
// http://www.unicode.org/reports/tr35/#Unicode_Language_and_Locale_Identifiers.
func Parse(s string) (loc ID, err error) {
	// TODO: consider supporting old-style locale key-value pairs.
	if s == "" {
		return und, errEmpty
	}
	loc = und
	if lang, ok := tagAlias[s]; ok {
		loc.lang = langID(lang)
		return
	}
	scan := makeScannerString(s)
	if len(scan.token) >= 4 {
		if !strings.EqualFold(s, "root") {
			return und, errInvalid
		}
		return und, nil
	}
	return parse(&scan, s)
}

func parse(scan *scanner, s string) (loc ID, err error) {
	loc = und
	var end int
	private := false
	if n := len(scan.token); n <= 1 {
		scan.toLower(0, len(scan.b))
		end = parsePrivate(scan)
		private = end > 0
	} else if n >= 4 {
		return und, errInvalid
	} else { // the usual case
		loc, end = parseTag(scan)
		if n := len(scan.token); n == 1 {
			loc.pExt = uint16(end)
			end = parseExtensions(scan)
			if end-int(loc.pExt) <= 1 {
				loc.pExt = 0
			}
		}
	}
	if end < len(scan.b) {
		scan.setErrorf("locale: invalid parts %q", scan.b[end:])
		scan.b = scan.b[:end]
	}
	if len(scan.b) <= len(s) {
		s = s[:len(scan.b)]
	}
	if len(s) > 0 && cmp(s, scan.b) == 0 {
		loc.str = &s
	} else if loc.pVariant > 0 || loc.pExt > 0 || private {
		s = string(scan.b)
		loc.str = &s
	}
	return loc, scan.err
}

// parseTag parses language, script, region and variants.
// It returns an ID and the end position in the input that was parsed.
func parseTag(scan *scanner) (ID, int) {
	loc := und
	// TODO: set an error if an unknown lang, script or region is encountered.
	loc.lang = getLangID(scan.token)
	scan.replace(loc.lang.String())
	langStart := scan.start
	end := scan.scan()
	for len(scan.token) == 3 && isAlpha(scan.token[0]) {
		// From http://tools.ietf.org/html/bcp47, <lang>-<extlang> tags are equivalent
		// to a tag of the form <extlang>.
		if lang := getLangID(scan.token); lang != unknownLang {
			loc.lang = lang
			copy(scan.b[langStart:], lang.String())
			scan.b[langStart+3] = '-'
			scan.start = langStart + 4
		}
		scan.gobble()
		end = scan.scan()
	}
	if len(scan.token) == 4 && isAlpha(scan.token[0]) {
		loc.script = getScriptID(script, scan.token)
		if loc.script == unknownScript {
			scan.gobble()
		}
		end = scan.scan()
	}
	if n := len(scan.token); n >= 2 && n <= 3 {
		loc.region = getRegionID(scan.token)
		if loc.region == unknownRegion {
			scan.gobble()
		} else {
			scan.replace(loc.region.String())
		}
		end = scan.scan()
	}
	scan.toLower(scan.start, len(scan.b))
	start := scan.start
	end = parseVariants(scan, end)
	if start < end {
		loc.pVariant = byte(start)
		loc.pExt = uint16(end)
	}
	return loc, end
}

// parseVariants scans tokens as long as each token is a valid variant string.
// Duplicate variants are removed.
func parseVariants(scan *scanner, end int) int {
	start := scan.start
	for ; len(scan.token) >= 4; scan.scan() {
		// TODO: validate and sort variants
		if bytes.Index(scan.b[start:scan.start], scan.token) != -1 {
			scan.gobble()
			continue
		}
		end = scan.end
		const maxVariantSize = 60000 // more than enough, ensures pExt will be valid.
		if end > maxVariantSize {
			break
		}
	}
	return end
}

type bytesSort [][]byte

func (b bytesSort) Len() int {
	return len(b)
}

func (b bytesSort) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b bytesSort) Less(i, j int) bool {
	return bytes.Compare(b[i], b[j]) == -1
}

// parseExtensions parses and normalizes the extensions in the buffer.
// It returns the last position of scan.b that is part of any extension.
func parseExtensions(scan *scanner) int {
	start := scan.start
	exts := [][]byte{}
	private := []byte{}
	end := scan.end
	for len(scan.token) == 1 {
		start := scan.start
		extension := []byte{}
		ext := scan.token[0]
		switch ext {
		case 'u':
			attrEnd := scan.acceptMinSize(3)
			end = attrEnd
			var key []byte
			for last := []byte{}; len(scan.token) == 2; last = key {
				key = scan.token
				end = scan.acceptMinSize(3)
				// TODO: check key value validity
				if bytes.Compare(key, last) != 1 {
					p := attrEnd + 1
					scan.next = p
					keys := [][]byte{}
					for scan.scan(); len(scan.token) == 2; {
						keyStart := scan.start
						end = scan.acceptMinSize(3)
						keys = append(keys, scan.b[keyStart:end])
					}
					sort.Sort(bytesSort(keys))
					copy(scan.b[p:], bytes.Join(keys, []byte{'-'}))
					break
				}
			}
		case 't':
			scan.scan()
			if n := len(scan.token); n >= 2 && n <= 3 && isAlpha(scan.token[1]) {
				_, end = parseTag(scan)
				scan.toLower(start, end)
			}
			for len(scan.token) == 2 && !isAlpha(scan.token[1]) {
				end = scan.acceptMinSize(3)
			}
		case 'x':
			end = scan.acceptMinSize(1)
		default:
			end = scan.acceptMinSize(2)
		}
		extension = scan.b[start:end]
		if len(extension) < 3 {
			scan.setErrorf("locale: empty extension %q", string(ext))
			continue
		} else if len(exts) == 0 && (ext == 'x' || scan.next >= len(scan.b)) {
			return end
		} else if ext == 'x' {
			private = extension
			break
		}
		exts = append(exts, extension)
	}
	if scan.next < len(scan.b) {
		scan.setErrorf("locale: invalid trailing characters %q", scan.b[scan.end:])
	}
	sort.Sort(bytesSort(exts))
	if len(private) > 0 {
		exts = append(exts, private)
	}
	scan.b = append(scan.b[:start], bytes.Join(exts, []byte{'-'})...)
	return len(scan.b)
}

func parsePrivate(scan *scanner) int {
	if len(scan.token) == 0 || scan.token[0] != 'x' {
		scan.setErrorf("locale: invalid locale %q", scan.b)
		return scan.start
	}
	return parseExtensions(scan)
}

// A Part identifies a part of the locale identifier string.
type Part byte

const (
	TagPart Part = iota // The identifier excluding extensions.
	LanguagePart
	ScriptPart
	RegionPart
	VariantPart
)

var partNames = []string{"Tag", "Language", "Script", "Region", "Variant"}

func (p Part) String() string {
	if p > VariantPart {
		return string(p)
	}
	return partNames[p]
}

// Extension returns the Part identifier for extension e, which must be 0-9 or a-z.
func Extension(e byte) Part {
	return Part(e)
}

var (
	errLang   = errors.New("locale: invalid Language")
	errScript = errors.New("locale: invalid Script")
	errRegion = errors.New("locale: invalid Region")
)

// Compose returns a Locale composed from the given parts or an error
// if any of the strings for the parts are ill-formed.
func Compose(m map[Part]string) (loc ID, err error) {
	loc = und
	var scan scanner
	scan.b = scan.bytes[:0]
	add := func(p Part) {
		if s, ok := m[p]; ok {
			if len(scan.b) > 0 {
				scan.b = append(scan.b, '-')
			}
			if p > VariantPart {
				scan.b = append(scan.b, byte(p), '-')
			}
			scan.b = append(scan.b, s...)
		}
	}
	for p := TagPart; p <= VariantPart; p++ {
		if p == TagPart && m[p] != "" {
			for i := LanguagePart; i <= VariantPart; i++ {
				if _, ok := m[i]; ok {
					return und, fmt.Errorf("locale: cannot specify both Tag and %s", partNames[i])
				}
			}
		}
		add(p)
	}
	for p := Part('0'); p < Part('9'); p++ {
		add(p)
	}
	for p := Part('a'); p < Part('w'); p++ {
		add(p)
	}
	for p := Part('y'); p < Part('z'); p++ {
		add(p)
	}
	add(Part('x'))
	scan.init()
	return parse(&scan, "")
}

// Part returns the part of the locale identifer indicated by t.
// The one-letter section identifier, if applicable, is not included.
// Components are separated by a '-'.
func (loc ID) Part(p Part) string {
	s := ""
	switch p {
	case TagPart:
		s = loc.String()
		if loc.pExt > 0 {
			s = s[:loc.pExt]
		}
	case LanguagePart:
		s = loc.lang.String()
	case ScriptPart:
		if loc.script != unknownScript {
			s = loc.script.String()
		}
	case RegionPart:
		if loc.region != unknownRegion {
			s = loc.region.String()
		}
	case VariantPart:
		if loc.pVariant > 0 {
			s = (*loc.str)[loc.pVariant:loc.pExt]
		}
	default:
		if loc.pExt > 0 {
			str := *loc.str
			for i := int(loc.pExt); i < len(str); {
				end, name, ext := getExtension(str, i)
				if name == byte(p) {
					return ext
				}
				i = end
			}
		} else if p == 'x' && loc.str != nil && strings.HasPrefix(*loc.str, "x-") {
			return (*loc.str)[2:]
		}
	}
	return s
}

// Parts returns all parts of the locale identifier in a map.
func (loc ID) Parts() map[Part]string {
	m := make(map[Part]string)
	m[LanguagePart] = loc.lang.String()
	if loc.script != unknownScript {
		m[ScriptPart] = loc.script.String()
	}
	if loc.region != unknownRegion {
		m[RegionPart] = loc.region.String()
	}
	if loc.str != nil {
		s := *loc.str
		if strings.HasPrefix(s, "x-") {
			m[Extension('x')] = s[2:]
		} else if loc.pExt > 0 {
			i := int(loc.pExt)
			if int(loc.pVariant) != i && loc.pVariant > 0 {
				m[VariantPart] = s[loc.pVariant:i]
			}
			for i < len(s) {
				end, name, ext := getExtension(s, i)
				m[Extension(name)] = ext
				i = end
			}
		}
	}
	return m
}

// getExtension returns the name, body and end position of the extension.
func getExtension(s string, p int) (end int, name byte, ext string) {
	p++
	if s[p] == 'x' {
		return len(s), s[p], s[p+2:]
	}
	end = nextExtension(s, p)
	return end, s[p], s[p+2 : end]
}

// nextExtension finds the next extension within the string, searching
// for the -<char>- pattern from position p.
// In the fast majority of cases, locale identifiers will have at most
// one extension and extensions tend to be small.
func nextExtension(s string, p int) int {
	for n := len(s) - 3; p < n; {
		if s[p] == '-' {
			if s[p+2] == '-' {
				return p
			}
			p += 3
		} else {
			p++
		}
	}
	return len(s)
}
