// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"runtime"
	"time"
)

const nAttrsInline = 5

// A Record holds information about a log event.
type Record struct {
	// The time at which the output method (Log, Info, etc.) was called.
	time time.Time

	// The log message.
	message string

	// The level of the event.
	level Level

	// The pc at the time the record was constructed, as determined
	// by runtime.Callers using the calldepth argument to NewRecord.
	pc uintptr

	// Allocation optimization: an inline array sized to hold
	// the majority of log calls (based on examination of open-source
	// code). The array holds the end of the sequence of Attrs.
	tail [nAttrsInline]Attr

	// The number of Attrs in tail.
	nTail int

	// The sequence of Attrs except for the tail, represented as a functional
	// list of arrays.
	attrs list[[nAttrsInline]Attr]
}

// MakeRecord creates a new Record from the given arguments.
// Use [Record.AddAttr] to add attributes to the Record.
// If calldepth is greater than zero, [Record.SourceLine] will
// return the file and line number at that depth,
// where 1 means the caller of MakeRecord.
//
// MakeRecord is intended for logging APIs that want to support a [Handler] as
// a backend.
func MakeRecord(t time.Time, level Level, msg string, calldepth int) Record {
	var p uintptr
	if calldepth > 0 {
		p = pc(calldepth + 1)
	}
	return Record{
		time:    t,
		message: msg,
		level:   level,
		pc:      p,
	}
}

func pc(depth int) uintptr {
	var pcs [1]uintptr
	runtime.Callers(depth, pcs[:])
	return pcs[0]
}

// Time returns the time of the log event.
func (r *Record) Time() time.Time { return r.time }

// Message returns the log message.
func (r *Record) Message() string { return r.message }

// Level returns the level of the log event.
func (r *Record) Level() Level { return r.level }

// SourceLine returns the file and line of the log event.
// If the Record was created without the necessary information,
// or if the location is unavailable, it returns ("", 0).
func (r *Record) SourceLine() (file string, line int) {
	fs := runtime.CallersFrames([]uintptr{r.pc})
	// TODO: error-checking?
	f, _ := fs.Next()
	return f.File, f.Line
}

// Attrs returns a copy of the sequence of Attrs in r.
func (r *Record) Attrs() []Attr {
	res := make([]Attr, 0, r.attrs.len()*nAttrsInline+r.nTail)
	r.attrs = r.attrs.normalize()
	for _, f := range r.attrs.front {
		res = append(res, f[:]...)
	}
	for _, a := range r.tail[:r.nTail] {
		res = append(res, a)
	}
	return res
}

// NumAttrs returns the number of Attrs in r.
func (r *Record) NumAttrs() int {
	return r.attrs.len()*nAttrsInline + r.nTail
}

// Attr returns the i'th Attr in r.
func (r *Record) Attr(i int) Attr {
	if r.attrs.back != nil {
		r.attrs = r.attrs.normalize()
	}
	alen := r.attrs.len() * nAttrsInline
	if i < alen {
		return r.attrs.at(i / nAttrsInline)[i%nAttrsInline]
	}
	return r.tail[i-alen]
}

// AddAttr appends an attributes to the record's list of attributes.
// It does not check for duplicate keys.
func (r *Record) AddAttr(a Attr) {
	if r.nTail == len(r.tail) {
		r.attrs = r.attrs.append(r.tail)
		r.nTail = 0
	}
	r.tail[r.nTail] = a
	r.nTail++
}
