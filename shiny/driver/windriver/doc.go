// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package windriver provides the Windows driver for accessing a screen.
package windriver

/*
Implementation Details

On Windows, UI can run on any thread, but any windows created
on a thread must only be manipulated from that thread. You can send
"window messages" to any window; when you send a window
message to a window owned by another thread, Windows will
temporarily switch to that thread to dispatch the message. As such,
windows serve as the communication endpoints between threads
on Windows. In addition, each thread that hosts UI must handle
incoming window messages from the OS through a "message pump".
These messages include paint events and input events.

windriver designates the thread that calls Main as the UI thread.
It locks this thread, creates a special window to handle screen.Screen
calls, runs the function passed to Main on another goroutine, and
runs a message pump.

The window that handles screen.Screen functions is currently called
the "utility window". A better name can be chosen later. This window
handles creating screen.Windows/Buffers/Textures. As such, all shiny
Windows are owned by a single thread.

Each function in windriver, be it on screen.Screen or screen.Window,
is translated into a window message and sent to a window, namely
the utility window and the screen.Window window, respectively.
This is how windriver remains thread-safe.

(TODO(andlabs): actually move per-window messages to the window itself)

Presently, the actual Windows API work is implemented in C. This is
to encapsulate Windows's data structures, ensure properly handling
signed -> unsigned conversions in constants, handle pointer casts
cleanly, and properly handle the "last error", which I will describe
later.

Here is a demonstration of all of the above. When you call
screen.NewWindow(opts), the Go code calls the C function
createWindow, which is implemented as something similar to

	HRESULT createWindow(newWindowOpts *opts, HWND *phwnd) {
		return (HRESULT) SendMessageW(utilityWindow,
			msgCreateWindow,
			(WPARAM) opts,
			(LPARAM) phwnd);
	}

HRESULT is another type for errors in Windows; I will again describe
this later. This function tells the utility window to make a new window,
using the given options, storing the window's OS handle in phwnd, and
returning any error directly to us through SendMessageW.

This code is running on another goroutine, which will definitely be
run on another OS thread. As such, Windows will switch to the UI
thread to dispatch this new window message. The code for the
implementation of the utility window (called a "window procedure")
contains something like this:

		case msgCreateWindow:
			return utilCreateWindow((newWindowOpts *) wParam,
				(HWND *) lParam);

and the utilCreateWindow function does the actual work:

	LRESULT utilCreateWindow(newWindowOpts *opts, HWND *phwnd) {
		*phwnd = CreateWindowExW(...);
		if (*phwnd == NULL) {
			return lastErrorAsLRESULT();
		}
		return lS_OK;
	}

When this returns, Windows switches back to the previous thread,
which can now use the window handle and error value.

Older Windows API functions return a Boolean flag to indicate if they
succeeded or failed, storing the actual reason for failure in what is
called the "last error". This is NOT contractual; functions are free to
fail without setting the last error, or free to clear the last error on
success.

To simplify error reporting, we instead convert all last errors to the
newer HRESULT error code system. The rules are simple: if the
function succeeded, we return the standard success code, S_OK.
If the function failed, we get the last error. If it's zero (no error),
we return the special value E_FAIL. Otherwise, we convert the last
error to an HRESULT (this is a well-defined operation that we can
reverse later when we're ready to report the error to the user).
This is all done by the C lastErrorToHRESULT function. Error
reporting on the Go side is handled by th winerror function.

Because window messages return LRESULTs, not HRESULTs,
the lastErrorToLRESULT and lS_OK macros are provided, which
automatically insert the necessary casts. An LRESULT (which is
pointer-sized) will always be either the same size as or larger than
an HRESULT (which is strictly 32 bits wide).
*/
