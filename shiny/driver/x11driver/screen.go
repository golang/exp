// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"errors"
	"fmt"
	"image"
	"log"
	"sync"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"

	"golang.org/x/exp/shiny/screen"
)

// TODO: check that xgb is safe to use concurrently from multiple goroutines.
// For example, its Conn.WaitForEvent concept is a method, not a channel, so
// it's not obvious how to interrupt it to service a NewWindow request.

type screenImpl struct {
	xc  *xgb.Conn
	xsi *xproto.ScreenInfo

	atomWMDeleteWindow xproto.Atom
	atomWMProtocols    xproto.Atom
	atomWMTakeFocus    xproto.Atom

	mu      sync.Mutex
	windows map[xproto.Window]*windowImpl
}

func (s *screenImpl) run() {
	for {
		ev, err := s.xc.WaitForEvent()
		if err != nil {
			log.Printf("x11driver: xproto.WaitForEvent: %v", err)
			continue
		}

		var xw xproto.Window
		switch ev := ev.(type) {
		default:
			continue
		case shm.CompletionEvent:
			// TODO.
		case xproto.ClientMessageEvent:
			xw = ev.Window
		case xproto.ConfigureNotifyEvent:
			xw = ev.Window
		case xproto.ExposeEvent:
			xw = ev.Window
		case xproto.FocusInEvent:
			xw = ev.Event
		case xproto.FocusOutEvent:
			xw = ev.Event
		case xproto.KeyPressEvent:
			xw = ev.Event
		case xproto.KeyReleaseEvent:
			xw = ev.Event
		case xproto.ButtonPressEvent:
			xw = ev.Event
		case xproto.ButtonReleaseEvent:
			xw = ev.Event
		case xproto.MotionNotifyEvent:
			xw = ev.Event
		}

		s.mu.Lock()
		w := s.windows[xw]
		s.mu.Unlock()

		if w == nil {
			log.Printf("x11driver: no window found for event %T", ev)
			continue
		}
		w.xevents <- ev
	}
}

var errTODO = errors.New("TODO: write the X11 driver")

const (
	maxShmSide = 0x00007fff // 32,767 pixels.
	maxShmSize = 0x10000000 // 268,435,456 bytes.
)

func (s *screenImpl) NewBuffer(size image.Point) (b screen.Buffer, retErr error) {
	// TODO: detect if the X11 server or connection cannot support SHM pixmaps,
	// and fall back to regular pixmaps.

	w, h := int64(size.X), int64(size.Y)
	if w < 0 || maxShmSide < w || h < 0 || maxShmSide < h || maxShmSize < 4*w*h {
		return nil, fmt.Errorf("x11driver: invalid buffer size %v", size)
	}
	xs, err := shm.NewSegId(s.xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: shm.NewSegId: %v", err)
	}

	bufLen := 4 * size.X * size.Y
	shmid, addr, err := shmOpen(bufLen)
	if err != nil {
		return nil, fmt.Errorf("x11driver: shmOpen: %v", err)
	}
	defer func() {
		if retErr != nil {
			shmClose(addr)
		}
	}()
	a := (*[maxShmSize]byte)(addr)
	buf := (*a)[:bufLen:bufLen]

	// readOnly is whether the shared memory is read-only from the X11 server's
	// point of view. We need false to use SHM pixmaps.
	const readOnly = false
	shm.Attach(s.xc, xs, uint32(shmid), readOnly)

	return &bufferImpl{
		s:    s,
		addr: addr,
		buf:  buf,
		rgba: image.RGBA{
			Pix:    buf,
			Stride: 4 * size.X,
			Rect:   image.Rectangle{Max: size},
		},
		size: size,
		xs:   xs,
	}, nil
}

func (s *screenImpl) NewTexture(size image.Point) (screen.Texture, error) {
	return nil, errTODO
}

func (s *screenImpl) NewWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	// TODO: look at opts.
	const width, height = 1024, 768

	xw, err := xproto.NewWindowId(s.xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: xproto.NewWindowId failed: %v", err)
	}
	xg, err := xproto.NewGcontextId(s.xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: xproto.NewGcontextId failed: %v", err)
	}
	xp, err := render.NewPictureId(s.xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: render.NewPictureId failed: %v", err)
	}

	w := &windowImpl{
		s:       s,
		xw:      xw,
		xg:      xg,
		xp:      xp,
		xevents: make(chan xgb.Event),
	}
	go w.run()

	s.mu.Lock()
	s.windows[xw] = w
	s.mu.Unlock()

	xproto.CreateWindow(s.xc, s.xsi.RootDepth, xw, s.xsi.Root,
		0, 0, width, height, 0,
		xproto.WindowClassInputOutput, s.xsi.RootVisual,
		xproto.CwEventMask,
		[]uint32{0 |
			xproto.EventMaskKeyPress |
			xproto.EventMaskKeyRelease |
			xproto.EventMaskButtonPress |
			xproto.EventMaskButtonRelease |
			xproto.EventMaskPointerMotion |
			xproto.EventMaskExposure |
			xproto.EventMaskStructureNotify |
			xproto.EventMaskFocusChange,
		},
	)
	s.setProperty(xw, s.atomWMProtocols, s.atomWMDeleteWindow, s.atomWMTakeFocus)
	xproto.CreateGC(s.xc, xg, xproto.Drawable(xw), 0, nil)
	// TODO: determine pictformat.
	// render.CreatePicture(s.xc, xp, xproto.Drawable(xw), pictformat, 0, nil)
	xproto.MapWindow(s.xc, xw)

	return w, nil
}

func (s *screenImpl) initAtoms() (err error) {
	s.atomWMDeleteWindow, err = s.internAtom("WM_DELETE_WINDOW")
	if err != nil {
		return err
	}
	s.atomWMProtocols, err = s.internAtom("WM_PROTOCOLS")
	if err != nil {
		return err
	}
	s.atomWMTakeFocus, err = s.internAtom("WM_TAKE_FOCUS")
	if err != nil {
		return err
	}
	return nil
}

func (s *screenImpl) internAtom(name string) (xproto.Atom, error) {
	r, err := xproto.InternAtom(s.xc, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, fmt.Errorf("x11driver: xproto.InternAtom failed: %v", err)
	}
	if r == nil {
		return 0, fmt.Errorf("x11driver: xproto.InternAtom failed")
	}
	return r.Atom, nil
}

func (s *screenImpl) setProperty(xw xproto.Window, prop xproto.Atom, values ...xproto.Atom) {
	b := make([]byte, len(values)*4)
	for i, v := range values {
		b[4*i+0] = uint8(v >> 0)
		b[4*i+1] = uint8(v >> 8)
		b[4*i+2] = uint8(v >> 16)
		b[4*i+3] = uint8(v >> 24)
	}
	xproto.ChangeProperty(s.xc, xproto.PropModeReplace, xw, prop, xproto.AtomAtom, 32, uint32(len(values)), b)
}
