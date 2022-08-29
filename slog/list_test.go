package slog

import (
	"testing"

	"golang.org/x/exp/slices"
)

func TestList(t *testing.T) {
	var l list[int]
	for i := 0; i < 10; i++ {
		l = l.append(i)
	}
	l = l.normalize()
	var got, want []int
	for i := 0; i < l.len(); i++ {
		want = append(want, i)
		got = append(got, l.at(i))
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestListAlloc(t *testing.T) {
	for n := 1; n < 100; n++ {
		got := testing.AllocsPerRun(1, func() {
			var l list[int]
			for i := 0; i < n; i++ {
				l = l.append(i)
			}
		})
		want := 1.5 * float64(n)
		if got > want {
			t.Fatalf("n=%d: got %f allocations, want <= %f",
				n, got, want)
		}
	}
}
