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

	"golang.org/x/exp/event/eventtest"
)

var (
	baseline = Hooks{
		AStart: func(ctx context.Context, a int) context.Context { return ctx },
		AEnd:   func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context { return ctx },
		BEnd:   func(ctx context.Context) {},
	}

	stdlibLog = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			logCtx(ctx).Printf(aMsgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			logCtx(ctx).Printf(bMsgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	stdlibPrintf = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			ctxPrintf(ctx, aMsgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			ctxPrintf(ctx, bMsgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

func BenchmarkBaseline(b *testing.B) {
	runBenchmark(b, context.Background(), Hooks{
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
	now := eventtest.TestNow()
	return context.WithValue(context.Background(), writerKey{},
		func(msg string, args ...interface{}) {
			fmt.Fprint(w, now().Format(timeFormat), " ")
			fmt.Fprintf(w, msg, args...)
			fmt.Fprintln(w)
		},
	)
}

func BenchmarkLogStdlib(b *testing.B) {
	runBenchmark(b, stdlibLogger(ioutil.Discard), stdlibLog)
}

func BenchmarkLogPrintf(b *testing.B) {
	runBenchmark(b, stdlibWriter(io.Discard), stdlibPrintf)
}

func TestLogStdlib(t *testing.T) {
	testBenchmark(t, stdlibLoggerNoTime, stdlibLog, `
a where A=0
b where B="A value"
a where A=1
b where B="Some other value"
a where A=22
b where B="Some other value"
a where A=333
b where B=""
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
	testBenchmark(t, stdlibWriter, stdlibPrintf, `
2020/03/05 14:27:48 a where A=0
2020/03/05 14:27:49 b where B="A value"
2020/03/05 14:27:50 a where A=1
2020/03/05 14:27:51 b where B="Some other value"
2020/03/05 14:27:52 a where A=22
2020/03/05 14:27:53 b where B="Some other value"
2020/03/05 14:27:54 a where A=333
2020/03/05 14:27:55 b where B=""
2020/03/05 14:27:56 a where A=4444
2020/03/05 14:27:57 b where B="prime count of values"
2020/03/05 14:27:58 a where A=55555
2020/03/05 14:27:59 b where B="V"
2020/03/05 14:28:00 a where A=666666
2020/03/05 14:28:01 b where B="A value"
2020/03/05 14:28:02 a where A=7777777
2020/03/05 14:28:03 b where B="A value"
`)
}
