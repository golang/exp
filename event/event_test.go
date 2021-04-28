// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
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
		expect: `time=2020-03-05T14:27:48 id=1 kind=log msg="a message"
`}, {
		name:   "log 1",
		events: func(ctx context.Context) { event.To(ctx).With(l1).Log("a message") },
		expect: `time=2020-03-05T14:27:48 id=1 kind=log msg="a message" l1=1`,
	}, {
		name:   "log 2",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).Log("a message") },
		expect: `time=2020-03-05T14:27:48 id=1 kind=log msg="a message" l1=1 l2=2`,
	}, {
		name:   "log 3",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).With(l3).Log("a message") },
		expect: `time=2020-03-05T14:27:48 id=1 kind=log msg="a message" l1=1 l2=2 l3=3`,
	}, {
		name: "span",
		events: func(ctx context.Context) {
			ctx, end := event.Start(ctx, "span")
			end()
		},
		expect: `
time=2020-03-05T14:27:48 id=1 kind=start msg=span
time=2020-03-05T14:27:49 id=2 span=1 kind=end
`}, {
		name: "span nested",
		events: func(ctx context.Context) {
			ctx, end := event.Start(ctx, "parent")
			defer end()
			child, end2 := event.Start(ctx, "child")
			defer end2()
			event.To(child).Log("message")
		},
		expect: `
time=2020-03-05T14:27:48 id=1 kind=start msg=parent
time=2020-03-05T14:27:49 id=2 span=1 kind=start msg=child
time=2020-03-05T14:27:50 id=3 span=2 kind=log msg=message
time=2020-03-05T14:27:51 id=4 span=2 kind=end
time=2020-03-05T14:27:52 id=5 span=1 kind=end
`}, {
		name:   "metric",
		events: func(ctx context.Context) { event.To(ctx).With(l1).Metric() },
		expect: `time=2020-03-05T14:27:48 id=1 kind=metric l1=1`,
	}, {
		name:   "metric 2",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).Metric() },
		expect: `time=2020-03-05T14:27:48 id=1 kind=metric l1=1 l2=2`,
	}, {
		name:   "annotate",
		events: func(ctx context.Context) { event.To(ctx).With(l1).Annotate() },
		expect: `time=2020-03-05T14:27:48 id=1 kind=annotate l1=1`,
	}, {
		name:   "annotate 2",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).Annotate() },
		expect: `time=2020-03-05T14:27:48 id=1 kind=annotate l1=1 l2=2`,
	}, {
		name: "multiple events",
		events: func(ctx context.Context) {
			b := event.To(ctx)
			b.Clone().With(keys.Int("myInt").Of(6)).Log("my event")
			b.With(keys.String("myString").Of("some string value")).Log("string event")
		},
		expect: `
time=2020-03-05T14:27:48 id=1 kind=log msg="my event" myInt=6
time=2020-03-05T14:27:49 id=2 kind=log msg="string event" myString="some string value"
`}} {
		buf := &strings.Builder{}
		h := event.Printer(buf)
		e := event.NewExporter(h)
		e.Now = eventtest.TestNow()
		ctx := event.WithExporter(ctx, e)
		test.events(ctx)
		got := strings.TrimSpace(buf.String())
		expect := strings.TrimSpace(test.expect)
		if got != expect {
			t.Errorf("%s failed\ngot   : %s\nexpect: %s", test.name, got, expect)
		}
	}
}

func ExampleLog() {
	e := event.NewExporter(event.Printer(os.Stdout))
	e.Now = eventtest.TestNow()
	ctx := event.WithExporter(context.Background(), e)
	event.To(ctx).With(keys.Int("myInt").Of(6)).Log("my event")
	event.To(ctx).With(keys.String("myString").Of("some string value")).Log("error event")
	// Output:
	// time=2020-03-05T14:27:48 id=1 kind=log msg="my event" myInt=6
	// time=2020-03-05T14:27:49 id=2 kind=log msg="error event" myString="some string value"
}
