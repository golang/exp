// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package bench_test

import (
	"testing"

	"golang.org/x/exp/event/bench"
)

func TestLogEventf(t *testing.T) {
	bench.TestBenchmark(t, eventPrint, eventLogf, `
time=2020-03-05T14:27:48 msg="a where A=0"
time=2020-03-05T14:27:49 msg="b where B=\"A value\""
time=2020-03-05T14:27:50 msg="a where A=1"
time=2020-03-05T14:27:51 msg="b where B=\"Some other value\""
time=2020-03-05T14:27:52 msg="a where A=22"
time=2020-03-05T14:27:53 msg="b where B=\"Some other value\""
time=2020-03-05T14:27:54 msg="a where A=333"
time=2020-03-05T14:27:55 msg="b where B=\"\""
time=2020-03-05T14:27:56 msg="a where A=4444"
time=2020-03-05T14:27:57 msg="b where B=\"prime count of values\""
time=2020-03-05T14:27:58 msg="a where A=55555"
time=2020-03-05T14:27:59 msg="b where B=\"V\""
time=2020-03-05T14:28:00 msg="a where A=666666"
time=2020-03-05T14:28:01 msg="b where B=\"A value\""
time=2020-03-05T14:28:02 msg="a where A=7777777"
time=2020-03-05T14:28:03 msg="b where B=\"A value\""
`)
}

func TestLogEvent(t *testing.T) {
	bench.TestBenchmark(t, eventPrint, eventLog, `
time=2020-03-05T14:27:48 A=0 msg=a
time=2020-03-05T14:27:49 B="A value" msg=b
time=2020-03-05T14:27:50 A=1 msg=a
time=2020-03-05T14:27:51 B="Some other value" msg=b
time=2020-03-05T14:27:52 A=22 msg=a
time=2020-03-05T14:27:53 B="Some other value" msg=b
time=2020-03-05T14:27:54 A=333 msg=a
time=2020-03-05T14:27:55 B="" msg=b
time=2020-03-05T14:27:56 A=4444 msg=a
time=2020-03-05T14:27:57 B="prime count of values" msg=b
time=2020-03-05T14:27:58 A=55555 msg=a
time=2020-03-05T14:27:59 B=V msg=b
time=2020-03-05T14:28:00 A=666666 msg=a
time=2020-03-05T14:28:01 B="A value" msg=b
time=2020-03-05T14:28:02 A=7777777 msg=a
time=2020-03-05T14:28:03 B="A value" msg=b
`)
}
