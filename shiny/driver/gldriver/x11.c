// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android

#include "_cgo_export.h"
#include <EGL/egl.h>
#include <stdio.h>
#include <stdlib.h>

EGLConfig e_config;
EGLContext e_ctx;
EGLDisplay e_dpy;
Colormap x_colormap;
Display *x_dpy;
XVisualInfo *x_visual_info;
Window x_root;

char *
eglGetErrorStr() {
	switch (eglGetError()) {
	case EGL_SUCCESS:
		return "EGL_SUCCESS";
	case EGL_NOT_INITIALIZED:
		return "EGL_NOT_INITIALIZED";
	case EGL_BAD_ACCESS:
		return "EGL_BAD_ACCESS";
	case EGL_BAD_ALLOC:
		return "EGL_BAD_ALLOC";
	case EGL_BAD_ATTRIBUTE:
		return "EGL_BAD_ATTRIBUTE";
	case EGL_BAD_CONFIG:
		return "EGL_BAD_CONFIG";
	case EGL_BAD_CONTEXT:
		return "EGL_BAD_CONTEXT";
	case EGL_BAD_CURRENT_SURFACE:
		return "EGL_BAD_CURRENT_SURFACE";
	case EGL_BAD_DISPLAY:
		return "EGL_BAD_DISPLAY";
	case EGL_BAD_MATCH:
		return "EGL_BAD_MATCH";
	case EGL_BAD_NATIVE_PIXMAP:
		return "EGL_BAD_NATIVE_PIXMAP";
	case EGL_BAD_NATIVE_WINDOW:
		return "EGL_BAD_NATIVE_WINDOW";
	case EGL_BAD_PARAMETER:
		return "EGL_BAD_PARAMETER";
	case EGL_BAD_SURFACE:
		return "EGL_BAD_SURFACE";
	case EGL_CONTEXT_LOST:
		return "EGL_CONTEXT_LOST";
	}
	return "unknown EGL error";
}

void
startDriver() {
	x_dpy = XOpenDisplay(NULL);
	if (!x_dpy) {
		fprintf(stderr, "XOpenDisplay failed\n");
		exit(1);
	}
	e_dpy = eglGetDisplay(x_dpy);
	if (!e_dpy) {
		fprintf(stderr, "eglGetDisplay failed: %s\n", eglGetErrorStr());
		exit(1);
	}
	EGLint e_major, e_minor;
	if (!eglInitialize(e_dpy, &e_major, &e_minor)) {
		fprintf(stderr, "eglInitialize failed: %s\n", eglGetErrorStr());
		exit(1);
	}
	if (!eglBindAPI(EGL_OPENGL_ES_API)) {
		fprintf(stderr, "eglBindAPI failed: %s\n", eglGetErrorStr());
		exit(1);
	}

	static const EGLint attribs[] = {
		EGL_RENDERABLE_TYPE, EGL_OPENGL_ES2_BIT,
		EGL_SURFACE_TYPE, EGL_WINDOW_BIT,
		EGL_BLUE_SIZE, 8,
		EGL_GREEN_SIZE, 8,
		EGL_RED_SIZE, 8,
		EGL_DEPTH_SIZE, 16,
		EGL_CONFIG_CAVEAT, EGL_NONE,
		EGL_NONE
	};
	EGLint num_configs;
	if (!eglChooseConfig(e_dpy, attribs, &e_config, 1, &num_configs)) {
		fprintf(stderr, "eglChooseConfig failed: %s\n", eglGetErrorStr());
		exit(1);
	}
	EGLint vid;
	if (!eglGetConfigAttrib(e_dpy, e_config, EGL_NATIVE_VISUAL_ID, &vid)) {
		fprintf(stderr, "eglGetConfigAttrib failed: %s\n", eglGetErrorStr());
		exit(1);
	}

	XVisualInfo visTemplate;
	visTemplate.visualid = vid;
	int num_visuals;
	x_visual_info = XGetVisualInfo(x_dpy, VisualIDMask, &visTemplate, &num_visuals);
	if (!x_visual_info) {
		fprintf(stderr, "XGetVisualInfo failed\n");
		exit(1);
	}

	x_root = RootWindow(x_dpy, DefaultScreen(x_dpy));
	x_colormap = XCreateColormap(x_dpy, x_root, x_visual_info->visual, AllocNone);
	if (!x_colormap) {
		fprintf(stderr, "XCreateColormap failed\n");
		exit(1);
	}

	static const EGLint ctx_attribs[] = {
		EGL_CONTEXT_CLIENT_VERSION, 2,
		EGL_NONE
	};
	e_ctx = eglCreateContext(e_dpy, e_config, EGL_NO_CONTEXT, ctx_attribs);
	if (!e_ctx) {
		fprintf(stderr, "eglCreateContext failed: %s\n", eglGetErrorStr());
		exit(1);
	}
}

