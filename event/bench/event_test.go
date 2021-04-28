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

func eventNoExporter() context.Context {
	return event.WithExporter(context.Background(), nil)
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
	event.SetEnabled(false)
	defer event.SetEnabled(true)
	runBenchmark(b, context.Background(), eventLog)
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
time=2020-03-05T14:27:48 id=1 kind=log msg="a where A=0"
time=2020-03-05T14:27:49 id=2 kind=log msg="b where B=\"A value\""
time=2020-03-05T14:27:50 id=3 kind=log msg="a where A=1"
time=2020-03-05T14:27:51 id=4 kind=log msg="b where B=\"Some other value\""
time=2020-03-05T14:27:52 id=5 kind=log msg="a where A=22"
time=2020-03-05T14:27:53 id=6 kind=log msg="b where B=\"Some other value\""
time=2020-03-05T14:27:54 id=7 kind=log msg="a where A=333"
time=2020-03-05T14:27:55 id=8 kind=log msg="b where B=\"\""
time=2020-03-05T14:27:56 id=9 kind=log msg="a where A=4444"
time=2020-03-05T14:27:57 id=10 kind=log msg="b where B=\"prime count of values\""
time=2020-03-05T14:27:58 id=11 kind=log msg="a where A=55555"
time=2020-03-05T14:27:59 id=12 kind=log msg="b where B=\"V\""
time=2020-03-05T14:28:00 id=13 kind=log msg="a where A=666666"
time=2020-03-05T14:28:01 id=14 kind=log msg="b where B=\"A value\""
time=2020-03-05T14:28:02 id=15 kind=log msg="a where A=7777777"
time=2020-03-05T14:28:03 id=16 kind=log msg="b where B=\"A value\""
`)
}

func TestLogEvent(t *testing.T) {
	testBenchmark(t, eventPrint, eventLog, `
time=2020-03-05T14:27:48 id=1 kind=log msg=a A=0
time=2020-03-05T14:27:49 id=2 kind=log msg=b B="A value"
time=2020-03-05T14:27:50 id=3 kind=log msg=a A=1
time=2020-03-05T14:27:51 id=4 kind=log msg=b B="Some other value"
time=2020-03-05T14:27:52 id=5 kind=log msg=a A=22
time=2020-03-05T14:27:53 id=6 kind=log msg=b B="Some other value"
time=2020-03-05T14:27:54 id=7 kind=log msg=a A=333
time=2020-03-05T14:27:55 id=8 kind=log msg=b B=""
time=2020-03-05T14:27:56 id=9 kind=log msg=a A=4444
time=2020-03-05T14:27:57 id=10 kind=log msg=b B="prime count of values"
time=2020-03-05T14:27:58 id=11 kind=log msg=a A=55555
time=2020-03-05T14:27:59 id=12 kind=log msg=b B=V
time=2020-03-05T14:28:00 id=13 kind=log msg=a A=666666
time=2020-03-05T14:28:01 id=14 kind=log msg=b B="A value"
time=2020-03-05T14:28:02 id=15 kind=log msg=a A=7777777
time=2020-03-05T14:28:03 id=16 kind=log msg=b B="A value"
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
