// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build 386 amd64
// +build !ios

package gldriver

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework OpenGL
#import <Carbon/Carbon.h> // for HIToolbox/Events.h
#import <Cocoa/Cocoa.h>
#include <pthread.h>
#include <stdint.h>

void startDriver();
void stopDriver();
void makeCurrentContext(uintptr_t ctx);
uintptr_t doNewWindow(int width, int height);
uintptr_t doShowWindow(uintptr_t id);
void doCloseWindow(uintptr_t id);
uint64_t threadID();
*/
import "C"

import (
	"log"
	"runtime"

	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/geom"
	"golang.org/x/mobile/gl"
)

var initThreadID C.uint64_t

func init() {
	// Lock the goroutine responsible for initialization to an OS thread.
	// This means the goroutine running main (and calling startDriver below)
	// is locked to the OS thread that started the program. This is
	// necessary for the correct delivery of Cocoa events to the process.
	//
	// A discussion on this topic:
	// https://groups.google.com/forum/#!msg/golang-nuts/IiWZ2hUuLDA/SNKYYZBelsYJ
	runtime.LockOSThread()
	initThreadID = C.threadID()
}

func newWindow(width, height int32) uintptr {
	return uintptr(C.doNewWindow(C.int(width), C.int(height)))
}

func showWindow(w *windowImpl) {
	w.glctxMu.Lock()
	w.glctx, w.worker = gl.NewContext()
	w.glctxMu.Unlock()

	w.ctx = uintptr(C.doShowWindow(C.uintptr_t(w.id)))
}

func closeWindow(id uintptr) {
	C.doCloseWindow(C.uintptr_t(id))
}

var mainCallback func(screen.Screen)

func main(f func(screen.Screen)) error {
	if tid := C.threadID(); tid != initThreadID {
		log.Fatalf("gldriver.Main called on thread %d, but gldriver.init ran on %d", tid, initThreadID)
	}

	mainCallback = f
	C.startDriver()
	return nil
}

//export driverStarted
func driverStarted() {
	go func() {
		mainCallback(theScreen)
		C.stopDriver()
	}()
}

//export drawgl
func drawgl(id uintptr) {
	theScreen.mu.Lock()
	w := theScreen.windows[id]
	theScreen.mu.Unlock()

	if w == nil {
		return // closing window
	}
	switch w.lifecycleStage {
	case lifecycle.StageFocused, lifecycle.StageVisible:
		w.Send(paint.Event{})
		<-w.drawDone
	}
}

// drawLoop is the primary drawing loop.
//
// After Cocoa has created an NSWindow on the initial OS thread for
// processing Cocoa events in doNewWindow, it starts drawLoop on another
// goroutine. It is locked to an OS thread for its OpenGL context.
//
// The screen is drawn every time a paint.Event is received, which can be
// triggered either by the user or by Cocoa via drawgl (for example, when
// the window is resized).
func drawLoop(w *windowImpl) {
	runtime.LockOSThread()
	C.makeCurrentContext(C.uintptr_t(w.ctx))

	workAvailable := w.worker.WorkAvailable()

	// TODO(crawshaw): exit this goroutine on Release.
	for {
		select {
		case <-workAvailable:
			w.worker.DoWork()
		case <-w.publish:
		loop:
			for {
				select {
				case <-workAvailable:
					w.worker.DoWork()
				default:
					break loop
				}
			}
			C.CGLFlushDrawable(C.CGLGetCurrentContext())
			select {
			case w.drawDone <- struct{}{}:
			default:
			}
		}
	}
}

//export setGeom
func setGeom(id uintptr, ppp float32, widthPx, heightPx int) {
	theScreen.mu.Lock()
	w := theScreen.windows[id]
	theScreen.mu.Unlock()

	if w == nil {
		return // closing window
	}

	sz := size.Event{
		WidthPx:     widthPx,
		HeightPx:    heightPx,
		WidthPt:     geom.Pt(float32(widthPx) / ppp),
		HeightPt:    geom.Pt(float32(heightPx) / ppp),
		PixelsPerPt: ppp,
	}

	w.szMu.Lock()
	w.sz = sz
	w.szMu.Unlock()

	w.Send(sz)
}

