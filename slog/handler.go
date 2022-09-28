// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slog/internal/buffer"
)

// A Handler handles log records produced by a Logger..
//
// A typical handler may print log records to standard error,
// or write them to a file or database, or perhaps augment them
// with additional attributes and pass them on to another handler.
//
// Any of the Handler's methods may be called concurrently with itself
// or with other methods. It is the responsibility of the Handler to
// manage this concurrency.
type Handler interface {
	// Enabled reports whether the handler handles records at the given level.
	// The handler ignores records whose level is lower.
	Enabled(Level) bool

	// Handle handles the Record.
	// Handle methods that produce output should observe the following rules:
	//   - If r.Time() is the zero time, ignore the time.
	//   - If an Attr's key is the empty string, ignore the Attr.
	Handle(r Record) error

	// With returns a new Handler whose attributes consist of
	// the receiver's attributes concatenated with the arguments.
	// The Handler owns the slice: it may retain, modify or discard it.
	With(attrs []Attr) Handler
}

type defaultHandler struct {
	attrs []Attr
}

func (*defaultHandler) Enabled(l Level) bool {
	return l >= InfoLevel
}

// Collect the level, attributes and message in a string and
// write it with the default log.Logger.
// Let the log.Logger handle time and file/line.
func (h *defaultHandler) Handle(r Record) error {
	var b strings.Builder
	b.WriteString(r.Level().String())
	b.WriteByte(' ')
	r.Attrs(func(a Attr) {
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(a.Value.Resolve().String())
		b.WriteByte(' ')
	})
	b.WriteString(r.Message())
	return log.Output(4, b.String())
}

func (d *defaultHandler) With(as []Attr) Handler {
	d2 := *d
	d2.attrs = concat(d2.attrs, as)
	return &d2
}

// HandlerOptions are options for a TextHandler or JSONHandler.
// A zero HandlerOptions consists entirely of default values.
type HandlerOptions struct {
	// Add a "source" attribute to the output whose value is of the form
	// "file:line".
	AddSource bool

	// Ignore records with levels below Level.Level().
	// The default is InfoLevel.
	Level Leveler

	// If set, ReplaceAttr is called on each attribute of the message,
	// and the returned value is used instead of the original. If the returned
	// key is empty, the attribute is omitted from the output.
	//
	// The built-in attributes with keys "time", "level", "source", and "msg"
	// are passed to this function first, except that time and level are omitted
	// if zero, and source is omitted if AddSourceLine is false.
	ReplaceAttr func(a Attr) Attr
}

type commonHandler struct {
	json              bool // true => output JSON; false => output text
	opts              HandlerOptions
	preformattedAttrs []byte
	mu                sync.Mutex
	w                 io.Writer
}

// Enabled reports whether l is greater than or equal to the
// minimum level.
func (h *commonHandler) Enabled(l Level) bool {
	minLevel := InfoLevel
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return l >= minLevel
}

func (h *commonHandler) with(as []Attr) *commonHandler {
	h2 := &commonHandler{
		json:              h.json,
		opts:              h.opts,
		preformattedAttrs: h.preformattedAttrs,
		w:                 h.w,
	}
	// Pre-format the attributes as an optimization.
	state := handleState{
		h:   h2,
		buf: (*buffer.Buffer)(&h2.preformattedAttrs),
		sep: "",
	}
	for _, a := range as {
		state.appendAttr(a)
	}
	return h2
}

func (h *commonHandler) handle(r Record) error {
	rep := h.opts.ReplaceAttr
	state := handleState{h: h, buf: buffer.New(), sep: ""}
	defer state.buf.Free()
	if h.json {
		state.buf.WriteByte('{')
	}
	// time
	if !r.Time().IsZero() {
		key := "time"
		val := r.Time().Round(0) // strip monotonic to match Attr behavior
		if rep == nil {
			state.appendKey(key)
			state.appendTime(val)
		} else {
			state.appendAttr(Time(key, val))
		}
	}
	// level
	key := "level"
	val := r.Level()
	if rep == nil {
		state.appendKey(key)
		state.appendString(val.String())
	} else {
		state.appendAttr(Any(key, val))
	}
	// source
	if h.opts.AddSource {
		file, line := r.SourceLine()
		if file != "" {
			key := "source"
			if rep == nil {
				state.appendKey(key)
				state.appendSource(file, line)
			} else {
				buf := buffer.New()
				buf.WriteString(file) // TODO: escape?
				buf.WriteByte(':')
				itoa((*[]byte)(buf), line, -1)
				s := buf.String()
				buf.Free()
				state.appendAttr(String(key, s))
			}
		}
	}
	key = "msg"
	msg := r.Message()
	if rep == nil {
		state.appendKey(key)
		state.appendString(msg)
	} else {
		state.appendAttr(String(key, msg))
	}
	// preformatted Attrs
	if len(h.preformattedAttrs) > 0 {
		state.buf.WriteString(state.sep)
		state.buf.Write(h.preformattedAttrs)
	}
	// Attrs in Record
	r.Attrs(func(a Attr) {
		state.appendAttr(a)
	})
	if h.json {
		state.buf.WriteByte('}')
	}
	state.buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(*state.buf)
	return err
}

