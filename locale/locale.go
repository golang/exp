// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The locale package provides a type to represent BCP 47 locale identifiers.
// It supports various canonicalizations defined in CLDR.
package locale

import "strings"

var (
	// Und represents the undefined langauge. It is also the root locale.
	Und   = und
	En    = en    // Default Locale for English.
	En_US = en_US // Default locale for American English.
	De    = de    // Default locale for German.
	// TODO: list of most common language identifiers.
)

var (
	Supported Set // All supported locales.
	Common    Set // A selection of common locales.
)

var (
	de    = ID{lang: getLangID([]byte("de")), region: unknownRegion, script: unknownScript}
	en    = ID{lang: getLangID([]byte("en")), region: unknownRegion, script: unknownScript}
	en_US = en
	und   = ID{lang: unknownLang, region: unknownRegion, script: unknownScript}
)

// ID represents a BCP 47 locale identifier. It can be used to
// select an instance for a specific locale. All Locale values are guaranteed
// to be well-formed.
type ID struct {
	// In most cases, just lang, region and script will be needed.  In such cases
	// str may be nil.
	lang     langID
	region   regionID
	script   scriptID
	pVariant byte   // offset in str
	pExt     uint16 // offset of first extension
	str      *string
}

// Make calls Parse and Canonicalize and returns the resulting ID.
// Any errors are ignored and a sensible default is returned.
// In most cases, locale IDs should be created using this method.
func Make(id string) ID {
	loc, _ := Parse(id)
	return loc.Canonicalize(All)
}

// IsRoot returns true if loc is equal to locale "und".
func (loc ID) IsRoot() bool {
	if loc.str != nil {
		n := len(*loc.str)
		if n > 0 && loc.pExt > 0 && int(loc.pExt) < n {
			return false
		}
		if uint16(loc.pVariant) != loc.pExt || strings.HasPrefix(*loc.str, "x-") {
			return false
		}
		loc.str = nil
	}
	return loc == und
}

// CanonType is can be used to enable or disable various types of canonicalization.
type CanonType int

const (
	// Replace deprecated values with their preferred ones.
	Deprecated CanonType = 1 << iota
	// Remove redundant scripts.
	SuppressScript
	// Map the dominant language of macro language group to the macro language identifier.
	// For example cmn -> zh.
	Macro
	// All canonicalizations prescribed by BCP 47.
	BCP47 = Deprecated | SuppressScript
	All   = BCP47 | Macro

	// TODO: LikelyScript, LikelyRegion: supress similar to ICU.
)

// Canonicalize replaces the identifier with its canonical equivalent.
func (loc ID) Canonicalize(t CanonType) ID {
	changed := false
	if t&SuppressScript != 0 {
		if loc.lang < langNoIndexOffset && uint8(loc.script) == suppressScript[loc.lang] {
			loc.script = unknownScript
			changed = true
		}
	}
	if t&Deprecated != 0 {
		l := normLang(langOldMap[:], loc.lang)
		if l != loc.lang {
			changed = true
		}
		loc.lang = l
	}
	if t&Macro != 0 {
		l := normLang(langMacroMap[:], loc.lang)
		if l != loc.lang {
			changed = true
		}
		loc.lang = l
	}
	if changed && loc.str != nil {
		ext := ""
		if loc.pExt > 0 {
			ext = (*loc.str)[loc.pExt+1:]
		}
		s := loc.makeString(loc.Part(VariantPart), ext)
		loc.str = &s
	}
	return loc
}

// Parent returns the direct parent for this locale, which is the locale
// from which this locale inherits any undefined values.
func (loc ID) Parent() ID {
	// TODO: implement
	return und
}

// Written strips qualifiers from the identifier until the resulting identfier
// inherits from root.
func (loc ID) Written() ID {
	// TODO: implement
	return und
}

// Confidence indicates the level of certainty for a given return value.
// For example, Serbian may be written in cyrillic or latin script.
// The confidence level indicates whether a value was explicitly specified,
// whether it is typically the only possible value, or whether there is
// an ambiguity.
type Confidence int

const (
	Not   Confidence = iota // full confidence that there was no match
	Low                     // most likely value picked out of a set of alternatives
	High                    // value inferred from a parent and is generally assumed to be the correct match
	Exact                   // exact match or explicitly specified value
)

