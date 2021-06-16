// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Exporter synchronizes the delivery of events to handlers.
type Exporter struct {
	opts ExporterOptions

	mu        sync.Mutex
	handler   Handler
	lastEvent uint64
	sources   sources
}

type ExporterOptions struct {
	// If non-nil, sets zero Event.At on delivery.
	Now func() time.Time

	// Disable some event types, for better performance.
	DisableLogging     bool
	DisableTracing     bool
	DisableAnnotations bool
	DisableMetrics     bool

	// Enable automatically setting the event Namespace to the calling package's
	// import path.
	EnableNamespaces bool
}

// contextKey is used as the key for storing a contextValue on the context.
type contextKeyType struct{}

var contextKey interface{} = contextKeyType{}

// contextValue is stored by value in the context to track the exporter and
// current parent event.
type contextValue struct {
	exporter  *Exporter
	parent    uint64
	startTime time.Time // for trace latency
}

var (
	defaultExporter unsafe.Pointer
)

// NewExporter creates an Exporter using the supplied handler and options.
// Event delivery is serialized to enable safe atomic handling.
func NewExporter(handler Handler, opts *ExporterOptions) *Exporter {
	if handler == nil {
		panic("handler must not be nil")
	}
	e := &Exporter{
		handler: handler,
		sources: newCallers(),
	}
	if opts != nil {
		e.opts = *opts
	}
	if e.opts.Now == nil {
		e.opts.Now = time.Now
	}
	return e
}

func setDefaultExporter(e *Exporter) {
	atomic.StorePointer(&defaultExporter, unsafe.Pointer(e))
}

func getDefaultExporter() *Exporter {
	return (*Exporter)(atomic.LoadPointer(&defaultExporter))
}

func newContext(ctx context.Context, exporter *Exporter, parent uint64, t time.Time) context.Context {
	return context.WithValue(ctx, contextKey, contextValue{exporter: exporter, parent: parent, startTime: t})
}

// FromContext returns the exporter, parentID and parent start time for the supplied context.
func FromContext(ctx context.Context) (*Exporter, uint64, time.Time) {
	if v, ok := ctx.Value(contextKey).(contextValue); ok {
		return v.exporter, v.parent, v.startTime
	}
	return getDefaultExporter(), 0, time.Time{}
}

// prepare events before delivering to the underlying handler.
// The event will be assigned a new ID.
// If the event does not have a timestamp, and the exporter has a Now function
// then the timestamp will be updated.
// If automatic namespaces are enabled and the event doesn't have a namespace,
// one based on the caller's import path will be provided.
// prepare must be called with the export mutex held.
func (e *Exporter) prepare(ev *Event) {
	if e.opts.Now != nil && ev.At.IsZero() {
		ev.At = e.opts.Now()
	}
	if e.opts.EnableNamespaces && ev.Namespace == "" {
		ev.Namespace = e.sources.scanStack().Space
	}
}

func (e *Exporter) loggingEnabled() bool     { return !e.opts.DisableLogging }
func (e *Exporter) annotationsEnabled() bool { return !e.opts.DisableAnnotations }
func (e *Exporter) tracingEnabled() bool     { return !e.opts.DisableTracing }
func (e *Exporter) metricsEnabled() bool     { return !e.opts.DisableMetrics }
