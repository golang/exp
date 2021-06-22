// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event_test

import (
	"context"
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
)

const thisImportPath = "golang.org/x/exp/event_test"

func TestNamespace(t *testing.T) {
	event.RegisterHelper(testHelperB)
	event.RegisterHelper(thisImportPath + ".testHelperC")
	h := &eventtest.CaptureHandler{}
	opt := eventtest.ExporterOptions()
	opt.EnableNamespaces = true
	ctx := event.WithExporter(context.Background(), event.NewExporter(h, opt))
	for _, test := range []struct {
		name   string
		do     func(context.Context)
		expect event.Source
	}{{
		name:   "simple",
		do:     testA,
		expect: event.Source{Space: thisImportPath, Name: "testA"},
	}, {
		name:   "pointer helper",
		do:     testB,
		expect: event.Source{Space: thisImportPath, Name: "testB"},
	}, {
		name:   "named helper",
		do:     testC,
		expect: event.Source{Space: thisImportPath, Name: "testC"},
	}, {
		name:   "method",
		do:     testD,
		expect: event.Source{Space: thisImportPath, Owner: "tester", Name: "D"},
	}} {
		t.Run(test.name, func(t *testing.T) {
			h.Got = h.Got[:0]
			test.do(ctx)
			if len(h.Got) != 1 {
				t.Fatalf("Expected 1 event, got %v", len(h.Got))
			}
			got := h.Got[0].Source
			if got.Space != test.expect.Space {
				t.Errorf("got namespace %q, want, %q", got.Space, test.expect.Space)
			}
			if got.Owner != test.expect.Owner {
				t.Errorf("got owner %q, want, %q", got.Owner, test.expect.Owner)
			}
			if got.Name != test.expect.Name {
				t.Errorf("got name %q, want, %q", got.Name, test.expect.Name)
			}
		})
	}
}

type tester struct{}

//go:noinline
func testA(ctx context.Context) { event.Log(ctx, "test A") }

//go:noinline
func testB(ctx context.Context) { testHelperB(ctx) }

//go:noinline
func testHelperB(ctx context.Context) { event.Log(ctx, "test B") }

//go:noinline
func testC(ctx context.Context) { testHelperC(ctx) }

//go:noinline
func testHelperC(ctx context.Context) { event.Log(ctx, "test C") }

//go:noinline
func testD(ctx context.Context) { tester{}.D(ctx) }

//go:noinline
func (tester) D(ctx context.Context) { event.Log(ctx, "test D") }
