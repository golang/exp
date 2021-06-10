// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
)

var (
	Method       = keys.String("method")
	RPCID        = keys.String("id")
	RPCDirection = keys.String("direction")
	StatusCode   = keys.String("status.code")
)

var (
	Started       = event.NewCounter("started", "Count of started RPCs.")
	Finished      = event.NewCounter("finished", "Count of finished RPCs (includes error).")
	ReceivedBytes = event.NewIntDistribution("received_bytes", "Bytes received.") //, unit.Bytes)
	SentBytes     = event.NewIntDistribution("sent_bytes", "Bytes sent.")         //, unit.Bytes)
	Latency       = event.NewDuration("latency", "Elapsed time of an RPC.")       //, unit.Milliseconds)
)

const (
	Inbound  = "in"
	Outbound = "out"
)
