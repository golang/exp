// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type Hooks struct {
	AStart func(ctx context.Context, a int) context.Context
	AEnd   func(ctx context.Context)
	BStart func(ctx context.Context, b string) context.Context
	BEnd   func(ctx context.Context)
}

var (
	initialList = []int{0, 1, 22, 333, 4444, 55555, 666666, 7777777}
	stringList  = []string{
		"A value",
		"Some other value",
		"A nice longer value but not too long",
		"V",
		"",
		"ı",
		"prime count of values",
	}
)

const (
	aName = "A"
	aMsg  = "A"
	aMsgf = aMsg + " where a=%d"
	bName = "B"
	bMsg  = "b"
	bMsgf = bMsg + " where b=%q"

	timeFormat = "2006/01/02 15:04:05"
)

type namedBenchmark struct {
	name string
	test func(ctx context.Context) func(*testing.B)
}

func benchA(ctx context.Context, hooks Hooks, a int) int {
	ctx = hooks.AStart(ctx, a)
	defer hooks.AEnd(ctx)
	return benchB(ctx, hooks, a, stringList[a%len(stringList)])
}

func benchB(ctx context.Context, hooks Hooks, a int, b string) int {
	ctx = hooks.BStart(ctx, b)
	defer hooks.BEnd(ctx)
	return a + len(b)
}

func runBenchmark(t testing.TB, ctx context.Context, hooks Hooks) {
	var acc int
	if b, ok := t.(*testing.B); ok {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, value := range initialList {
				acc += benchA(ctx, hooks, value)
			}
		}
		return
	}
	for _, value := range initialList {
		acc += benchA(ctx, hooks, value)
	}
}

func testBenchmark(t testing.TB, f func(io.Writer) context.Context, hooks Hooks, expect string) {
	buf := strings.Builder{}
	runBenchmark(t, f(&buf), hooks)
	got := strings.TrimSpace(buf.String())
	expect = strings.TrimSpace(expect)
	if diff := cmp.Diff(got, expect); diff != "" {
		t.Error(diff)
	}
}
