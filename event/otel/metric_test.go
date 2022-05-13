// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package otel_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metrictest"
	"go.opentelemetry.io/otel/sdk/metric/number"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/otel"
)

func TestMeter(t *testing.T) {
	ctx := context.Background()
	mp, exp := metrictest.NewTestMeterProvider()
	mh := otel.NewMetricHandler(mp.Meter("test"))
	ctx = event.WithExporter(ctx, event.NewExporter(mh, nil))
	recordMetrics(ctx)

	exp.Collect(ctx)
	lib := metrictest.Library{InstrumentationName: "test"}
	got := exp.Records
	want := []metrictest.ExportRecord{
		{
			InstrumentName:         "golang.org/x/exp/event/otel_test/hits",
			Sum:                    number.NewInt64Number(8),
			Attributes:             nil,
			InstrumentationLibrary: lib,
			AggregationKind:        aggregation.SumKind,
			NumberKind:             number.Int64Kind,
		},
		{
			InstrumentName: "golang.org/x/exp/event/otel_test/temp",
			Sum:            number.NewFloat64Number(-100),
			Attributes: []attribute.KeyValue{
				{
					Key:   attribute.Key("location"),
					Value: attribute.StringValue("Mare Imbrium"),
				},
			},
			InstrumentationLibrary: lib,
			AggregationKind:        aggregation.SumKind,
			NumberKind:             number.Float64Kind,
		},
		{
			InstrumentName:         "golang.org/x/exp/event/otel_test/latency",
			Sum:                    number.NewInt64Number(int64(2503 * time.Millisecond)),
			Count:                  2,
			Attributes:             nil,
			InstrumentationLibrary: lib,
			AggregationKind:        aggregation.HistogramKind,
			NumberKind:             number.Int64Kind,
			Histogram: aggregation.Buckets{
				Boundaries: []float64{5000, 10000, 25000, 50000, 100000, 250000, 500000, 1e+06, 2.5e+06, 5e+06, 1e+07},
				Counts:     []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
			},
		},
	}

	if diff := cmp.Diff(want, got, cmp.Comparer(valuesEqual)); diff != "" {
		t.Errorf("mismatch (-want, got):\n%s", diff)
	}
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
