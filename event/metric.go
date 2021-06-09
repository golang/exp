// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
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

func (m *MetricDescriptor) Name() string      { return m.name }
func (m *MetricDescriptor) Namespace() string { return m.namespace }

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

func (c *Counter) To(ctx context.Context) CounterBuilder {
	b := CounterBuilder{builderCommon: builderCommon{ctx: ctx}, c: c}
	b.data = newBuilder(ctx)
	if b.data != nil {
		b.builderID = b.data.id
	}
	return b
}

type CounterBuilder struct {
	builderCommon
	c *Counter
}

func (b CounterBuilder) With(label Label) CounterBuilder {
	b.addLabel(label)
	return b
}

func (b CounterBuilder) WithAll(labels ...Label) CounterBuilder {
	b.addLabels(labels)
	return b
}

func (b CounterBuilder) Record(v uint64) {
	record(b.builderCommon, b.c, Uint64Of(v))
}

func record(b builderCommon, m Metric, v Value) {
	if b.data == nil {
		return
	}
	checkValid(b.data, b.builderID)
	if b.data.exporter.metricsEnabled() {
		b.data.exporter.mu.Lock()
		defer b.data.exporter.mu.Unlock()
		b.data.Event.Labels = append(b.data.Event.Labels, MetricValue.Of(v), MetricKey.Of(ValueOf(m)))
		b.data.exporter.prepare(&b.data.Event)
		b.data.exporter.handler.Metric(b.ctx, &b.data.Event)
	}
	b.done()
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

func (g *FloatGauge) To(ctx context.Context) FloatGaugeBuilder {
	b := FloatGaugeBuilder{builderCommon: builderCommon{ctx: ctx}, g: g}
	b.data = newBuilder(ctx)
	if b.data != nil {
		b.builderID = b.data.id
	}
	return b
}

type FloatGaugeBuilder struct {
	builderCommon
	g *FloatGauge
}

func (b FloatGaugeBuilder) With(label Label) FloatGaugeBuilder {
	b.addLabel(label)
	return b
}

func (b FloatGaugeBuilder) WithAll(labels ...Label) FloatGaugeBuilder {
	b.addLabels(labels)
	return b
}

func (b FloatGaugeBuilder) Record(v float64) {
	record(b.builderCommon, b.g, Float64Of(v))
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

func (d *Duration) To(ctx context.Context) DurationBuilder {
	b := DurationBuilder{builderCommon: builderCommon{ctx: ctx}, d: d}
	b.data = newBuilder(ctx)
	if b.data != nil {
		b.builderID = b.data.id
	}
	return b
}

type DurationBuilder struct {
	builderCommon
	d *Duration
}

func (b DurationBuilder) With(label Label) DurationBuilder {
	b.addLabel(label)
	return b
}

func (b DurationBuilder) WithAll(labels ...Label) DurationBuilder {
	b.addLabels(labels)
	return b
}

func (b DurationBuilder) Record(v time.Duration) {
	record(b.builderCommon, b.d, DurationOf(v))
}
