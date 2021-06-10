// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package eventtest

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/event/adapter/logfmt"
)

type Info struct {
	Name string
	Msg  string
	Msgf string
}

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
		" ",
		"ı",
		"prime count of values",
	}

	A = Info{
		Name: "A",
		Msg:  "a",
		Msgf: "a where A=%d",
	}

	B = Info{
		Name: "B",
		Msg:  "b",
		Msgf: "b where B=%q",
	}
)

const (
	TimeFormat = logfmt.TimeFormat

	LogfmtOutput = `
time="2020/03/05 14:27:48" level=info A=0 msg=a
time="2020/03/05 14:27:49" level=info B="A value" msg=b
time="2020/03/05 14:27:50" level=info A=1 msg=a
time="2020/03/05 14:27:51" level=info B="Some other value" msg=b
time="2020/03/05 14:27:52" level=info A=22 msg=a
time="2020/03/05 14:27:53" level=info B="Some other value" msg=b
time="2020/03/05 14:27:54" level=info A=333 msg=a
time="2020/03/05 14:27:55" level=info B=" " msg=b
time="2020/03/05 14:27:56" level=info A=4444 msg=a
time="2020/03/05 14:27:57" level=info B="prime count of values" msg=b
time="2020/03/05 14:27:58" level=info A=55555 msg=a
time="2020/03/05 14:27:59" level=info B=V msg=b
time="2020/03/05 14:28:00" level=info A=666666 msg=a
time="2020/03/05 14:28:01" level=info B="A value" msg=b
time="2020/03/05 14:28:02" level=info A=7777777 msg=a
time="2020/03/05 14:28:03" level=info B="A value" msg=b
`

	LogfOutput = `
time="2020/03/05 14:27:48" level=info msg="a where A=0"
time="2020/03/05 14:27:49" level=info msg="b where B=\"A value\""
time="2020/03/05 14:27:50" level=info msg="a where A=1"
time="2020/03/05 14:27:51" level=info msg="b where B=\"Some other value\""
time="2020/03/05 14:27:52" level=info msg="a where A=22"
time="2020/03/05 14:27:53" level=info msg="b where B=\"Some other value\""
time="2020/03/05 14:27:54" level=info msg="a where A=333"
time="2020/03/05 14:27:55" level=info msg="b where B=\" \""
time="2020/03/05 14:27:56" level=info msg="a where A=4444"
time="2020/03/05 14:27:57" level=info msg="b where B=\"prime count of values\""
time="2020/03/05 14:27:58" level=info msg="a where A=55555"
time="2020/03/05 14:27:59" level=info msg="b where B=\"V\""
time="2020/03/05 14:28:00" level=info msg="a where A=666666"
time="2020/03/05 14:28:01" level=info msg="b where B=\"A value\""
time="2020/03/05 14:28:02" level=info msg="a where A=7777777"
time="2020/03/05 14:28:03" level=info msg="b where B=\"A value\""
`
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

func runOnce(ctx context.Context, hooks Hooks) {
	var acc int
	for _, value := range initialList {
		acc += benchA(ctx, hooks, value)
	}
}

func RunBenchmark(b *testing.B, ctx context.Context, hooks Hooks) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runOnce(ctx, hooks)
	}
}

func TestBenchmark(t *testing.T, f func(io.Writer) context.Context, hooks Hooks, expect string) {
	buf := strings.Builder{}
	ctx := f(&buf)
	runOnce(ctx, hooks)
	got := strings.TrimSpace(buf.String())
	expect = strings.TrimSpace(expect)
	if diff := cmp.Diff(got, expect); diff != "" {
		t.Error(diff)
	}
}

func TestAllocs(t *testing.T, f func(io.Writer) context.Context, hooks Hooks, expect int) {
	t.Helper()
	var acc int
	ctx := f(io.Discard)
	got := int(testing.AllocsPerRun(5, func() {
		for _, value := range initialList {
			acc += benchA(ctx, hooks, value)
		}
	}))
	if got != expect {
		t.Errorf("Got %d allocs, expect %d", got, expect)
	}
}
