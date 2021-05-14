// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"golang.org/x/exp/event"
)

func Method(v string) event.Label       { return event.String("method", v) }
func RPCID(v string) event.Label        { return event.String("id", v) }
func RPCDirection(v string) event.Label { return event.String("direction", v) }
func StatusCode(v string) event.Label   { return event.String("status.code", v) }

var (
	Started       = event.NewCounter("started", &event.MetricOptions{Description: "Count of started RPCs."})
	Finished      = event.NewCounter("finished", &event.MetricOptions{Description: "Count of finished RPCs (includes error)."})
	ReceivedBytes = event.NewIntDistribution("received_bytes", &event.MetricOptions{
		Description: "Bytes received.",
		Unit:        event.UnitBytes,
	})
	SentBytes = event.NewIntDistribution("sent_bytes", &event.MetricOptions{
		Description: "Bytes sent.",
		Unit:        event.UnitBytes,
	})
	Latency = event.NewDuration("latency", &event.MetricOptions{
		Description: "Elapsed time of an RPC.",
		Unit:        event.UnitMilliseconds,
	})
)

const (
	Inbound  = "in"
	Outbound = "out"
)
