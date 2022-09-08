// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO: verify that the output of Marshal{Text,JSON} is suitably escaped.

package slog

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slog/internal/buffer"
)

func TestWith(t *testing.T) {
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
	tm := time.Now()
	r := MakeRecord(tm, InfoLevel, "message", 1)
	r.AddAttr(String("a", "one"))
	r.AddAttr(Int("b", 2))
	r.AddAttr(Any("", "ignore me"))

	for _, test := range []struct {
		name string
		h    *commonHandler
		want map[string]any
	}{
		{
			name: "basic",
			h:    &commonHandler{},
			want: map[string]any{
				"msg":   "message",
				"time":  tm.Round(0), // strip monotonic
				"level": "INFO",
				"a":     "one",
				"b":     int64(2),
			},
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
			want: map[string]any{
				"MSG":   "message",
				"TIME":  tm.Round(0), // strip monotonic
				"LEVEL": "INFO",
				"A":     "one",
				"B":     int64(2),
			},
		},
		{
			name: "remove all",
			h: &commonHandler{
				opts: HandlerOptions{
					ReplaceAttr: func(a Attr) Attr { return Attr{} },
				},
			},
			want: map[string]any{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ma := &memAppender{m: map[string]any{}}
			test.h.w = &bytes.Buffer{}
			test.h.newAppender = func(*buffer.Buffer) appender { return ma }
			if err := test.h.handle(r); err != nil {
				t.Fatal(err)
			}
			if got := ma.m; !reflect.DeepEqual(got, test.want) {
				t.Errorf("\ngot  %#v\nwant %#v\n", got, test.want)
			}
		})
	}
}

type memAppender struct {
	key string
	m   map[string]any
}

func (a *memAppender) set(v any) { a.m[a.key] = v }

func (a *memAppender) appendStart()          {}
func (a *memAppender) appendSep()            {}
func (a *memAppender) appendEnd()            {}
func (a *memAppender) appendKey(key string)  { a.key = key }
func (a *memAppender) appendString(s string) { a.set(s) }

func (a *memAppender) appendTime(t time.Time) error {
	a.set(t)
	return nil
}

func (a *memAppender) appendSource(file string, line int) {
	a.set(fmt.Sprintf("%s:%d", file, line))
}

func (a *memAppender) appendAttrValue(at Attr) error {
	a.set(at.Value())
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
