// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package iconvg implements a compact, binary format for simple vector graphics:
icons, logos, glyphs and emoji.

WARNING: THIS FORMAT IS EXPERIMENTAL AND SUBJECT TO INCOMPATIBLE CHANGES.

It is similar in concept to SVG (Scalable Vector Graphics) but much simpler.
Compared to "SVG Tiny", it does not have features for text, multimedia,
interactivity, linking, scripting, animation, XSLT, DOM, combination with
raster graphics such as JPEG formatted textures, etc.

It is a format for efficient presentation, not an authoring format. For
example, it does not provide grouping individual paths into higher level
objects. Instead, the anticipated workflow is that artists use other tools and
authoring formats like Inkscape and SVG, or commercial equivalents, and export
IconVG versions of their assets, the same way that they would produce PNG
versions of their vector art. It is not a goal to be able to recover the
original SVG from a derived IconVG.

It is not a pixel-exact format. Different implementations may produce slightly
different renderings, due to implementation-specific rounding errors in the
mathematical computations when rasterizing vector paths to pixels. Artifacts
may appear when scaling up to extreme sizes, say 1 million by 1 million pixels.
Nonetheless, at typical scales, e.g. up to 4096 × 4096, such differences are
not expected to be perceptible to the naked eye.


Structure

An IconVG graphic consists of a magic identifier, one or more metadata bytes
then a sequence of variable length instructions for a virtual machine.

