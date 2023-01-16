// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO: verify that the output of Marshal{Text,JSON} is suitably escaped.

package slog

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog/internal/buffer"
)

func TestDefaultHandle(t *testing.T) {
	preAttrs := []Attr{Int("pre", 0)}
	attrs := []Attr{Int("a", 1), String("b", "two")}
	for _, test := range []struct {
		name  string
		with  func(Handler) Handler
		attrs []Attr
		want  string
	}{
		{
			name: "no attrs",
			want: "NOTICE message",
		},
		{
			name:  "attrs",
			attrs: attrs,
			want:  "NOTICE message a=1 b=two",
		},
		{
			name:  "preformatted",
			with:  func(h Handler) Handler { return h.WithAttrs(preAttrs) },
			attrs: attrs,
			want:  "NOTICE message pre=0 a=1 b=two",
		},
		{
			name: "groups",
			attrs: []Attr{
				Int("a", 1),
				Group("g",
					Int("b", 2),
					Group("h", Int("c", 3)),
					Int("d", 4)),
				Int("e", 5),
			},
			want: "NOTICE message a=1 g.b=2 g.h.c=3 g.d=4 e=5",
		},
		{
			name:  "group",
			with:  func(h Handler) Handler { return h.WithAttrs(preAttrs).WithGroup("s") },
			attrs: attrs,
			want:  "NOTICE message pre=0 s.a=1 s.b=two",
		},
		{
			name: "preformatted groups",
			with: func(h Handler) Handler {
				return h.WithAttrs([]Attr{Int("p1", 1)}).
					WithGroup("s1").
					WithAttrs([]Attr{Int("p2", 2)}).
					WithGroup("s2")
			},
			attrs: attrs,
			want:  "NOTICE message p1=1 s1.p2=2 s1.s2.a=1 s1.s2.b=two",
		},
		{
			name: "two with-groups",
			with: func(h Handler) Handler {
				return h.WithAttrs([]Attr{Int("p1", 1)}).
					WithGroup("s1").
					WithGroup("s2")
			},
			attrs: attrs,
			want:  "NOTICE message p1=1 s1.s2.a=1 s1.s2.b=two",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got string
			var h Handler = newDefaultHandler(func(_ int, s string) error {
				got = s
				return nil
			})
			if test.with != nil {
				h = test.with(h)
			}
			r := NewRecord(time.Time{}, LevelNotice, "message", 0, nil)
			r.AddAttrs(test.attrs...)
			if err := h.Handle(r); err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("\ngot  %s\nwant %s", got, test.want)
			}
		})
	}
}

