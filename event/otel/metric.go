// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package otel

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/exp/event"
)

// MetricHandler is an event.Handler for OpenTelemetry metrics.
// Its Event method handles Metric events and ignores all others.
type MetricHandler struct {
	meter metric.Meter
	mu    sync.Mutex
	// A map from event.Metrics to, effectively, otel Meters.
	// But since the only thing we need from the Meter is recording a value, we
	// use a function for that that closes over the Meter itself.
	recordFuncs map[event.Metric]recordFunc
}

type recordFunc func(context.Context, event.Label, []attribute.KeyValue)

var _ event.Handler = (*MetricHandler)(nil)

// NewMetricHandler creates a new MetricHandler.
func NewMetricHandler(m metric.Meter) *MetricHandler {
	return &MetricHandler{
		meter:       m,
		recordFuncs: map[event.Metric]recordFunc{},
	}
}

func (m *MetricHandler) Event(ctx context.Context, e *event.Event) context.Context {
	if e.Kind != event.MetricKind {
		return ctx
	}
	// Get the otel instrument corresponding to the event's MetricDescriptor,
	// or create a new one.
	mi, ok := event.MetricKey.Find(e)
	if !ok {
		panic(errors.New("no metric key for metric event"))
	}
	em := mi.(event.Metric)
	lval := e.Find(event.MetricVal)
	if !lval.HasValue() {
		panic(errors.New("no metric value for metric event"))
	}
	rf := m.getRecordFunc(em)
	if rf == nil {
		panic(fmt.Errorf("unable to record for metric %v", em))
	}
	rf(ctx, lval, labelsToAttributes(e.Labels))
	return ctx
}

func (m *MetricHandler) getRecordFunc(em event.Metric) recordFunc {
	m.mu.Lock()
	defer m.mu.Unlock()
	if f, ok := m.recordFuncs[em]; ok {
		return f
	}
	f, err := m.newRecordFunc(em)
	if err != nil {
		panic(err)
	}
	m.recordFuncs[em] = f
	return f
}

func (m *MetricHandler) newRecordFunc(em event.Metric) (recordFunc, error) {
	opts := em.Options()
	name := opts.Namespace + "/" + em.Name()
	switch em.(type) {
	case *event.Counter:
		opts := []metric.Int64CounterOption{
			metric.WithDescription(opts.Description),
			metric.WithUnit(string(opts.Unit)),
		}
		c, err := m.meter.Int64Counter(name, opts...)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, l event.Label, attrs []attribute.KeyValue) {
			c.Add(ctx, l.Int64(), metric.WithAttributes(attrs...))
		}, nil

	case *event.FloatGauge:
		g, err := m.meter.Float64UpDownCounter(name,
			metric.WithDescription(opts.Description),
			metric.WithUnit(string(opts.Unit)))
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, l event.Label, attrs []attribute.KeyValue) {
			g.Add(ctx, l.Float64(), metric.WithAttributes(attrs...))
		}, nil

	case *event.DurationDistribution:
		r, err := m.meter.Int64Histogram(name,
			metric.WithDescription(opts.Description),
			metric.WithUnit(string(opts.Unit)))
		if err != nil {
			return nil, err
		}

		return func(ctx context.Context, l event.Label, attrs []attribute.KeyValue) {
			r.Record(ctx, l.Duration().Nanoseconds(), metric.WithAttributes(attrs...))
		}, nil

	default:
		return nil, nil
	}
}

func labelsToAttributes(ls []event.Label) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	for _, l := range ls {
		if l.Name == string(event.MetricKey) || l.Name == string(event.MetricVal) {
			continue
		}
		attrs = append(attrs, labelToAttribute(l))
	}
	return attrs
}

func labelToAttribute(l event.Label) attribute.KeyValue {
	switch {
	case l.IsString():
		return attribute.String(l.Name, l.String())
	case l.IsInt64():
		return attribute.Int64(l.Name, l.Int64())
	case l.IsFloat64():
		return attribute.Float64(l.Name, l.Float64())
	case l.IsBool():
		return attribute.Bool(l.Name, l.Bool())
	default: // including uint64
		panic(fmt.Errorf("cannot convert label value of type %T to attribute.KeyValue", l.Interface()))
	}
}
