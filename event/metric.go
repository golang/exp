// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import "fmt"

// MetricKind represents the kind of a Metric.
type MetricKind int

const (
	// A Counter is a metric that always increases, usually by 1.
	Counter MetricKind = iota
	// A Gauge is a metric that may go up or down.
	Gauge
	// A Distribution is a metric for which a summary of values is tracked.
	Distribution
)

func (k MetricKind) String() string {
	switch k {
	case Counter:
		return "Counter"
	case Gauge:
		return "Gauge"
	case Distribution:
		return "Distribution"
	default:
		return "!unknownMetricKind"
	}
}

type Metric struct {
	kind        MetricKind
	namespace   string
	name        string
	description string
	// For unit, follow otel, or define Go types for common units? We don't need
	// a time unit because we'll use time.Duration, and the only other unit otel
	// currently defines (besides dimensionless) is bytes.
}

func NewMetric(kind MetricKind, name, description string) *Metric {
	if name == "" {
		panic("name cannot be empty")
	}
	m := &Metric{
		kind:        kind,
		name:        name,
		description: description,
	}
	// Set namespace to the caller's import path.
	// Depth:
	//   0  runtime.Callers
	//   1  importPath
	//   2  this function
	//   3  caller of this function
	m.namespace = importPath(3, nil)
	return m
}

func (m *Metric) String() string {
	return fmt.Sprintf("%s(\"%s/%s\")", m.kind, m.namespace, m.name)
}

func (m *Metric) WithNamespace(ns string) *Metric {
	if ns == "" {
		panic("namespace cannot be empty")
	}
	m.namespace = ns
	return m
}

func (m *Metric) Kind() MetricKind    { return m.kind }
func (m *Metric) Name() string        { return m.name }
func (m *Metric) Namespace() string   { return m.namespace }
func (m *Metric) Description() string { return m.description }