// handleState holds state for a single call to commonHandler.handle.
// The initial value of sep determines whether to emit a separator
// before the next key, after which it stays true.
type handleState struct {
	h   *commonHandler
	buf *buffer.Buffer
	sep string // write between Attrs
}

// appendAttr appends the Attr's key and value using app.
// If sep is true, it also prepends a separator.
// It handles replacement and checking for an empty key.
// It sets sep to true if it actually did the append (if the key was non-empty
// after replacement).
func (s *handleState) appendAttr(a Attr) {
	if rep := s.h.opts.ReplaceAttr; rep != nil {
		a = rep(a)
	}
	if a.Key == "" {
		return
	}
	s.appendKey(a.Key)
	s.appendValue(a.Value)
}

func (s *handleState) appendError(err error) {
	s.appendString(fmt.Sprintf("!ERROR:%v", err))
}

func (s *handleState) appendKey(key string) {
	s.buf.WriteString(s.sep)
	s.appendString(key)
	if s.h.json {
		s.buf.WriteByte(':')
		s.sep = ","
	} else {
		s.buf.WriteByte('=')
		s.sep = " "
	}
}

func (s *handleState) appendSource(file string, line int) {
	if s.h.json {
		s.buf.WriteByte('"')
		*s.buf = appendEscapedJSONString(*s.buf, file)
		s.buf.WriteByte(':')
		itoa((*[]byte)(s.buf), line, -1)
		s.buf.WriteByte('"')
	} else {
		// text
		if needsQuoting(file) {
			s.appendString(file + ":" + strconv.Itoa(line))
		} else {
			// common case: no quoting needed.
			s.appendString(file)
			s.buf.WriteByte(':')
			itoa((*[]byte)(s.buf), line, -1)
		}
	}
}

func (s *handleState) appendString(str string) {
	if s.h.json {
		s.buf.WriteByte('"')
		*s.buf = appendEscapedJSONString(*s.buf, str)
		s.buf.WriteByte('"')
	} else {
		// text
		if needsQuoting(str) {
			*s.buf = strconv.AppendQuote(*s.buf, str)
		} else {
			s.buf.WriteString(str)
		}
	}
}

func (s *handleState) appendValue(v Value) {
	v = v.Resolve()
	var err error
	if s.h.json {
		err = appendJSONValue(s, v)
	} else {
		err = appendTextValue(s, v)
	}
	if err != nil {
		s.appendError(err)
	}
}

func (s *handleState) appendTime(t time.Time) {
	if s.h.json {
		appendJSONTime(s, t)
	} else {
		*s.buf = appendTimeRFC3339Millis(*s.buf, t)
	}
}

// This takes half the time of Time.AppendFormat.
func appendTimeRFC3339Millis(buf []byte, t time.Time) []byte {
	// TODO: try to speed up by indexing the buffer.
	char := func(b byte) {
		buf = append(buf, b)
	}

	year, month, day := t.Date()
	itoa(&buf, year, 4)
	char('-')
	itoa(&buf, int(month), 2)
	char('-')
	itoa(&buf, day, 2)
	char('T')
	hour, min, sec := t.Clock()
	itoa(&buf, hour, 2)
	char(':')
	itoa(&buf, min, 2)
	char(':')
	itoa(&buf, sec, 2)
	ns := t.Nanosecond()
	char('.')
	itoa(&buf, ns/1e6, 3)
	_, offsetSeconds := t.Zone()
	if offsetSeconds == 0 {
		char('Z')
	} else {
		offsetMinutes := offsetSeconds / 60
		if offsetMinutes < 0 {
			char('-')
			offsetMinutes = -offsetMinutes
		} else {
			char('+')
		}
		itoa(&buf, offsetMinutes/60, 2)
		char(':')
		itoa(&buf, offsetMinutes%60, 2)
	}
	return buf
}