void
processEvents() {
	while (XPending(x_dpy)) {
		XEvent ev;
		XNextEvent(x_dpy, &ev);
		switch (ev.type) {
		case ButtonPress:
		case ButtonRelease:
			onMouse(ev.xbutton.window, ev.xbutton.x, ev.xbutton.y, ev.xbutton.state, ev.xbutton.button,
				ev.type == ButtonPress ? 1 : 2);
			break;
		case MotionNotify:
			onMouse(ev.xmotion.window, ev.xmotion.x, ev.xmotion.y, ev.xmotion.state, 0, 0);
			break;
		case Expose:
			// A non-zero Count means that there are more expose events coming. For
			// example, a non-rectangular exposure (e.g. from a partially overlapped
			// window) will result in multiple expose events whose dirty rectangles
			// combine to define the dirty region. Go's paint events do not provide
			// dirty regions, so we only pass on the final X11 expose event.
			if (ev.xexpose.count == 0) {
				onExpose(ev.xexpose.window);
			}
			break;
		case ConfigureNotify:
			onResize(ev.xconfigure.window, ev.xconfigure.width, ev.xconfigure.height);
			break;
		}
	}
}

void
makeCurrent(uintptr_t surface) {
	EGLSurface surf = (EGLSurface)(surface);
	if (!eglMakeCurrent(e_dpy, surf, surf, e_ctx)) {
		fprintf(stderr, "eglMakeCurrent failed: %s\n", eglGetErrorStr());
		exit(1);
	}
}

void
swapBuffers(uintptr_t surface) {
	EGLSurface surf = (EGLSurface)(surface);
	if (!eglSwapBuffers(e_dpy, surf)) {
		fprintf(stderr, "eglSwapBuffers failed: %s\n", eglGetErrorStr());
		exit(1);
	}
}

uintptr_t
doNewWindow(int width, int height) {
	XSetWindowAttributes attr;
	attr.colormap = x_colormap;
	attr.event_mask =
		ButtonPressMask |
		ButtonReleaseMask |
		PointerMotionMask |
		ButtonMotionMask |
		ExposureMask |
		StructureNotifyMask;
	Window win = XCreateWindow(
		x_dpy, x_root, 0, 0, width, height, 0, x_visual_info->depth, InputOutput,
		x_visual_info->visual, CWColormap | CWEventMask, &attr);
	XSizeHints sizehints;
	sizehints.width = width;
	sizehints.height = height;
	sizehints.flags = USSize;
	XSetNormalHints(x_dpy, win, &sizehints);
	XSetStandardProperties(x_dpy, win, "App", "App", None, (char **)NULL, 0, &sizehints);
	return win;
}

uintptr_t
doShowWindow(uintptr_t id) {
	Window win = (Window)(id);
	XMapWindow(x_dpy, win);
	EGLSurface surf = eglCreateWindowSurface(e_dpy, e_config, win, NULL);
	if (!surf) {
		fprintf(stderr, "eglCreateWindowSurface failed: %s\n", eglGetErrorStr());
		exit(1);
	}
	return (uintptr_t)(surf);
}
