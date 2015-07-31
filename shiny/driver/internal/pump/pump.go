// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pump provides an infinitely buffered event channel.
package pump

// Make returns a new Pump. Call Release to stop pumping events.
func Make() Pump {
	p := Pump{
		in:      make(chan interface{}),
		out:     make(chan interface{}),
		release: make(chan struct{}),
	}
	go p.run()
	return p
}

// Pump is an event pump, such that calling Send(e) will eventually send e on
// the event channel, in order, but Send will always complete soon, even if
// nothing is receiving on the event channel. It is effectively an infinitely
// buffered channel.
//
// In particular, goroutine A calling p.Send will not deadlock even if
// goroutine B that's responsible for receiving on p.Events() is currently
// blocked trying to send to A on a separate channel.
type Pump struct {
	in      chan interface{}
	out     chan interface{}
	release chan struct{}
}

// Events returns the event channel.
func (p *Pump) Events() <-chan interface{} {
	return p.out
}

// Send sends an event on the event channel.
func (p *Pump) Send(event interface{}) {
	select {
	case p.in <- event:
	case <-p.release:
	}
}

// Release stops the event pump. Pending events may or may not be delivered on
// the event channel. Calling Release will not close the event channel.
func (p *Pump) Release() {
	close(p.release)
}

func (p *Pump) run() {
	// initialSize is the initial size of the circular buffer. It must be a
	// power of 2.
	const initialSize = 16
	i, j, buf, mask := 0, 0, make([]interface{}, initialSize), initialSize-1
	for {
		maybeOut := p.out
		if i == j {
			maybeOut = nil
		}
		select {
		case maybeOut <- buf[i&mask]:
			buf[i&mask] = nil
			i++
		case e := <-p.in:
			// Allocate a bigger buffer if necessary.
			if i+len(buf) == j {
				b := make([]interface{}, 2*len(buf))
				n := copy(b, buf[j&mask:])
				copy(b[n:], buf[:j&mask])
				i, j = 0, len(buf)
				buf, mask = b, len(b)-1
			}
			buf[j&mask] = e
			j++
		case <-p.release:
			return
		}
	}
}
