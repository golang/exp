// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"sync"
)

// Builder is a fluent builder for construction of new events.
type Builder struct {
	exporter *Exporter
	index    int
	Event    Event
}

var builderPool = sync.Pool{New: func() interface{} { return &Builder{} }}

// Clone returns a copy of this builder.
// The two copies can be independently delivered.
func (b *Builder) Clone() *Builder {
	if b == nil {
		return nil
	}
	clone := builderPool.Get().(*Builder)
	*clone = *b
	if len(b.Event.Dynamic) > 0 {
		clone.Event.Dynamic = make([]Label, len(b.Event.Dynamic))
		copy(clone.Event.Dynamic, b.Event.Dynamic)
	}
	return clone
}

// With adds a new label to the event being constructed.
func (b *Builder) With(label Label) *Builder {
	if b == nil {
		return nil
	}
	if b.index < len(b.Event.Static) {
		b.Event.Static[b.index] = label
		b.index++
		return b
	}
	b.Event.Dynamic = append(b.Event.Dynamic, label)
	b.index++
	return b
}

// WithAll adds all the supplied labels to the event being constructed.
func (b *Builder) WithAll(labels ...Label) *Builder {
	if b == nil || len(labels) == 0 {
		return b
	}
	if len(b.Event.Dynamic) == 0 {
		b.Event.Dynamic = labels
		return b
	}
	b.Event.Dynamic = append(b.Event.Dynamic, labels...)
	return b
}

// Deliver sends the constructed event to the exporter.
func (b *Builder) Deliver(kind Kind, message string) uint64 {
	if b == nil {
		return 0
	}
	b.Event.Kind = kind
	b.Event.Message = message
	id := b.exporter.Deliver(&b.Event)
	*b = Builder{}
	builderPool.Put(b)
	return id
}

// Log is a helper that calls Deliver with LogKind.
func (b *Builder) Log(message string) {
	b.Deliver(LogKind, message)
}

// Logf is a helper that uses fmt.Sprint to build the message and then
// calls Deliver with LogKind.
func (b *Builder) Logf(template string, args ...interface{}) {
	b.Deliver(LogKind, fmt.Sprintf(template, args...))
}

// Start is a helper that calls Deliver with StartKind.
func (b *Builder) Start(name string) uint64 {
	return b.Deliver(StartKind, name)
}

// End is a helper that calls Deliver with EndKind.
func (b *Builder) End() {
	b.Deliver(EndKind, "")
}

// Metric is a helper that calls Deliver with MetricKind.
func (b *Builder) Metric() {
	b.Deliver(MetricKind, "")
}

// Annotate is a helper that calls Deliver with AnnotateKind.
func (b *Builder) Annotate() {
	b.Deliver(AnnotateKind, "")
}
