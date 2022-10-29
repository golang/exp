// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"bytes"
	"context"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

const timeRE = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}(Z|[+-]\d{2}:\d{2})`

func TestLogTextHandler(t *testing.T) {
	var buf bytes.Buffer

	l := New(NewTextHandler(&buf))

	check := func(want string) {
		t.Helper()
		if want != "" {
			want = "time=" + timeRE + " " + want
		}
		checkLogOutput(t, buf.String(), want)
		buf.Reset()
	}

	l.Info("msg", "a", 1, "b", 2)
	check(`level=INFO msg=msg a=1 b=2`)

	// By default, debug messages are not printed.
	l.Debug("bg", Int("a", 1), "b", 2)
	check("")

	l.Warn("w", Duration("dur", 3*time.Second))
	check(`level=WARN msg=w dur=3s`)

	l.Error("bad", io.EOF, "a", 1)
	check(`level=ERROR msg=bad a=1 err=EOF`)

	l.Log(WarnLevel+1, "w", Int("a", 1), String("b", "two"))
	check(`level=WARN\+1 msg=w a=1 b=two`)

	l.LogAttrs(InfoLevel+1, "a b c", Int("a", 1), String("b", "two"))
	check(`level=INFO\+1 msg="a b c" a=1 b=two`)
}

func TestConnections(t *testing.T) {
	var logbuf, slogbuf bytes.Buffer

	// The default slog.Logger's handler uses the log package's default output.
	log.SetOutput(&logbuf)
	log.SetFlags(log.Flags() | log.Lshortfile)
	Info("msg", "a", 1)
	checkLogOutput(t, logbuf.String(),
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} logger_test.go:\d\d: INFO msg a=1`)
	logbuf.Reset()
	Warn("msg", "b", 2)
	checkLogOutput(t, logbuf.String(),
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} logger_test.go:\d\d: WARN msg b=2`)
	// Levels below Info are not printed.
	logbuf.Reset()
	Debug("msg", "c", 3)
	checkLogOutput(t, logbuf.String(), "")

	// Once slog.SetDefault is called, the direction is reversed: the default
	// log.Logger's output goes through the handler.
	SetDefault(New(NewTextHandler(&slogbuf)))
	log.Print("msg2")
	checkLogOutput(t, slogbuf.String(), "time="+timeRE+` level=INFO msg=msg2`)

	// Setting log's output again breaks the connection.
	logbuf.Reset()
	slogbuf.Reset()
	log.SetOutput(&logbuf)
	log.SetFlags(log.LstdFlags)
	log.Print("msg3")
	checkLogOutput(t, logbuf.String(),
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} msg3`)
	if got := slogbuf.String(); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestAttrs(t *testing.T) {
	check := func(got []Attr, want ...Attr) {
		t.Helper()
		if !attrsEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	l1 := New(&captureHandler{}).With("a", 1)
	l2 := New(l1.Handler()).With("b", 2)
	l2.Info("m", "c", 3)
	h := l2.Handler().(*captureHandler)
	check(h.attrs, Int("a", 1), Int("b", 2))
	check(attrsSlice(h.r), Int("c", 3))
}

func TestCallDepth(t *testing.T) {
	h := &captureHandler{}
	var startLine int

	check := func(count int) {
		t.Helper()
		const wantFile = "logger_test.go"
		wantLine := startLine + count*2
		gotFile, gotLine := h.r.SourceLine()
		gotFile = filepath.Base(gotFile)
		if gotFile != wantFile || gotLine != wantLine {
			t.Errorf("got (%s, %d), want (%s, %d)", gotFile, gotLine, wantFile, wantLine)
		}
	}

	logger := New(h)
	SetDefault(logger)

	// Calls to check must be one line apart.
	// Determine line where calls start.
	f, _ := runtime.CallersFrames([]uintptr{pc(2)}).Next()
	startLine = f.Line + 4
	// Do not change the number of lines between here and the call to check(0).

	logger.Log(InfoLevel, "")
	check(0)
	logger.LogAttrs(InfoLevel, "")
	check(1)
	logger.Debug("")
	check(2)
	logger.Info("")
	check(3)
	logger.Warn("")
	check(4)
	logger.Error("", nil)
	check(5)
	Debug("")
	check(6)
	Info("")
	check(7)
	Warn("")
	check(8)
	Error("", nil)
	check(9)
	Log(InfoLevel, "")
	check(10)
	LogAttrs(InfoLevel, "")
	check(11)
}

func TestAlloc(t *testing.T) {
	dl := New(discardHandler{})
	defer func(d Logger) { SetDefault(d) }(Default())
	SetDefault(dl)

	t.Run("Info", func(t *testing.T) {
		wantAllocs(t, 0, func() { Info("hello") })
	})
	// t.Run("Error", func(t *testing.T) {
	// 	wantAllocs(t, 0, func() { Error("hello", io.EOF) })
	// })
	t.Run("logger.Info", func(t *testing.T) {
		wantAllocs(t, 0, func() { dl.Info("hello") })
	})
	t.Run("logger.Log", func(t *testing.T) {
		wantAllocs(t, 0, func() { dl.Log(DebugLevel, "hello") })
	})
	t.Run("2 pairs", func(t *testing.T) {
		s := "abc"
		i := 2000
		wantAllocs(t, 2, func() {
			dl.Info("hello",
				"n", i,
				"s", s,
			)
		})
	})
	t.Run("2 pairs disabled inline", func(t *testing.T) {
		l := New(discardHandler{disabled: true})
		s := "abc"
		i := 2000
		wantAllocs(t, 2, func() {
			l.Log(InfoLevel, "hello",
				"n", i,
				"s", s,
			)
		})
	})
	t.Run("2 pairs disabled", func(t *testing.T) {
		l := New(discardHandler{disabled: true})
		s := "abc"
		i := 2000
		wantAllocs(t, 0, func() {
			if l.Enabled(InfoLevel) {
				l.Log(InfoLevel, "hello",
					"n", i,
					"s", s,
				)
			}
		})
	})
	t.Run("9 kvs", func(t *testing.T) {
		s := "abc"
		i := 2000
		d := time.Second
		wantAllocs(t, 11, func() {
			dl.Info("hello",
				"n", i, "s", s, "d", d,
				"n", i, "s", s, "d", d,
				"n", i, "s", s, "d", d)
		})
	})
	t.Run("pairs", func(t *testing.T) {
		wantAllocs(t, 0, func() { dl.Info("", "error", io.EOF) })
	})
	t.Run("attrs1", func(t *testing.T) {
		wantAllocs(t, 0, func() { dl.LogAttrs(InfoLevel, "", Int("a", 1)) })
		wantAllocs(t, 0, func() { dl.LogAttrs(InfoLevel, "", Any("error", io.EOF)) })
	})
	t.Run("attrs3", func(t *testing.T) {
		wantAllocs(t, 0, func() {
			dl.LogAttrs(InfoLevel, "hello", Int("a", 1), String("b", "two"), Duration("c", time.Second))
		})
	})
	t.Run("attrs3 disabled", func(t *testing.T) {
		logger := New(discardHandler{disabled: true})
		wantAllocs(t, 0, func() {
			logger.LogAttrs(InfoLevel, "hello", Int("a", 1), String("b", "two"), Duration("c", time.Second))
		})
	})
	t.Run("attrs6", func(t *testing.T) {
		wantAllocs(t, 1, func() {
			dl.LogAttrs(InfoLevel, "hello",
				Int("a", 1), String("b", "two"), Duration("c", time.Second),
				Int("d", 1), String("e", "two"), Duration("f", time.Second))
		})
	})
	t.Run("attrs9", func(t *testing.T) {
		wantAllocs(t, 1, func() {
			dl.LogAttrs(InfoLevel, "hello",
				Int("a", 1), String("b", "two"), Duration("c", time.Second),
				Int("d", 1), String("e", "two"), Duration("f", time.Second),
				Int("d", 1), String("e", "two"), Duration("f", time.Second))
		})
	})
}

func TestSetAttrs(t *testing.T) {
	for _, test := range []struct {
		args []any
		want []Attr
	}{
		{nil, nil},
		{[]any{"a", 1}, []Attr{Int("a", 1)}},
		{[]any{"a", 1, "b", "two"}, []Attr{Int("a", 1), String("b", "two")}},
		{[]any{"a"}, []Attr{String(badKey, "a")}},
		{[]any{"a", 1, "b"}, []Attr{Int("a", 1), String(badKey, "b")}},
		{[]any{"a", 1, 2, 3}, []Attr{Int("a", 1), Int(badKey, 2), Int(badKey, 3)}},
	} {
		r := NewRecord(time.Time{}, 0, "", 0, nil)
		r.setAttrsFromArgs(test.args)
		got := attrsSlice(r)
		if !attrsEqual(got, test.want) {
			t.Errorf("%v:\ngot  %v\nwant %v", test.args, got, test.want)
		}
	}
}

func checkLogOutput(t *testing.T, got, wantRegexp string) {
	t.Helper()
	got = clean(got)
	wantRegexp = "^" + wantRegexp + "$"
	matched, err := regexp.MatchString(wantRegexp, got)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("\ngot  %s\nwant %s", got, wantRegexp)
	}
}

