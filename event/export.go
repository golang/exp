// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
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
type contextKeyType struct{}

var contextKey interface{} = contextKeyType{}

// contextValue is stored by value in the context to track the exporter and
// current parent event.
type contextValue struct {
	exporter *Exporter
	parent   uint64
}

type defaultHandler struct{}

var (
	enabled         int32 = 1
	defaultExporter       = unsafe.Pointer(&Exporter{
		Now:     time.Now,
		handler: defaultHandler{},
	})
)

// NewExporter creates an Exporter using the supplied handler.
// Event delivery is serialized to enable safe atomic handling.
// It also marks the event system as active.
func NewExporter(h Handler) *Exporter {
	return &Exporter{
		Now:     time.Now,
		handler: h,
	}
}

// SetEnabled can be used to enable or disable the entire event system.
func SetEnabled(value bool) {
	if value {
		atomic.StoreInt32(&enabled, 1)
	} else {
		atomic.StoreInt32(&enabled, 0)
	}
}

func isDisabled() bool {
	return atomic.LoadInt32(&enabled) == 0
}

func setDefaultExporter(e *Exporter) {
	atomic.StorePointer(&defaultExporter, unsafe.Pointer(e))
}

func getDefaultExporter() *Exporter {
	return (*Exporter)(atomic.LoadPointer(&defaultExporter))
}

func newContext(ctx context.Context, exporter *Exporter, parent uint64) context.Context {
	return context.WithValue(ctx, contextKey, contextValue{exporter: exporter, parent: parent})
}

func fromContext(ctx context.Context) (*Exporter, uint64) {
	if v, ok := ctx.Value(contextKey).(contextValue); ok {
		return v.exporter, v.parent
	}
	return getDefaultExporter(), 0
}

// WithExporter returns a context with the exporter attached.
// The exporter is called synchronously from the event call site, so it should
// return quickly so as not to hold up user code.
func WithExporter(ctx context.Context, e *Exporter) context.Context {
	return newContext(ctx, e, 0)
}

// Builder returns a new builder for the exporter.
func (e *Exporter) Builder() *Builder {
	if e == nil {
		return nil
	}
	b := builderPool.Get().(*Builder)
	b.exporter = e
	b.Event.Labels = b.labels[:0]
	return b
}

// To initializes a builder from the values stored in a context.
func To(ctx context.Context) *Builder {
	if isDisabled() {
		return nil
	}
	exporter, parent := fromContext(ctx)
	b := exporter.Builder()
	if b != nil {
		b.Event.Parent = parent
	}
	return b
}

// Start delivers a start event with the given name and labels.
// Its second return value is a function that should be called to deliver the
// matching end event.
// All events created from the returned context will have this start event
// as their parent.
func Start(ctx context.Context, name string, labels ...Label) (_ context.Context, end func()) {
	if isDisabled() {
		return nil, func() {}
	}
	exporter, parent := fromContext(ctx)
	b := exporter.Builder()
	if b == nil {
		return ctx, func() {}
	}
	b.Event.Parent = parent
	span := b.WithAll(labels...).Deliver(StartKind, name)
	ctx = newContext(ctx, exporter, span)
	return ctx, func() {
		eb := exporter.Builder()
		eb.Event.Parent = span
		eb.Deliver(EndKind, "")
	}
}

// Deliver events to the underlying handler.
// The event will be assigned a new ID before being delivered, and the new ID
// will be returned.
// If the event does not have a timestamp, and the exporter has a Now function
// then the timestamp will be updated.
func (e *Exporter) Deliver(ev *Event) uint64 {
	if e == nil {
		return 0
	}
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

func (defaultHandler) Handle(ev *Event) {
	if ev.Kind != LogKind {
		return
	}
	//TODO: split between stdout and stderr?
	fmt.Fprintln(os.Stdout, ev)
}
