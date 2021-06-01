// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Builder is a fluent builder for construction of new events.
type Builder struct {
	ctx       context.Context
	data      *builder
	builderID uint64 // equals data.id if all is well
}

// preallocateLabels controls the space reserved for labels in a builder.
// Storing the first few labels directly in builders can avoid an allocation at
// all for the very common cases of simple events. The length needs to be large
// enough to cope with the majority of events but no so large as to cause undue
// stack pressure.
const preallocateLabels = 6

type builder struct {
	exporter *Exporter
	Event    Event
	labels   [preallocateLabels]Label
	id       uint64
}

var builderPool = sync.Pool{New: func() interface{} { return &builder{} }}

// To initializes a builder from the values stored in a context.
func To(ctx context.Context) Builder {
	b := Builder{ctx: ctx}
	exporter, parent := FromContext(ctx)
	if exporter == nil {
		return b
	}
	b.data = allocBuilder()
	b.builderID = b.data.id
	b.data.exporter = exporter
	b.data.Event.Labels = b.data.labels[:0]
	b.data.Event.Parent = parent
	return b
}

var builderID uint64 // atomic

func allocBuilder() *builder {
	b := builderPool.Get().(*builder)
	b.id = atomic.AddUint64(&builderID, 1)
	return b
}

// Clone returns a copy of this builder.
// The two copies can be independently delivered.
func (b Builder) Clone() Builder {
	if b.data == nil {
		return b
	}
	bb := allocBuilder()
	bbid := bb.id
	clone := Builder{ctx: b.ctx, data: bb, builderID: bb.id}
	*clone.data = *b.data
	clone.data.id = bbid
	if len(b.data.Event.Labels) == 0 || &b.data.labels[0] == &b.data.Event.Labels[0] {
		clone.data.Event.Labels = clone.data.labels[:len(b.data.Event.Labels)]
	} else {
		clone.data.Event.Labels = make([]Label, len(b.data.Event.Labels))
		copy(clone.data.Event.Labels, b.data.Event.Labels)
	}
	return clone
}

// With adds a new label to the event being constructed.
func (b Builder) With(label Label) Builder {
	if b.data != nil {
		checkValid(b.data, b.builderID)
		b.data.Event.Labels = append(b.data.Event.Labels, label)
	}
	return b
}

// WithAll adds all the supplied labels to the event being constructed.
func (b Builder) WithAll(labels ...Label) Builder {
	if b.data == nil || len(labels) == 0 {
		return b
	}
	checkValid(b.data, b.builderID)
	if len(b.data.Event.Labels) == 0 {
		b.data.Event.Labels = labels
	} else {
		b.data.Event.Labels = append(b.data.Event.Labels, labels...)
	}
	return b
}

func (b Builder) At(t time.Time) Builder {
	if b.data != nil {
		checkValid(b.data, b.builderID)
		b.data.Event.At = t
	}
	return b
}

func (b Builder) Namespace(ns string) Builder {
	if b.data != nil {
		checkValid(b.data, b.builderID)
		b.data.Event.Namespace = ns
	}
	return b
}

// Log is a helper that calls Deliver with LogKind.
func (b Builder) Log(message string) {
	if b.data == nil {
		return
	}
	checkValid(b.data, b.builderID)
	if b.data.exporter.loggingEnabled() {
		b.data.exporter.mu.Lock()
		defer b.data.exporter.mu.Unlock()
		b.data.Event.Labels = append(b.data.Event.Labels, Message.Of(message))
		b.data.exporter.prepare(&b.data.Event)
		b.data.exporter.handler.Log(b.ctx, &b.data.Event)
	}
	b.done()
}

