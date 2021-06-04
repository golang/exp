// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"testing"

	"golang.org/x/exp/event/adapter/eventtest"
	"golang.org/x/exp/event/bench"
)

var (
	baseline = bench.Hooks{
		AStart: func(ctx context.Context, a int) context.Context { return ctx },
		AEnd:   func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context { return ctx },
		BEnd:   func(ctx context.Context) {},
	}

	stdlibLog = bench.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			logCtx(ctx).Printf(bench.A.Msgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			logCtx(ctx).Printf(bench.B.Msgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	stdlibPrintf = bench.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			ctxPrintf(ctx, bench.A.Msgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			ctxPrintf(ctx, bench.B.Msgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

func BenchmarkBaseline(b *testing.B) {
	bench.RunBenchmark(b, context.Background(), bench.Hooks{
		AStart: func(ctx context.Context, a int) context.Context { return ctx },
		AEnd:   func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context { return ctx },
		BEnd:   func(ctx context.Context) {},
	})
}

type stdlibLogKey struct{}

func logCtx(ctx context.Context) *log.Logger {
	return ctx.Value(stdlibLogKey{}).(*log.Logger)
}

func stdlibLogger(w io.Writer) context.Context {
	logger := log.New(w, "", log.LstdFlags)
	return context.WithValue(context.Background(), stdlibLogKey{}, logger)
}

func stdlibLoggerNoTime(w io.Writer) context.Context {
	// there is no way to fixup the time, so we have to suppress it
	logger := log.New(w, "", 0)
	return context.WithValue(context.Background(), stdlibLogKey{}, logger)
}

type writerKey struct{}

func ctxPrintf(ctx context.Context, msg string, args ...interface{}) {
	ctx.Value(writerKey{}).(func(string, ...interface{}))(msg, args...)
}

func stdlibWriter(w io.Writer) context.Context {
	now := eventtest.ExporterOptions().Now
	return context.WithValue(context.Background(), writerKey{},
		func(msg string, args ...interface{}) {
			fmt.Fprintf(w, "time=%q level=info msg=%q\n",
				now().Format(bench.TimeFormat),
				fmt.Sprintf(msg, args...))
		},
	)
}

func BenchmarkStdlibLogfDiscard(b *testing.B) {
	bench.RunBenchmark(b, stdlibLogger(ioutil.Discard), stdlibLog)
}

func BenchmarkStdlibPrintfDiscard(b *testing.B) {
	bench.RunBenchmark(b, stdlibWriter(io.Discard), stdlibPrintf)
}

func TestLogStdlib(t *testing.T) {
	bench.TestBenchmark(t, stdlibLoggerNoTime, stdlibLog, `
a where A=0
b where B="A value"
a where A=1
b where B="Some other value"
a where A=22
b where B="Some other value"
a where A=333
b where B=" "
a where A=4444
b where B="prime count of values"
a where A=55555
b where B="V"
a where A=666666
b where B="A value"
a where A=7777777
b where B="A value"
`)
}

func TestLogPrintf(t *testing.T) {
	bench.TestBenchmark(t, stdlibWriter, stdlibPrintf, bench.LogfOutput)
}
