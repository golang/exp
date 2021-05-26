// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package bench_test

import (
	"testing"
)

func TestLogEventf(t *testing.T) {
	testBenchmark(t, eventPrint, eventLogf, `
time=2020-03-05T14:27:48 id=1 kind=log msg="a where A=0"
time=2020-03-05T14:27:49 id=2 kind=log msg="b where B=\"A value\""
time=2020-03-05T14:27:50 id=3 kind=log msg="a where A=1"
time=2020-03-05T14:27:51 id=4 kind=log msg="b where B=\"Some other value\""
time=2020-03-05T14:27:52 id=5 kind=log msg="a where A=22"
time=2020-03-05T14:27:53 id=6 kind=log msg="b where B=\"Some other value\""
time=2020-03-05T14:27:54 id=7 kind=log msg="a where A=333"
time=2020-03-05T14:27:55 id=8 kind=log msg="b where B=\"\""
time=2020-03-05T14:27:56 id=9 kind=log msg="a where A=4444"
time=2020-03-05T14:27:57 id=10 kind=log msg="b where B=\"prime count of values\""
time=2020-03-05T14:27:58 id=11 kind=log msg="a where A=55555"
time=2020-03-05T14:27:59 id=12 kind=log msg="b where B=\"V\""
time=2020-03-05T14:28:00 id=13 kind=log msg="a where A=666666"
time=2020-03-05T14:28:01 id=14 kind=log msg="b where B=\"A value\""
time=2020-03-05T14:28:02 id=15 kind=log msg="a where A=7777777"
time=2020-03-05T14:28:03 id=16 kind=log msg="b where B=\"A value\""
`)
}

func TestLogEvent(t *testing.T) {
	testBenchmark(t, eventPrint, eventLog, `
time=2020-03-05T14:27:48 id=1 kind=log A=0 msg=a
time=2020-03-05T14:27:49 id=2 kind=log B="A value" msg=b
time=2020-03-05T14:27:50 id=3 kind=log A=1 msg=a
time=2020-03-05T14:27:51 id=4 kind=log B="Some other value" msg=b
time=2020-03-05T14:27:52 id=5 kind=log A=22 msg=a
time=2020-03-05T14:27:53 id=6 kind=log B="Some other value" msg=b
time=2020-03-05T14:27:54 id=7 kind=log A=333 msg=a
time=2020-03-05T14:27:55 id=8 kind=log B="" msg=b
time=2020-03-05T14:27:56 id=9 kind=log A=4444 msg=a
time=2020-03-05T14:27:57 id=10 kind=log B="prime count of values" msg=b
time=2020-03-05T14:27:58 id=11 kind=log A=55555 msg=a
time=2020-03-05T14:27:59 id=12 kind=log B=V msg=b
time=2020-03-05T14:28:00 id=13 kind=log A=666666 msg=a
time=2020-03-05T14:28:01 id=14 kind=log B="A value" msg=b
time=2020-03-05T14:28:02 id=15 kind=log A=7777777 msg=a
time=2020-03-05T14:28:03 id=16 kind=log B="A value" msg=b
`)
}
