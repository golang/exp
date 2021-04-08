// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench_test

import (
	"context"
	"io"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/event/eventtest"
)

var (
	zapLog = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			zapCtx(ctx).Info(aMsg, zap.Int(aName, a))
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			zapCtx(ctx).Info(aMsg, zap.String(bName, b))
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
	zapLogf = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			zapCtx(ctx).Sugar().Infof(aMsgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			zapCtx(ctx).Sugar().Infof(bMsgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

type zapKey struct{}

func zapCtx(ctx context.Context) *zap.Logger {
	return ctx.Value(zapKey{}).(*zap.Logger)
}

func zapPrint(w io.Writer) context.Context {
	now := eventtest.TestNow()
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeDuration = zapcore.NanosDurationEncoder
	timeEncoder := zapcore.TimeEncoderOfLayout(timeFormat)
	ec.EncodeTime = func(t time.Time, a zapcore.PrimitiveArrayEncoder) {
		timeEncoder(now(), a)
	}
	enc := zapcore.NewConsoleEncoder(ec)
	logger := zap.New(zapcore.NewCore(
		enc,
		zapcore.AddSync(w),
		zap.InfoLevel,
	))
	return context.WithValue(context.Background(), zapKey{}, logger)
}

func BenchmarkLogZap(b *testing.B) {
	runBenchmark(b, zapPrint(io.Discard), zapLog)
}

func BenchmarkLogZapf(b *testing.B) {
	runBenchmark(b, zapPrint(io.Discard), zapLogf)
}

func TestLogZapf(t *testing.T) {
	testBenchmark(t, zapPrint, zapLogf, `
2020/03/05 14:27:48	info	A where a=0
2020/03/05 14:27:49	info	b where b="A value"
2020/03/05 14:27:50	info	A where a=1
2020/03/05 14:27:51	info	b where b="Some other value"
2020/03/05 14:27:52	info	A where a=22
2020/03/05 14:27:53	info	b where b="Some other value"
2020/03/05 14:27:54	info	A where a=333
2020/03/05 14:27:55	info	b where b=""
2020/03/05 14:27:56	info	A where a=4444
2020/03/05 14:27:57	info	b where b="prime count of values"
2020/03/05 14:27:58	info	A where a=55555
2020/03/05 14:27:59	info	b where b="V"
2020/03/05 14:28:00	info	A where a=666666
2020/03/05 14:28:01	info	b where b="A value"
2020/03/05 14:28:02	info	A where a=7777777
2020/03/05 14:28:03	info	b where b="A value"
`)
}
