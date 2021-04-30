// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event

import (
	"context"
	"fmt"
	"sync"
)

// Builder is a fluent builder for construction of new events.
type Builder struct {
	data *builder
}

// SpanBuilder is a specialized Builder for construction of new span events.
type SpanBuilder struct {
	ctx  context.Context
	data *spanBuilder
}

// preallocateLabels controls the space reserved for labels in a builder.
// Storing the first few labels directly in builders can avoid an allocation at
// all for the very common cases of simple events. The length needs to be large
// enough to cope with the majority of events but no so large as to cause undue
// stack pressure.
const preallocateLabels = 4

type builder struct {
	exporter *Exporter
	ctx      context.Context
	Event    Event
	labels   [preallocateLabels]Label
}

var builderPool = sync.Pool{New: func() interface{} { return &builder{} }}

type spanBuilder struct {
	exporter *Exporter
	Event    Event
	labels   [preallocateLabels]Label
}

var spanBuilderPool = sync.Pool{New: func() interface{} { return &spanBuilder{} }}

// To initializes a builder from the values stored in a context.
func To(ctx context.Context) Builder {
	return Builder{data: newBuilder(ctx)}
}

func newBuilder(ctx context.Context) *builder {
	exporter, parent := fromContext(ctx)
	if exporter == nil {
		return nil
	}
	b := builderPool.Get().(*builder)
	b.exporter = exporter
	b.ctx = ctx
	b.Event.Labels = b.labels[:0]
	b.Event.Parent = parent
	return b
}

// Span initializes a span builder from the values stored in a context.
func Span(ctx context.Context) SpanBuilder {
	b := SpanBuilder{ctx: ctx}
	exporter, parent := fromContext(ctx)
	if exporter == nil {
		return b
	}
	b.data = spanBuilderPool.Get().(*spanBuilder)
	b.data.exporter = exporter
	b.data.Event.Labels = b.data.labels[:0]
	b.data.Event.Parent = parent
	return b
}

// Clone returns a copy of this builder.
// The two copies can be independently delivered.
func (b Builder) Clone() Builder {
	if b.data == nil {
		return b
	}
	clone := Builder{data: builderPool.Get().(*builder)}
	*clone.data = *b.data
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
		b.data.Event.Labels = append(b.data.Event.Labels, label)
	}
	return b
}

// WithAll adds all the supplied labels to the event being constructed.
func (b Builder) WithAll(labels ...Label) Builder {
	if b.data != nil || len(labels) == 0 {
		return b
	}
	if len(b.data.Event.Labels) == 0 {
		b.data.Event.Labels = labels
	} else {
		b.data.Event.Labels = append(b.data.Event.Labels, labels...)
	}
	return b
}

// Log is a helper that calls Deliver with LogKind.
func (b Builder) Log(message string) {
	if b.data == nil {
		return
	}
	b.data.exporter.mu.Lock()
	defer b.data.exporter.mu.Unlock()
	b.data.Event.Message = message
	b.data.exporter.prepare(&b.data.Event)
	b.data.exporter.handler.Log(b.data.ctx, &b.data.Event)
	b.done()
}

// Logf is a helper that uses fmt.Sprint to build the message and then
// calls Deliver with LogKind.
func (b Builder) Logf(template string, args ...interface{}) {
	if b.data == nil {
		return
	}
	b.data.exporter.mu.Lock()
	defer b.data.exporter.mu.Unlock()
	b.data.Event.Message = fmt.Sprintf(template, args...)
	b.data.exporter.prepare(&b.data.Event)
	b.data.exporter.handler.Log(b.data.ctx, &b.data.Event)
	b.done()
}

// Metric is a helper that calls Deliver with MetricKind.
func (b Builder) Metric() {
	if b.data == nil {
		return
	}
	b.data.exporter.mu.Lock()
	defer b.data.exporter.mu.Unlock()
	b.data.exporter.prepare(&b.data.Event)
	b.data.exporter.handler.Metric(b.data.ctx, &b.data.Event)
	b.done()
}

// Annotate is a helper that calls Deliver with AnnotateKind.
func (b Builder) Annotate() {
	if b.data == nil {
		return
	}
	b.data.exporter.mu.Lock()
	defer b.data.exporter.mu.Unlock()
	b.data.exporter.prepare(&b.data.Event)
	b.data.exporter.handler.Annotate(b.data.ctx, &b.data.Event)
	b.done()
}

// End is a helper that calls Deliver with EndKind.
func (b Builder) End() {
	if b.data == nil {
		return
	}
	b.data.exporter.mu.Lock()
	defer b.data.exporter.mu.Unlock()
	b.data.exporter.prepare(&b.data.Event)
	b.data.exporter.handler.End(b.data.ctx, &b.data.Event)
	b.done()
}

// Event returns a copy of the event currently being built.
func (b Builder) Event() *Event {
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

// WithAll adds all the supplied labels to the event being constructed.
func (b SpanBuilder) WithAll(labels ...Label) SpanBuilder {
	if b.data != nil || len(labels) == 0 {
		return b
	}
	if len(b.data.Event.Labels) == 0 {
		b.data.Event.Labels = labels
	} else {
		b.data.Event.Labels = append(b.data.Event.Labels, labels...)
	}
	return b
}

// Start delivers a start event with the given name and labels.
// Its second return value is a function that should be called to deliver the
// matching end event.
// All events created from the returned context will have this start event
// as their parent.
func (b SpanBuilder) Start(name string) (context.Context, func()) {
	if b.data == nil {
		return b.ctx, func() {}
	}
	b.data.exporter.mu.Lock()
	defer b.data.exporter.mu.Unlock()
	b.data.exporter.prepare(&b.data.Event)
	exporter, parent := b.data.exporter, b.data.Event.ID
	b.data.Event.Message = name
	ctx := newContext(b.ctx, exporter, parent)
	ctx = b.data.exporter.handler.Start(ctx, &b.data.Event)
	b.done()
	return ctx, func() {
		b := Builder{}
		b.data = builderPool.Get().(*builder)
		b.data.exporter = exporter
		b.data.Event.Parent = parent
		b.End()
	}
}

func (b SpanBuilder) done() {
	*b.data = spanBuilder{}
	spanBuilderPool.Put(b.data)
}
