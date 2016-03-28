// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package spi allows users to read from and write to an SPI device.
package spi // import "golang.org/x/exp/io/spi"

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
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

const (
	magic = 107

	nrbits   = 8
	typebits = 8
	sizebits = 13
	dirbits  = 3

	nrshift   = 0
	typeshift = nrshift + nrbits
	sizeshift = typeshift + typebits
	dirshift  = sizeshift + sizebits

	none  = 0
	read  = 2
	write = 4
)

type Device struct {
	f           *os.File
	mode        uint8
	speedHz     uint32
	bitsPerWord uint8
}

type payload struct {
	tx          uint64
	rx          uint64
	length      uint32
	speedHz     uint32
	delay       uint16
	bitsPerWord uint8
	csChange    uint8
	txNBits     uint8
	rxNBits     uint8
	pad         uint16
}

// SetMode sets the SPI mode. SPI mode is a combination of polarity and phases.
// CPOL is the high order bit, CPHA is the low order. Pre-computed mode
// values are Mode0, Mode1, Mode2 and Mode3.
// The value can be changed by SPI device's driver.
func (d *Device) SetMode(mode Mode) error {
	m := uint8(mode)
	if err := d.ioctl(requestCode(write, magic, 1, 1), uintptr(unsafe.Pointer(&m))); err != nil {
		return fmt.Errorf("error setting mode to %v: %v", mode, err)
	}
	d.mode = m
	return nil
}

// SetMaxSpeed sets the maximum clock speed in Hz.
// The value can be overriden by SPI device's driver.
func (d *Device) SetMaxSpeed(speedHz int) error {
	s := uint32(speedHz)
	if err := d.ioctl(requestCode(write, magic, 4, 4), uintptr(unsafe.Pointer(&s))); err != nil {
		return fmt.Errorf("error setting speed to %v: %v", speedHz, err)
	}
	d.speedHz = s
	return nil
}

// SetBitsPerWord sets how many bits it takes to represent a word.
// e.g. 8 represents 8-bit words.
// The default is 8 bits per word if none is set.
func (d *Device) SetBitsPerWord(bits int) error {
	b := uint8(bits)
	if err := d.ioctl(requestCode(write, magic, 3, 1), uintptr(unsafe.Pointer(&b))); err != nil {
		return fmt.Errorf("error setting bits per word to %v: %v", bits, err)
	}
	d.bitsPerWord = b
	return nil
}

// Do performs a duplex transmission to write to the SPI device and read
// len(buf) numbers of bytes.
// It is user's responsibility to not to mutate the buffer until
// this call returns.
func (d *Device) Do(buf []byte, delay time.Duration) error {
	p := payload{
		tx:          uint64(uintptr(unsafe.Pointer(&buf[0]))),
		rx:          uint64(uintptr(unsafe.Pointer(&buf[0]))),
		length:      uint32(len(buf)),
		speedHz:     d.speedHz,
		delay:       uint16(delay.Nanoseconds() / 1000),
		bitsPerWord: d.bitsPerWord,
	}
	// TODO: Rename Do as Transfer and provide bidirectional transfer.
	return d.ioctl(msgRequestCode(1), uintptr(unsafe.Pointer(&p)))
}

// Open opens a device with the specified bus identifier and chip select
// by using the given driver. If an empty string provided for the driver,
// the default driver (devfs) is used.
// Mode is the SPI mode. SPI mode is a combination of polarity and phases.
// CPOL is the high order bit, CPHA is the low order. Pre-computed mode
// values are Mode0, Mode1, Mode2 and Mode3. The value of the mode argument
// can be overriden by the device's driver.
// Max clock speed is in Hz and can be overriden by the device's driver.
func Open(driver string, bus, cs int, mode Mode, maxSpeedHz int) (*Device, error) {
	// TODO(jbd): Don't depend on devfs. Allow multiple backends and
	// those who may depend on proprietary APIs. devfs backend
	// could be the default backend.
	n := fmt.Sprintf("/dev/spidev%d.%d", bus, cs)
	f, err := os.OpenFile(n, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	dev := &Device{f: f}
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
	return d.f.Close()
}

// requestCode returns the device specific request code for the specified direction,
// type, number and size to be used in the ioctl call.
func requestCode(dir, typ, nr, size uintptr) uintptr {
	return (dir << dirshift) | (typ << typeshift) | (nr << nrshift) | (size << sizeshift)
}

// msgRequestCode returns the device specific value for the SPI
// message payload to be used in the ioctl call.
// n represents the number of messages.
func msgRequestCode(n uint32) uintptr {
	return uintptr(0x40006B00 + (n * 0x200000))
}

// ioctl makes an IOCTL on the open device file descriptor.
func (d *Device) ioctl(a1, a2 uintptr) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL, d.f.Fd(), a1, a2,
	)
	if errno != 0 {
		return syscall.Errno(errno)
	}
	return nil
}
