package ranges

import (
	"testing"
)

func TestRangeList(t *testing.T) {
	wants := []struct {
		Start, Stop, Step float64
	}{
		{0, 10, 1},
		{0, -10, -1},
		{-5, -125, -5},
		{0, 0, 1},
		{-1.5, -25.5, -1.5},
		{1, 1, 0},
	}

	for _, want := range wants {
		t.Log(RangeList(want.Start, want.Stop, want.Step))
	}
}

func BenchmarkRangeList(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RangeList(0, i, 1)
	}
}
