// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build 386 amd64
// +build !ios

#include "_cgo_export.h"
#include <pthread.h>
#include <stdio.h>

#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>
#import <OpenGL/gl3.h>
#import <QuartzCore/CVReturn.h>
#import <QuartzCore/CVBase.h>

static CVReturn displayLinkDraw(CVDisplayLinkRef displayLink, const CVTimeStamp* now, const CVTimeStamp* outputTime, CVOptionFlags flagsIn, CVOptionFlags* flagsOut, void* displayLinkContext)
{
	drawgl((GoUintptr)displayLinkContext);
	return kCVReturnSuccess;
}

void makeCurrentContext(uintptr_t context) {
	NSOpenGLContext* ctx = (NSOpenGLContext*)context;
	[ctx makeCurrentContext];
}

uint64 threadID() {
	uint64 id;
	if (pthread_threadid_np(pthread_self(), &id)) {
		abort();
	}
	return id;
}

@interface ScreenGLView : NSOpenGLView<NSWindowDelegate>
{
	// Each window maintains its own display link, so that windows on
	// different displays honor their current display.
	//
	// TODO(crawshaw): test that display links transfer properly, if I
	// ever find a machine with more than one display.
	@public CVDisplayLinkRef displayLink;
}
@end

@implementation ScreenGLView
- (void)prepareOpenGL {
	[self setWantsBestResolutionOpenGLSurface:YES];
	GLint swapInt = 1;
	[[self openGLContext] setValues:&swapInt forParameter:NSOpenGLCPSwapInterval];

	CVDisplayLinkCreateWithActiveCGDisplays(&displayLink);
	CVDisplayLinkSetOutputCallback(displayLink, &displayLinkDraw, self);

	CGLContextObj cglContext = [[self openGLContext] CGLContextObj];
	CGLPixelFormatObj cglPixelFormat = [[self pixelFormat] CGLPixelFormatObj];
	CVDisplayLinkSetCurrentCGDisplayFromOpenGLContext(displayLink, cglContext, cglPixelFormat);

	// Using attribute arrays in OpenGL 3.3 requires the use of a VBA.
	// But VBAs don't exist in ES 2. So we bind a default one.
	GLuint vba;
	glGenVertexArrays(1, &vba);
	glBindVertexArray(vba);
}

- (void)reshape {
	[super reshape];

	// Calculate screen PPI.
	//
	// Note that the backingScaleFactor converts from logical
	// pixels to actual pixels, but both of these units vary
	// independently from real world size. E.g.
	//
	// 13" Retina Macbook Pro, 2560x1600, 227ppi, backingScaleFactor=2, scale=3.15
	// 15" Retina Macbook Pro, 2880x1800, 220ppi, backingScaleFactor=2, scale=3.06
	// 27" iMac,               2560x1440, 109ppi, backingScaleFactor=1, scale=1.51
	// 27" Retina iMac,        5120x2880, 218ppi, backingScaleFactor=2, scale=3.03
	NSScreen *screen = [NSScreen mainScreen];
	double screenPixW = [screen frame].size.width * [screen backingScaleFactor];

	CGDirectDisplayID display = (CGDirectDisplayID)[[[screen deviceDescription] valueForKey:@"NSScreenNumber"] intValue];
	CGSize screenSizeMM = CGDisplayScreenSize(display); // in millimeters
	float ppi = 25.4 * screenPixW / screenSizeMM.width;
	float pixelsPerPt = ppi/72.0;

	// The width and height reported to the geom package are the
	// bounds of the OpenGL view. Several steps are necessary.
	// First, [self bounds] gives us the number of logical pixels
	// in the view. Multiplying this by the backingScaleFactor
	// gives us the number of actual pixels.
	NSRect r = [self bounds];
	int w = r.size.width * [screen backingScaleFactor];
	int h = r.size.height * [screen backingScaleFactor];

	setGeom((GoUintptr)self, pixelsPerPt, w, h);
}

- (void)drawRect:(NSRect)theRect {
	// Called during resize. Do an extra draw if we are visible.
	// This gets rid of flicker when resizing.
	if (CVDisplayLinkIsRunning(displayLink)) {
		drawgl((GoUintptr)self);
	}
}

- (void)mouseEventNS:(NSEvent *)theEvent {
	double scale = [[NSScreen mainScreen] backingScaleFactor];
	NSPoint p = [theEvent locationInWindow];
	mouseEvent((GoUintptr)self, p.x * scale, p.y * scale, theEvent.type, theEvent.buttonNumber);
}