func (loc *ID) makeString(vars, ext string) string {
	buf := [128]byte{}
	n := loc.lang.stringToBuf(buf[:])
	if loc.script != unknownScript {
		n += copy(buf[n:], "-")
		n += copy(buf[n:], loc.script.String())
	}
	if loc.region != unknownRegion {
		n += copy(buf[n:], "-")
		n += copy(buf[n:], loc.region.String())
	}
	b := buf[:n]
	if vars != "" {
		b = append(b, '-')
		loc.pVariant = byte(len(b))
		b = append(b, vars...)
		loc.pExt = uint16(len(b))
	}
	if ext != "" {
		loc.pExt = uint16(len(b))
		b = append(b, '-')
		b = append(b, ext...)
	}
	return string(b)
}

// String returns the canonical string representation of the locale.
func (loc ID) String() string {
	if loc.str == nil {
		return loc.makeString("", "")
	}
	return *loc.str
}

// Language returns the language for the locale.
func (loc ID) Language() Language {
	// TODO: implement
	return Language{0}
}

// Script infers the script for the locale.  If it was not explictly given, it will infer
// a most likely candidate from the parent locales.
// If more than one script is commonly used for a language, the most likely one
// is returned with a low confidence indication. For example, it returns (Cyrl, Low)
// for Serbian.
// Note that an inferred script is never guaranteed to be the correct one. Latn is
// almost exclusively used for Afrikaans, but Arabic has been used for some texts
// in the past.  Also, the script that is commonly used may change over time.
func (loc ID) Script() (Script, Confidence) {
	// TODO: implement
	return Script{0}, Exact
}

// Region returns the region for l.  If it was not explicitly given, it will
// infer a most likely candidate from the parent locales.
func (loc ID) Region() (Region, Confidence) {
	// TODO: implement
	return Region{0}, Exact
}

// Variant returns the variant specified explicitly for this locale
// or nil if no variant was specified.
func (loc ID) Variant() Variant {
	return Variant{""}
}

// Scope returns a Set that indicates the common variants for which the
// locale may be applicable.
// Locales will returns all valid sublocales. Languages will return the language
// for this locale.  Regions will return all regions for which a locale with
// this language is defined.  And Scripts will return all scripts that are
// commonly used for this locale.
// If any of these properties is explicitly specified, the respective lists
// will be constraint.  For example, for sr_Latn Scripts will return [Latn]
// instead of [Cyrl Latn].
func (loc ID) Scope() Set {
	// TODO: implement
	return nil
}

// TypeForKey returns the type associated with the given key, where key
// is one of the allowed values defined for the Unicode locale extension ('u') in
// http://www.unicode.org/reports/tr35/#Unicode_Language_and_Locale_Identifiers.
// TypeForKey will traverse the inheritance chain to get the correct value.
func (loc ID) TypeForKey(key string) string {
	// TODO: implement
	return ""
}

// KeyValueString returns a string to be set with KeyValuePart.
// Error handling is done by Compose.
func KeyValueString(m map[string]string) (string, error) {
	// TODO: implement
	return "", nil
}

// SimplifyOptions removes options in loc that it would inherit
// by default from its parent.
func (loc ID) SimplifyOptions() ID {
	// TODO: implement
	return ID{}
}

// Language is an ISO 639 language identifier.
type Language struct {
	langID
}

// Scope returns a Set of all pre-defined sublocales for this language.
func (l Language) Scope() Set {
	// TODO: implement
	return nil
}

// Script is a 4-letter ISO 15924 code for representing scripts.
// It is idiomatically represented in title case.
type Script struct {
	scriptID
}

// Scope returns a Set of all pre-defined sublocales applicable to the script.
func (s Script) Scope() Set {
	// TODO: implement
	return nil
}

// Region is an ISO 3166-1 or UN M.49 code for representing countries and regions.
type Region struct {
	regionID
}

// IsCountry returns whether this region is a country.
func (r Region) IsCountry() bool {
	// TODO: implement
	return true
}

// Scope returns a Set of all pre-defined sublocales applicable to the region.
func (r Region) Scope() Set {
	// TODO: implement
	return nil
}

// Variant represents a registered variant of a language as defined by BCP 47.
type Variant struct {
	// TODO: implement
	variant string
}

// String returns the string representation of the variant.
func (v Variant) String() string {
	// TODO: implement
	return v.variant
}

// Currency is an ISO 4217 currency designator.
type Currency struct {
	currencyID
}

// Set provides information about a set of locales.
type Set interface {
	Locales() []ID
	Languages() []Language
	Regions() []Region
	Scripts() []Script
	Currencies() []Currency
}