Those instructions encode a sequence of filled paths, similar to SVG path data
(https://www.w3.org/TR/SVG/paths.html#PathData). Rendering involves switching
between two modes: a styling mode where color registers are set, and a drawing
mode where a path's geometry is defined. The virtual machine starts in the
styling mode.

In both modes, rendering proceeds by reading a one byte opcode followed by a
variable number of data bytes for that opcode. The mapping from byte values to
opcodes depends on whether the renderer is in the styling or drawing mode. A
0x07 byte value means setting the color selector register in the styling mode,
and means adding multiple lineto segments in the drawing mode.


Level of Detail

The machine state includes 2 level of detail registers, denoted LOD0 and LOD1,
both float32 values, initialized to +0 and +infinity. Drawing mode opcodes have
no effect (other than leaving drawing mode) unless the height H in pixels of
the rasterization satisfies LOD0 <= H and H < LOD1.

This allows the format to provide a simpler version for small rasterizations
(e.g. below 32 pixels) and a more complex version for large rasterizations
(e.g. 32 and above pixels).


Registers

The machine state includes 64 color registers (denoted CREG[0], CREG[1], ...,
CREG[63]) and 64 number registers (denoted NREG[0], NREG[1], ..., NREG[63]).
Register indexing is done modulo 64, so CREG[70] is the same as CREG[6], and
CREG[-1] is the same as CREG[63].

Each CREG and NREG register is 32 bits wide. The CREG registers are initialized
to the custom palette (see below); the NREG registers are initialized to 0. The
machine state also includes two selector registers, denoted CSEL and NSEL. They
are effectively 6 bit integers, as they index CREG and NREG, and are also
initialized to 0.

Color registers are four uint8 values: red, green, blue and alpha.

Number registers are float32 values.


Colors and Gradients

IconVG graphics work in 32 bit alpha-premultiplied color, with 8 bits for red,
green, blue and alpha. Alpha-premultiplication means that c00000c0 represents a
75%-opaque, fully saturated red.

It also means that some RGBA combinations (where e.g. the red value is greater
than the alpha value) are nonsensical. The virtual machine re-purposes some of
those values to represent gradients instead of flat colors. Any color register
whose alpha value is zero but whose blue value is at least 128 is a gradient.
Its remaining bits are reinterpreted such that:

The low 6 bits of the red value is the number of color/offset stops, NSTOPS.

The high 2 bits of the red value are reserved.

The low 6 bits of the green value is the color register base, CBASE.

The high 2 bits of the green value is how to spread the gradient past its
nominal bounds (from offset being 0.0 to offset being 1.0). The high two bits
being 0, 1, 2 or 3 mean none, pad, reflect and repeat respectively. None means
that offsets outside of the [0.0, 1.0] range map to transparent black. Pad
means that offsets below 0.0 and above 1.0 map to the colors that 0.0 and 1.0
would map to. Reflect means that the offset mapping is reflected start-to-end,
end-to-start, start-to-end, etc. Repeat means that the offset mapping is
repeated start-to-end, start-to-end, start-to-end, etc.

The low 6 bits of the blue value is the number register base, NBASE.

The remaining bit (the 0x40 bit) of the blue value denotes the gradient shape:
0 means a linear gradient and 1 means a radial gradient.

The gradient has NSTOPS color/offset stops. The first stop has color
CREG[CBASE+0] and offset NREG[NBASE+0], the second stop has color CREG[CBASE+1]
and offset NREG[NBASE+1], and so on.

The gradient also uses the six numbers from NREG[NBASE-6] to NREG[NBASE-1],
which form an affine transformation matrix [a, b, c; d, e, f] such that
a=NREG[NBASE-6], b=NREG[NBASE-5], c=NREG[NBASE-4], etc. This matrix maps from
graphic coordinate space (defined by the metadata's viewBox) to gradient
coordinate space. Gradient coordinate space is where a linear gradient ranges
from x=0 to x=1, and a radial gradient has center (0, 0) and radius 1.

The graphic coordinate (px, py) maps to the gradient coordinate (dx, dy) by:

	dx = a*px + b*py + c
	dy = d*px + e*py + f

The appendix below gives explicit formulae for the [a, b, c; d, e, f] affine
transformation matrix for common gradient geometry, such as a linear gradient
defined by two points.

At the time a gradient is used to fill a path, it is invalid for any of the
stop colors to itself be a gradient, or for any stop offset to be less than or
equal to a previous offset, or outside the range [0, 1].


Colors

Color register values are always 32 bits, or 4 bytes. Colors in the instruction
byte stream can be encoded more compactly, and are encoded in either 1, 2, 3 or
4 bytes, depending on context. For example, some opcodes are followed by a 1
byte color, others by a 2 byte color. There are two forms of 3 byte colors:
direct and indirect.

For a 1 byte encoding, byte values in the range [0, 125) encode the RGBA color
where the red, green and blue values come from the base-5 encoding of that byte
value such that 0, 1, 2, 3 and 4 map to 0x00, 0x40, 0x80, 0xc0, 0xff. The alpha
value is 0xff. For example, the color 40ffc0ff can be encoded as 0x30, as
decimal 48 equals 1*25 + 4*5 + 3. A byte value of 125, 126 or 127 mean the
colors c0c0c0c0, 80808080 and 00000000 respectively. Byte values in the range
[128, 192) mean a color from the custom palette (indexed by that byte value
minus 128). Byte values in the range [192, 256) mean the value of a CREG color
register (with CREG indexed by that byte value minus 192).

For a 2 byte encoding, the red, green, blue and alpha values are all 4 bit
values. For example, the color 338800ff can be encoded as 0x38 0x0f.

For a 3 byte direct encoding, the red, green and blue values are all 8 bit
values. The alpha value is implicitly 255. For example, the color 306607ff can
be encoded as 0x30 0x66 0x07.

For a 4 byte encoding, the red, green, blue and alpha values are all 8 bit
values. For example, the color 30660780 is simply 0x30 0x66 0x07 0x80.

For a 3 byte indirect encoding, the first byte is an integer value in the range
[0, 255] (denoted T) and the second and third bytes are each a 1 byte encoded
color (denoted C0 and C1). The resultant color's red channel value is given by:
	RESULTANT.RED = (((255-T) * C0.RED) + (T * C1.RED) + 128) / 255
rounded down to an integer value (the mathematical floor function), and
likewise for the green, blue and alpha channels. For example, if the custom
palette's third entry is a fully opaque orange, then 0x40 0x7f 0x82 encodes a
25% percent opaque orange: a blend of 75% times fully transparent (1 byte color
0x7f) and 25% times a fully opaque orange (1 byte color 0x82).

It is valid for some encodings to yield a color value where the red, green or
blue value is greater than the alpha value, as this may be a gradient. If it
isn't a gradient, the subsequent rendering is undefined.


Palettes

Rendering an IconVG graphic can be varied by a 64 color palette. For example,
an emoji graphic may be rendered with palette color 0 for skin and palette
color 1 for hair. Decorative variations, such as different clothing, can be
implemented by palette colors possibly set to completely transparent to switch
paths off.

Rendering software should allow users to pass an optional 64 color palette. If
one isn't provided, the suggested palette (either given in the metadata or the
default consisting entirely of opaque black) should be used. Whichever palette
ends up being used is designated the custom palette.

Some user-given colors may be nonsensical as alpha-premultiplied colors, where
e.g. the red value is greater than the alpha value. Such colors are replaced by
opaque black and not re-interpreted as gradients.

Assigning names such as "skin", "hair" or "bow_tie" to the integer indexes of
that 64 color palette is out of scope of the IconVG format per se.


Numbers

Like colors, numbers are encoded in the instruction stream in either 1, 2 or 4
bytes. Unlike colors, the encoding length is not determined by context.
Instead, the low two bits of the first byte indicate the encoding length.

If the least significant bit (the 0x01 bit) of the first byte is 0, the number
is encoded in 1 byte. Otherwise, it is encoded in 2 or 4 bytes depending on the
second least significant bit (the 0x02 bit) of the first byte being 0 or 1.


Natural Numbers

For a 1 byte encoding, the remaining 7 bits form an integer value in the range
[0, 1<<7). For example, 0x28 encodes the value 0x14 or, in decimal, 20.

For a 2 byte encoding, the remaining 14 bits, interpreted as little endian,
form an integer in the range [0, 1<<14). For example, 0x59 0x83 encodes the
value 0x20d6 or, in decimal, 8406.

For a 4 byte encoding, the remaining 30 bits, interpreted as little endian,
form an integer in the range [0, 1<<30). For example, 0x07 0x00 0x80 0x3f
encodes the value 0xfe00001 or, in decimal, 266338305.


Real Numbers

The encoding of a real number resembles the encoding of a natural number. For 1
and 2 byte encodings, the decoded real number equals the decoded natural
number. For example, 0x28 encodes the real number 20.0, just as it encodes the
natural number 20.

For a 4 byte encoding, the decoded natural number, in the range [0, 1<<30), is
shifted left by 2, to make a uint32 value that is a multiple of 4. The decoded
real number is the floating point number corresponding to the IEEE 754 binary
representation of that uint32 (i.e. a reinterpretation as a float32). For
example, 0x07 0x00 0x80 0x3f encodes the value 1.000000476837158203125.

It is valid for the 4 byte encoding to represent infinities and NaNs, but if
not loaded into LOD0 or LOD1, the subsequent rendering is undefined.


Coordinate Numbers

The encoding of a coordinate number resembles the encoding of a real number.
For 1 and 2 byte encodings, the decoded coordinate number equals (R*scale -
bias), where R is the decoded real number as above. The scale and bias depends
on the number of bytes in the encoding.

For a 1 byte encoding, the scale is 1 and the bias is 64, so that a 1 byte
coordinate ranges in [-64, +64) at integer granularity. For example, the
coordinate 7 can be encoded as 0x8e.

For a 2 byte encoding, the scale is 1/64 and the bias is 128, so that a 2 byte
coordinate ranges in [-128, +128) at 1/64 granularity. For example, the
coordinate 7.5 can be encoded as 0x81 0x87.

For a 4 byte encoding, the decoded coordinate number simply equals R. For
example, the coordinate 7.5 can also be encoded as 0x03 0x00 0xf0 0x40.


Zero-to-One Numbers

A zero-to-one number is a real number that is typically in the range [0, 1],
although it is valid for a value to be outside of that range. For example,
angles are expressed as a zero-to-one number: a fraction of a complete
revolution (360 degrees). Gradient stop offsets are another example.

The encoding of a zero-to-one number resembles the encoding of a real number.
For 1 and 2 byte encodings, the decoded number equals R*scale, where R is the
decoded real number as above. The scale depends on the number of bytes in the
encoding.

For a 1 byte encoding, the real number (ranging up to 128) is scaled by 1/120.
The denominator is 2*2*2 * 3 * 5, so that 15 degrees (2*π/24 radians) can be
encoded as 0x0a.

For a 2 byte encoding, the real number (ranging up to 16384) is scaled by
1/15120. The denominator is 2*2*2*2 * 3*3*3 * 5 * 7. For example, 40 degrees
(2*π/9 radians) can be encoded as 0x41 0x1a.

For a 4 byte encoding, the decoded zero-to-one number simply equals R. For
example, 1 degree (2*π/360 radians), or 0.002777777..., can be approximated by
the encoding 0x63 0x0b 0x36 0x3b.


Magic Identifier

An IconVG graphic starts with the four bytes 0x89 0x49 0x56 0x47 ("\x89IVG").


Metadata

The encoded metadata starts with a natural number (see encoding above) of the
number of metadata chunks in the metadata, followed by that many chunks. Each
chunk starts with the length remaining in the chunk (again, encoded as a
natural number), not including the chunk length itself. After that is a MID
(Metadata Identifier) natural number, then MID-specific data. Chunks must be
presented in increasing MID order. MIDs cannot be repeated. All MIDs are
optional.


MID 0 - ViewBox

Metadata Identifier 0 means that the MID-specific data contains four coordinate
values (see above for the coordinate encoding). These are the minX, minY, maxX,
maxY of the graphic's viewBox: its bounding rectangle in (scalable) vector
space. Note that these are abstract units, and not necessarily 1:1 with pixels.
If this MID is not present, the viewBox defaults to (-32, -32, +32, +32). A
viewBox is invalid if minX > maxX or if minY > maxY or if at least one of those
four values are a NaN or an infinity.


MID 1 - Suggested Palette

Metadata Identifier 1 means that the MID-specific data contains a suggested
palette, e.g. to provide a default rendering of variable colors such as an
emoji's skin and hair. The suggested palette is encoded in at least one byte.
The low 6 bits of that byte form a number N. The high 2 bits denote the palette
color format: those high 2 bits being 0, 1, 2 or 3 mean 1, 2, 3 (direct) or 4
byte colors (see above for the color encoding). The chunk then contains N+1
explicit colors, in that 1, 2, 3 or 4 byte encoding. A palette has exactly 64
colors, the 63-N implicit colors of the suggested palette are set to opaque
black. A 1 byte color that refers to the custom palette or a CREG color
register resolves to opaque black. If this MID is not present, the suggested
palette consists entirely of opaque black, as black is always fashionable.


Styling Opcodes

Some opcode descriptions refer to an adjustment value, ADJ. That value is the
low three bits of the opcode, nominally in the range [0, 8), although in
practice the range is [0, 7) as no ADJ-using opcode has the low three bits set.

Opcodes 0x00 to 0x3f sets CSEL to the low 6 bits of the opcode.

Opcodes 0x40 to 0x7f sets NSEL to the low 6 bits of the opcode.

Opcodes 0x80 to 0x86 sets CREG[CSEL-ADJ] to the 1 byte encoded color following
the opcode.

Opcodes 0x88 to 0x8e sets CREG[CSEL-ADJ] to the 2 byte encoded color following
the opcode.

Opcodes 0x90 to 0x96 sets CREG[CSEL-ADJ] to the 3 byte direct encoded color
following the opcode.

Opcodes 0x98 to 0x9e sets CREG[CSEL-ADJ] to the 4 byte encoded color following
the opcode.

Opcodes 0xa0 to 0xa6 sets CREG[CSEL-ADJ] to the 3 byte indirect encoded color
following the opcode.

Opcodes 0x87, 0x8f, 0x97, 0x9f and 0xa7 sets CREG[CSEL] to the 1, 2, 3
(direct), 4 and 3 (indirect) byte encoded color, following the opcode, and then
increments CSEL by 1.

Opcodes 0xa8 to 0xae sets NREG[NSEL-ADJ] to the real number following the
opcode.

Opcodes 0xb0 to 0xb6 sets NREG[NSEL-ADJ] to the coordinate number following the
opcode.

Opcodes 0xb8 to 0xbe sets NREG[NSEL-ADJ] to the zero-to-one number following
the opcode.

Opcode 0xaf, 0xb7 and 0xbf sets NREG[NSEL] to the real, coordinate and
zero-to-one number following the opcode, and then increments NSEL by 1.

Opcodes 0xc0 to 0xc6 switches to the drawing mode, and is followed by two
coordinates that is the path's starting location. In effect, there is an
implicit M (absolute moveto) op. CREG[CSEL-ADJ], either a flat color or a
gradient, will fill the path once it is complete.

Opcode 0xc7 sets the Level of Detail bounds LOD0 and LOD1 to the two real
numbers following the opcode.

All other opcodes are reserved.


Drawing Opcodes

The drawing model is based on SVG path data
(https://www.w3.org/TR/SVG/paths.html#PathData) and this description re-uses
SVG's one-letter mnemonics: M means absolute moveto, m means relative moveto, L
means absolute lineto, l means relative lineto, H means absolute horizontal
lineto, etc. Upper and lower case mean absolute and relative coordinates. The
upper case mnemonics of the SVG operations used in IconVG's drawing mode are:
M, Z, L, H, V, C, S, Q, T, A.

IconVG differs from SVG with multiple consecutive moveto ops. SVG treats all
but the first one as lineto ops. IconVG treats them all as moveto ops.

Almost all opcodes, i.e. those in the range [0x00, 0xdf], come in contiguous
groups of 16 or 32. For example, there are 16 Q (absolute quadratic Bézier
curveto) opcodes, from 0x60 to 0x6f. Those opcodes' meaning differ only in
their repeat count RC: how often that drawing operation is repeated. The lowest
valued opcode has a repeat count of 1, the next lowest has a repeat count of 2
and so on. For example, the opcode 0x68 means 9 consecutive Q drawing ops.

Opcodes 0x00 to 0x1f means RC consecutive L ops, for RC in [1, 32]. The opcode
is followed by 2*RC coordinates, RC sets of (x, y).

Opcodes 0x20 to 0x3f are like the previous paragraph, except L (absolute)
becomes l (relative).

Opcodes 0x40 to 0x4f means RC consecutive T ops, for RC in [1, 16]. The opcode
is followed by 2*RC coordinates, RC sets of (x, y).

Opcodes 0x50 to 0x5f are like the previous paragraph, except T (absolute)
becomes t (relative).

Opcodes 0x60 to 0x6f means RC consecutive Q ops, for RC in [1, 16]. The opcode
is followed by 4*RC coordinates, RC sets of (x1, y1, x, y).

Opcodes 0x70 to 0x7f are like the previous paragraph, except Q (absolute)
becomes q (relative).

Opcodes 0x80 to 0x8f means RC consecutive S ops, for RC in [1, 16]. The opcode
is followed by 4*RC coordinates, RC sets of (x2, y2, x, y).

Opcodes 0x90 to 0x9f are like the previous paragraph, except S (absolute)
becomes s (relative).

Opcodes 0xa0 to 0xaf means RC consecutive C ops, for RC in [1, 16]. The opcode
is followed by 6*RC coordinates, RC sets of (x1, y1, x2, y2, x, y).

Opcodes 0xb0 to 0xbf are like the previous paragraph, except C (absolute)
becomes c (relative).

Opcodes 0xc0 to 0xcf means RC consecutive A ops, for RC in [1, 16]. The opcode
is followed by 6*RC numbers, RC sets of (rx, ry, xAxisRotation, flags, x, y).
The rx, ry, x and y numbers are coordinates. The xAxisRotation number is an
angle (a zero-to-one number being a fraction of 360 degrees). The flags are
encoded as a natural number. The 0x01 bit of the decoded natural number is the
large-arc-flag and the 0x02 bit is the sweep-flag.

Opcodes 0xd0 to 0xdf are like the previous paragraph, except A (absolute)
becomes a (relative).

Opcode 0xe0 is reserved. (A future version of IconVG may use this opcode to
mean the same as 0xe1 without the one z op, which will matter for stroked
paths).

Opcode 0xe1 means one z op and then end the path: fill the path with the color
chosen when we switched to the drawing mode, and switch back to the styling
mode. (The Z and z ops are equivalent).

Opcode 0xe2 means one z op and then an implicit M op to the (x, y) coordinates
after the opcode.

Opcode 0xe3 means one z op and then an implicit m op to the (x, y) coordinates
after the opcode.

Opcodes 0xe4 and 0xe5 are reserved. (A future version of IconVG may use these
for M and m ops, if we allow stroked paths).

Opcode 0xe6 means one H op to the x coordinate after the opcode.

Opcode 0xe7 means one h op to the x coordinate after the opcode.

Opcode 0xe8 means one V op to the y coordinate after the opcode.

Opcode 0xe9 means one v op to the y coordinate after the opcode.

All other opcodes are reserved.

These opcode descriptions all assume that the Level of Detail constraint (see
above) is satisfied. If not, the opcodes and their variable length data are
consumed, but no further action is taken (other than leaving drawing mode).


Example

The production version of the "action/info" icon from the Material Design icon
set is defined by the following SVG, also available at
https://github.com/google/material-design-icons/blob/master/action/svg/production/ic_info_48px.svg:

	<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 48 48">
	<path d="M24 4C12.95 4 4 12.95 4 24s8.95 20 20 20 20-8.95 20-20S35.05 4 24 4z
	m2 30h-4V22h4v12zm0-16h-4v-4h4v4z"/></svg>

This is 202 bytes, or 174 bytes after "gzip --best". The PNG renderings at
various sizes:

	18x18: 207 bytes
	24x24: 222 bytes
	36x36: 321 bytes
	48x48: 412 bytes

The corresponding IconVG is 73 bytes:

	89 49 56 47 02 0a 00 50 50 b0 b0 c0 80 58 a0 cf
	cc 30 c1 58 58 cf cc 30 c1 58 80 91 37 33 0f 41
	a8 a8 a8 a8 37 33 0f c1 a8 58 80 cf cc 30 41 58
	80 58 e3 84 bc e7 78 e8 7c e7 88 e9 98 e3 80 60
	e7 78 e9 78 e7 88 e9 88 e1

The annotated version is below. Note that the IconVG viewBox ranges from -24 to
+24 while the SVG viewBox ranges from 0 to 48.

	89 49 56 47   IconVG Magic identifier
	02            Number of metadata chunks: 1
	0a            Metadata chunk length: 5
	00            Metadata Identifier: 0 (viewBox)
	50                -24
	50                -24
	b0                +24
	b0                +24
	c0            Start path, filled with CREG[CSEL-0]; M (absolute moveTo)
	80                +0
	58                -20
	a0            C (absolute cubeTo), 1 reps
	cf cc 30 c1       -11.049999
	58                -20
	58                -20
	cf cc 30 c1       -11.049999
	58                -20
	80                +0
	91            s (relative smooth cubeTo), 2 reps
	37 33 0f 41       +8.950001
	a8                +20
	a8                +20
	a8                +20
	              s (relative smooth cubeTo), implicit
	a8                +20
	37 33 0f c1       -8.950001
	a8                +20
	58                -20
	80            S (absolute smooth cubeTo), 1 reps
	cf cc 30 41       +11.049999
	58                -20
	80                +0
	58                -20
	e3            z (closePath); m (relative moveTo)
	84                +2
	bc                +30
	e7            h (relative horizontal lineTo)
	78                -4
	e8            V (absolute vertical lineTo)
	7c                -2
	e7            h (relative horizontal lineTo)
	88                +4
	e9            v (relative vertical lineTo)
	98                +12
	e3            z (closePath); m (relative moveTo)
	80                +0
	60                -16
	e7            h (relative horizontal lineTo)
	78                -4
	e9            v (relative vertical lineTo)
	78                -4
	e7            h (relative horizontal lineTo)
	88                +4
	e9            v (relative vertical lineTo)
	88                +4
	e1            z (closePath); end path

There are more examples in the ./testdata directory.


Appendix - Gradient Transformation Matrices

This appendix derives the affine transformation matrices [a, b, c; d, e, f] for
linear, circular and elliptical gradients.


Linear Gradients

For a linear gradient from (x1, y1) to (x2, y2), let dx, dy = x2-x1, y2-y1. In
gradient coordinate space, the y-coordinate is ignored, so the transformation
matrix simplifies to [a, b, c; 0, 0, 0]. It satisfies the three simultaneous
equations:

	a*(x1   ) + b*(y1   ) + c = 0   (eq L.0)
	a*(x1+dy) + b*(y1-dx) + c = 0   (eq L.1)
	a*(x1+dx) + b*(y1+dy) + c = 1   (eq L.2)

Subtracting equation L.0 from equations L.1 and L.2 yields:

	a*dy - b*dx = 0
	a*dx + b*dy = 1

So that

	a*dy*dy - b*dx*dy = 0
	a*dx*dx + b*dx*dy = dx

Overall:

	a = dx / (dx*dx + dy*dy)
	b = dy / (dx*dx + dy*dy)
	c = -a*x1 - b*y1
	d = 0
	e = 0
	f = 0


Circular Gradients

For a circular gradient with center (cx, cy) and radius vector (rx, ry), such
that (cx+rx, cy+ry) is on the circle, let

	r = math.Sqrt(rx*rx + ry*ry)

The transformation matrix maps (cx, cy) to (0, 0), maps (cx+r, cy) to (1, 0)
and maps (cx, cy+r) to (0, 1). Solving those six simultaneous equations give:

	a = +1  / r
	b = +0  / r
	c = -cx / r
	d = +0  / r
	e = +1  / r
	f = -cy / r


Elliptical Gradients

For an elliptical gradient with center (cx, cy) and axis vectors (rx, ry) and
(sx, sy), such that (cx+rx, cx+ry) and (cx+sx, cx+sy) are on the ellipse, the
transformation matrix satisfies the six simultaneous equations:

	a*(cx   ) + b*(cy   ) + c = 0   (eq E.0)
	a*(cx+rx) + b*(cy+ry) + c = 1   (eq E.1)
	a*(cx+sx) + b*(cy+sy) + c = 0   (eq E.2)
	d*(cx   ) + e*(cy   ) + f = 0   (eq E.3)
	d*(cx+rx) + e*(cy+ry) + f = 0   (eq E.4)
	d*(cx+sx) + e*(cy+sy) + f = 1   (eq E.5)

Subtracting equation E.0 from equations E.1 and E.2 yields:

	a*rx + b*ry = 1
	a*sx + b*sy = 0

Solving these two simultaneous equations yields:

	a = +sy / (rx*sy - sx*ry)
	b = -sx / (rx*sy - sx*ry)

Re-arranging E.0 yields:

	c = -a*cx - b*cy

Similarly for d, e and f so that, overall:

	a = +sy / (rx*sy - sx*ry)
	b = -sx / (rx*sy - sx*ry)
	c = -a*cx - b*cy
	d = -ry / (rx*sy - sx*ry)
	e = +rx / (rx*sy - sx*ry)
	f = -d*cx - e*cy

Note that if rx = r, ry = 0, sx = 0 and sy = r then this simplifies to the
circular gradient transformation matrix formula, above.
*/
package iconvg

// TODO: shapes (circles, rects) and strokes? Or can we assume that authoring
// tools will convert shapes and strokes to paths?

// TODO: mark somehow that a graphic (such as a back arrow) should be flipped
// horizontally or its paths otherwise varied when presented in a Right-To-Left
// context, such as among Arabic and Hebrew text? Or should that be the
// responsibility of higher layers, selecting different IconVG graphics based
// on context, the way they would select different PNG graphics.

// TODO: hinting?
