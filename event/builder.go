// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"sync"
)

// Builder is a fluent builder for construction of new events.
//
// Storing the first few labels directly can avoid an allocation at all for the
// very common cases of simple events. The length needs to be large enough to
// cope with the majority of events but no so large as to cause undue stack
// pressure.
type Builder struct {
	exporter *Exporter
	Event    Event
	labels   [4]Label
}

var builderPool = sync.Pool{New: func() interface{} { return &Builder{} }}

// Clone returns a copy of this builder.
// The two copies can be independently delivered.
func (b *Builder) Clone() *Builder {
	if b == nil {
		return nil
	}
	clone := builderPool.Get().(*Builder)
	clone.exporter = b.exporter
	clone.Event = b.Event
	n := len(b.Event.Labels)
	if n <= len(b.labels) {
		clone.Event.Labels = clone.labels[:n]
	} else {
		clone.Event.Labels = make([]Label, n)
	}
	copy(clone.Event.Labels, b.Event.Labels)
	return clone
}

// With adds a new label to the event being constructed.
func (b *Builder) With(label Label) *Builder {
	if b == nil {
		return nil
	}
	b.Event.Labels = append(b.Event.Labels, label)
	return b
}

// WithAll adds all the supplied labels to the event being constructed.
func (b *Builder) WithAll(labels ...Label) *Builder {
	if b == nil || len(labels) == 0 {
		return b
	}
	// TODO: this can cause the aliasing check based on length to fail,
	// so find another way to check.
	if len(b.Event.Labels) == 0 {
		b.Event.Labels = labels
		return b
	}
	b.Event.Labels = append(b.Event.Labels, labels...)
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
