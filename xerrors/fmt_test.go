// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fmt_test

import (
	gofmt "fmt"
	"io"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/exp/errors"
	"golang.org/x/exp/errors/fmt"
)

func TestErrorf(t *testing.T) {
	chained := &wrapped{"chained", nil}
	chain := func(s ...string) (a []string) {
		for _, s := range s {
			a = append(a, cleanPath(s))
		}
		return a
	}
	testCases := []struct {
		got  error
		want []string
	}{{
		fmt.Errorf("no args"),
		chain("no args/path.TestErrorf/path.go:xxx"),
	}, {
		fmt.Errorf("no args: %s"),
		chain("no args: %!s(MISSING)/path.TestErrorf/path.go:xxx"),
	}, {
		fmt.Errorf("nounwrap: %s", "simple"),
		chain(`nounwrap: simple/path.TestErrorf/path.go:xxx`),
	}, {
		fmt.Errorf("nounwrap: %v", "simple"),
		chain(`nounwrap: simple/path.TestErrorf/path.go:xxx`),
	}, {
		fmt.Errorf("%s failed: %v", "foo", chained),
		chain("foo failed/path.TestErrorf/path.go:xxx",
			"chained/somefile.go:xxx"),
	}, {
		fmt.Errorf("no wrap: %s", chained),
		chain("no wrap/path.TestErrorf/path.go:xxx",
			"chained/somefile.go:xxx"),
	}, {
		fmt.Errorf("%s failed: %w", "foo", chained),
		chain("wraps:foo failed/path.TestErrorf/path.go:xxx",
			"chained/somefile.go:xxx"),
	}, {
		fmt.Errorf("nowrapv: %v", chained),
		chain("nowrapv/path.TestErrorf/path.go:xxx",
			"chained/somefile.go:xxx"),
	}, {
		fmt.Errorf("wrapw: %w", chained),
		chain("wraps:wrapw/path.TestErrorf/path.go:xxx",
			"chained/somefile.go:xxx"),
	}, {
		fmt.Errorf("not wrapped: %+v", chained),
		chain("not wrapped: chained: somefile.go:123/path.TestErrorf/path.go:xxx"),
	}}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i)+"/"+path.Join(tc.want...), func(t *testing.T) {
			got := errToParts(tc.got)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Format:\n got: %#v\nwant: %#v", got, tc.want)
			}

			gotStr := tc.got.Error()
			wantStr := fmt.Sprint(tc.got)
			if gotStr != wantStr {
				t.Errorf("Error:\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestErrorFormatter(t *testing.T) {
	var (
		simple   = &wrapped{"simple", nil}
		elephant = &wrapped{
			"can't adumbrate elephant",
			detailed{},
		}
		nonascii = &wrapped{"café", nil}
		newline  = &wrapped{"msg with\nnewline",
			&wrapped{"and another\none", nil}}
		fallback  = &wrapped{"fallback", os.ErrNotExist}
		oldAndNew = &wrapped{"new style", formatError("old style")}
		framed    = &withFrameAndMore{
			frame: errors.Caller(0),
		}
		opaque = &wrapped{"outer",
			errors.Opaque(&wrapped{"mid",
				&wrapped{"inner", nil}})}
	)
	testCases := []struct {
		err    error
		fmt    string
		want   string
		regexp bool
	}{{
		err:  simple,
		fmt:  "%s",
		want: "simple",
	}, {
		err:  elephant,
		fmt:  "%s",
		want: "can't adumbrate elephant: out of peanuts",
	}, {
		err:  &wrapped{"a", &wrapped{"b", &wrapped{"c", nil}}},
		fmt:  "%s",
		want: "a: b: c",
	}, {
		err: simple,
		fmt: "%+v",
		want: "simple:" +
			"\n    somefile.go:123",
	}, {
		err: elephant,
		fmt: "%+v",
		want: "can't adumbrate elephant:" +
			"\n    somefile.go:123" +
			"\n  - out of peanuts:" +
			"\n    the elephant is on strike" +
			"\n    and the 12 monkeys" +
			"\n    are laughing",
	}, {
		err:  simple,
		fmt:  "%#v",
		want: "&fmt_test.wrapped{msg:\"simple\", err:error(nil)}",
	}, {
		err: framed,
		fmt: "%+v",
		want: "something:" +
			"\n    golang.org/x/exp/errors/fmt_test.TestErrorFormatter" +
			"\n        .+/golang.org/x/exp/errors/fmt/errors_test.go:98" +
			"\n    something more",
		regexp: true,
	}, {
		err:  fmtTwice("Hello World!"),
		fmt:  "%#v",
		want: "2 times Hello World!",
	}, {
		err:  fallback,
		fmt:  "%s",
		want: "fallback: file does not exist",
	}, {
		err: fallback,
		fmt: "%+v",
		// Note: no colon after the last error, as there are no details.
		want: "fallback:" +
			"\n    somefile.go:123" +
			"\n  - file does not exist",
	}, {
		err:  opaque,
		fmt:  "%s",
		want: "outer: mid: inner",
	}, {
		err: opaque,
		fmt: "%+v",
		want: "outer:" +
			"\n    somefile.go:123" +
			"\n  - mid:" +
			"\n    somefile.go:123" +
			"\n  - inner:" +
			"\n    somefile.go:123",
	}, {
		err:  oldAndNew,
		fmt:  "%v",
		want: "new style: old style",
	}, {
		err:  oldAndNew,
		fmt:  "%q",
		want: `"new style: old style"`,
	}, {
		err: oldAndNew,
		fmt: "%+v",
		// Note the extra indentation.
		// Colon for old style error is rendered by the fmt.Formatter
		// implementation of the old-style error.
		want: "new style:" +
			"\n    somefile.go:123" +
			"\n  - old style:" +
			"\n    otherfile.go:456",
	}, {
		err:  simple,
		fmt:  "%-12s",
		want: "simple      ",
	}, {
		// Don't use formatting flags for detailed view.
		err: simple,
		fmt: "%+12v",
		want: "simple:" +
			"\n    somefile.go:123",
	}, {
		err:  elephant,
		fmt:  "%+50s",
		want: "          can't adumbrate elephant: out of peanuts",
	}, {
		err:  nonascii,
		fmt:  "%q",
		want: `"café"`,
	}, {
		err:  nonascii,
		fmt:  "%+q",
		want: `"caf\u00e9"`,
	}, {
		err:  simple,
		fmt:  "% x",
		want: "73 69 6d 70 6c 65",
	}, {
		err: newline,
		fmt: "%s",
		want: "msg with" +
			"\nnewline: and another" +
			"\none",
	}, {
		err: newline,
		fmt: "%+v",
		want: "msg with" +
			"\n    newline:" +
			"\n    somefile.go:123" +
			"\n  - and another" +
			"\n    one:" +
			"\n    somefile.go:123",
	}, {
		err: wrapped{"", wrapped{"inner message", nil}},
		fmt: "%+v",
		want: "somefile.go:123" +
			"\n  - inner message:" +
			"\n    somefile.go:123",
	}, {
		err:  spurious(""),
		fmt:  "%s",
		want: "spurious",
	}, {
		err:  spurious(""),
		fmt:  "%+v",
		want: "spurious",
	}, {
		err:  spurious("extra"),
		fmt:  "%s",
		want: "spurious",
	}, {
		err: spurious("extra"),
		fmt: "%+v",
		want: "spurious:\n" +
			"    extra",
	}, {
		err:  nil,
		fmt:  "%+v",
		want: "<nil>",
	}, {
		err:  (*wrapped)(nil),
		fmt:  "%+v",
		want: "<nil>",
	}, {
		err:  simple,
		fmt:  "%T",
		want: "*fmt_test.wrapped",
	}, {
		err:  simple,
		fmt:  "%🤪",
		want: "%!🤪(*fmt_test.wrapped=&{simple <nil>})",
	}, {
		err:  formatError("use fmt.Formatter"),
		fmt:  "%#v",
		want: "use fmt.Formatter",
	}, {
		err: wrapped{"using errors.Formatter",
			formatError("use fmt.Formatter")},
		fmt:  "%#v",
		want: "fmt_test.wrapped{msg:\"using errors.Formatter\", err:\"use fmt.Formatter\"}",
	}, {
		err:  fmtTwice("%s %s", "ok", panicValue{}),
		fmt:  "%s",
		want: "ok %!s(PANIC=panic)/ok %!s(PANIC=panic)",
	}, {
		err:  fmtTwice("%o %s", panicValue{}, "ok"),
		fmt:  "%s",
		want: "{} ok/{} ok",
	}, {
		err: adapted{"adapted", nil},
		fmt: "%+v",
		want: "adapted:" +
			"\n    detail",
	}, {
		err: adapted{"outer", adapted{"mid", adapted{"inner", nil}}},
		fmt: "%+v",
		want: "outer:" +
			"\n    detail" +
			"\n  - mid:" +
			"\n    detail" +
			"\n  - inner:" +
			"\n    detail",
	}}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d/%s", i, tc.fmt), func(t *testing.T) {
			got := fmt.Sprintf(tc.fmt, tc.err)
			var ok bool
			if tc.regexp {
				var err error
				ok, err = regexp.MatchString(tc.want, got)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				ok = got == tc.want
			}
			if !ok {
				t.Errorf("\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestAdaptor(t *testing.T) {
	testCases := []struct {
		err    error
		fmt    string
		want   string
		regexp bool
	}{{
		err: adapted{"adapted", nil},
		fmt: "%+v",
		want: "adapted:" +
			"\n    detail",
	}, {
		err: adapted{"outer", adapted{"mid", adapted{"inner", nil}}},
		fmt: "%+v",
		want: "outer:" +
			"\n    detail" +
			"\n  - mid:" +
			"\n    detail" +
			"\n  - inner:" +
			"\n    detail",
	}}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d/%s", i, tc.fmt), func(t *testing.T) {
			got := gofmt.Sprintf(tc.fmt, tc.err)
			if got != tc.want {
				t.Errorf("\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

var _ errors.Formatter = wrapped{}

type wrapped struct {
	msg string
	err error
}

func (e wrapped) Error() string { return "should call Format" }

func (e wrapped) FormatError(p errors.Printer) (next error) {
	p.Print(e.msg)
	p.Detail()
	p.Print("somefile.go:123")
	return e.err
}

var _ errors.Formatter = detailed{}

type detailed struct{}

func (e detailed) Error() string { return fmt.Sprint(e) }

func (detailed) FormatError(p errors.Printer) (next error) {
	p.Printf("out of %s", "peanuts")
	p.Detail()
	p.Print("the elephant is on strike\n")
	p.Printf("and the %d monkeys\nare laughing", 12)
	return nil
}

type withFrameAndMore struct {
	frame errors.Frame
}

func (e *withFrameAndMore) Error() string { return fmt.Sprint(e) }

func (e *withFrameAndMore) FormatError(p errors.Printer) (next error) {
	p.Print("something")
	if p.Detail() {
		e.frame.Format(p)
		p.Print("something more")
	}
	return nil
}

type spurious string

func (e spurious) Error() string { return fmt.Sprint(e) }

func (e spurious) Format(fmt.State, rune) {
	panic("should never be called by one of the tests")
}

func (e spurious) FormatError(p errors.Printer) (next error) {
	p.Print("spurious")
	p.Detail() // Call detail even if we don't print anything
	if e == "" {
		p.Print()
	} else {
		p.Print("\n", string(e)) // print extraneous leading newline
	}
	return nil
}

type adapted struct {
	msg string
	err error
}

func (e adapted) Error() string { return string(e.msg) }

func (e adapted) Format(s fmt.State, verb rune) {
	fmt.FormatError(s, verb, e)
}

func (e adapted) FormatError(p errors.Printer) error {
	p.Print(e.msg)
	p.Detail()
	p.Print("detail")
	return e.err
}

// formatError is an error implementing Format instead of errors.Formatter.
// The implementation mimics the implementation of github.com/pkg/errors.
type formatError string

func (e formatError) Error() string { return string(e) }

func (e formatError) Format(s fmt.State, verb rune) {
	// Body based on pkg/errors/errors.go
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, string(e))
			fmt.Fprintf(s, ":\n%s", "otherfile.go:456")
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, string(e))
	case 'q':
		fmt.Fprintf(s, "%q", string(e))
	}
}

func (e formatError) GoString() string {
	panic("should never be called")
}

type fmtTwiceErr struct {
	format string
	args   []interface{}
}

func fmtTwice(format string, a ...interface{}) error {
	return fmtTwiceErr{format, a}
}

func (e fmtTwiceErr) Error() string { return fmt.Sprint(e) }

func (e fmtTwiceErr) FormatError(p errors.Printer) (next error) {
	p.Printf(e.format, e.args...)
	p.Print("/")
	p.Printf(e.format, e.args...)
	return nil
}

func (e fmtTwiceErr) GoString() string {
	return "2 times " + fmt.Sprintf(e.format, e.args...)
}

type panicValue struct{}

func (panicValue) String() string { panic("panic") }

var rePath = regexp.MustCompile(`( [^ ]*)fmt.*test\.`)
var reLine = regexp.MustCompile(":[0-9]*\n?$")

func cleanPath(s string) string {
	s = rePath.ReplaceAllString(s, "/path.")
	s = reLine.ReplaceAllString(s, ":xxx")
	s = strings.Replace(s, "\n   ", "", -1)
	s = strings.Replace(s, " /", "/", -1)
	return s
}

func errToParts(err error) (a []string) {
	for err != nil {
		var p testPrinter
		if errors.Unwrap(err) != nil {
			p.str += "wraps:"
		}
		f, ok := err.(errors.Formatter)
		if !ok {
			a = append(a, err.Error())
			break
		}
		err = f.FormatError(&p)
		a = append(a, cleanPath(p.str))
	}
	return a

}

type testPrinter struct {
	str string
}

func (p *testPrinter) Print(a ...interface{}) {
	p.str += fmt.Sprint(a...)
}

func (p *testPrinter) Printf(format string, a ...interface{}) {
	p.str += fmt.Sprintf(format, a...)
}

func (p *testPrinter) Detail() bool {
	p.str += " /"
	return true
}
