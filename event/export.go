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
	lastEvent uint64 // accessed using atomic, must be 64 bit aligned
	opts      ExporterOptions

	mu      sync.Mutex
	handler Handler
	sources sources
}

// target is a bound exporter.
// Normally you get a target by looking in the context using To.
type target struct {
	exporter  *Exporter
	parent    uint64
	startTime time.Time // for trace latency
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

var (
	defaultTarget unsafe.Pointer
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
	atomic.StorePointer(&defaultTarget, unsafe.Pointer(&target{exporter: e}))
}

func getDefaultTarget() *target {
	return (*target)(atomic.LoadPointer(&defaultTarget))
}

func newContext(ctx context.Context, exporter *Exporter, parent uint64, start time.Time) context.Context {
	var t *target
	if exporter != nil {
		t = &target{exporter: exporter, parent: parent, startTime: start}
	}
	return context.WithValue(ctx, contextKey, t)
}

// prepare events before delivering to the underlying handler.
// it is safe to call this more than once (trace events have to call it early)
// If the event does not have a timestamp, and the exporter has a Now function
// then the timestamp will be updated.
// If automatic namespaces are enabled and the event doesn't have a namespace,
// one based on the caller's import path will be provided.
func (ev *Event) prepare() {
	e := ev.target.exporter
	if ev.ID == 0 {
		ev.ID = atomic.AddUint64(&e.lastEvent, 1)
	}
	if e.opts.Now != nil && ev.At.IsZero() {
		ev.At = e.opts.Now()
	}
	if e.opts.EnableNamespaces && ev.Source.Space == "" {
		ev.Source = e.sources.scanStack()
	}
}

func (e *Exporter) loggingEnabled() bool     { return !e.opts.DisableLogging }
func (e *Exporter) annotationsEnabled() bool { return !e.opts.DisableAnnotations }
func (e *Exporter) tracingEnabled() bool     { return !e.opts.DisableTracing }
func (e *Exporter) metricsEnabled() bool     { return !e.opts.DisableMetrics }
