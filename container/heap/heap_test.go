package heap_test

import (
	"testing"

	"golang.org/x/exp/container/heap"
)

func TestIntHeap(t *testing.T) {
	in := []int{5, 2, -10, 6, 4, 3}
	h := heap.NewMin(in...)
	heap.Init(h)

	var n int
	n = heap.Pop(h)
	if n != -10 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 2 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 3 {
		t.Fail()
	}

	heap.Push(h, 1)

	n = heap.Pop(h)
	if n != 1 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 4 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 5 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 6 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 0 {
		t.Fail()
	}
}

func TestFloatHeap(t *testing.T) {
	in := []float64{5.1, 22.0, 10.1, 6.8, 4.9, 3.7}
	h := heap.NewMin(in...)
	heap.Init(h)

	var n float64
	n = heap.Pop(h)
	if n != 3.7 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 4.9 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 5.1 {
		t.Fail()
	}

	heap.Push(h, 11.2)

	n = heap.Pop(h)
	if n != 6.8 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 10.1 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 11.2 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 22 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 0 {
		t.Fail()
	}
}

func TestUintHeap(t *testing.T) {
	in := []uint{15, 2, 10, 6, 8, 13}
	h := heap.NewMax(in...)
	heap.Init(h)

	var n uint
	n = heap.Pop(h)
	if n != 15 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 13 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 10 {
		t.Fail()
	}

	heap.Push(h, 4)

	n = heap.Pop(h)
	if n != 8 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 6 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 4 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 2 {
		t.Fail()
	}
	n = heap.Pop(h)
	if n != 0 {
		t.Fail()
	}
}
