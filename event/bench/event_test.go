// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench_test

import (
	"context"
	"io"
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/eventtest"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/keys"
)

var (
	aValue  = keys.Int(aName)
	bValue  = keys.String(bName)
	aCount  = keys.Int64("aCount")
	aStat   = keys.Int("aValue")
	bCount  = keys.Int64("B")
	bLength = keys.Int("BLen")

	eventLog = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			event.To(ctx).With(aValue.Of(a)).Log(aMsg)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			event.To(ctx).With(bValue.Of(b)).Log(bMsg)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	eventLogf = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			event.To(ctx).Logf(aMsgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			event.To(ctx).Logf(bMsgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	eventTrace = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			ctx, _ = event.To(ctx).Start(aMsg)
			event.To(ctx).With(aValue.Of(a)).Annotate()
			return ctx
		},
		AEnd: func(ctx context.Context) {
			event.To(ctx).End()
		},
		BStart: func(ctx context.Context, b string) context.Context {
			ctx, _ = event.To(ctx).Start(bMsg)
			event.To(ctx).With(bValue.Of(b)).Annotate()
			return ctx
		},
		BEnd: func(ctx context.Context) {
			event.To(ctx).End()
		},
	}

	eventMetric = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			event.To(ctx).With(aStat.Of(a)).Metric()
			event.To(ctx).With(aCount.Of(1)).Metric()
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			event.To(ctx).With(bLength.Of(len(b))).Metric()
			event.To(ctx).With(bCount.Of(1)).Metric()
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

func eventNoExporter() context.Context {
	return event.WithExporter(context.Background(), nil)
}

func eventNoop() context.Context {
	e := event.NewExporter(noopHandler{})
	e.Now = eventtest.TestNow()
	return event.WithExporter(context.Background(), e)
}

func eventPrint(w io.Writer) context.Context {
	e := event.NewExporter(logfmt.NewHandler(w))
	e.Now = eventtest.TestNow()
	return event.WithExporter(context.Background(), e)
}

func BenchmarkLogEventNoExporter(b *testing.B) {
	runBenchmark(b, eventNoExporter(), eventLog)
}

func BenchmarkLogEventNoop(b *testing.B) {
	runBenchmark(b, eventNoop(), eventLog)
}

func BenchmarkLogEventDiscard(b *testing.B) {
	runBenchmark(b, eventPrint(io.Discard), eventLog)
}

func BenchmarkLogEventfDiscard(b *testing.B) {
	runBenchmark(b, eventPrint(io.Discard), eventLogf)
}

func BenchmarkTraceEventNoop(b *testing.B) {
	runBenchmark(b, eventPrint(io.Discard), eventTrace)
}

func BenchmarkMetricEventNoop(b *testing.B) {
	runBenchmark(b, eventPrint(io.Discard), eventMetric)
}

type noopHandler struct{}

func (noopHandler) Log_(ctx context.Context, ev *event.Event)     {}
func (noopHandler) Metric(ctx context.Context, ev *event.Event)   {}
func (noopHandler) Annotate(ctx context.Context, ev *event.Event) {}
func (noopHandler) End(ctx context.Context, ev *event.Event)      {}
func (noopHandler) Start(ctx context.Context, ev *event.Event) context.Context {
	return ctx
}
