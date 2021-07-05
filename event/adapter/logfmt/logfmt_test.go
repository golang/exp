// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logfmt_test

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
)

func TestPrint(t *testing.T) {
	var p logfmt.Printer
	buf := &strings.Builder{}
	// the exporter is not used, but we need it to build the events
	at, _ := time.Parse(logfmt.TimeFormat, "2020/03/05 14:27:48")
	for _, test := range []struct {
		name   string
		event  event.Event
		expect string
	}{{
		name:   "empty",
		event:  event.Event{},
		expect: ``,
	}, {
		name:   "span",
		event:  event.Event{ID: 34, Kind: event.StartKind},
		expect: `trace=34`,
	}, {
		name:   "parent",
		event:  event.Event{Parent: 14},
		expect: `parent=14`,
	}, {
		name:   "namespace",
		event:  event.Event{Source: event.Source{Space: "golang.org/x/exp/event"}},
		expect: `in=golang.org/x/exp/event`,
	}, {
		name:   "at",
		event:  event.Event{At: at},
		expect: `time="2020/03/05 14:27:48"`,
	}, {
		name:   "message",
		event:  event.Event{Labels: []event.Label{event.String("msg", "a message")}},
		expect: `msg="a message"`,
	}, {
		name:   "end",
		event:  event.Event{Kind: event.EndKind},
		expect: `end`,
	}, {
		name: "string",
		event: event.Event{
			Labels: []event.Label{
				event.String("v1", "text"),
				event.String("v2", "text with quotes"),
				event.String("empty", ""),
			},
		},
		expect: `v1=text v2="text with quotes" empty=""`,
	}, {
		name: "int",
		event: event.Event{
			Labels: []event.Label{
				event.Int64("value", 67),
			},
		},
		expect: `value=67`,
	}, {
		name: "float",
		event: event.Event{
			Labels: []event.Label{
				event.Float64("value", 263.2),
			},
		},
		expect: `value=263.2`,
	}, {
		name: "bool",
		event: event.Event{
			Labels: []event.Label{
				event.Bool("v1", true),
				event.Bool("v2", false),
			},
		},
		expect: `v1=true v2=false`,
	}, {
		name: "value",
		event: event.Event{
			Labels: []event.Label{
				event.Value("v1", notString{"simple"}),
				event.Value("v2", notString{"needs quoting"}),
			},
		},
		expect: `v1=simple v2="needs quoting"`,
	}, {
		name: "empty label",
		event: event.Event{
			Labels: []event.Label{
				event.Value("before", nil),
				event.String("", "text"),
				event.Value("after", nil),
			},
		},
		expect: `before after`,
	}, {
		name: "quoted ident",
		event: event.Event{
			Labels: []event.Label{
				event.String("name with space", "text"),
			},
		},
		expect: `"name with space"=text`,
	}, {
		name:   "quoting quote",
		event:  event.Event{Labels: []event.Label{event.String("msg", `with"middle`)}},
		expect: `msg="with\"middle"`,
	}, {
		name:   "quoting newline",
		event:  event.Event{Labels: []event.Label{event.String("msg", "with\nmiddle")}},
		expect: `msg="with\nmiddle"`,
	}, {
		name:   "quoting slash",
		event:  event.Event{Labels: []event.Label{event.String("msg", `with\middle`)}},
		expect: `msg="with\\middle"`,
	}, {
		name: "quoting bytes",
		event: event.Event{
			Labels: []event.Label{
				event.Bytes("value", ([]byte)(`bytes "need" quote`)),
			},
		},
		expect: `value="bytes \"need\" quote"`,
	}} {
		t.Run(test.name, func(t *testing.T) {
			buf.Reset()
			p.Event(buf, &test.event)
			got := strings.TrimSpace(buf.String())
			if got != test.expect {
				t.Errorf("got: \n%q\nexpect:\n%q\n", got, test.expect)
			}
		})
	}
}

type notString struct {
	value string
}

func (v notString) String() string { return v.value }

func TestPrinterFlags(t *testing.T) {
	var reference logfmt.Printer
	buf := &strings.Builder{}
	// the exporter is not used, but we need it to build the events
	for _, test := range []struct {
		name    string
		printer logfmt.Printer
		event   event.Event
		before  string
		after   string
	}{{
		name:    "quote values",
		printer: logfmt.Printer{QuoteValues: true},
		event: event.Event{
			Labels: []event.Label{
				event.String("value", "text"),
			},
		},
		before: `value=text`,
		after:  `value="text"`,
	}, {
		name:    "suppress namespace",
		printer: logfmt.Printer{SuppressNamespace: true},
		event: event.Event{
			Source: event.Source{Space: "golang.org/x/exp/event"},
			Labels: []event.Label{event.String("msg", "some text")},
		},
		before: `in=golang.org/x/exp/event msg="some text"`,
		after:  `msg="some text"`,
	}} {
		t.Run(test.name, func(t *testing.T) {
			buf.Reset()
			reference.Event(buf, &test.event)
			gotBefore := strings.TrimSpace(buf.String())
			buf.Reset()
			test.printer.Event(buf, &test.event)
			gotAfter := strings.TrimSpace(buf.String())
			if gotBefore != test.before {
				t.Errorf("got: \n%q\nexpect:\n%q\n", gotBefore, test.before)
			}
			if gotAfter != test.after {
				t.Errorf("got: \n%q\nexpect:\n%q\n", gotAfter, test.after)
			}
		})
	}
}
