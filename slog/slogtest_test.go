package slog_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"golang.org/x/exp/slog"
	"golang.org/x/exp/slog/slogtest"
)

func TestSlogtest(t *testing.T) {
	for _, test := range []struct {
		name  string
		new   func(io.Writer) slog.Handler
		parse func([]byte) (map[string]any, error)
	}{
		{"JSON", func(w io.Writer) slog.Handler { return slog.NewJSONHandler(w) }, parseJSON},
		{"Text", func(w io.Writer) slog.Handler { return slog.NewTextHandler(w) }, parseText},
	} {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := test.new(&buf)
			results := func() []map[string]any {
				ms, err := parseLines(buf.Bytes(), test.parse)
				if err != nil {
					t.Fatal(err)
				}
				return ms
			}
			if err := slogtest.TestHandler(h, results); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func parseLines(bs []byte, parse func([]byte) (map[string]any, error)) ([]map[string]any, error) {
	var ms []map[string]any
	for _, line := range bytes.Split(bs, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		m, err := parse(line)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", string(line), err)
		}
		ms = append(ms, m)
	}
	return ms, nil
}

func parseJSON(bs []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(bs, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// parseText parses the output of a single call to TextHandler.Handle.
// It can parse the output of the tests in this package,
// but it doesn't handle quoted keys or values.
// It doesn't need to handle all cases, because slogtest deliberately
// uses simple inputs so handler writers can focus on testing
// handler behavior, not parsing.
func parseText(bs []byte) (map[string]any, error) {
	top := map[string]any{}
	s := string(bytes.TrimSpace(bs))
	for len(s) > 0 {
		kv, rest, _ := strings.Cut(s, " ")
		k, value, found := strings.Cut(kv, "=")
		if !found {
			return nil, fmt.Errorf("no '=' in %q", kv)
		}
		keys := strings.Split(k, ".")
		m := top
		for _, key := range keys[:len(keys)-1] {
			x, ok := m[key]
			var m2 map[string]any
			if !ok {
				m2 = map[string]any{}
				m[key] = m2
			} else {
				m2, ok = x.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("value for %q in composite key %q is not map[string]any", key, k)

				}
			}
			m = m2
		}
		m[keys[len(keys)-1]] = value
		s = rest
	}
	return top, nil
}
