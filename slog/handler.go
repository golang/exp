// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"fmt"
	"io"
	"log"
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
	//   - If r.Level() is Level(0), ignore the level.
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

func (*defaultHandler) Enabled(Level) bool { return true }

// Collect the level, attributes and message in a string and
// write it with the default log.Logger.
// Let the log.Logger handle time and file/line.
func (h *defaultHandler) Handle(r Record) error {
	var b strings.Builder
	if r.Level() > 0 {
		b.WriteString(r.Level().String())
		b.WriteByte(' ')
	}
	r.Attrs(func(a Attr) {
		fmt.Fprint(&b, a) // Attr.Format will print key=value
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

	// Ignore records with levels above Level.Level.
	// If nil, accept all levels.
	Level *AtomicLevel

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
	opts              HandlerOptions
	app               appender
	attrSep           byte // char separating attrs from each other
	preformattedAttrs []byte
	mu                sync.Mutex
	w                 io.Writer
}

// Enabled reports whether l is less than or equal to the
// maximum level.
func (h *commonHandler) Enabled(l Level) bool {
	return l <= h.opts.Level.Level()
}

func (h *commonHandler) with(as []Attr) *commonHandler {
	h2 := &commonHandler{
		app:               h.app,
		attrSep:           h.attrSep,
		opts:              h.opts,
		preformattedAttrs: h.preformattedAttrs,
		w:                 h.w,
	}
	// Pre-format the attributes as an optimization.
	state := handleState{
		h2,
		(*buffer.Buffer)(&h2.preformattedAttrs),
		false,
	}
	for _, a := range as {
		state.appendAttr(a)
	}
	return h2
}

func (h *commonHandler) handle(r Record) error {
	rep := h.opts.ReplaceAttr
	state := handleState{h, buffer.New(), false}
	defer state.buf.Free()
	h.app.appendStart(state.buf)
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
	if r.Level() != 0 {
		key := "level"
		val := r.Level()
		if rep == nil {
			state.appendKey(key)
			state.appendString(val.String())
		} else {
			state.appendAttr(Any(key, val))
		}
	}
	// source
	if h.opts.AddSource {
		file, line := r.SourceLine()
		if file != "" {
			key := "source"
			if rep == nil {
				state.appendKey(key)
				h.app.appendSource(state.buf, file, line)
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
	// message
	key := "msg"
	val := r.Message()
	if rep == nil {
		state.appendKey(key)
		state.appendString(val)
	} else {
		state.appendAttr(String(key, val))
	}
	// preformatted Attrs
	if len(h.preformattedAttrs) > 0 {
		state.appendSep()
		state.buf.Write(h.preformattedAttrs)
	}
	// Attrs in Record
	r.Attrs(func(a Attr) {
		state.appendAttr(a)
	})
	h.app.appendEnd(state.buf)
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
	sep bool // Append separator before next Attr?
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
	if a.Key() == "" {
		return
	}
	s.appendKey(a.Key())
	s.appendAttrValue(a)
}

func (s *handleState) appendError(err error) {
	s.appendString(fmt.Sprintf("!ERROR:%v", err))
}

type appender interface {
	appendStart(*buffer.Buffer)                 // start of output
	appendEnd(*buffer.Buffer)                   // end of output
	appendKey(*buffer.Buffer, string)           // append key and key-value separator
	appendString(*buffer.Buffer, string)        // append a string
	appendSource(*buffer.Buffer, string, int)   // append a filename and line
	appendTime(*buffer.Buffer, time.Time) error // append a time
	appendAttrValue(*buffer.Buffer, Attr) error // append Attr's value (but not key)
}

func (s *handleState) appendSep() {
	if s.sep {
		s.buf.WriteByte(s.h.attrSep)
	}
}

func (s *handleState) appendKey(key string) {
	s.appendSep()
	s.h.app.appendKey(s.buf, key)
	s.sep = true
}

func (s *handleState) appendString(str string) {
	s.h.app.appendString(s.buf, str)
}

func (s *handleState) appendAttrValue(a Attr) {
	if err := s.h.app.appendAttrValue(s.buf, a); err != nil {
		s.appendError(err)
	}
}

func (s *handleState) appendTime(t time.Time) {
	if err := s.h.app.appendTime(s.buf, t); err != nil {
		s.appendError(err)
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
