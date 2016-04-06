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

// Read reads at most len(buf) number of bytes from the device. n represents
// the total number of bytes read.
func (d *Device) Read(buf []byte) (n int, err error) {
	return d.conn.Read(buf)
}

// Write writes the given byte buffer to the device. n represents the total
// number of bytes written.
func (d *Device) Write(buf []byte) (n int, err error) {
	return d.conn.Write(buf)
}

// Close closes the device and releases the underlying sources.
// All devices must be closed once they are no longer in use.
func (d *Device) Close() error {
	return d.conn.Close()
}

// Open opens an I2C device with the given I2C address on the specified bus.
func Open(o driver.Opener, bus, addr int) (*Device, error) {
	conn, err := o.Open(bus, addr)
	if err != nil {
		return nil, err
	}
	return &Device{conn: conn}, nil
}
