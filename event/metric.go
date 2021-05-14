// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"time"
)

// A Unit is a unit of measurement for a metric.
type Unit string

const (
	UnitDimensionless Unit = "1"
	UnitBytes         Unit = "By"
	UnitMilliseconds  Unit = "ms"
)

// A Metric represents a kind of recorded measurement.
type Metric interface {
	Name() string
	Options() MetricOptions
}

type MetricOptions struct {
	// A string that should be common for all metrics of an application or
	// service. Defaults to the import path of the package calling
	// the metric construction function (NewCounter, etc.).
	Namespace string

	// Optional description of the metric.
	Description string

	// Optional unit for the metric. Defaults to UnitDimensionless.
	Unit Unit
}

// A Counter is a metric that counts something cumulatively.
type Counter struct {
	name string
	opts MetricOptions
}

func initOpts(popts *MetricOptions) MetricOptions {
	var opts MetricOptions
	if popts != nil {
		opts = *popts
	}
	if opts.Namespace == "" {
		opts.Namespace = scanStack().Space
	}
	if opts.Unit == "" {
		opts.Unit = UnitDimensionless
	}
	return opts
}

// NewCounter creates a counter with the given name.
func NewCounter(name string, opts *MetricOptions) *Counter {
	return &Counter{name, initOpts(opts)}
}

func (c *Counter) Name() string           { return c.name }
func (c *Counter) Options() MetricOptions { return c.opts }

// Record delivers a metric event with the given metric, value and labels to the
// exporter in the context.
func (c *Counter) Record(ctx context.Context, v int64, labels ...Label) {
	ev := New(ctx, MetricKind)
	if ev != nil {
		record(ev, c, Int64(string(MetricVal), v))
		ev.Labels = append(ev.Labels, labels...)
		ev.Deliver()
	}
}

// A FloatGauge records a single floating-point value that may go up or down.
// TODO(generics): Gauge[T]
type FloatGauge struct {
	name string
	opts MetricOptions
}

// NewFloatGauge creates a new FloatGauge with the given name.
func NewFloatGauge(name string, opts *MetricOptions) *FloatGauge {
	return &FloatGauge{name, initOpts(opts)}
}

func (g *FloatGauge) Name() string           { return g.name }
func (g *FloatGauge) Options() MetricOptions { return g.opts }

// Record converts its argument into a Value and returns a MetricValue with the
// receiver and the value.
func (g *FloatGauge) Record(ctx context.Context, v float64, labels ...Label) {
	ev := New(ctx, MetricKind)
	if ev != nil {
		record(ev, g, Float64(string(MetricVal), v))
		ev.Labels = append(ev.Labels, labels...)
		ev.Deliver()
	}
}

// A DurationDistribution records a distribution of durations.
// TODO(generics): Distribution[T]
type DurationDistribution struct {
	name string
	opts MetricOptions
}

// NewDuration creates a new Duration with the given name.
func NewDuration(name string, opts *MetricOptions) *DurationDistribution {
	return &DurationDistribution{name, initOpts(opts)}
}

func (d *DurationDistribution) Name() string           { return d.name }
func (d *DurationDistribution) Options() MetricOptions { return d.opts }

// Record converts its argument into a Value and returns a MetricValue with the
// receiver and the value.
func (d *DurationDistribution) Record(ctx context.Context, v time.Duration, labels ...Label) {
	ev := New(ctx, MetricKind)
	if ev != nil {
		record(ev, d, Duration(string(MetricVal), v))
		ev.Labels = append(ev.Labels, labels...)
		ev.Deliver()
	}
}

// An IntDistribution records a distribution of int64s.
type IntDistribution struct {
	name string
	opts MetricOptions
}

func (d *IntDistribution) Name() string           { return d.name }
func (d *IntDistribution) Options() MetricOptions { return d.opts }

// NewIntDistribution creates a new IntDistribution with the given name.
func NewIntDistribution(name string, opts *MetricOptions) *IntDistribution {
	return &IntDistribution{name, initOpts(opts)}
}

// Record converts its argument into a Value and returns a MetricValue with the
// receiver and the value.
func (d *IntDistribution) Record(ctx context.Context, v int64, labels ...Label) {
	ev := New(ctx, MetricKind)
	if ev != nil {
		record(ev, d, Int64(string(MetricVal), v))
		ev.Labels = append(ev.Labels, labels...)
		ev.Deliver()
	}
}

func record(ev *Event, m Metric, l Label) {
	ev.Labels = append(ev.Labels, l, MetricKey.Of(m))
}
