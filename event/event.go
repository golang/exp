// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
)

// Event holds the information about an event that occurred.
// It combines the event metadata with the user supplied labels.
type Event struct {
	Kind    Kind
	ID      uint64    // unique for this process id of the event
	Parent  uint64    // id of the parent event for this event
	At      time.Time // time at which the event is delivered to the exporter.
	Message string
	Labels  []Label
}

// Kind indicates the type of event.
type Kind byte

const (
	// UnknownKind is the default event kind, a real kind should always be chosen.
	UnknownKind = Kind(iota)
	// LogKind is a Labels kind that indicates a log event.
	LogKind
	// StartKind is a Labels kind that indicates a span start event.
	StartKind
	// EndKind is a Labels kind that indicates a span end event.
	EndKind
	// MetricKind is a Labels kind that indicates a metric record event.
	MetricKind
	// AnnotateKind is a Labels kind that reports label values at a point in time.
	AnnotateKind
)

// Format prints the value in a standard form.
func (e *Event) Format(f fmt.State, verb rune) {
	buf := bufPool.Get().(*buffer)
	e.format(f.(writer), buf.data[:0])
	bufPool.Put(buf)
}

// Format prints the value in a standard form.
func (e *Event) format(w writer, buf []byte) {
	const timeFormat = "2006/01/02 15:04:05"
	if !e.At.IsZero() {
		w.Write(e.At.AppendFormat(buf[:0], timeFormat))
		w.WriteString("\t")
	}
	//TODO: pick a standard format for the event id and parent
	w.WriteString("[")
	w.Write(strconv.AppendUint(buf[:0], e.ID, 10))
	if e.Parent != 0 {
		w.WriteString(":")
		w.Write(strconv.AppendUint(buf[:0], e.Parent, 10))
	}
	w.WriteString("]")

	//TODO: pick a standard format for the kind
	w.WriteString("\t")
	e.Kind.format(w, buf)

	if e.Message != "" {
		w.WriteString("\t")
		w.WriteString(e.Message)
	}

	first := true
	for _, l := range e.Labels {
		if l.Name == "" {
			continue
		}
		if first {
			w.WriteString("\t{")
			first = false
		} else {
			w.WriteString(", ")
		}
		l.format(w, buf)
	}
	if !first {
		w.WriteString("}")
	}
}

func (k Kind) Format(f fmt.State, verb rune) {
	buf := bufPool.Get().(*buffer)
	k.format(f.(writer), buf.data[:0])
	bufPool.Put(buf)
}

func (k Kind) format(w writer, buf []byte) {
	switch k {
	case LogKind:
		w.WriteString("log")
	case StartKind:
		w.WriteString("start")
	case EndKind:
		w.WriteString("end")
	case MetricKind:
		w.WriteString("metric")
	case AnnotateKind:
		w.WriteString("annotate")
	default:
		w.Write(strconv.AppendUint(buf[:0], uint64(k), 10))
	}
}

// Printer returns a handler that prints the events to the supplied writer.
// Each event is printed in normal %v mode on its own line.
func Printer(to io.Writer) Handler {
	return &printHandler{to: to}
}

type printHandler struct {
	to io.Writer
}

func (h *printHandler) Handle(ev *Event) {
	fmt.Fprintln(h.to, ev)
}

//TODO: some actual research into what this arbritray optimization number should be
const bufCap = 50

type buffer struct{ data [bufCap]byte }

var bufPool = sync.Pool{New: func() interface{} { return new(buffer) }}

type writer interface {
	io.Writer
	io.StringWriter
}
