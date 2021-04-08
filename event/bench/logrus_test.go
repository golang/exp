// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/event/eventtest"
)

var (
	logrusLog = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			logrusCtx(ctx).WithField(aName, a).Info(aMsg)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			logrusCtx(ctx).WithField(bName, b).Info(bMsg)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	logrusLogf = Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			logrusCtx(ctx).Infof(aMsgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			logrusCtx(ctx).Infof(bMsgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

type logrusKey struct{}
type logrusTimeFormatter struct {
	now     func() time.Time
	wrapped logrus.Formatter
}

func (f *logrusTimeFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Time = f.now()
	return f.wrapped.Format(entry)
}

func logrusCtx(ctx context.Context) *logrus.Logger {
	return ctx.Value(logrusKey{}).(*logrus.Logger)
}

func logrusPrint(w io.Writer) context.Context {
	logger := &logrus.Logger{
		Out:   w,
		Level: logrus.InfoLevel,
		Formatter: &logrusTimeFormatter{
			now: eventtest.TestNow(),
			wrapped: &logrus.TextFormatter{
				FullTimestamp:   true,
				TimestampFormat: timeFormat,
				DisableSorting:  true,
				DisableColors:   true,
			},
		},
	}
	return context.WithValue(context.Background(), logrusKey{}, logger)
}

func BenchmarkLogrus(b *testing.B) {
	runBenchmark(b, logrusPrint(io.Discard), logrusLog)
}

func BenchmarkLogrusf(b *testing.B) {
	runBenchmark(b, logrusPrint(io.Discard), logrusLogf)
}

func TestLogrusf(t *testing.T) {
	testBenchmark(t, logrusPrint, logrusLogf, `
time="2020/03/05 14:27:48" level=info msg="A where a=0"
time="2020/03/05 14:27:49" level=info msg="b where b=\"A value\""
time="2020/03/05 14:27:50" level=info msg="A where a=1"
time="2020/03/05 14:27:51" level=info msg="b where b=\"Some other value\""
time="2020/03/05 14:27:52" level=info msg="A where a=22"
time="2020/03/05 14:27:53" level=info msg="b where b=\"Some other value\""
time="2020/03/05 14:27:54" level=info msg="A where a=333"
time="2020/03/05 14:27:55" level=info msg="b where b=\"\""
time="2020/03/05 14:27:56" level=info msg="A where a=4444"
time="2020/03/05 14:27:57" level=info msg="b where b=\"prime count of values\""
time="2020/03/05 14:27:58" level=info msg="A where a=55555"
time="2020/03/05 14:27:59" level=info msg="b where b=\"V\""
time="2020/03/05 14:28:00" level=info msg="A where a=666666"
time="2020/03/05 14:28:01" level=info msg="b where b=\"A value\""
time="2020/03/05 14:28:02" level=info msg="A where a=7777777"
time="2020/03/05 14:28:03" level=info msg="b where b=\"A value\""
`)
}
