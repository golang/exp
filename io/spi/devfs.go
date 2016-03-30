// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spi

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/exp/io/spi/driver"
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

type payload struct {
	tx       uint64
	rx       uint64
	length   uint32
	speed    uint32
	delay    uint16
	bits     uint8
	csChange uint8
	txNBits  uint8
	rxNBits  uint8
	pad      uint16
}

// DevFS is an SPI driver that works against the devfs.
type DevFS struct{}

// Open opens /dev/spidev<bus>.<chip> and returns a connection.
func (d *DevFS) Open(bus, chip int) (driver.Conn, error) {
	n := fmt.Sprintf("/dev/spidev%d.%d", bus, chip)
	f, err := os.OpenFile(n, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &devfsConn{f: f}, nil
}

type devfsConn struct {
	f     *os.File
	mode  uint8
	speed uint32
	bits  uint8
}

func (c *devfsConn) Configure(mode, bits, speed int) error {
	if mode > -1 {
		m := uint8(mode)
		if err := c.ioctl(requestCode(write, magic, 1, 1), uintptr(unsafe.Pointer(&m))); err != nil {
			return fmt.Errorf("error setting mode to %v: %v", mode, err)
		}
		c.mode = m
	}
	if bits > -1 {
		b := uint8(bits)
		if err := c.ioctl(requestCode(write, magic, 3, 1), uintptr(unsafe.Pointer(&b))); err != nil {
			return fmt.Errorf("error setting bits per word to %v: %v", bits, err)
		}
		c.bits = b
	}
	if speed > -1 {
		s := uint32(speed)
		if err := c.ioctl(requestCode(write, magic, 4, 4), uintptr(unsafe.Pointer(&s))); err != nil {
			return fmt.Errorf("error setting speed to %v: %v", speed, err)
		}
		c.speed = s
	}
	return nil
}

func (c *devfsConn) Transfer(tx, rx []byte, delay time.Duration) error {
	p := payload{
		tx:     uint64(uintptr(unsafe.Pointer(&tx[0]))),
		rx:     uint64(uintptr(unsafe.Pointer(&rx[0]))),
		length: uint32(len(tx)),
		speed:  c.speed,
		delay:  uint16(delay.Nanoseconds() / 1000),
		bits:   c.bits,
	}
	// TODO(jbd): Read from the device and fill rx.
	return c.ioctl(msgRequestCode(1), uintptr(unsafe.Pointer(&p)))
}

func (c *devfsConn) Close() error {
	return c.f.Close()
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
func (c *devfsConn) ioctl(a1, a2 uintptr) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL, c.f.Fd(), a1, a2,
	)
	if errno != 0 {
		return syscall.Errno(errno)
	}
	return nil
}
