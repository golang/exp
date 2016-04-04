// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package driver contains interfaces to be implemented by various SPI implementations.
package driver // import "golang.org/x/exp/io/spi/driver"

const (
	Mode = iota
	Bits
	Speed
	Order
	Delay
)

// Opener is an interface to be implemented by the SPI driver to open
// a connection an SPI device with the specified bus and chip number.
type Opener interface {
	Open(bus, chip int) (Conn, error)
}

// Conn is a connection to an SPI device.
// TODO(jbd): Extend the interface to query configuration values.
type Conn interface {
	// Configure configures the SPI device. Available keys are Mode (as the SPI mode),
	// Bits (as bits per word), Speed (as max clock speed in Hz), Order
	// (as bit order to be used in transfers) and Delay (in usecs).
	//
	// Some SPI devices require a minimum amount of wait time after
	// each frame write. If set, Delay amount of usecs are inserted after
	// each write.
	//
	// SPI devices can override these values.
	Configure(k, v int) error

	// Transfer transfers tx and reads into rx.
	Transfer(tx, rx []byte) error

	// Close frees the underlying resources and closes the connection.
	Close() error
}
