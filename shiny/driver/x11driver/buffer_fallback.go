package x11driver

import (
	"image"
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"golang.org/x/exp/shiny/driver/internal/swizzle"
)

const (
	xPutImageReqSizeMax   = (1 << 16) * 4
	xPutImageReqSizeFixed = 28
	xPutImageReqDataSize  = xPutImageReqSizeMax - xPutImageReqSizeFixed
)

type bufferFailbackImpl struct {
	xc *xgb.Conn

	buf  []byte
	rgba image.RGBA
	size image.Point
}

func (b *bufferFailbackImpl) Release()                {}
func (b *bufferFailbackImpl) Size() image.Point       { return b.size }
func (b *bufferFailbackImpl) Bounds() image.Rectangle { return image.Rectangle{Max: b.size} }
func (b *bufferFailbackImpl) RGBA() *image.RGBA       { return &b.rgba }

func (b *bufferFailbackImpl) preUpload() {
	// Check that the program hasn't tried to modify the rgba field via the
	// pointer returned by the bufferImpl.RGBA method. This check doesn't catch
	// 100% of all cases; it simply tries to detect some invalid uses of a
	// screen.Buffer such as:
	//	*buffer.RGBA() = anotherImageRGBA
	if len(b.buf) != 0 && len(b.rgba.Pix) != 0 && &b.buf[0] != &b.rgba.Pix[0] {
		panic("x11driver: invalid Buffer.RGBA modification")
	}

	swizzle.BGRA(b.buf)
}

func (b *bufferFailbackImpl) upload(xd xproto.Drawable, xg xproto.Gcontext, depth uint8, dp image.Point, sr image.Rectangle) {
	originalSRMin := sr.Min
	sr = sr.Intersect(b.Bounds())
	if sr.Empty() {
		return
	}
	dp = dp.Add(sr.Min.Sub(originalSRMin))
	b.preUpload()

	err := b.putImage(xd, xg, depth, dp, sr)
	if err != nil {
		log.Printf("x11driver: xproto.PutImage: %v", err)
	}
}

// request xproto.PutImage in batches
func (b *bufferFailbackImpl) putImage(xd xproto.Drawable, xg xproto.Gcontext, depth uint8, dp image.Point, sr image.Rectangle) error {
	widthPerReq := b.size.X
	rowPerReq := xPutImageReqDataSize / (widthPerReq * 4)
	dataPerReq := rowPerReq * widthPerReq * 4
	dstX := dp.X
	dstY := dp.Y
	start := 0
	end := 0

	var heightPerReq int
	var data []byte

	for end < len(b.buf) {
		end = start + dataPerReq
		if end > len(b.buf) {
			end = len(b.buf)
		}

		data = b.buf[start:end]
		heightPerReq = len(data) / 4 / widthPerReq

		err := xproto.PutImageChecked(
			b.xc, xproto.ImageFormatZPixmap, xd, xg,
			uint16(widthPerReq), uint16(heightPerReq),
			int16(dstX), int16(dstY),
			0, depth, data).Check()
		if err != nil {
			return err
		}

		// prepare next request
		start = end
		dstY += rowPerReq
	}

	return nil
}
