// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Handler is a the type for something that handles events as they occur.
type Handler interface {
	// Handle is called for each event delivered to the system.
	Handle(*Event)
}

// Exporter synchronizes the delivery of events to handlers.
type Exporter struct {
	Now func() time.Time

	mu        sync.Mutex
	handler   Handler
	lastEvent uint64
}

// contextKey is used as the key for storing a contextValue on the context.
type contextKey struct{}

// contextValue is stored by value in the context to track the exporter and
// current parent event.
type contextValue struct {
	exporter *Exporter
	parent   uint64
}

var (
	activeExporters int32 // used atomically to shortcut the entire system
)

// NewExporter creates an Exporter using the supplied handler.
// Event delivery is serialized to enable safe atomic handling.
// It also marks the event system as active.
func NewExporter(h Handler) *Exporter {
	atomic.StoreInt32(&activeExporters, 1)
	return &Exporter{
		Now:     time.Now,
		handler: h,
	}
}

// WithExporter returns a context with the exporter attached.
// The exporter is called synchronously from the event call site, so it should
// return quickly so as not to hold up user code.
func WithExporter(ctx context.Context, e *Exporter) context.Context {
	atomic.StoreInt32(&activeExporters, 1)
	return context.WithValue(ctx, contextKey{}, contextValue{exporter: e})
}

// Disable turns off the exporters, until the next WithExporter call.
func Disable() {
	atomic.StoreInt32(&activeExporters, 0)
}

// Builder returns a new builder for the exporter.
func (e *Exporter) Builder() *Builder {
	b := builderPool.Get().(*Builder)
	b.exporter = e
	b.Event.Labels = b.labels[:0]
	return b
}

// To initializes a builder from the values stored in a context.
func To(ctx context.Context) *Builder {
	if atomic.LoadInt32(&activeExporters) == 0 {
		return nil
	}
	v, ok := ctx.Value(contextKey{}).(contextValue)
	if !ok || v.exporter == nil {
		return nil
	}
	b := v.exporter.Builder()
	b.Event.Parent = v.parent
	return b
}

// Start delivers a start event with the given name and labels.
// Its second return value is a function that should be called to deliver the
// matching end event.
// All events created from the returned context will have this start event
// as their parent.
func Start(ctx context.Context, name string, labels ...Label) (_ context.Context, end func()) {
	b := To(ctx)
	if b == nil || b.exporter == nil {
		return ctx, func() {}
	}
	v := contextValue{exporter: b.exporter}
	v.parent = b.WithAll(labels...).Deliver(StartKind, name)
	return context.WithValue(ctx, contextKey{}, v), func() {
		eb := v.exporter.Builder()
		eb.Event.Parent = v.parent
		eb.Deliver(EndKind, "")
	}
}

// Deliver events to the underlying handler.
// The event will be assigned a new ID before being delivered, and the new ID
// will be returned.
// If the event does not have a timestamp, and the exporter has a Now function
// then the timestamp will be updated.
func (e *Exporter) Deliver(ev *Event) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastEvent++
	id := e.lastEvent
	ev.ID = id
	if e.Now != nil && ev.At.IsZero() {
		ev.At = e.Now()
	}
	e.handler.Handle(ev)
	return id
}