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
		expect: `
2020/03/05 14:27:48	[1]	log	a message
`}, {
		name:   "log 1",
		events: func(ctx context.Context) { event.To(ctx).With(l1).Log("a message") },
		expect: `2020/03/05 14:27:48	[1]	log	a message	{"l1":1}`}, {
		name:   "simple",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).Log("a message") },
		expect: `2020/03/05 14:27:48	[1]	log	a message	{"l1":1, "l2":2}`,
	}, {
		name:   "simple",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).With(l3).Log("a message") },
		expect: `2020/03/05 14:27:48	[1]	log	a message	{"l1":1, "l2":2, "l3":3}`,
	}, {
		name: "span",
		events: func(ctx context.Context) {
			ctx, end := event.Start(ctx, "span")
			end()
		},
		expect: `
2020/03/05 14:27:48	[1]	start	span
2020/03/05 14:27:49	[2:1]	end
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
2020/03/05 14:27:48	[1]	start	parent
2020/03/05 14:27:49	[2:1]	start	child
2020/03/05 14:27:50	[3:2]	log	message
2020/03/05 14:27:51	[4:2]	end
2020/03/05 14:27:52	[5:1]	end
`}, {
		name:   "metric",
		events: func(ctx context.Context) { event.To(ctx).With(l1).Metric() },
		expect: `2020/03/05 14:27:48	[1]	metric	{"l1":1}`,
	}, {
		name:   "metric 2",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).Metric() },
		expect: `2020/03/05 14:27:48	[1]	metric	{"l1":1, "l2":2}`,
	}, {
		name:   "annotate",
		events: func(ctx context.Context) { event.To(ctx).With(l1).Annotate() },
		expect: `2020/03/05 14:27:48	[1]	annotate	{"l1":1}`,
	}, {
		name:   "annotate 2",
		events: func(ctx context.Context) { event.To(ctx).With(l1).With(l2).Annotate() },
		expect: `2020/03/05 14:27:48	[1]	annotate	{"l1":1, "l2":2}`,
	}, {
		name: "multiple events",
		events: func(ctx context.Context) {
			b := event.To(ctx)
			b.Clone().With(keys.Int("myInt").Of(6)).Log("my event")
			b.With(keys.String("myString").Of("some string value")).Log("string event")
		},
		expect: `
2020/03/05 14:27:48	[1]	log	my event	{"myInt":6}
2020/03/05 14:27:49	[2]	log	string event	{"myString":"some string value"}
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
	// 2020/03/05 14:27:48	[1]	log	my event	{"myInt":6}
	// 2020/03/05 14:27:49	[2]	log	error event	{"myString":"some string value"}
}
