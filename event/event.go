// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"time"
)

// Event holds the information about an event that occurred.
// It combines the event metadata with the user supplied labels.
type Event struct {
	TraceID   uint64
	Parent    uint64    // id of the parent event for this event
	Namespace string    // namespace of event; if empty, set by exporter to import path
	At        time.Time // time at which the event is delivered to the exporter.
	Kind      Kind
	Message   string
	Name      string
	Error     error
	Labels    []Label
}

// Handler is a the type for something that handles events as they occur.
type Handler interface {
	// Log indicates a logging event.
	Log(context.Context, *Event)
	// Metric indicates a metric record event.
	Metric(context.Context, *Event)
	// Annotate reports label values at a point in time.
	Annotate(context.Context, *Event)
	// Start indicates a trace start event.
	Start(context.Context, *Event) context.Context
	// End indicates a trace end event.
	End(context.Context, *Event)
}

// Matcher is the interface to something that can check if an event matches
// a condition.
type Matcher interface {
	Matches(ev *Event) bool
}

// WithExporter returns a context with the exporter attached.
// The exporter is called synchronously from the event call site, so it should
// return quickly so as not to hold up user code.
func WithExporter(ctx context.Context, e *Exporter) context.Context {
	return newContext(ctx, e, 0, time.Time{})
}

// SetDefaultExporter sets an exporter that is used if no exporter can be
// found on the context.
func SetDefaultExporter(e *Exporter) {
	setDefaultExporter(e)
}

// Is uses the matcher to check if the event is a match.
// This is a simple helper to convert code like
//   event.End.Matches(ev)
// to the more readable
//   ev.Is(event.End)
func (ev *Event) Is(m Matcher) bool {
	return m.Matches(ev)
}

// NopHandler is a handler that does nothing. It can be used for tests, or
// embedded in a struct to avoid having to implement all the Handler methods.
type NopHandler struct{}

func (NopHandler) Log(context.Context, *Event)      {}
func (NopHandler) Metric(context.Context, *Event)   {}
func (NopHandler) Annotate(context.Context, *Event) {}
func (NopHandler) End(context.Context, *Event)      {}
func (NopHandler) Start(ctx context.Context, _ *Event) context.Context {
	return ctx
}
