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
	io.Writer
	io.StringWriter

	buf     [bufCap]byte
	needSep bool
}

type stringWriter struct {
	io.Writer
}

// NewPrinter returns a handler that prints the events to the supplied writer.
// Each event is printed in logfmt format on a single line.
func NewPrinter(to io.Writer) *Printer {
	p := &Printer{Writer: to}
	ok := false
	p.StringWriter, ok = to.(io.StringWriter)
	if !ok {
		p.StringWriter = &stringWriter{to}
	}
	return p
}

func (p *Printer) Log(ctx context.Context, ev *event.Event) {
	p.Event("log", ev)
}

func (p *Printer) Metric(ctx context.Context, ev *event.Event) {
	p.Event("metric", ev)
}

func (p *Printer) Annotate(ctx context.Context, ev *event.Event) {
	p.Event("annotate", ev)
}

func (p *Printer) Start(ctx context.Context, ev *event.Event) context.Context {
	p.Event("start", ev)
	return ctx
}

func (p *Printer) End(ctx context.Context, ev *event.Event) {
	p.Event("end", ev)
}

func (p *Printer) Event(kind string, ev *event.Event) {
	const timeFormat = "2006-01-02T15:04:05"
	p.needSep = false
	if !ev.At.IsZero() {
		p.label("time", event.BytesOf(ev.At.AppendFormat(p.buf[:0], timeFormat)))
	}

	p.label("id", event.BytesOf(strconv.AppendUint(p.buf[:0], ev.ID, 10)))
	if ev.Parent != 0 {
		p.label("span", event.BytesOf(strconv.AppendUint(p.buf[:0], ev.Parent, 10)))
	}

	p.label("kind", event.StringOf(kind))

	for _, l := range ev.Labels {
		if l.Name == "" {
			continue
		}
		p.Label(l)
	}
	p.WriteString("\n")
}

func (p *Printer) Label(l event.Label) {
	p.label(l.Name, l.Value)
}

func (p *Printer) Value(v event.Value) {
	switch {
	case v.IsString():
		p.Quote(v.String())
	case v.IsBytes():
		p.Bytes(v.Bytes())
	case v.IsInt64():
		p.Write(strconv.AppendInt(p.buf[:0], v.Int64(), 10))
	case v.IsUint64():
		p.Write(strconv.AppendUint(p.buf[:0], v.Uint64(), 10))
	case v.IsFloat64():
		p.Write(strconv.AppendFloat(p.buf[:0], v.Float64(), 'g', -1, 64))
	case v.IsBool():
		if v.Bool() {
			p.WriteString("true")
		} else {
			p.WriteString("false")
		}
	default:
		fmt.Fprint(p, v.Interface())
	}
}

func (p *Printer) Ident(s string) {
	//TODO: this should also escape = if it occurs in an ident?
	p.Quote(s)
}

func (p *Printer) Quote(s string) {
	if s == "" {
		p.WriteString(`""`)
		return
	}
	if !needQuote(s) {
		p.WriteString(s)
		return
	}
	// string needs quoting
	p.WriteString(`"`)
	written := 0
	for o, r := range s {
		q := quoteRune(r)
		if len(q) == 0 {
			continue
		}
		// write out any prefix
		p.WriteString(s[written:o])
		written = o + 1 // we can plus 1 because all runes we escape are ascii
		// and write out the quoted rune
		p.WriteString(q)
	}
	p.WriteString(s[written:])
	p.WriteString(`"`)
}

// Bytes writes a byte array in string form to the printer.
func (p *Printer) Bytes(buf []byte) {
	//TODO: non asci chars need escaping
	p.Write(buf)
}

func (p *Printer) label(name string, value event.Value) {
	if name == "" {
		return
	}
	if p.needSep {
		p.WriteString(" ")
	}
	p.needSep = true
	p.Ident(name)
	if value.HasValue() {
		p.WriteString("=")
		p.Value(value)
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

func (w *stringWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
