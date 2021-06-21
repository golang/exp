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
	"golang.org/x/exp/event/severity"
)

var (
	eventLog = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			severity.Info.Log(ctx, eventtest.A.Msg, event.Int64(eventtest.A.Name, int64(a)))
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			severity.Info.Log(ctx, eventtest.B.Msg, event.String(eventtest.B.Name, b))
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	eventLogf = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			severity.Info.Logf(ctx, eventtest.A.Msgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			severity.Info.Logf(ctx, eventtest.B.Msgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	eventTrace = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			ctx = event.Start(ctx, eventtest.A.Msg, event.Int64(eventtest.A.Name, int64(a)))
			return ctx
		},
		AEnd: func(ctx context.Context) {
			event.End(ctx)
		},
		BStart: func(ctx context.Context, b string) context.Context {
			ctx = event.Start(ctx, eventtest.B.Msg, event.String(eventtest.B.Name, b))
			return ctx
		},
		BEnd: func(ctx context.Context) {
			event.End(ctx)
		},
	}

	eventMetric = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			gauge.Record(ctx, 1, event.Int64("aValue", int64(a)))
			gauge.Record(ctx, 1, event.Int64("aCount", 1))
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			gauge.Record(ctx, 1, event.Int64("BLen", int64(len(b))))
			gauge.Record(ctx, 1, event.Int64("B", 1))
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

func eventNoExporter() context.Context {
	return event.WithExporter(context.Background(), nil)
}

func eventNoop() context.Context {
	return event.WithExporter(context.Background(), event.NewExporter(nopHandler{}, eventtest.ExporterOptions()))
}

func eventPrint(w io.Writer) context.Context {
	return event.WithExporter(context.Background(), event.NewExporter(logfmt.NewHandler(w), eventtest.ExporterOptions()))
}

func eventPrintSource(w io.Writer) context.Context {
	opts := eventtest.ExporterOptions()
	opts.EnableNamespaces = true
	return event.WithExporter(context.Background(), event.NewExporter(logfmt.NewHandler(w), opts))
}

type nopHandler struct{}

func (nopHandler) Event(ctx context.Context, _ *event.Event) context.Context { return ctx }

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
