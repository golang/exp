// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"io"
	"strconv"
)

// Printer is the interface to something capable of printing standard types
// used in events.
// It is used to enable zero allocation printing support.
// Implementations can be directly used as an Exporter.
type Printer interface {
	Handler
	Label(l Label)
	String(v string)
	Quote(v string)
	Int(v int64)
	Uint(v uint64)
	Float(v float64)
	Value(v interface{})
}

// NewPrinter returns a Printer that prints to the supplied writer.
func NewPrinter(w io.Writer) Printer {
	return &printer{writer: w}
}

type printer struct {
	buf    [128]byte
	writer io.Writer
}

func (p *printer) Handle(ev *Event) {
	const timeFormat = "2006/01/02 15:04:05"
	if !ev.At.IsZero() {
		p.writer.Write(ev.At.AppendFormat(p.buf[:0], timeFormat))
	}
	io.WriteString(p.writer, " [")
	io.WriteString(p.writer, ev.Kind.String())
	io.WriteString(p.writer, ":")
	p.writer.Write(strconv.AppendUint(p.buf[:0], ev.ID, 10))
	if ev.Parent != 0 {
		io.WriteString(p.writer, ":")
		p.writer.Write(strconv.AppendUint(p.buf[:0], ev.Parent, 10))
	}
	io.WriteString(p.writer, "]")
	if ev.Message != "" {
		io.WriteString(p.writer, " ")
		io.WriteString(p.writer, ev.Message)
	}
	for _, l := range ev.Static {
		if !l.Valid() {
			continue
		}
		io.WriteString(p.writer, "\n\t")
		p.Label(l)
	}
	for _, l := range ev.Dynamic {
		if !l.Valid() {
			continue
		}
		io.WriteString(p.writer, "\n\t")
		p.Label(l)
	}
	io.WriteString(p.writer, "\n")
}

func (p *printer) Label(l Label) {
	io.WriteString(p.writer, l.key)
	if l.dispatch != nil {
		io.WriteString(p.writer, "=")
		l.dispatch(p, l)
	}
}

func (p *printer) String(v string) {
	io.WriteString(p.writer, v)
}

func (p *printer) Quote(v string) {
	p.writer.Write(strconv.AppendQuote(p.buf[:0], v))
}

func (p *printer) Int(v int64) {
	p.writer.Write(strconv.AppendInt(p.buf[:0], v, 10))
}

func (p *printer) Uint(v uint64) {
	p.writer.Write(strconv.AppendUint(p.buf[:0], v, 10))
}

func (p *printer) Float(v float64) {
	p.writer.Write(strconv.AppendFloat(p.buf[:0], v, 'E', -1, 32))
}

func (p *printer) Value(v interface{}) {
	fmt.Fprint(p.writer, v)
}