// Verify the common parts of TextHandler and JSONHandler.
func TestJSONAndTextHandlers(t *testing.T) {

	// ReplaceAttr functions

	// remove all Attrs
	removeAll := func(_ []string, a Attr) Attr { return Attr{} }

	attrs := []Attr{String("a", "one"), Int("b", 2), Any("", "ignore me")}
	preAttrs := []Attr{Int("pre", 3), String("x", "y")}

	for _, test := range []struct {
		name     string
		replace  func([]string, Attr) Attr
		with     func(Handler) Handler
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
			replace:  removeAll,
			attrs:    attrs,
			wantText: "",
			wantJSON: `{}`,
		},
		{
			name:     "preformatted",
			with:     func(h Handler) Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: "time=2000-01-02T03:04:05.000Z level=INFO msg=message pre=3 x=y a=one b=2",
			wantJSON: `{"time":"2000-01-02T03:04:05Z","level":"INFO","msg":"message","pre":3,"x":"y","a":"one","b":2}`,
		},
		{
			name:     "preformatted cap keys",
			replace:  upperCaseKey,
			with:     func(h Handler) Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: "TIME=2000-01-02T03:04:05.000Z LEVEL=INFO MSG=message PRE=3 X=y A=one B=2",
			wantJSON: `{"TIME":"2000-01-02T03:04:05Z","LEVEL":"INFO","MSG":"message","PRE":3,"X":"y","A":"one","B":2}`,
		},
		{
			name:     "preformatted remove all",
			replace:  removeAll,
			with:     func(h Handler) Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: "",
			wantJSON: "{}",
		},
		{
			name:     "remove built-in",
			replace:  removeKeys(TimeKey, LevelKey, MessageKey),
			attrs:    attrs,
			wantText: "a=one b=2",
			wantJSON: `{"a":"one","b":2}`,
		},
		{
			name:     "preformatted remove built-in",
			replace:  removeKeys(TimeKey, LevelKey, MessageKey),
			with:     func(h Handler) Handler { return h.WithAttrs(preAttrs) },
			attrs:    attrs,
			wantText: "pre=3 x=y a=one b=2",
			wantJSON: `{"pre":3,"x":"y","a":"one","b":2}`,
		},
		{
			name:    "groups",
			replace: removeKeys(TimeKey, LevelKey), // to simplify the result
			attrs: []Attr{
				Int("a", 1),
				Group("g",
					Int("b", 2),
					Group("h", Int("c", 3)),
					Int("d", 4)),
				Int("e", 5),
			},
			wantText: "msg=message a=1 g.b=2 g.h.c=3 g.d=4 e=5",
			wantJSON: `{"msg":"message","a":1,"g":{"b":2,"h":{"c":3},"d":4},"e":5}`,
		},
		{
			name:     "empty group",
			replace:  removeKeys(TimeKey, LevelKey),
			attrs:    []Attr{Group("g"), Group("h", Int("a", 1))},
			wantText: "msg=message h.a=1",
			wantJSON: `{"msg":"message","g":{},"h":{"a":1}}`,
		},
		{
			name:    "escapes",
			replace: removeKeys(TimeKey, LevelKey),
			attrs: []Attr{
				String("a b", "x\t\n\000y"),
				Group(" b.c=\"\\x2E\t",
					String("d=e", "f.g\""),
					Int("m.d", 1)), // dot is not escaped
			},
			wantText: `msg=message "a b"="x\t\n\x00y" " b.c=\"\\x2E\t.d=e"="f.g\"" " b.c=\"\\x2E\t.m.d"=1`,
			wantJSON: `{"msg":"message","a b":"x\t\n\u0000y"," b.c=\"\\x2E\t":{"d=e":"f.g\"","m.d":1}}`,
		},
		{
			name:    "LogValuer",
			replace: removeKeys(TimeKey, LevelKey),
			attrs: []Attr{
				Int("a", 1),
				Any("name", logValueName{"Ren", "Hoek"}),
				Int("b", 2),
			},
			wantText: "msg=message a=1 name.first=Ren name.last=Hoek b=2",
			wantJSON: `{"msg":"message","a":1,"name":{"first":"Ren","last":"Hoek"},"b":2}`,
		},
		{
			name:     "with-group",
			replace:  removeKeys(TimeKey, LevelKey),
			with:     func(h Handler) Handler { return h.WithAttrs(preAttrs).WithGroup("s") },
			attrs:    attrs,
			wantText: "msg=message pre=3 x=y s.a=one s.b=2",
			wantJSON: `{"msg":"message","pre":3,"x":"y","s":{"a":"one","b":2}}`,
		},
		{
			name:    "preformatted with-groups",
			replace: removeKeys(TimeKey, LevelKey),
			with: func(h Handler) Handler {
				return h.WithAttrs([]Attr{Int("p1", 1)}).
					WithGroup("s1").
					WithAttrs([]Attr{Int("p2", 2)}).
					WithGroup("s2")
			},
			attrs:    attrs,
			wantText: "msg=message p1=1 s1.p2=2 s1.s2.a=one s1.s2.b=2",
			wantJSON: `{"msg":"message","p1":1,"s1":{"p2":2,"s2":{"a":"one","b":2}}}`,
		},
		{
			name:    "two with-groups",
			replace: removeKeys(TimeKey, LevelKey),
			with: func(h Handler) Handler {
				return h.WithAttrs([]Attr{Int("p1", 1)}).
					WithGroup("s1").
					WithGroup("s2")
			},
			attrs:    attrs,
			wantText: "msg=message p1=1 s1.s2.a=one s1.s2.b=2",
			wantJSON: `{"msg":"message","p1":1,"s1":{"s2":{"a":"one","b":2}}}`,
		},
		{
			name:     "GroupValue as Attr value",
			replace:  removeKeys(TimeKey, LevelKey),
			attrs:    []Attr{{"v", AnyValue(IntValue(3))}},
			wantText: "msg=message v=3",
			wantJSON: `{"msg":"message","v":3}`,
		},
		{
			name:     "byte slice",
			replace:  removeKeys(TimeKey, LevelKey),
			attrs:    []Attr{Any("bs", []byte{1, 2, 3, 4})},
			wantText: `msg=message bs="\x01\x02\x03\x04"`,
			wantJSON: `{"msg":"message","bs":"AQIDBA=="}`,
		},
		{
			name:     "json.RawMessage",
			replace:  removeKeys(TimeKey, LevelKey),
			attrs:    []Attr{Any("bs", json.RawMessage([]byte("1234")))},
			wantText: `msg=message bs="1234"`,
			wantJSON: `{"msg":"message","bs":1234}`,
		},
	} {
		r := NewRecord(testTime, LevelInfo, "message", 1, nil)
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
					h := handler.h
					if test.with != nil {
						h = test.with(h)
					}
					buf.Reset()
					if err := h.Handle(r); err != nil {
						t.Fatal(err)
					}
					got := strings.TrimSuffix(buf.String(), "\n")
					if got != handler.want {
						t.Errorf("\ngot  %s\nwant %s\n", got, handler.want)
					}
				})
			}
		})
	}
}

