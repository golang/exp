// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package driver contains interfaces to be implemented by various SPI implementations.
package driver // import "golang.org/x/exp/io/spi/driver"

import "time"

const (
	Mode = iota
	Bits
	Speed
	Order
)

// Opener is an interface to be implemented by the SPI driver to open
// a connection an SPI device with the specified bus and chip number.
type Opener interface {
	Open(bus, chip int) (Conn, error)
}

// Conn is a connection to an SPI device.
// TODO(jbd): Expand the interface to query mode, bits per word and clock speed.
type Conn interface {
	// Configure configures the SPI device. Available keys are Mode (as the SPI mode),
	// Bits (as bits per word), Speed (as max clock speed in Hz) and Order
	// (as bit order to be used in transfers).
	//
	// SPI devices can override these values.
	//
	// If a negative value is provided, it preserves the previous state
	// of the setting, e.g. Configure(-1, -1, 10000) will only modify the
	// speed.
	Configure(k, v int) error

	// Transfer transfers tx and reads into rx.
	// Some SPI devices require a minimum amount of wait time after
	// each frame write. "delay" amount of nanoseconds are inserted after
	// each write.
	Transfer(tx, rx []byte, delay time.Duration) error

	// Close frees the underlying resources and closes the connection.
	Close() error
}
