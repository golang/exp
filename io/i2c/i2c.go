// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package i2c allows users to read from an write to a slave I2C device.
package i2c // import "golang.org/x/exp/io/i2c"

import (
	"golang.org/x/exp/io/i2c/driver"
)

// Device represents an I2C device. Devices must be closed once
// they are no longer in use.
type Device struct {
	conn driver.Conn
}

// TOOD(jbd): Do we need higher level I2C packet writers and readers?
// TODO(jbd): Support bidirectional communication.
// TODO(jbd): Investigate if command-less read/write is valid.
//            Tweak interfaces not to require the cmd arg if so.
// TODO(jbd): How do we support 10-bit addresses and how to enable 10-bit on devfs?

// Read reads at most len(buf) number of bytes from the device for the given command.
func (d *Device) Read(cmd byte, buf []byte) error {
	return d.conn.Read(cmd, buf)
}

// Write writes the buffer for the given command to the device.
func (d *Device) Write(cmd byte, buf []byte) (err error) {
	return d.conn.Write(cmd, buf)
}

// Close closes the device and releases the underlying sources.
// All devices must be closed once they are no longer in use.
func (d *Device) Close() error {
	return d.conn.Close()
}

// Open opens an I2C device with the given I2C address on the specified bus.
func Open(o driver.Opener, bus, addr int) (*Device, error) {
	if o == nil {
		o = &Devfs{}
	}
	conn, err := o.Open(bus, addr)
	if err != nil {
		return nil, err
	}
	return &Device{conn: conn}, nil
}
