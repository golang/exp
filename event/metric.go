// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
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
	Description string
	// TODO: deal with units. Follow otel, or define Go types for common units.
	// We don't need a time unit because we'll use time.Duration, and the only
	// other unit otel currently defines (besides dimensionless) is bytes.
}

// NewMetricDescriptor creates a MetricDescriptor with the given name.
// The namespace defaults to the import path of the caller of NewMetricDescriptor.
// Use SetNamespace to provide a different one.
// Neither the name nor the namespace can be empty.
func NewMetricDescriptor(name string) *MetricDescriptor {
	return newMetricDescriptor(name)
}

func newMetricDescriptor(name string) *MetricDescriptor {
	if name == "" {
		panic("name cannot be empty")
	}
	return &MetricDescriptor{
		name: name,
		// Set namespace to the caller's import path.
		// Depth:
		//   0  runtime.Callers
		//   1  importPath
		//   2  this function
		//   3  caller of this function (one of the NewXXX methods in this package)
		//   4  caller's caller
		namespace: importPath(4, nil),
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

func (m *MetricDescriptor) Name() string      { return m.name }
func (m *MetricDescriptor) Namespace() string { return m.namespace }

// A MetricValue is a pair of a Metric and a Value.
type MetricValue struct {
	m Metric
	v Value
}

// A Counter is a metric that counts something cumulatively.
type Counter struct {
	*MetricDescriptor
}

// NewCounter creates a counter with the given name.
func NewCounter(name string) *Counter {
	return &Counter{newMetricDescriptor(name)}
}

// Descriptor returns the receiver's MetricDescriptor.
func (c *Counter) Descriptor() *MetricDescriptor {
	return c.MetricDescriptor
}

// Record converts its argument into a Value and returns a MetricValue with the
// receiver and the value. It is intended to be used as an argument to
// Builder.Metric.
func (c *Counter) Record(v uint64) MetricValue {
	return MetricValue{c, Uint64Of(v)}
}

// A FloatGauge records a single floating-point value that may go up or down.
// TODO(generics): Gauge[T]
type FloatGauge struct {
	*MetricDescriptor
}

// NewFloatGauge creates a new FloatGauge with the given name.
func NewFloatGauge(name string) *FloatGauge {
	return &FloatGauge{newMetricDescriptor(name)}
}

// Descriptor returns the receiver's MetricDescriptor.
func (g *FloatGauge) Descriptor() *MetricDescriptor {
	return g.MetricDescriptor
}

// Record converts its argument into a Value and returns a MetricValue with the
// receiver and the value. It is intended to be used as an argument to
// Builder.Metric.
func (g *FloatGauge) Record(v float64) MetricValue {
	return MetricValue{g, Float64Of(v)}
}

// A Duration records a distribution of durations.
// TODO(generics): Distribution[T]
type Duration struct {
	*MetricDescriptor
}

// NewDuration creates a new Duration with the given name.
func NewDuration(name string) *Duration {
	return &Duration{newMetricDescriptor(name)}
}

// Descriptor returns the receiver's MetricDescriptor.
func (d *Duration) Descriptor() *MetricDescriptor {
	return d.MetricDescriptor
}

// Record converts its argument into a Value and returns a MetricValue with the
// receiver and the value. It is intended to be used as an argument to
// Builder.Metric.
func (d *Duration) Record(v time.Duration) MetricValue {
	return MetricValue{d, DurationOf(v)}
}
