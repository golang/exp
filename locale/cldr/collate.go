// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cldr

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// RuleProcessor can be passed to Collator's Process method, which
// parses the rules and calls the respective method for each rule found.
type RuleProcessor interface {
	Reset(anchor string, before int) error
	Insert(level int, str, context, extend string) error
	Index(id string)
}

// cldrIndex is a Unicode-reserved sentinel value used to mark the start
// of a grouping within an index.
// We ignore any rule that starts with this rune.
// See http://unicode.org/reports/tr35/#Collation_Elements for details.
const cldrIndex = "\uFDD0"

var lmap = map[byte]int{
	'p': 1,
	's': 2,
	't': 3,
	'i': 5,
}

type rulesElem struct {
	Rules struct {
		Common
		Any []*struct {
			XMLName xml.Name
			rule
		} `xml:",any"`
	} `xml:"rules"`
}

type rule struct {
	Value  string `xml:",innerxml"`
	Before string `xml:"before,attr"`
	Any    []*struct {
		XMLName xml.Name
		rule
	} `xml:",any"` // for <x> elements
}

var tagRe = regexp.MustCompile(`< *([a-z_]*)  */>`)

func (r *rule) value() string {
	// Convert hexadecimal Unicode codepoint notation to a string.
	r.Value = charRe.ReplaceAllStringFunc(r.Value, replaceUnicode)

	// Strip spaces from reset positions.
	r.Value = tagRe.ReplaceAllString(r.Value, "<$1/>")
	return r.Value
}

func (r rule) process(p RuleProcessor, name, context, extend string) error {
	v := r.value()
	switch name {
	case "p", "s", "t", "i":
		if strings.HasPrefix(v, cldrIndex) {
			p.Index(v[len(cldrIndex):])
			return nil
		}
		if err := p.Insert(lmap[name[0]], v, context, extend); err != nil {
			return err
		}
	case "pc", "sc", "tc", "ic":
		level := lmap[name[0]]
		for _, s := range v {
			if err := p.Insert(level, string(s), context, extend); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("cldr: unsupported tag: %q", name)
	}
	return nil
}

// Process parses the rules for the tailorings of this collation
// and calls the respective methods of p for each rule found.
func (c Collation) Process(p RuleProcessor) error {
	// Collation is generated and defined in xml.go.
	for _, r := range c.Rules.Any {
		switch r.XMLName.Local {
		case "reset":
			level := 0
			switch r.Before {
			case "primary", "1":
				level = 1
			case "secondary", "2":
				level = 2
			case "tertiary", "3":
				level = 3
			case "":
			default:
				return fmt.Errorf("cldr: unknown level %q", r.Before)
			}
			if err := p.Reset(r.value(), level); err != nil {
				return err
			}
		case "x":
			var context, extend string
			for _, r1 := range r.Any {
				switch r1.XMLName.Local {
				case "context":
					context = r1.value()
				case "extend":
					extend = r1.value()
				}
			}
			for _, r1 := range r.Any {
				if t := r1.XMLName.Local; t == "context" || t == "extend" {
					continue
				}
				r1.rule.process(p, r1.XMLName.Local, context, extend)
			}
		default:
			if err := r.rule.process(p, r.XMLName.Local, "", ""); err != nil {
				return err
			}
		}
	}
	return nil
}
