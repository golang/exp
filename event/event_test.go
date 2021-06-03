// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/eventtest"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/keys"
)

var (
	l1 = keys.Int("l1").Of(1)
	l2 = keys.Int("l2").Of(2)
	l3 = keys.Int("l3").Of(3)
)

func TestPrint(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name   string
		events func(context.Context)
		expect string
	}{{
		name:   "simple",
		events: func(ctx context.Context) { event.To(ctx).Log("a message") },
		expect: `time=2020-03-05T14:27:48 msg="a message"
`}, {
		name:   "log 1",
		events: func(ctx context.Context) { event.To(ctx).With(l1).Log("a message") },
		expect: `time=2020-03-05T14:27:48 l1=1 msg="a message"`,
	}, {
		name:   "log 2",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).Log("a message") },
		expect: `time=2020-03-05T14:27:48 l1=1 l2=2 msg="a message"`,
	}, {
		name:   "log 3",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).With(l3).Log("a message") },
		expect: `time=2020-03-05T14:27:48 l1=1 l2=2 l3=3 msg="a message"`,
	}, {
		name: "span",
		events: func(ctx context.Context) {
			ctx, end := event.To(ctx).Start("span")
			end()
		},
		expect: `
time=2020-03-05T14:27:48 trace=1 name=span
time=2020-03-05T14:27:49 parent=1 end
`}, {
		name: "span nested",
		events: func(ctx context.Context) {
			ctx, end := event.To(ctx).Start("parent")
			defer end()
			child, end2 := event.To(ctx).Start("child")
			defer end2()
			event.To(child).Log("message")
		},
		expect: `
time=2020-03-05T14:27:48 trace=1 name=parent
time=2020-03-05T14:27:49 parent=1 trace=2 name=child
time=2020-03-05T14:27:50 parent=2 msg=message
time=2020-03-05T14:27:51 parent=2 end
time=2020-03-05T14:27:52 parent=1 end
`}, {
		name:   "metric",
		events: func(ctx context.Context) { event.To(ctx).With(l1).Metric() },
		expect: `time=2020-03-05T14:27:48 l1=1 metric`,
	}, {
		name:   "metric 2",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).Metric() },
		expect: `time=2020-03-05T14:27:48 l1=1 l2=2 metric`,
	}, {
		name:   "annotate",
		events: func(ctx context.Context) { event.To(ctx).With(l1).Annotate() },
		expect: `time=2020-03-05T14:27:48 l1=1`,
	}, {
		name:   "annotate 2",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).Annotate() },
		expect: `time=2020-03-05T14:27:48 l1=1 l2=2`,
	}, {
		name: "multiple events",
		events: func(ctx context.Context) {
			b := event.To(ctx)
			b.Clone().With(keys.Int("myInt").Of(6)).Log("my event")
			b.With(keys.String("myString").Of("some string value")).Log("string event")
		},
		expect: `
time=2020-03-05T14:27:48 myInt=6 msg="my event"
time=2020-03-05T14:27:49 myString="some string value" msg="string event"
`}} {
		buf := &strings.Builder{}
		ctx := event.WithExporter(ctx, event.NewExporter(logfmt.NewHandler(buf)))
		eventtest.FixedNow(ctx)
		test.events(ctx)
		got := strings.TrimSpace(buf.String())
		expect := strings.TrimSpace(test.expect)
		if got != expect {
			t.Errorf("%s failed\ngot   : %s\nexpect: %s", test.name, got, expect)
		}
	}
}

func ExampleLog() {
	ctx := event.WithExporter(context.Background(), event.NewExporter(logfmt.NewHandler(os.Stdout)))
	eventtest.FixedNow(ctx)
	event.To(ctx).With(keys.Int("myInt").Of(6)).Log("my event")
	event.To(ctx).With(keys.String("myString").Of("some string value")).Log("error event")
	// Output:
	// time=2020-03-05T14:27:48 myInt=6 msg="my event"
	// time=2020-03-05T14:27:49 myString="some string value" msg="error event"
}