//export windowClosing
func windowClosing(id uintptr) {
	sendLifecycle(id, lifecycle.StageDead)
	theScreen.mu.Lock()
	w := theScreen.windows[id]
	delete(theScreen.windows, id)
	theScreen.mu.Unlock()
	w.releaseCleanup()
}

func sendWindowEvent(id uintptr, e interface{}) {
	theScreen.mu.Lock()
	w := theScreen.windows[id]
	theScreen.mu.Unlock()

	if w == nil {
		return // closing window
	}
	w.Send(e)
}

var mods = [...]struct {
	flags uint32
	code  uint16
	mod   key.Modifiers
}{
	// Left and right variants of modifier keys have their own masks,
	// but they are not documented. These were determined empirically.
	{1<<17 | 0x102, C.kVK_Shift, key.ModShift},
	{1<<17 | 0x104, C.kVK_RightShift, key.ModShift},
	{1<<18 | 0x101, C.kVK_Control, key.ModControl},
	// TODO key.ControlRight
	{1<<19 | 0x120, C.kVK_Option, key.ModAlt},
	{1<<19 | 0x140, C.kVK_RightOption, key.ModAlt},
	{1<<20 | 0x108, C.kVK_Command, key.ModMeta},
	{1<<20 | 0x110, C.kVK_Command, key.ModMeta}, // TODO: missing kVK_RightCommand
}

func cocoaMods(flags uint32) (m key.Modifiers) {
	for _, mod := range mods {
		if flags&mod.flags == mod.flags {
			m |= mod.mod
		}
	}
	return m
}

func cocoaMouseDir(ty int32) mouse.Direction {
	switch ty {
	case C.NSLeftMouseDown, C.NSRightMouseDown, C.NSOtherMouseDown:
		return mouse.DirPress
	case C.NSLeftMouseUp, C.NSRightMouseUp, C.NSOtherMouseUp:
		return mouse.DirRelease
	default: // dragged
		return mouse.DirNone
	}
}

func cocoaMouseButton(button int32) mouse.Button {
	switch button {
	case 0:
		return mouse.ButtonLeft
	case 1:
		return mouse.ButtonRight
	case 2:
		return mouse.ButtonMiddle
	default:
		return mouse.ButtonNone
	}
}

//export mouseEvent
func mouseEvent(id uintptr, x, y float32, ty, button int32, flags uint32) {
	sendWindowEvent(id, mouse.Event{
		X:         x,
		Y:         y,
		Button:    cocoaMouseButton(button),
		Direction: cocoaMouseDir(ty),
		Modifiers: cocoaMods(flags),
	})
}

func sendLifecycle(id uintptr, to lifecycle.Stage) {
	theScreen.mu.Lock()
	w := theScreen.windows[id]
	theScreen.mu.Unlock()

	if w.lifecycleStage == to {
		return
	}
	w.Send(lifecycle.Event{
		From:        w.lifecycleStage,
		To:          to,
		DrawContext: w.glctx,
	})
	w.lifecycleStage = to
}

func sendLifecycleAll(to lifecycle.Stage) {
	theScreen.mu.Lock()
	defer theScreen.mu.Unlock()
	for _, w := range theScreen.windows {
		if w.lifecycleStage == to {
			continue
		}
		w.Send(lifecycle.Event{
			From:        w.lifecycleStage,
			To:          to,
			DrawContext: w.glctx,
		})
		w.lifecycleStage = to
	}
}

//export lifecycleDeadAll
func lifecycleDeadAll() { sendLifecycleAll(lifecycle.StageDead) }

//export lifecycleAliveAll
func lifecycleAliveAll() { sendLifecycleAll(lifecycle.StageAlive) }

//export lifecycleVisible
func lifecycleVisible(id uintptr) { sendLifecycle(id, lifecycle.StageVisible) }

//export lifecycleFocused
func lifecycleFocused(id uintptr) { sendLifecycle(id, lifecycle.StageFocused) }
