// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package spi allows users to read from and write to an SPI device.
package spi // import "golang.org/x/exp/io/spi"

import (
	"time"

	"golang.org/x/exp/io/spi/driver"
)

// Mode represents the SPI mode number where clock parity (CPOL)
// is the high order and clock edge (CPHA) is the low order bit.
type Mode int

const (
	Mode0 = Mode(0)
	Mode1 = Mode(1)
	Mode2 = Mode(2)
	Mode3 = Mode(3)
)

type Device struct {
	conn driver.Conn
}

// SetMode sets the SPI mode. SPI mode is a combination of polarity and phases.
// CPOL is the high order bit, CPHA is the low order. Pre-computed mode
// values are Mode0, Mode1, Mode2 and Mode3.
// The value can be changed by SPI device's driver.
func (d *Device) SetMode(mode Mode) error {
	return d.conn.Configure(int(mode), -1, -1)
}

// SetMaxSpeed sets the maximum clock speed in Hz.
// The value can be overriden by SPI device's driver.
func (d *Device) SetMaxSpeed(speedHz int) error {
	return d.conn.Configure(-1, -1, speedHz)
}

// SetBitsPerWord sets how many bits it takes to represent a word.
// e.g. 8 represents 8-bit words.
// The default is 8 bits per word if none is set.
func (d *Device) SetBitsPerWord(bits int) error {
	return d.conn.Configure(-1, bits, -1)
}

// Transfer performs a duplex transmission to write to the SPI device
// and read len(rx) bytes to rx.
// User should not mutate the tx and rx until this call returns.
func (d *Device) Transfer(tx, rx []byte, delay time.Duration) error {
	return d.conn.Transfer(tx, rx, delay)
}

// Open opens a device with the specified bus identifier and chip select
// by using the given driver name. If an empty string provided for the driver name,
// the default driver (devfs) is used.
// Mode is the SPI mode. SPI mode is a combination of polarity and phases.
// CPOL is the high order bit, CPHA is the low order. Pre-computed mode
// values are Mode0, Mode1, Mode2 and Mode3. The value of the mode argument
// can be overriden by the device's driver.
// Max clock speed is in Hz and can be overriden by the device's driver.
func Open(o driver.Opener, bus, cs int, mode Mode, maxSpeedHz int) (*Device, error) {
	if o == nil {
		o = &DevFS{}
	}

	conn, err := o.Open(bus, cs)
	if err != nil {
		return nil, err
	}

	dev := &Device{conn: conn}
	if err := dev.SetMode(mode); err != nil {
		dev.Close()
		return nil, err
	}
	if err := dev.SetMaxSpeed(maxSpeedHz); err != nil {
		dev.Close()
		return nil, err
	}
	return dev, nil
}

// Close closes the SPI device and releases the related resources.
func (d *Device) Close() error {
	return d.conn.Close()
}
