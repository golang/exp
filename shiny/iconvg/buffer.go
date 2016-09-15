// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iconvg

import (
	"math"
)

// TODO: decoding and encoding colors, not just numbers.

// buffer holds an encoded IconVG graphic.
//
// The decodeXxx methods return the decoded value and an integer n, the number
// of bytes that value was encoded in. They return n == 0 if an error occured.
//
// The encodeXxx methods append to the buffer, modifying the slice in place.
type buffer []byte

func (b buffer) decodeNatural() (u uint32, n int) {
	if len(b) < 1 {
		return 0, 0
	}
	x := b[0]
	if x&0x01 == 0 {
		return uint32(x) >> 1, 1
	}
	if x&0x02 == 0 {
		if len(b) >= 2 {
			y := uint16(b[0]) | uint16(b[1])<<8
			return uint32(y) >> 2, 2
		}
		return 0, 0
	}
	if len(b) >= 4 {
		y := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
		return y >> 2, 4
	}
	return 0, 0
}

func (b buffer) decodeReal() (f float32, n int) {
	switch u, n := b.decodeNatural(); n {
	case 0:
		return 0, n
	case 1:
		return float32(u), n
	case 2:
		return float32(u), n
	default:
		return math.Float32frombits(u << 2), n
	}
}

func (b buffer) decodeCoordinate() (f float32, n int) {
	switch u, n := b.decodeNatural(); n {
	case 0:
		return 0, n
	case 1:
		return float32(int32(u) - 64), n
	case 2:
		return float32(int32(u)-64*128) / 64, n
	default:
		return math.Float32frombits(u << 2), n
	}
}

func (b buffer) decodeZeroToOne() (f float32, n int) {
	switch u, n := b.decodeNatural(); n {
	case 0:
		return 0, n
	case 1:
		return float32(u) / 120, n
	case 2:
		return float32(u) / 15120, n
	default:
		return math.Float32frombits(u << 2), n
	}
}

func (b *buffer) encodeNatural(u uint32) {
	if u < 1<<7 {
		u = (u << 1)
		*b = append(*b, uint8(u))
		return
	}
	if u < 1<<14 {
		u = (u << 2) | 1
		*b = append(*b, uint8(u), uint8(u>>8))
		return
	}
	u = (u << 2) | 3
	*b = append(*b, uint8(u), uint8(u>>8), uint8(u>>16), uint8(u>>24))
}

func (b *buffer) encodeReal(f float32) {
	if u := uint32(f); float32(u) == f && u < 1<<14 {
		if u < 1<<7 {
			u = (u << 1)
			*b = append(*b, uint8(u))
		} else {
			u = (u << 2) | 1
			*b = append(*b, uint8(u), uint8(u>>8))
		}
		return
	}
	b.encode4ByteReal(f)
}

func (b *buffer) encode4ByteReal(f float32) {
	u := math.Float32bits(f)

	// Round the fractional bits (the low 23 bits) to the nearest multiple of
	// 4, being careful not to overflow into the upper bits.
	v := u & 0x007fffff
	if v < 0x007fffffe {
		v += 2
	}
	u = (u & 0xff800000) | v

	// A 4 byte encoding has the low two bits set.
	u |= 0x03
	*b = append(*b, uint8(u), uint8(u>>8), uint8(u>>16), uint8(u>>24))
}

func (b *buffer) encodeCoordinate(f float32) {
	if i := int32(f); -64 <= i && i < +64 && float32(i) == f {
		u := uint32(i + 64)
		u = (u << 1)
		*b = append(*b, uint8(u))
		return
	}
	if i := int32(f * 64); -128*64 <= i && i < +128*64 && float32(i) == f*64 {
		u := uint32(i + 128*64)
		u = (u << 2) | 1
		*b = append(*b, uint8(u), uint8(u>>8))
		return
	}
	b.encode4ByteReal(f)
}

func (b *buffer) encodeZeroToOne(f float32) {
	if u := uint32(f * 15120); float32(u) == f*15120 && u < 15120 {
		if u%126 == 0 {
			u = ((u / 126) << 1)
			*b = append(*b, uint8(u))
		} else {
			u = (u << 2) | 1
			*b = append(*b, uint8(u), uint8(u>>8))
		}
		return
	}
	b.encode4ByteReal(f)
}