- (void)mouseDown:(NSEvent *)theEvent         { [self mouseEventNS:theEvent]; }
- (void)mouseUp:(NSEvent *)theEvent           { [self mouseEventNS:theEvent]; }
- (void)mouseDragged:(NSEvent *)theEvent      { [self mouseEventNS:theEvent]; }
- (void)rightMouseDown:(NSEvent *)theEvent    { [self mouseEventNS:theEvent]; }
- (void)rightMouseUp:(NSEvent *)theEvent      { [self mouseEventNS:theEvent]; }
- (void)rightMouseDragged:(NSEvent *)theEvent { [self mouseEventNS:theEvent]; }
- (void)otherMouseDown:(NSEvent *)theEvent    { [self mouseEventNS:theEvent]; }
- (void)otherMouseUp:(NSEvent *)theEvent      { [self mouseEventNS:theEvent]; }
- (void)otherMouseDragged:(NSEvent *)theEvent { [self mouseEventNS:theEvent]; }

- (void)windowDidBecomeKey:(NSNotification *)notification {
	lifecycleFocused();
}

- (void)windowDidResignKey:(NSNotification *)notification {
	if (![NSApp isHidden]) {
		lifecycleVisible();
	}
}

- (void)windowWillClose:(NSNotification *)notification {
	CVDisplayLinkStop(displayLink);
	lifecycleAlive();
	// TODO(crawshaw): remove w from theScreen.windows
}
@end

@interface WindowResponder : NSResponder
{
}
@end

@implementation WindowResponder
// TODO(crawshaw): key events
@end

@interface AppDelegate : NSObject<NSApplicationDelegate>
{
}
@end

@implementation AppDelegate
- (void)applicationDidFinishLaunching:(NSNotification *)aNotification {
	driverStarted();
	lifecycleAlive();
	[[NSRunningApplication currentApplication] activateWithOptions:(NSApplicationActivateAllWindows | NSApplicationActivateIgnoringOtherApps)];
	lifecycleVisible();
}

- (void)applicationWillTerminate:(NSNotification *)aNotification {
	lifecycleDead();
}

- (void)applicationDidHide:(NSNotification *)aNotification {
	// TODO stop all window display links
	lifecycleAlive();
}

- (void)applicationWillUnhide:(NSNotification *)notification {
	// TODO start all window display links
	lifecycleVisible();
}
@end

uintptr_t doNewWindow(int width, int height) {
	__block int w = width;
	__block int h = height;
	__block ScreenGLView* view = NULL;

	dispatch_barrier_sync(dispatch_get_main_queue(), ^{
		id menuBar = [[NSMenu new] autorelease];
		id menuItem = [[NSMenuItem new] autorelease];
		[menuBar addItem:menuItem];
		[NSApp setMainMenu:menuBar];

		id menu = [[NSMenu new] autorelease];
		NSString* name = [[NSProcessInfo processInfo] processName];

		id hideMenuItem = [[[NSMenuItem alloc] initWithTitle:@"Hide"
			action:@selector(hide:) keyEquivalent:@"h"]
			autorelease];
		[menu addItem:hideMenuItem];

		id quitMenuItem = [[[NSMenuItem alloc] initWithTitle:@"Quit"
			action:@selector(terminate:) keyEquivalent:@"q"]
			autorelease];
		[menu addItem:quitMenuItem];
		[menuItem setSubmenu:menu];

		NSRect rect = NSMakeRect(0, 0, w, h);

		// TODO: release window object when closed
		NSWindow* window = [[NSWindow alloc] initWithContentRect:rect
				styleMask:NSTitledWindowMask
				backing:NSBackingStoreBuffered
				defer:NO];
		window.styleMask |= NSResizableWindowMask;
		window.styleMask |= NSMiniaturizableWindowMask ;
		window.styleMask |= NSClosableWindowMask;
		window.title = name;
		[window cascadeTopLeftFromPoint:NSMakePoint(20,20)];

		NSOpenGLPixelFormatAttribute attr[] = {
			NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion3_2Core,
			NSOpenGLPFAColorSize,     24,
			NSOpenGLPFAAlphaSize,     8,
			NSOpenGLPFADepthSize,     16,
			NSOpenGLPFAAccelerated,
			NSOpenGLPFADoubleBuffer,
			NSOpenGLPFAAllowOfflineRenderers,
			0
		};
		id pixFormat = [[NSOpenGLPixelFormat alloc] initWithAttributes:attr];
		view = [[ScreenGLView alloc] initWithFrame:rect pixelFormat:pixFormat];
		[window setContentView:view];
		[window setDelegate:view];

		window.nextResponder = [[[WindowResponder alloc] init] autorelease];
	});

	return (uintptr_t)view;
}

uintptr_t doShowWindow(uintptr_t viewID) {
	ScreenGLView* view = (ScreenGLView*)viewID;
	__block uintptr_t ret = 0;
	dispatch_barrier_sync(dispatch_get_main_queue(), ^{
		[view.window makeKeyAndOrderFront:view.window];
		CVDisplayLinkStart(view->displayLink);
		ret = (uintptr_t)[view openGLContext];
	});
	return ret;
}

void startDriver() {
	[NSAutoreleasePool new];
	[NSApplication sharedApplication];
	[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
	AppDelegate* delegate = [[AppDelegate alloc] init];
	[NSApp setDelegate:delegate];
	[NSApp run];
}

void stopDriver() {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp terminate:nil];
	});
}