// removeKeys returns a function suitable for HandlerOptions.ReplaceAttr
// that removes all Attrs with the given keys.
func removeKeys(keys ...string) func([]string, Attr) Attr {
	return func(_ []string, a Attr) Attr {
		for _, k := range keys {
			if a.Key == k {
				return Attr{}
			}
		}
		return a
	}
}

func upperCaseKey(_ []string, a Attr) Attr {
	a.Key = strings.ToUpper(a.Key)
	return a
}

type logValueName struct {
	first, last string
}

func (n logValueName) LogValue() Value {
	return GroupValue(
		String("first", n.first),
		String("last", n.last))
}

func TestHandlerEnabled(t *testing.T) {
	levelVar := func(l Level) *LevelVar {
		var al LevelVar
		al.Set(l)
		return &al
	}

	for _, test := range []struct {
		leveler Leveler
		want    bool
	}{
		{nil, true},
		{LevelWarn, false},
		{&LevelVar{}, true}, // defaults to notice 0 info is 42
		{levelVar(LevelWarn), false},
		{LevelDebug, true},
		{levelVar(LevelDebug), true},
	} {
		h := &commonHandler{opts: HandlerOptions{Level: test.leveler}}
		got := h.enabled(LevelNotice)
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

func TestSecondWith(t *testing.T) {
	// Verify that a second call to Logger.With does not corrupt
	// the original.
	var buf bytes.Buffer
	h := HandlerOptions{ReplaceAttr: removeKeys(TimeKey)}.NewTextHandler(&buf)
	logger := New(h).With(
		String("app", "playground"),
		String("role", "tester"),
		Int("data_version", 2),
	)
	appLogger := logger.With("type", "log") // this becomes type=met
	_ = logger.With("type", "metric")
	appLogger.Notice("foo")
	got := strings.TrimSpace(buf.String())
	want := `level=NOTICE msg=foo app=playground role=tester data_version=2 type=log`
	if got != want {
		t.Errorf("\ngot  %s\nwant %s", got, want)
	}
}

func TestReplaceAttrGroups(t *testing.T) {
	// Verify that ReplaceAttr is called with the correct groups.
	type ga struct {
		groups string
		key    string
		val    string
	}

	var got []ga

	h := HandlerOptions{ReplaceAttr: func(gs []string, a Attr) Attr {
		v := a.Value.String()
		if a.Key == TimeKey {
			v = "<now>"
		}
		got = append(got, ga{strings.Join(gs, ","), a.Key, v})
		return a
	}}.NewTextHandler(io.Discard)
	New(h).
		With(Int("a", 1)).
		WithGroup("g1").
		With(Int("b", 2)).
		WithGroup("g2").
		With(
			Int("c", 3),
			Group("g3", Int("d", 4)),
			Int("e", 5)).
		Notice("m",
			Int("f", 6),
			Group("g4", Int("h", 7)),
			Int("i", 8))

	want := []ga{
		{"", "a", "1"},
		{"g1", "b", "2"},
		{"g1,g2", "c", "3"},
		{"g1,g2,g3", "d", "4"},
		{"g1,g2", "e", "5"},
		{"", "time", "<now>"},
		{"", "level", "NOTICE"},
		{"", "msg", "m"},
		{"g1,g2", "f", "6"},
		{"g1,g2,g4", "h", "7"},
		{"g1,g2", "i", "8"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("\ngot  %v\nwant %v", got, want)
	}
}

const rfc3339Millis = "2006-01-02T15:04:05.000Z07:00"

func TestWriteTimeRFC3339(t *testing.T) {
	for _, tm := range []time.Time{
		time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
		time.Date(2000, 1, 2, 3, 4, 5, 400, time.Local),
		time.Date(2000, 11, 12, 3, 4, 500, 5e7, time.UTC),
	} {
		want := tm.Format(rfc3339Millis)
		buf := buffer.New()
		defer buf.Free()
		writeTimeRFC3339Millis(buf, tm)
		got := buf.String()
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

func BenchmarkWriteTime(b *testing.B) {
	buf := buffer.New()
	defer buf.Free()
	tm := time.Date(2022, 3, 4, 5, 6, 7, 823456789, time.Local)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writeTimeRFC3339Millis(buf, tm)
		buf.Reset()
	}
}
