// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO: verify that the output of Marshal{Text,JSON} is suitably escaped.

package slog

import (
	"bytes"
	"fmt"
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

func TestCommonHandle(t *testing.T) {
	tm := time.Date(2022, 9, 18, 8, 26, 33, 0, time.UTC)
	r := NewRecord(tm, InfoLevel, "message", 1)
	r.AddAttrs(String("a", "one"), Int("b", 2), Any("", "ignore me"))

	for _, test := range []struct {
		name string
		h    *commonHandler
		want string
	}{
		{
			name: "basic",
			h:    &commonHandler{},
			want: "(time=2022-09-18T08:26:33.000Z;level=INFO;msg=message;a=one;b=2)",
		},
		{
			name: "cap keys",
			h: &commonHandler{
				opts: HandlerOptions{
					ReplaceAttr: func(a Attr) Attr {
						return a.WithKey(strings.ToUpper(a.Key()))
					},
				},
			},
			want: "(TIME=2022-09-18T08:26:33.000Z;LEVEL=INFO;MSG=message;A=one;B=2)",
		},
		{
			name: "remove all",
			h: &commonHandler{
				opts: HandlerOptions{
					ReplaceAttr: func(a Attr) Attr { return Attr{} },
				},
			},
			// TODO: fix this. The correct result is "()".
			want: "(;;)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			test.h.w = &buf
			test.h.newAppender = func(buf *buffer.Buffer) appender {
				return &testAppender{buf}
			}
			if err := test.h.handle(r); err != nil {
				t.Fatal(err)
			}
			got := strings.TrimSuffix(buf.String(), "\n")
			if got != test.want {
				t.Errorf("\ngot  %#v\nwant %#v\n", got, test.want)
			}
		})
	}
}

type testAppender struct {
	buf *buffer.Buffer
}

func (a *testAppender) appendStart() { a.buf.WriteByte('(') }
func (a *testAppender) appendSep()   { a.buf.WriteByte(';') }
func (a *testAppender) appendEnd()   { a.buf.WriteByte(')') }

func (a *testAppender) appendKey(key string) {
	a.appendString(key)
	a.buf.WriteByte('=')
}
func (a *testAppender) appendString(s string) { a.buf.WriteString(s) }

func (a *testAppender) appendTime(t time.Time) error {
	*a.buf = appendTimeRFC3339Millis(*a.buf, t)
	return nil
}

func (a *testAppender) appendSource(file string, line int) {
	a.appendString(fmt.Sprintf("%s:%d", file, line))
}

func (a *testAppender) appendAttrValue(at Attr) error {
	switch at.Kind() {
	case StringKind:
		a.appendString(at.str())
	case TimeKind:
		a.appendTime(at.Time())
	default:
		*a.buf = at.appendValue(*a.buf)
	}
	return nil
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
