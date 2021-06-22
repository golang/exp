// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"fmt"
	"time"
)

// A Metric represents a kind of recorded measurement.
type Metric interface {
	Descriptor() *MetricDescriptor
}

// A MetricDescriptor describes a metric.
type MetricDescriptor struct {
	namespace   string
	name        string
	description string
	// TODO: deal with units. Follow otel, or define Go types for common units.
	// We don't need a time unit because we'll use time.Duration, and the only
	// other unit otel currently defines (besides dimensionless) is bytes.
}

// NewMetricDescriptor creates a MetricDescriptor with the given name.
// The namespace defaults to the import path of the caller of NewMetricDescriptor.
// Use SetNamespace to provide a different one.
// Neither the name nor the namespace can be empty.
func NewMetricDescriptor(name, description string) *MetricDescriptor {
	return newMetricDescriptor(name, description)
}

func newMetricDescriptor(name, description string) *MetricDescriptor {
	if name == "" {
		panic("name cannot be empty")
	}
	return &MetricDescriptor{
		name:        name,
		namespace:   scanStack().Space,
		description: description,
	}
}

// SetNamespace sets the namespace of m to a non-empty string.
func (m *MetricDescriptor) SetNamespace(ns string) {
	if ns == "" {
		panic("namespace cannot be empty")
	}
	m.namespace = ns
}

func (m *MetricDescriptor) String() string {
	return fmt.Sprintf("Metric(\"%s/%s\")", m.namespace, m.name)
}

func (m *MetricDescriptor) Name() string        { return m.name }
func (m *MetricDescriptor) Namespace() string   { return m.namespace }
func (m *MetricDescriptor) Description() string { return m.description }

// A Counter is a metric that counts something cumulatively.
type Counter struct {
	*MetricDescriptor
}

// NewCounter creates a counter with the given name.
func NewCounter(name, description string) *Counter {
	return &Counter{newMetricDescriptor(name, description)}
}

// Descriptor returns the receiver's MetricDescriptor.
func (c *Counter) Descriptor() *MetricDescriptor {
	return c.MetricDescriptor
}

// Record delivers a metric event with the given metric, value and labels to the
// exporter in the context.
func (c *Counter) Record(ctx context.Context, v int64, labels ...Label) {
	ev := New(ctx, MetricKind)
	if ev != nil {
		record(ev, c, Int64(MetricVal, v))
		ev.Labels = append(ev.Labels, labels...)
		ev.Deliver()
	}
}

// A FloatGauge records a single floating-point value that may go up or down.
// TODO(generics): Gauge[T]
type FloatGauge struct {
	*MetricDescriptor
}

// NewFloatGauge creates a new FloatGauge with the given name.
func NewFloatGauge(name, description string) *FloatGauge {
	return &FloatGauge{newMetricDescriptor(name, description)}
}

// Descriptor returns the receiver's MetricDescriptor.
func (g *FloatGauge) Descriptor() *MetricDescriptor {
	return g.MetricDescriptor
}

// Record converts its argument into a Value and returns a MetricValue with the
// receiver and the value. It is intended to be used as an argument to
// Builder.Metric.
func (g *FloatGauge) Record(ctx context.Context, v float64, labels ...Label) {
	ev := New(ctx, MetricKind)
	if ev != nil {
		record(ev, g, Float64(MetricVal, v))
		ev.Labels = append(ev.Labels, labels...)
		ev.Deliver()
	}
}

// A DurationDistribution records a distribution of durations.
// TODO(generics): Distribution[T]
type DurationDistribution struct {
	*MetricDescriptor
}

// NewDuration creates a new Duration with the given name.
func NewDuration(name, description string) *DurationDistribution {
	return &DurationDistribution{newMetricDescriptor(name, description)}
}

// Descriptor returns the receiver's MetricDescriptor.
func (d *DurationDistribution) Descriptor() *MetricDescriptor {
	return d.MetricDescriptor
}

// Record converts its argument into a Value and returns a MetricValue with the
// receiver and the value. It is intended to be used as an argument to
// Builder.Metric.
func (d *DurationDistribution) Record(ctx context.Context, v time.Duration, labels ...Label) {
	ev := New(ctx, MetricKind)
	if ev != nil {
		record(ev, d, Duration(MetricVal, v))
		ev.Labels = append(ev.Labels, labels...)
		ev.Deliver()
	}
}

// An IntDistribution records a distribution of int64s.
type IntDistribution struct {
	*MetricDescriptor
}

// NewIntDistribution creates a new IntDistribution with the given name.
func NewIntDistribution(name, description string) *IntDistribution {
	return &IntDistribution{newMetricDescriptor(name, description)}
}

// Descriptor returns the receiver's MetricDescriptor.
func (d *IntDistribution) Descriptor() *MetricDescriptor {
	return d.MetricDescriptor
}

// Record converts its argument into a Value and returns a MetricValue with the
// receiver and the value. It is intended to be used as an argument to
// Builder.Metric.
func (d *IntDistribution) Record(ctx context.Context, v int64, labels ...Label) {
	ev := New(ctx, MetricKind)
	if ev != nil {
		record(ev, d, Int64(MetricVal, v))
		ev.Labels = append(ev.Labels, labels...)
		ev.Deliver()
	}
}

func record(ev *Event, m Metric, l Label) {
	ev.Labels = append(ev.Labels, l, Value(MetricKey, m))
}
