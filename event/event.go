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
	ID     uint64    // unique for this process id of the event
	Parent uint64    // id of the parent event for this event
	At     time.Time // time at which the event is delivered to the exporter.
	Labels []Label
}

// LogHandler is a the type for something that handles log events as they occur.
type LogHandler interface {
	// Log indicates a logging event.
	Log(context.Context, *Event)
}

// MetricHandler is a the type for something that handles metric events as they
// occur.
type MetricHandler interface {
	// Metric indicates a metric record event.
	Metric(context.Context, *Event)
}

// AnnotateHandler is a the type for something that handles annotate events as
// they occur.
type AnnotateHandler interface {
	// Annotate reports label values at a point in time.
	Annotate(context.Context, *Event)
}

// TraceHandler is a the type for something that handles start and end events as
// they occur.
type TraceHandler interface {
	// Start indicates a trace start event.
	Start(context.Context, *Event) context.Context
	// End indicates a trace end event.
	End(context.Context, *Event)
}

// WithExporter returns a context with the exporter attached.
// The exporter is called synchronously from the event call site, so it should
// return quickly so as not to hold up user code.
func WithExporter(ctx context.Context, e *Exporter) context.Context {
	return newContext(ctx, e, 0)
}

// SetDefaultExporter sets an exporter that is used if no exporter can be
// found on the context.
func SetDefaultExporter(e *Exporter) {
	setDefaultExporter(e)
}
