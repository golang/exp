// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package otel_test

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/event"
)

func TestMeter(t *testing.T) {
	t.Skip("package go.opentelemetry.io/otel/metric/metrictest removed")

	// ctx := context.Background()
	// mp := metrictest.NewMeterProvider()
	// mh := otel.NewMetricHandler(mp.Meter("test"))
	// ctx = event.WithExporter(ctx, event.NewExporter(mh, nil))
	// recordMetrics(ctx)

	// lib := metrictest.Library{InstrumentationName: "test"}
	// emptyLabels := map[attribute.Key]attribute.Value{}
	// got := metrictest.AsStructs(mp.MeasurementBatches)
	// want := []metrictest.Measured{
	// 	{
	// 		Name:    "golang.org/x/exp/event/otel_test/hits",
	// 		Number:  number.NewInt64Number(8),
	// 		Labels:  emptyLabels,
	// 		Library: lib,
	// 	},
	// 	{
	// 		Name:    "golang.org/x/exp/event/otel_test/temp",
	// 		Number:  number.NewFloat64Number(-100),
	// 		Labels:  map[attribute.Key]attribute.Value{"location": attribute.StringValue("Mare Imbrium")},
	// 		Library: lib,
	// 	},
	// 	{
	// 		Name:    "golang.org/x/exp/event/otel_test/latency",
	// 		Number:  number.NewInt64Number(int64(1248 * time.Millisecond)),
	// 		Labels:  emptyLabels,
	// 		Library: lib,
	// 	},
	// 	{
	// 		Name:    "golang.org/x/exp/event/otel_test/latency",
	// 		Number:  number.NewInt64Number(int64(1255 * time.Millisecond)),
	// 		Labels:  emptyLabels,
	// 		Library: lib,
	// 	},
	// }

	// if diff := cmp.Diff(want, got, cmp.Comparer(valuesEqual)); diff != "" {
	// 	t.Errorf("mismatch (-want, got):\n%s", diff)
	// }
}

func valuesEqual(v1, v2 attribute.Value) bool {
	return v1.AsInterface() == v2.AsInterface()
}

func recordMetrics(ctx context.Context) {
	c := event.NewCounter("hits", &event.MetricOptions{Description: "Earth meteorite hits"})
	g := event.NewFloatGauge("temp", &event.MetricOptions{Description: "moon surface temperature in Kelvin"})
	d := event.NewDuration("latency", &event.MetricOptions{Description: "Earth-moon comms lag, milliseconds"})

	c.Record(ctx, 8)
	g.Record(ctx, -100, event.String("location", "Mare Imbrium"))
	d.Record(ctx, 1248*time.Millisecond)
	d.Record(ctx, 1255*time.Millisecond)
}
