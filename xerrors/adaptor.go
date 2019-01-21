// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xerrors

import (
	"bytes"
	"fmt"
)

func fmtError(p *pp, verb rune, err error) (handled bool) {
	var (
		sep = " " // separator before next error
		w   = p   // print buffer where error text is written
	)
	switch {
	// Note that this switch must match the preference order
	// for ordinary string printing (%#v before %+v, and so on).

	case p.fmt.sharpV:
		if stringer, ok := p.arg.(GoStringer); ok {
			// Print the result of GoString unadorned.
			p.fmt.fmtS(stringer.GoString())
			return true
		}
		return false

	case p.fmt.plusV:
		sep = "\n  - "
		w.fmt.fmtFlags = fmtFlags{plusV: p.fmt.plusV} // only keep detail flag

		// The width or precision of a detailed view could be the number of
		// errors to print from a list.

	default:
		// Use an intermediate buffer in the rare cases that precision,
		// truncation, or one of the alternative verbs (q, x, and X) are
		// specified.
		switch verb {
		case 's', 'v':
			if (!w.fmt.widPresent || w.fmt.wid == 0) && !w.fmt.precPresent {
				break
			}
			fallthrough
		case 'q', 'x', 'X':
			w = newPrinter()
			defer w.free()
		default:
			w.badVerb(verb)
			return true
		}
	}

loop:
	for {
		w.fmt.inDetail = false
		switch v := err.(type) {
		case Formatter:
			err = v.FormatError((*errPP)(w))
		case fmt.Formatter:
			if w.fmt.plusV {
				v.Format((*errPPState)(w), 'v') // indent new lines
			} else {
				v.Format(w, 'v') // do not indent new lines
			}
			break loop
		default:
			w.fmtString(v.Error(), 's')
			break loop
		}
		if err == nil {
			break
		}
		if !w.fmt.inDetail || !p.fmt.plusV {
			w.buf.WriteByte(':')
		}
		// Strip last newline of detail.
		if bytes.HasSuffix([]byte(w.buf), detailSep) {
			w.buf = w.buf[:len(w.buf)-len(detailSep)]
		}
		w.buf.WriteString(sep)
		w.fmt.inDetail = false
	}

	if w != p {
		p.fmtString(string(w.buf), verb)
	}
	return true
}

var detailSep = []byte("\n    ")

// errPPState wraps a pp to implement State with indentation. It is used
// for errors implementing fmt.Formatter.
type errPPState pp

func (p *errPPState) Width() (wid int, ok bool)      { return (*pp)(p).Width() }
func (p *errPPState) Precision() (prec int, ok bool) { return (*pp)(p).Precision() }
func (p *errPPState) Flag(c int) bool                { return (*pp)(p).Flag(c) }

func (p *errPPState) Write(b []byte) (n int, err error) {
	if !p.fmt.inDetail || p.fmt.plusV {
		if len(b) == 0 {
			return 0, nil
		}
		if p.fmt.inDetail && p.fmt.needNewline {
			p.fmt.needNewline = false
			p.buf.WriteByte(':')
			p.buf.Write(detailSep)
			if b[0] == '\n' {
				b = b[1:]
			}
		}
		k := 0
		for i, c := range b {
			if c == '\n' {
				p.buf.Write(b[k:i])
				p.buf.Write(detailSep)
				k = i + 1
			}
		}
		p.buf.Write(b[k:])
		p.fmt.needNewline = !p.fmt.inDetail
	}
	return len(b), nil
}

// errPP wraps a pp to implement a Printer.
type errPP pp

func (p *errPP) Print(args ...interface{}) {
	if !p.fmt.inDetail || p.fmt.plusV {
		if p.fmt.plusV {
			Fprint((*errPPState)(p), args...)
		} else {
			(*pp)(p).doPrint(args)
		}
	}
}

func (p *errPP) Printf(format string, args ...interface{}) {
	if !p.fmt.inDetail || p.fmt.plusV {
		if p.fmt.plusV {
			Fprintf((*errPPState)(p), format, args...)
		} else {
			(*pp)(p).doPrintf(format, args)
		}
	}
}

func (p *errPP) Detail() bool {
	p.fmt.inDetail = true
	return p.fmt.plusV
}
