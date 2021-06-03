// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logfmt

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"golang.org/x/exp/event"
)

//TODO: some actual research into what this arbritray optimization number should be
const bufCap = 50

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
	const timeFormat = "2006-01-02T15:04:05"
	p.needSep = false
	if !ev.At.IsZero() {
		p.label(w, "time", event.BytesOf(ev.At.AppendFormat(p.buf[:0], timeFormat)))
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
	//TODO: this should also escape = if it occurs in an ident?
	p.Quote(w, s)
}

func (p *Printer) Quote(w io.Writer, s string) {
	if s == "" {
		io.WriteString(w, `""`)
		return
	}
	if !needQuote(s) {
		io.WriteString(w, s)
		return
	}
	// string needs quoting
	io.WriteString(w, `"`)
	written := 0
	for o, r := range s {
		q := quoteRune(r)
		if len(q) == 0 {
			continue
		}
		// write out any prefix
		io.WriteString(w, s[written:o])
		written = o + 1 // we can plus 1 because all runes we escape are ascii
		// and write out the quoted rune
		io.WriteString(w, q)
	}
	io.WriteString(w, s[written:])
	io.WriteString(w, `"`)
}

// Bytes writes a byte array in string form to the printer.
func (p *Printer) Bytes(w io.Writer, buf []byte) {
	//TODO: non asci chars need escaping
	w.Write(buf)
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

func needQuote(s string) bool {
	for _, r := range s {
		if len(quoteRune(r)) > 0 {
			return true
		}
	}
	return false
}

func quoteRune(r rune) string {
	switch r {
	case '"':
		return `\"`
	case ' ':
		return ` ` // does not change but forces quoting
	case '\n':
		return `\n`
	case '\\':
		return `\\`
	default:
		return ``
	}
}
