// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

TEXT ·bgra16(SB),NOSPLIT,$0-24
	MOVQ	p+0(FP), SI
	MOVQ	len+8(FP), DI

	// Sanity check that len is a multiple of 16.
	MOVQ	DI, AX
	ANDQ	$15, AX
	CMPQ	AX, $0
	JNE	done

	// Make the shuffle control mask (16-byte register X0) look like this,
	// where the low order byte comes first:
	//
	// 02 01 00 03  06 05 04 07  0a 09 08 0b  0e 0d 0c 0f
	//
	// Load the bottom 8 bytes into X0, the top into X1, then interleave them
	// into X0.
	MOVQ	$0x0704050603000102, AX
	MOVQ	AX, X0
	MOVQ	$0x0f0c0d0e0b08090a, AX
	MOVQ	AX, X1
	PUNPCKLQDQ	X1, X0

	ADDQ	SI, DI
loop:
	CMPQ	SI, DI
	JEQ	done

	MOVOU	(SI), X1
	PSHUFB	X0, X1
	MOVOU	X1, (SI)

	ADDQ	$16, SI
	JMP	loop
done:
	RET
