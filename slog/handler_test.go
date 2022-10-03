// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO: verify that the output of Marshal{Text,JSON} is suitably escaped.

package slog

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slog/internal/buffer"
)

func TestDefaultWith(t *testing.T) {
	d := &defaultHandler{}
	if g := len(d.attrs); g != 0 {
		t.Errorf("got %d, want 0", g)
	}
	a1 := []Attr{Int("a", 1)}
	d2 := d.With(a1)
	if g := d2.(*defaultHandler).attrs; !attrsEqual(g, a1) {
		t.Errorf("got %v, want %v", g, a1)
	}
	d3 := d2.With([]Attr{String("b", "two")})
	want := append(a1, String("b", "two"))
	if g := d3.(*defaultHandler).attrs; !attrsEqual(g, want) {
		t.Errorf("got %v, want %v", g, want)
	}
}

// Verify the common parts of TextHandler and JSONHandler.
func TestJSONAndTextHandlers(t *testing.T) {
	removeAttr := func(a Attr) Attr { return Attr{} }

	attrs := []Attr{String("a", "one"), Int("b", 2), Any("", "ignore me")}
	preAttrs := []Attr{Int("pre", 3), String("x", "y")}

	for _, test := range []struct {
		name     string
		replace  func(Attr) Attr
		preAttrs []Attr
		attrs    []Attr
		wantText string
		wantJSON string
	}{
		{
			name:     "basic",
			attrs:    attrs,
			wantText: "time=2000-01-02T03:04:05.000Z level=INFO msg=message a=one b=2",
			wantJSON: `{"time":"2000-01-02T03:04:05Z","level":"INFO","msg":"message","a":"one","b":2}`,
		},
		{
			name:     "cap keys",
			replace:  upperCaseKey,
			attrs:    attrs,
			wantText: "TIME=2000-01-02T03:04:05.000Z LEVEL=INFO MSG=message A=one B=2",
			wantJSON: `{"TIME":"2000-01-02T03:04:05Z","LEVEL":"INFO","MSG":"message","A":"one","B":2}`,
		},
		{
			name:     "remove all",
			replace:  removeAttr,
			attrs:    attrs,
			wantText: "",
			wantJSON: `{}`,
		},
		{
			name:     "preformatted",
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: "time=2000-01-02T03:04:05.000Z level=INFO msg=message pre=3 x=y a=one b=2",
			wantJSON: `{"time":"2000-01-02T03:04:05Z","level":"INFO","msg":"message","pre":3,"x":"y","a":"one","b":2}`,
		},
		{
			name:     "preformatted cap keys",
			replace:  upperCaseKey,
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: "TIME=2000-01-02T03:04:05.000Z LEVEL=INFO MSG=message PRE=3 X=y A=one B=2",
			wantJSON: `{"TIME":"2000-01-02T03:04:05Z","LEVEL":"INFO","MSG":"message","PRE":3,"X":"y","A":"one","B":2}`,
		},
		{
			name:     "preformatted remove all",
			replace:  removeAttr,
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: "",
			wantJSON: "{}",
		},
	} {
		r := NewRecord(testTime, InfoLevel, "message", 1)
		r.AddAttrs(test.attrs...)
		var buf bytes.Buffer
		opts := HandlerOptions{ReplaceAttr: test.replace}
		t.Run(test.name, func(t *testing.T) {
			for _, handler := range []struct {
				name string
				h    Handler
				want string
			}{
				{"text", opts.NewTextHandler(&buf), test.wantText},
				{"json", opts.NewJSONHandler(&buf), test.wantJSON},
			} {
				t.Run(handler.name, func(t *testing.T) {
					h := handler.h.With(test.preAttrs)
					buf.Reset()
					if err := h.Handle(r); err != nil {
						t.Fatal(err)
					}
					got := strings.TrimSuffix(buf.String(), "\n")
					if got != handler.want {
						t.Errorf("\ngot  %#v\nwant %#v\n", got, handler.want)
					}
				})
			}
		})
	}
}

func upperCaseKey(a Attr) Attr {
	a.Key = strings.ToUpper(a.Key)
	return a
}

func TestHandlerEnabled(t *testing.T) {
	atomicLevel := func(l Level) *AtomicLevel {
		var al AtomicLevel
		al.Set(l)
		return &al
	}

	for _, test := range []struct {
		leveler Leveler
		want    bool
	}{
		{nil, true},
		{WarnLevel, false},
		{&AtomicLevel{}, true}, // defaults to Info
		{atomicLevel(WarnLevel), false},
		{DebugLevel, true},
		{atomicLevel(DebugLevel), true},
	} {
		h := &commonHandler{opts: HandlerOptions{Level: test.leveler}}
		got := h.Enabled(InfoLevel)
		if got != test.want {
			t.Errorf("%v: got %t, want %t", test.leveler, got, test.want)
		}
	}
}

func TestAppendSource(t *testing.T) {
	for _, test := range []struct {
		file               string
		wantText, wantJSON string
	}{
		{"a/b.go", "a/b.go:1", `"a/b.go:1"`},
		{"a b.go", `"a b.go:1"`, `"a b.go:1"`},
		{`C:\windows\b.go`, `C:\windows\b.go:1`, `"C:\\windows\\b.go:1"`},
	} {
		check := func(json bool, want string) {
			t.Helper()
			var buf []byte
			state := handleState{
				h:   &commonHandler{json: json},
				buf: (*buffer.Buffer)(&buf),
			}
			state.appendSource(test.file, 1)
			got := string(buf)
			if got != want {
				t.Errorf("%s, json=%t:\ngot  %s\nwant %s", test.file, json, got, want)
			}
		}
		check(false, test.wantText)
		check(true, test.wantJSON)
	}
}

const rfc3339Millis = "2006-01-02T15:04:05.000Z07:00"

func TestAppendTimeRFC3339(t *testing.T) {
	for _, tm := range []time.Time{
		time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
		time.Date(2000, 1, 2, 3, 4, 5, 400, time.Local),
		time.Date(2000, 11, 12, 3, 4, 500, 5e7, time.UTC),
	} {
		want := tm.Format(rfc3339Millis)
		var buf []byte
		buf = appendTimeRFC3339Millis(buf, tm)
		got := string(buf)
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

func BenchmarkAppendTime(b *testing.B) {
	buf := make([]byte, 0, 100)
	tm := time.Date(2022, 3, 4, 5, 6, 7, 823456789, time.Local)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = appendTimeRFC3339Millis(buf, tm)
		buf = buf[:0]
	}
}
