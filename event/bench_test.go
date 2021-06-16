// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event_test

import (
	"context"
	"io"
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/keys"
	"golang.org/x/exp/event/severity"
)

var (
	aValue  = keys.Int(eventtest.A.Name)
	bValue  = keys.String(eventtest.B.Name)
	aCount  = keys.Int64("aCount")
	aStat   = keys.Int("aValue")
	bCount  = keys.Int64("B")
	bLength = keys.Int("BLen")

	eventLog = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			event.To(ctx).With(severity.Info).With(aValue.Of(a)).Log(eventtest.A.Msg)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			event.To(ctx).With(severity.Info).With(bValue.Of(b)).Log(eventtest.B.Msg)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	eventLogf = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			event.To(ctx).With(severity.Info).Logf(eventtest.A.Msgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			event.To(ctx).With(severity.Info).Logf(eventtest.B.Msgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	eventTrace = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			ctx, _ = event.To(ctx).Start(eventtest.A.Msg)
			event.To(ctx).With(aValue.Of(a)).Annotate()
			return ctx
		},
		AEnd: func(ctx context.Context) {
			event.To(ctx).End()
		},
		BStart: func(ctx context.Context, b string) context.Context {
			ctx, _ = event.To(ctx).Start(eventtest.B.Msg)
			event.To(ctx).With(bValue.Of(b)).Annotate()
			return ctx
		},
		BEnd: func(ctx context.Context) {
			event.To(ctx).End()
		},
	}

	eventMetric = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			event.To(ctx).With(aStat.Of(a)).Metric(gauge.Record(1))
			event.To(ctx).With(aCount.Of(1)).Metric(gauge.Record(1))
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			event.To(ctx).With(bLength.Of(len(b))).Metric(gauge.Record(1))
			event.To(ctx).With(bCount.Of(1)).Metric(gauge.Record(1))
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

func eventNoExporter() context.Context {
	return event.WithExporter(context.Background(), nil)
}

func eventNoop() context.Context {
	return event.WithExporter(context.Background(), event.NewExporter(event.NopHandler{}, eventtest.ExporterOptions()))
}

func eventPrint(w io.Writer) context.Context {
	return event.WithExporter(context.Background(), event.NewExporter(logfmt.NewHandler(w), eventtest.ExporterOptions()))
}

func eventPrintSource(w io.Writer) context.Context {
	opts := eventtest.ExporterOptions()
	opts.EnableNamespaces = true
	return event.WithExporter(context.Background(), event.NewExporter(logfmt.NewHandler(w), opts))
}

func BenchmarkEventLogNoExporter(b *testing.B) {
	eventtest.RunBenchmark(b, eventNoExporter(), eventLog)
}

func BenchmarkEventLogNoop(b *testing.B) {
	eventtest.RunBenchmark(b, eventNoop(), eventLog)
}

func BenchmarkEventLogDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, eventPrint(io.Discard), eventLog)
}

func BenchmarkEventLogSourceDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, eventPrintSource(io.Discard), eventLog)
}

func BenchmarkEventLogfDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, eventPrint(io.Discard), eventLogf)
}

func BenchmarkEventTraceNoop(b *testing.B) {
	eventtest.RunBenchmark(b, eventNoop(), eventTrace)
}

func BenchmarkEventTraceDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, eventPrint(io.Discard), eventTrace)
}

func BenchmarkEventMetricNoop(b *testing.B) {
	eventtest.RunBenchmark(b, eventNoop(), eventMetric)
}

func BenchmarkEventMetricDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, eventPrint(io.Discard), eventMetric)
}