// clean prepares log output for comparison.
func clean(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return strings.ReplaceAll(s, "\n", "~")
}

type captureHandler struct {
	r     Record
	attrs []Attr
}

func (h *captureHandler) Handle(r Record) error {
	h.r = r
	return nil
}

func (*captureHandler) Enabled(Level) bool { return true }

func (c *captureHandler) WithAttrs(as []Attr) Handler {
	c2 := *c
	c2.attrs = concat(c2.attrs, as)
	return &c2
}

func (h *captureHandler) WithGroup(name string) Handler {
	panic("unimplemented")
}

type discardHandler struct {
	disabled bool
	attrs    []Attr
}

func (d discardHandler) Enabled(Level) bool { return !d.disabled }
func (discardHandler) Handle(Record) error  { return nil }
func (d discardHandler) WithAttrs(as []Attr) Handler {
	d.attrs = concat(d.attrs, as)
	return d
}
func (h discardHandler) WithGroup(name string) Handler {
	return h
}

// This is a simple benchmark. See the benchmarks subdirectory for more extensive ones.
func BenchmarkNopLog(b *testing.B) {
	b.ReportAllocs()
	l := New(&captureHandler{})
	b.Run("attrs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			l.LogAttrs(InfoLevel, "msg", Int("a", 1), String("b", "two"), Bool("c", true))
		}
	})
	b.Run("attrs-parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.LogAttrs(InfoLevel, "msg", Int("a", 1), String("b", "two"), Bool("c", true))
			}
		})
	})
	b.Run("keys-values", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			l.Log(InfoLevel, "msg", "a", 1, "b", "two", "c", true)
		}
	})
}

func TestSetDefault(t *testing.T) {
	// Verify that setting the default to itself does not result in deadlock.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer func(w io.Writer) { log.SetOutput(w) }(log.Writer())
	log.SetOutput(io.Discard)
	go func() {
		Info("A")
		SetDefault(Default())
		Info("B")
		cancel()
	}()
	<-ctx.Done()
	if err := ctx.Err(); err != context.Canceled {
		t.Errorf("wanted canceled, got %v", err)
	}
}

// concat returns a new slice with the elements of s1 followed
// by those of s2. The slice has no additional capacity.
func concat[T any](s1, s2 []T) []T {
	s := make([]T, len(s1)+len(s2))
	copy(s, s1)
	copy(s[len(s1):], s2)
	return s
}
