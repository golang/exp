// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The locale package provides a type to represent BCP 47 locale identifiers.
// It supports various canonicalizations defined in CLDR.
package locale

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
	de    = ID{lang: getLangID([]byte("de"))}
	en    = ID{lang: getLangID([]byte("en"))}
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
	return loc.Canonicalize()
}

// Canonicalize replaces the identifier with its canonical equivalent.
func (loc ID) Canonicalize() ID {
	return und
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

// String returns the canonical string representation of the locale.
func (loc ID) String() string {
	if loc.str == nil {
		s := loc.lang.String()
		if loc.script != unknownScript {
			s += "-" + loc.script.String()
		}
		if loc.region != unknownRegion {
			s += "-" + loc.region.String()
		}
		loc.str = &s
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
	// TODO: implement
	lang int
}

// String returns the shortest possible ISO country code.
func (l Language) String() string {
	// TODO: implement
	return ""
}

// ISO3 returns the ISO 639-3 language code.
func (l Language) ISO3() string {
	// TODO: implement
	return ""
}

// Scope returns a Set of all pre-defined sublocales for this language.
func (l Language) Scope() Set {
	// TODO: implement
	return nil
}

// Script is a 4-letter ISO 15924 code for representing scripts.
// It is idiomatically represented in title case.
type Script struct {
	// TODO: implement
	script int
}

// String returns the script code in title case.
func (s Script) String() string {
	// TODO: implement
	return ""
}

// Scope returns a Set of all pre-defined sublocales applicable to the script.
func (s Script) Scope() Set {
	// TODO: implement
	return nil
}

// Region is an ISO 3166-1 or UN M.49 code for representing countries and regions.
type Region struct {
	// TODO: implement
	region int
}

// String returns the canonical representation of the region.
func (r Region) String() string {
	// TODO: implement
	return ""
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
	// TODO: implement
	currency int
}

// String returns the lower-cased representation of the currency.
func (c Currency) String() string {
	// TODO: implement
	return ""
}

// Num returns the 3-digit numerical representation of the currency.
func (c Currency) Num() int {
	// TODO: implement
	return 0
}

// Set provides information about a set of locales.
type Set interface {
	Locales() []ID
	Languages() []Language
	Regions() []Region
	Scripts() []Script
	Currencies() []Currency
}
