// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"sync"
	"time"
)

// Event holds the information about an event that occurred.
// It combines the event metadata with the user supplied labels.
type Event struct {
	ID     uint64
	Parent uint64    // id of the parent event for this event
	Source Source    // source of event; if empty, set by exporter to import path
	At     time.Time // time at which the event is delivered to the exporter.
	Kind   Kind
	Labels []Label

	ctx    context.Context
	target *target
	labels [preallocateLabels]Label
}

// Handler is a the type for something that handles events as they occur.
type Handler interface {
	// Event is called with each event.
	Event(context.Context, *Event) context.Context
}

// preallocateLabels controls the space reserved for labels in a builder.
// Storing the first few labels directly in builders can avoid an allocation at
// all for the very common cases of simple events. The length needs to be large
// enough to cope with the majority of events but no so large as to cause undue
// stack pressure.
const preallocateLabels = 6

var eventPool = sync.Pool{New: func() interface{} { return &Event{} }}

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

// New prepares a new event.
// This is intended to avoid allocations in the steady state case, to do this
// it uses a pool of events.
// Events are returned to the pool when Deliver is called. Failure to call
// Deliver will exhaust the pool and cause allocations.
// It returns nil if there is no active exporter for this kind of event.
func New(ctx context.Context, kind Kind) *Event {
	var t *target
	if v, ok := ctx.Value(contextKey).(*target); ok {
		t = v
	} else {
		t = getDefaultTarget()
	}
	if t == nil {
		return nil
	}
	//TODO: we can change this to a much faster test
	switch kind {
	case LogKind:
		if !t.exporter.loggingEnabled() {
			return nil
		}
	case MetricKind:
		if !t.exporter.metricsEnabled() {
			return nil
		}
	case StartKind, EndKind:
		if !t.exporter.tracingEnabled() {
			return nil
		}
	}
	ev := eventPool.Get().(*Event)
	*ev = Event{
		ctx:    ctx,
		target: t,
		Kind:   kind,
		Parent: t.parent,
	}
	ev.Labels = ev.labels[:0]
	return ev
}

// Clone makes a deep copy of the Event.
// Deliver can be called on both Events independently.
func (ev *Event) Clone() *Event {
	ev2 := eventPool.Get().(*Event)
	*ev2 = *ev
	ev2.Labels = append(ev2.labels[:0], ev.Labels...)
	return ev2
}

func (ev *Event) Trace() {
	ev.prepare()
	ev.ctx = newContext(ev.ctx, ev.target.exporter, ev.ID, ev.At)
}

// Deliver the event to the exporter that was found in New.
// This also returns the event to the pool, it is an error to do anything
// with the event after it is delivered.
func (ev *Event) Deliver() context.Context {
	// get the event ready to send
	ev.prepare()
	ctx := ev.deliver()
	eventPool.Put(ev)
	return ctx
}

func (ev *Event) deliver() context.Context {
	// hold the lock while we deliver the event
	e := ev.target.exporter
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.handler.Event(ev.ctx, ev)
}

func (ev *Event) Find(name string) Label {
	for _, l := range ev.Labels {
		if l.Name == name {
			return l
		}
	}
	return Label{}
}
