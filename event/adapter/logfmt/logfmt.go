// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logfmt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"unicode"
	"unicode/utf8"

	"golang.org/x/exp/event"
)

//TODO: some actual research into what this arbritray optimization number should be
const bufCap = 50

const TimeFormat = "2006/01/02 15:04:05"

type Printer struct {
	QuoteValues       bool
	SuppressNamespace bool
	buf               [bufCap]byte
	needSep           bool
	w                 bytes.Buffer
}

type Handler struct {
	to io.Writer
	Printer
}

// NewHandler returns a handler that prints the events to the supplied writer.
// Each event is printed in logfmt format on a single line.
func NewHandler(to io.Writer) *Handler {
	return &Handler{to: to}
}

func (h *Handler) Event(ctx context.Context, ev *event.Event) context.Context {
	h.Printer.Event(h.to, ev)
	return ctx
}

func (p *Printer) Event(w io.Writer, ev *event.Event) {
	p.needSep = false
	if !ev.At.IsZero() {
		p.Label(w, event.Bytes("time", ev.At.AppendFormat(p.buf[:0], TimeFormat)))
	}

	if !p.SuppressNamespace && ev.Namespace != "" {
		p.Label(w, event.String("in", ev.Namespace))
	}

	if ev.Parent != 0 {
		p.Label(w, event.Bytes("parent", strconv.AppendUint(p.buf[:0], ev.Parent, 10)))
	}
	for _, l := range ev.Labels {
		if l.Name == "" {
			continue
		}
		p.Label(w, l)
	}

	if ev.TraceID != 0 {
		p.Label(w, event.Uint64("trace", ev.TraceID))
	}

	if ev.Kind == event.EndKind {
		p.Label(w, event.Value("end", nil))
	}

	io.WriteString(w, "\n")
}

func (p *Printer) Label(w io.Writer, l event.Label) {
	if l.Name == "" {
		return
	}
	if p.needSep {
		io.WriteString(w, " ")
	}
	p.needSep = true
	p.Ident(w, l.Name)
	if l.HasValue() {
		io.WriteString(w, "=")
		switch {
		case l.IsString():
			s := l.String()
			if p.QuoteValues || stringNeedQuote(s) {
				p.quoteString(w, s)
			} else {
				io.WriteString(w, s)
			}
		case l.IsBytes():
			buf := l.Bytes()
			if p.QuoteValues || bytesNeedQuote(buf) {
				p.quoteBytes(w, buf)
			} else {
				w.Write(buf)
			}
		case l.IsInt64():
			w.Write(strconv.AppendInt(p.buf[:0], l.Int64(), 10))
		case l.IsUint64():
			w.Write(strconv.AppendUint(p.buf[:0], l.Uint64(), 10))
		case l.IsFloat64():
			w.Write(strconv.AppendFloat(p.buf[:0], l.Float64(), 'g', -1, 64))
		case l.IsBool():
			if l.Bool() {
				io.WriteString(w, "true")
			} else {
				io.WriteString(w, "false")
			}
		default:
			if p.w.Cap() == 0 {
				// we rely on the inliner to cause this to not allocate
				p.w = *bytes.NewBuffer(p.buf[:0])
			}
			fmt.Fprint(&p.w, l.Interface())
			b := p.w.Bytes()
			p.w.Reset()
			if p.QuoteValues || bytesNeedQuote(b) {
				p.quoteBytes(w, b)
			} else {
				w.Write(b)
			}
		}
	}
}

func (p *Printer) Ident(w io.Writer, s string) {
	if !stringNeedQuote(s) {
		io.WriteString(w, s)
		return
	}
	p.quoteString(w, s)
}

func (p *Printer) quoteString(w io.Writer, s string) {
	io.WriteString(w, `"`)
	written := 0
	for offset, r := range s {
		q := quoteRune(r)
		if len(q) == 0 {
			continue
		}
		// write out any prefix
		io.WriteString(w, s[written:offset])
		written = offset + utf8.RuneLen(r)
		// and write out the quoted rune
		io.WriteString(w, q)
	}
	io.WriteString(w, s[written:])
	io.WriteString(w, `"`)
}

// Bytes writes a byte array in string form to the printer.
func (p *Printer) quoteBytes(w io.Writer, buf []byte) {
	io.WriteString(w, `"`)
	written := 0
	for offset := 0; offset < len(buf); {
		r, size := utf8.DecodeRune(buf[offset:])
		offset += size
		q := quoteRune(r)
		if len(q) == 0 {
			continue
		}
		// write out any prefix
		w.Write(buf[written : offset-size])
		written = offset
		// and write out the quoted rune
		io.WriteString(w, q)
	}
	w.Write(buf[written:])
	io.WriteString(w, `"`)
}

func stringNeedQuote(s string) bool {
	if len(s) == 0 {
		return true
	}
	for _, r := range s {
		if runeForcesQuote(r) {
			return true
		}
	}
	return false
}

func bytesNeedQuote(buf []byte) bool {
	for offset := 0; offset < len(buf); {
		r, size := utf8.DecodeRune(buf[offset:])
		offset += size
		if runeForcesQuote(r) {
			return true
		}
	}
	return false
}

func runeForcesQuote(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsNumber(r)
}

func quoteRune(r rune) string {
	switch r {
	case '"':
		return `\"`
	case '\n':
		return `\n`
	case '\\':
		return `\\`
	default:
		return ``
	}
}
