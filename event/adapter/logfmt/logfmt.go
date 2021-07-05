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
	"time"
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
		p.separator(w)
		io.WriteString(w, `time="`)
		p.time(w, ev.At)
		io.WriteString(w, `"`)
	}

	if !p.SuppressNamespace && ev.Source.Space != "" {
		p.Label(w, event.String("in", ev.Source.Space))
	}
	if ev.Source.Owner != "" {
		p.Label(w, event.String("owner", ev.Source.Owner))
	}
	if ev.Source.Name != "" {
		p.Label(w, event.String("name", ev.Source.Name))
	}

	if ev.Parent != 0 {
		p.separator(w)
		io.WriteString(w, `parent=`)
		w.Write(strconv.AppendUint(p.buf[:0], ev.Parent, 10))
	}
	for _, l := range ev.Labels {
		if l.Name == "" {
			continue
		}
		p.Label(w, l)
	}

	if ev.ID != 0 && ev.Kind == event.StartKind {
		p.separator(w)
		io.WriteString(w, `trace=`)
		w.Write(strconv.AppendUint(p.buf[:0], ev.ID, 10))
	}

	if ev.Kind == event.EndKind {
		p.separator(w)
		io.WriteString(w, `end`)
	}

	io.WriteString(w, "\n")
}

func (p *Printer) separator(w io.Writer) {
	if p.needSep {
		io.WriteString(w, " ")
	}
	p.needSep = true
}

func (p *Printer) Label(w io.Writer, l event.Label) {
	if l.Name == "" {
		return
	}
	p.separator(w)
	p.Ident(w, l.Name)
	if l.HasValue() {
		io.WriteString(w, "=")
		switch {
		case l.IsString():
			p.string(w, l.String())
		case l.IsBytes():
			p.bytes(w, l.Bytes())
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
			v := l.Interface()
			switch v := v.(type) {
			case string:
				p.string(w, v)
			case fmt.Stringer:
				p.string(w, v.String())
			default:
				if p.w.Cap() == 0 {
					// we rely on the inliner to cause this to not allocate
					p.w = *bytes.NewBuffer(p.buf[:0])
				}
				fmt.Fprint(&p.w, v)
				b := p.w.Bytes()
				p.w.Reset()
				p.bytes(w, b)
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

func (p *Printer) string(w io.Writer, s string) {
	if p.QuoteValues || stringNeedQuote(s) {
		p.quoteString(w, s)
	} else {
		io.WriteString(w, s)
	}
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

func (p *Printer) bytes(w io.Writer, buf []byte) {
	if p.QuoteValues || stringNeedQuote(string(buf)) {
		p.quoteBytes(w, buf)
	} else {
		w.Write(buf)
	}
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

// time writes a timstamp in the same format as
func (p *Printer) time(w io.Writer, t time.Time) {
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	p.padInt(w, int64(year), 4)
	io.WriteString(w, `/`)
	p.padInt(w, int64(month), 2)
	io.WriteString(w, `/`)
	p.padInt(w, int64(day), 2)
	io.WriteString(w, ` `)
	p.padInt(w, int64(hour), 2)
	io.WriteString(w, `:`)
	p.padInt(w, int64(minute), 2)
	io.WriteString(w, `:`)
	p.padInt(w, int64(second), 2)
}

func (p *Printer) padInt(w io.Writer, v int64, width int) {
	b := strconv.AppendInt(p.buf[:0], int64(v), 10)
	if len(b) < width {
		io.WriteString(w, "0000"[:width-len(b)])
	}
	w.Write(b)
}

func stringNeedQuote(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= utf8.RuneSelf || c == ' ' || c == '"' || c == '\n' || c == '\\' {
			return true
		}
	}
	return false
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