// Logf is a helper that uses fmt.Sprint to build the message and then
// calls Deliver with LogKind.
func (b Builder) Logf(template string, args ...interface{}) {
	if b.data == nil {
		return
	}
	checkValid(b.data, b.builderID)
	if b.data.exporter.loggingEnabled() {
		message := fmt.Sprintf(template, args...)
		// Duplicate code from Log so Exporter.deliver's invocation of runtime.Callers is correct.
		b.data.exporter.mu.Lock()
		defer b.data.exporter.mu.Unlock()
		b.data.Event.Labels = append(b.data.Event.Labels, Message.Of(message))
		b.data.exporter.prepare(&b.data.Event)
		b.data.exporter.handler.Log(b.ctx, &b.data.Event)
	}
	b.done()
}

// Metric is a helper that calls Deliver with MetricKind.
func (b Builder) Metric() {
	if b.data == nil {
		return
	}
	checkValid(b.data, b.builderID)
	if b.data.exporter.metricsEnabled() {
		b.data.exporter.mu.Lock()
		defer b.data.exporter.mu.Unlock()
		b.data.Event.Labels = append(b.data.Event.Labels, Metric.Value())
		b.data.exporter.prepare(&b.data.Event)
		b.data.exporter.handler.Metric(b.ctx, &b.data.Event)
	}
	b.done()
}

// Annotate is a helper that calls Deliver with AnnotateKind.
func (b Builder) Annotate() {
	if b.data == nil {
		return
	}
	checkValid(b.data, b.builderID)
	if b.data.exporter.annotationsEnabled() {
		b.data.exporter.mu.Lock()
		defer b.data.exporter.mu.Unlock()
		b.data.exporter.prepare(&b.data.Event)
		b.data.exporter.handler.Annotate(b.ctx, &b.data.Event)
	}
	b.done()
}

// End is a helper that calls Deliver with EndKind.
func (b Builder) End() {
	if b.data == nil {
		return
	}
	checkValid(b.data, b.builderID)
	if b.data.exporter.tracingEnabled() {
		b.data.exporter.mu.Lock()
		defer b.data.exporter.mu.Unlock()
		b.data.Event.Labels = append(b.data.Event.Labels, End.Value())
		b.data.exporter.prepare(&b.data.Event)
		b.data.exporter.handler.End(b.ctx, &b.data.Event)
	}
	b.done()
}

// Event returns a copy of the event currently being built.
func (b Builder) Event() *Event {
	checkValid(b.data, b.builderID)
	clone := b.data.Event
	if len(b.data.Event.Labels) > 0 {
		clone.Labels = make([]Label, len(b.data.Event.Labels))
		copy(clone.Labels, b.data.Event.Labels)
	}
	return &clone
}

func (b Builder) done() {
	*b.data = builder{}
	builderPool.Put(b.data)
}

// Start delivers a start event with the given name and labels.
// Its second return value is a function that should be called to deliver the
// matching end event.
// All events created from the returned context will have this start event
// as their parent.
func (b Builder) Start(name string) (context.Context, func()) {
	if b.data == nil {
		return b.ctx, func() {}
	}
	checkValid(b.data, b.builderID)
	ctx := b.ctx
	end := func() {}
	if b.data.exporter.tracingEnabled() {
		b.data.exporter.mu.Lock()
		defer b.data.exporter.mu.Unlock()
		b.data.exporter.lastEvent++
		traceID := b.data.exporter.lastEvent
		b.data.Event.Labels = append(b.data.Event.Labels, Trace.Of(traceID))
		b.data.exporter.prepare(&b.data.Event)
		// create the end builder
		eb := Builder{}
		eb.data = allocBuilder()
		eb.builderID = eb.data.id
		eb.data.exporter = b.data.exporter
		eb.data.Event.Parent = traceID
		// and now deliver the start event
		b.data.Event.Labels = append(b.data.Event.Labels, Name.Of(name))
		ctx = newContext(ctx, b.data.exporter, traceID)
		ctx = b.data.exporter.handler.Start(ctx, &b.data.Event)
		eb.ctx = ctx
		end = eb.End
	}
	b.done()
	return ctx, end
}

func checkValid(b *builder, wantID uint64) {
	if b.exporter == nil || b.id != wantID {
		panic("Builder already delivered an event; missing call to Clone")
	}
}
