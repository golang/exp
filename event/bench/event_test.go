// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench_test

import (
	"context"
	"io"
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
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
			ctx, _ = event.Start(ctx, aMsg)
			event.To(ctx).With(aValue.Of(a)).Annotate()
			return ctx
		},
		AEnd: func(ctx context.Context) {
			event.To(ctx).Deliver(event.EndKind, "")
		},
		BStart: func(ctx context.Context, b string) context.Context {
			ctx, _ = event.Start(ctx, bMsg)
			event.To(ctx).With(bValue.Of(b)).Annotate()
			return ctx
		},
		BEnd: func(ctx context.Context) {
			event.To(ctx).Deliver(event.EndKind, "")
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

func eventDisabled() context.Context {
	event.Disable()
	return context.Background()
}

func eventNoExporter() context.Context {
	// register an exporter to turn the system on, but not in this context
	event.NewExporter(noopHandler{})
	return context.Background()
}

func eventNoop() context.Context {
	e := event.NewExporter(noopHandler{})
	e.Now = eventtest.TestNow()
	return event.WithExporter(context.Background(), e)
}

func eventPrint(w io.Writer) context.Context {
	e := event.NewExporter(event.Printer(w))
	e.Now = eventtest.TestNow()
	return event.WithExporter(context.Background(), e)
}

func BenchmarkLogEventDisabled(b *testing.B) {
	runBenchmark(b, eventDisabled(), eventLog)
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

func TestLogEventf(t *testing.T) {
	testBenchmark(t, eventPrint, eventLogf, `
2020/03/05 14:27:48	[1]	log	a where A=0
2020/03/05 14:27:49	[2]	log	b where B="A value"
2020/03/05 14:27:50	[3]	log	a where A=1
2020/03/05 14:27:51	[4]	log	b where B="Some other value"
2020/03/05 14:27:52	[5]	log	a where A=22
2020/03/05 14:27:53	[6]	log	b where B="Some other value"
2020/03/05 14:27:54	[7]	log	a where A=333
2020/03/05 14:27:55	[8]	log	b where B=""
2020/03/05 14:27:56	[9]	log	a where A=4444
2020/03/05 14:27:57	[10]	log	b where B="prime count of values"
2020/03/05 14:27:58	[11]	log	a where A=55555
2020/03/05 14:27:59	[12]	log	b where B="V"
2020/03/05 14:28:00	[13]	log	a where A=666666
2020/03/05 14:28:01	[14]	log	b where B="A value"
2020/03/05 14:28:02	[15]	log	a where A=7777777
2020/03/05 14:28:03	[16]	log	b where B="A value"
`)
}

func TestLogEvent(t *testing.T) {
	testBenchmark(t, eventPrint, eventLog, `
2020/03/05 14:27:48	[1]	log	a	{"A":0}
2020/03/05 14:27:49	[2]	log	b	{"B":"A value"}
2020/03/05 14:27:50	[3]	log	a	{"A":1}
2020/03/05 14:27:51	[4]	log	b	{"B":"Some other value"}
2020/03/05 14:27:52	[5]	log	a	{"A":22}
2020/03/05 14:27:53	[6]	log	b	{"B":"Some other value"}
2020/03/05 14:27:54	[7]	log	a	{"A":333}
2020/03/05 14:27:55	[8]	log	b	{"B":""}
2020/03/05 14:27:56	[9]	log	a	{"A":4444}
2020/03/05 14:27:57	[10]	log	b	{"B":"prime count of values"}
2020/03/05 14:27:58	[11]	log	a	{"A":55555}
2020/03/05 14:27:59	[12]	log	b	{"B":"V"}
2020/03/05 14:28:00	[13]	log	a	{"A":666666}
2020/03/05 14:28:01	[14]	log	b	{"B":"A value"}
2020/03/05 14:28:02	[15]	log	a	{"A":7777777}
2020/03/05 14:28:03	[16]	log	b	{"B":"A value"}
`)
}

func BenchmarkTraceEventNoop(b *testing.B) {
	runBenchmark(b, eventPrint(io.Discard), eventTrace)
}

func BenchmarkMetricEventNoop(b *testing.B) {
	runBenchmark(b, eventPrint(io.Discard), eventMetric)
}

type noopHandler struct{}

func (noopHandler) Handle(ev *event.Event) {}
