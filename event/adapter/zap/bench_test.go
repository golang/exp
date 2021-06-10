// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zap_test

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
	zapLog = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			zapCtx(ctx).Info(eventtest.A.Msg, zap.Int(eventtest.A.Name, a))
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			zapCtx(ctx).Info(eventtest.B.Msg, zap.String(eventtest.B.Name, b))
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
	zapLogf = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			zapCtx(ctx).Sugar().Infof(eventtest.A.Msgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			zapCtx(ctx).Sugar().Infof(eventtest.B.Msgf, b)
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
	now := eventtest.ExporterOptions().Now
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeDuration = zapcore.NanosDurationEncoder
	timeEncoder := zapcore.TimeEncoderOfLayout(eventtest.TimeFormat)
	ec.EncodeTime = func(_ time.Time, a zapcore.PrimitiveArrayEncoder) {
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

func BenchmarkZapLogDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, zapPrint(io.Discard), zapLog)
}

func BenchmarkZapLogfDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, zapPrint(io.Discard), zapLogf)
}

func TestZapLogfDiscard(t *testing.T) {
	eventtest.TestBenchmark(t, zapPrint, zapLogf, `
2020/03/05 14:27:48	info	a where A=0
2020/03/05 14:27:49	info	b where B="A value"
2020/03/05 14:27:50	info	a where A=1
2020/03/05 14:27:51	info	b where B="Some other value"
2020/03/05 14:27:52	info	a where A=22
2020/03/05 14:27:53	info	b where B="Some other value"
2020/03/05 14:27:54	info	a where A=333
2020/03/05 14:27:55	info	b where B=" "
2020/03/05 14:27:56	info	a where A=4444
2020/03/05 14:27:57	info	b where B="prime count of values"
2020/03/05 14:27:58	info	a where A=55555
2020/03/05 14:27:59	info	b where B="V"
2020/03/05 14:28:00	info	a where A=666666
2020/03/05 14:28:01	info	b where B="A value"
2020/03/05 14:28:02	info	a where A=7777777
2020/03/05 14:28:03	info	b where B="A value"
`)
}
func TestLogZap(t *testing.T) {
	eventtest.TestBenchmark(t, zapPrint, zapLog, `
2020/03/05 14:27:48	info	a	{"A": 0}
2020/03/05 14:27:49	info	b	{"B": "A value"}
2020/03/05 14:27:50	info	a	{"A": 1}
2020/03/05 14:27:51	info	b	{"B": "Some other value"}
2020/03/05 14:27:52	info	a	{"A": 22}
2020/03/05 14:27:53	info	b	{"B": "Some other value"}
2020/03/05 14:27:54	info	a	{"A": 333}
2020/03/05 14:27:55	info	b	{"B": " "}
2020/03/05 14:27:56	info	a	{"A": 4444}
2020/03/05 14:27:57	info	b	{"B": "prime count of values"}
2020/03/05 14:27:58	info	a	{"A": 55555}
2020/03/05 14:27:59	info	b	{"B": "V"}
2020/03/05 14:28:00	info	a	{"A": 666666}
2020/03/05 14:28:01	info	b	{"B": "A value"}
2020/03/05 14:28:02	info	a	{"A": 7777777}
2020/03/05 14:28:03	info	b	{"B": "A value"}
`)
}
