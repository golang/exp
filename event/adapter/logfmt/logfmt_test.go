// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logfmt_test

import (
	"errors"
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
		event:  event.Event{TraceID: 34},
		expect: `trace=34`,
	}, {
		name:   "parent",
		event:  event.Event{Parent: 14},
		expect: `parent=14`,
	}, {
		name:   "namespace",
		event:  event.Event{Namespace: "golang.org/x/exp/event"},
		expect: `in="golang.org/x/exp/event"`,
	}, {
		name:   "name",
		event:  event.Event{Name: "named"},
		expect: `name=named`,
	}, {
		name:   "at",
		event:  event.Event{At: at},
		expect: `time="2020/03/05 14:27:48"`,
	}, {
		name:   "message",
		event:  event.Event{Message: "a message"},
		expect: `msg="a message"`,
	}, {
		name:   "error",
		event:  event.Event{Error: errors.New("an error")},
		expect: `err="an error"`,
	}, {
		name:   "end",
		event:  event.Event{Kind: event.TraceKind},
		expect: `end`,
	}, {
		name: "string",
		event: event.Event{
			Labels: []event.Label{{
				Name:  "v1",
				Value: event.StringOf("text"),
			}, {
				Name:  "v2",
				Value: event.StringOf("text with quotes"),
			}, {
				Name:  "empty",
				Value: event.StringOf(""),
			}},
		},
		expect: `v1=text v2="text with quotes" empty=""`,
	}, {
		name: "int",
		event: event.Event{
			Labels: []event.Label{{
				Name:  "value",
				Value: event.Int64Of(67),
			}},
		},
		expect: `value=67`,
	}, {
		name: "float",
		event: event.Event{
			Labels: []event.Label{{
				Name:  "value",
				Value: event.Float64Of(263.2),
			}},
		},
		expect: `value=263.2`,
	}, {
		name: "bool",
		event: event.Event{
			Labels: []event.Label{{
				Name:  "v1",
				Value: event.BoolOf(true),
			}, {
				Name:  "v2",
				Value: event.BoolOf(false),
			}},
		},
		expect: `v1=true v2=false`,
	}, {
		name: "value",
		event: event.Event{
			Labels: []event.Label{{
				Name:  "v1",
				Value: event.ValueOf(notString{"simple"}),
			}, {
				Name:  "v2",
				Value: event.ValueOf(notString{"needs quoting"}),
			}},
		},
		expect: `v1=simple v2="needs quoting"`,
	}, {
		name: "empty label",
		event: event.Event{
			Labels: []event.Label{{
				Name: "before",
			}, {
				Name:  "",
				Value: event.StringOf("text"),
			}, {
				Name: "after",
			}},
		},
		expect: `before after`,
	}, {
		name: "quoted ident",
		event: event.Event{
			Labels: []event.Label{{
				Name:  "name with space",
				Value: event.StringOf("text"),
			}},
		},
		expect: `"name with space"=text`,
	}, {
		name:   "quoting quote",
		event:  event.Event{Message: `with"middle`},
		expect: `msg="with\"middle"`,
	}, {
		name:   "quoting newline",
		event:  event.Event{Message: "with\nmiddle"},
		expect: `msg="with\nmiddle"`,
	}, {
		name:   "quoting slash",
		event:  event.Event{Message: `with\middle`},
		expect: `msg="with\\middle"`,
	}, {
		name: "quoting bytes",
		event: event.Event{
			Labels: []event.Label{{
				Name:  "value",
				Value: event.BytesOf(([]byte)(`bytes "need" quote`)),
			}},
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
			Labels: []event.Label{{
				Name:  "value",
				Value: event.StringOf("text"),
			}},
		},
		before: `value=text`,
		after:  `value="text"`,
	}, {
		name:    "suppress namespace",
		printer: logfmt.Printer{SuppressNamespace: true},
		event: event.Event{
			Namespace: "golang.org/x/exp/event",
			Message:   "some text",
		},
		before: `in="golang.org/x/exp/event" msg="some text"`,
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
