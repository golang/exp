// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"time"
)

type Metric interface {
	Descriptor() MetricDescriptor
}

type MetricDescriptor struct {
	namespace   string
	name        string
	Description string
	// For unit, follow otel, or define Go types for common units? We don't need
	// a time unit because we'll use time.Duration, and the only other unit otel
	// currently defines (besides dimensionless) is bytes.
}

// TODO: how to force a non-empty namespace?

func NewMetricDescriptor(name string) MetricDescriptor {
	if name == "" {
		panic("name cannot be empty")
	}
	m := MetricDescriptor{name: name}
	// TODO: make this work right whether called from in this package or externally.
	// Set namespace to the caller's import path.
	// Depth:
	//   0  runtime.Callers
	//   1  importPath
	//   2  this function
	//   3  caller of this function (one of the NewXXX methods in this package)
	//   4  caller's caller
	m.namespace = importPath(4, nil)
	return m
}

func (m *MetricDescriptor) String() string {
	return fmt.Sprintf("Metric(\"%s/%s\")", m.namespace, m.name)
}

func (m *MetricDescriptor) WithNamespace(ns string) *MetricDescriptor {
	if ns == "" {
		panic("namespace cannot be empty")
	}
	m.namespace = ns
	return m
}

func (m MetricDescriptor) Name() string      { return m.name }
func (m MetricDescriptor) Namespace() string { return m.namespace }

// A counter is a metric that counts something cumulatively.
type Counter struct {
	MetricDescriptor
}

func NewCounter(name string) *Counter {
	return &Counter{NewMetricDescriptor(name)}
}

func (c *Counter) Descriptor() MetricDescriptor {
	return c.MetricDescriptor
}

type MetricValue struct {
	m Metric
	v Value
}

func (c *Counter) Record(v uint64) MetricValue {
	return MetricValue{c, Uint64Of(v)}
}

// A FloatGauge records a single floating-point value that may go up or down.
// TODO(generics): Gauge[T]
type FloatGauge struct {
	MetricDescriptor
}

func NewFloatGauge(name string) *FloatGauge {
	return &FloatGauge{NewMetricDescriptor(name)}
}

func (g *FloatGauge) Descriptor() MetricDescriptor {
	return g.MetricDescriptor
}

func (g *FloatGauge) Record(v float64) MetricValue {
	return MetricValue{g, Float64Of(v)}
}

// A Duration records a distribution of durations.
// TODO(generics): Distribution[T]
type Duration struct {
	MetricDescriptor
}

func NewDuration(name string) *Duration {
	return &Duration{NewMetricDescriptor(name)}
}

func (d *Duration) Descriptor() MetricDescriptor {
	return d.MetricDescriptor
}

func (d *Duration) Record(v time.Duration) MetricValue {
	return MetricValue{d, DurationOf(v)}
}
