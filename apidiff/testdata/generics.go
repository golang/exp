package p

//// Generics

// old
type G[T any] []T

// new
// OK: param name change
type G[A any] []A

// old
type GM[A, B comparable] map[A]B

// new
// i GM: changed from map[A]B to map[B]A
type GM[A, B comparable] map[B]A

// old
type GT[V any] struct {
}

func (GT[V]) M(*GT[V]) {}

// new
// OK
type GT[V any] struct {
}

func (GT[V]) M(*GT[V]) {}

// old
type GT2[V any] struct {
}

func (GT2[V]) M(*GT2[V]) {}

// new
// i GT2: changed from GT2[V any] to GT2[V comparable]
type GT2[V comparable] struct {
}

func (GT2[V]) M(*GT2[V]) {}

// both
type custom interface {
	int
}

type GT3[E custom] map[E]int
