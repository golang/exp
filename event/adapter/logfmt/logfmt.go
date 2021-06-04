// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logfmt

import (
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
	buf     [bufCap]byte
	needSep bool
}

type Handler struct {
	to      io.Writer
	printer Printer
}

// NewHandler returns a handler that prints the events to the supplied writer.
// Each event is printed in logfmt format on a single line.
func NewHandler(to io.Writer) *Handler {
	return &Handler{to: to}
}

func (h *Handler) Log(ctx context.Context, ev *event.Event) {
	h.printer.Event(h.to, ev)
}

func (h *Handler) Metric(ctx context.Context, ev *event.Event) {
	h.printer.Event(h.to, ev)
}

func (h *Handler) Annotate(ctx context.Context, ev *event.Event) {
	h.printer.Event(h.to, ev)
}

func (h *Handler) Start(ctx context.Context, ev *event.Event) context.Context {
	h.printer.Event(h.to, ev)
	return ctx
}

func (h *Handler) End(ctx context.Context, ev *event.Event) {
	h.printer.Event(h.to, ev)
}

func (p *Printer) Event(w io.Writer, ev *event.Event) {
	p.needSep = false
	if !ev.At.IsZero() {
		p.label(w, "time", event.BytesOf(ev.At.AppendFormat(p.buf[:0], TimeFormat)))
	}

	if ev.Parent != 0 {
		p.label(w, "parent", event.BytesOf(strconv.AppendUint(p.buf[:0], ev.Parent, 10)))
	}

	for _, l := range ev.Labels {
		if l.Name == "" {
			continue
		}
		p.Label(w, l)
	}
	io.WriteString(w, "\n")
}

func (p *Printer) Label(w io.Writer, l event.Label) {
	p.label(w, l.Name, l.Value)
}

func (p *Printer) Value(w io.Writer, v event.Value) {
	switch {
	case v.IsString():
		p.Quote(w, v.String())
	case v.IsBytes():
		p.Bytes(w, v.Bytes())
	case v.IsInt64():
		w.Write(strconv.AppendInt(p.buf[:0], v.Int64(), 10))
	case v.IsUint64():
		w.Write(strconv.AppendUint(p.buf[:0], v.Uint64(), 10))
	case v.IsFloat64():
		w.Write(strconv.AppendFloat(p.buf[:0], v.Float64(), 'g', -1, 64))
	case v.IsBool():
		if v.Bool() {
			io.WriteString(w, "true")
		} else {
			io.WriteString(w, "false")
		}
	default:
		fmt.Fprint(w, v.Interface())
	}
}

func (p *Printer) Ident(w io.Writer, s string) {
	p.Quote(w, s)
}

func (p *Printer) Quote(w io.Writer, s string) {
	if !stringNeedQuote(s) {
		io.WriteString(w, s)
		return
	}
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
func (p *Printer) Bytes(w io.Writer, buf []byte) {
	if !bytesNeedQuote(buf) {
		w.Write(buf)
		return
	}
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

func (p *Printer) label(w io.Writer, name string, value event.Value) {
	if name == "" {
		return
	}
	if p.needSep {
		io.WriteString(w, " ")
	}
	p.needSep = true
	p.Ident(w, name)
	if value.HasValue() {
		io.WriteString(w, "=")
		p.Value(w, value)
	}
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
