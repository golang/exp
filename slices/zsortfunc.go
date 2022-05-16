// Code generated; DO NOT EDIT.

// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slices

type lessFunc[E any] func(a, b E) bool

func (lt lessFunc[E]) sortFast(list []E) {
	size := len(list)
	chance := log2Ceil(uint(size)) * 3 / 2
	if size > 50 {
		a, b, c := size/4, size/2, size*3/4
		a, ha := lt.median(list, a-1, a, a+1)
		b, hb := lt.median(list, b-1, b, b+1)
		c, hc := lt.median(list, c-1, c, c+1)
		m, hint := lt.median(list, a, b, c)
		hint &= ha & hb & hc

		pivot := list[m]
		if hint == hintRevered {
			reverse(list)
			hint = hintSorted
		}
		if hint == hintSorted {
			for i := 1; i < size; i++ {
				if lt(list[i], list[i-1]) {
					hint = 0
					break
				}
			}
			if hint == hintSorted {
				return
			}
		}

		l, r := 0, size-1
		for {
			for lt(list[l], pivot) {
				l++
			}
			for lt(pivot, list[r]) {
				r--
			}
			if l >= r {
				break
			}
			list[l], list[r] = list[r], list[l]
			l++
			r--
		}

		if l > size/2 {
			lt.introSort(list[l:], chance)
			list = list[:l]
		} else {
			lt.introSort(list[:l], chance)
			list = list[l:]
		}
	}
	lt.introSort(list, chance)
}

func (lt lessFunc[E]) median(list []E, a, b, c int) (int, uint8) {

	if lt(list[b], list[a]) {
		if lt(list[c], list[b]) {
			return b, hintRevered
		} else if lt(list[c], list[a]) {
			return c, 0
		} else {
			return a, 0
		}
	} else {
		if lt(list[c], list[a]) {
			return a, 0
		} else if lt(list[c], list[b]) {
			return c, 0
		} else {
			return b, hintSorted
		}
	}
}

func (lt lessFunc[E]) simpleSort(list []E) {
	if len(list) < 2 {
		return
	}
	for i := 1; i < len(list); i++ {
		curr := list[i]
		if lt(curr, list[0]) {
			for j := i; j > 0; j-- {
				list[j] = list[j-1]
			}
			list[0] = curr
		} else {
			pos := i
			for ; lt(curr, list[pos-1]); pos-- {
				list[pos] = list[pos-1]
			}
			list[pos] = curr
		}
	}
}

func (lt lessFunc[E]) heapSort(list []E) {
	for idx := len(list)/2 - 1; idx >= 0; idx-- {
		lt.heapDown(list, idx)
	}
	for end := len(list) - 1; end > 0; end-- {
		list[0], list[end] = list[end], list[0]
		lt.heapDown(list[:end], 0)
	}
}

func (lt lessFunc[E]) heapDown(list []E, pos int) {
	curr := list[pos]
	kid, last := pos*2+1, len(list)-1
	for kid < last {
		if lt(list[kid], list[kid+1]) {
			kid++
		}
		if !lt(curr, list[kid]) {
			break
		}
		list[pos] = list[kid]
		pos, kid = kid, kid*2+1
	}
	if kid == last && lt(curr, list[kid]) {
		list[pos], pos = list[kid], kid
	}
	list[pos] = curr
}

func (lt lessFunc[E]) sortIndex5(list []E,
	a, b, c, d, e int) (int, int, int, int, int) {
	if lt(list[b], list[a]) {
		a, b = b, a
	}
	if lt(list[d], list[c]) {
		c, d = d, c
	}
	if lt(list[c], list[a]) {
		a, c = c, a
		b, d = d, b
	}
	if lt(list[c], list[e]) {
		if lt(list[d], list[e]) {
			if lt(list[b], list[d]) {
				if lt(list[c], list[b]) {
					return a, c, b, d, e
				} else {
					return a, b, c, d, e
				}
			} else if lt(list[b], list[e]) {
				return a, c, d, b, e
			} else {
				return a, c, d, e, b
			}
		} else {
			if lt(list[b], list[e]) {
				if lt(list[c], list[b]) {
					return a, c, b, e, d
				} else {
					return a, b, c, e, d
				}
			} else if lt(list[b], list[d]) {
				return a, c, e, b, d
			} else {
				return a, c, e, d, b
			}
		}
	} else {
		if lt(list[b], list[c]) {
			if lt(list[e], list[a]) {
				return e, a, b, c, d
			} else if lt(list[e], list[b]) {
				return a, e, b, c, d
			} else {
				return a, b, e, c, d
			}
		} else {
			if lt(list[a], list[e]) {
				a, e = e, a
			}
			if lt(list[d], list[b]) {
				b, d = d, b
			}
			return e, a, c, b, d
		}
	}
}

func (lt lessFunc[E]) triPartition(list []E) (l, r int) {
	size := len(list)
	m, s := size/2, size/4

	x, l, _, r, y := lt.sortIndex5(list, m-s, m-1, m, m+1, m+s)

	s = size - 1
	pivotL, pivotR := list[l], list[r]
	list[l], list[r] = list[0], list[s]
	list[1], list[x] = list[x], list[1]
	list[s-1], list[y] = list[y], list[s-1]

	l, r = 2, s-2
	for {
		for lt(list[l], pivotL) {
			l++
		}
		for lt(pivotR, list[r]) {
			r--
		}
		if lt(pivotR, list[l]) {
			list[l], list[r] = list[r], list[l]
			r--
			if lt(list[l], pivotL) {
				l++
				continue
			}
		}
		break
	}

	for k := l + 1; k <= r; k++ {
		if lt(pivotR, list[k]) {
			for lt(pivotR, list[r]) {
				r--
			}
			if k >= r {
				break
			}
			if lt(list[r], pivotL) {
				list[l], list[k], list[r] = list[r], list[l], list[k]
				l++
			} else {
				list[k], list[r] = list[r], list[k]
			}
			r--
		} else if lt(list[k], pivotL) {
			list[k], list[l] = list[l], list[k]
			l++
		}
	}

	l--
	r++
	list[0], list[l] = list[l], pivotL
	list[s], list[r] = list[r], pivotR
	return l, r
}

func (lt lessFunc[E]) introSort(list []E, chance int) {
	for len(list) > 14 {
		if chance--; chance < 0 {
			lt.heapSort(list)
			return
		}

		l, r := lt.triPartition(list)
		lt.introSort(list[:l], chance)
		lt.introSort(list[r+1:], chance)
		if !lt(list[l], list[r]) {
			return
		}
		list = list[l+1 : r]
	}
	lt.simpleSort(list)
}

func (lt lessFunc[E]) sortStable(list []E) {
	if size := len(list); size < 16 {
		lt.simpleSort(list)
	} else {
		step := 8
		a, b := 0, step
		for b <= size {
			lt.simpleSort(list[a:b])
			a = b
			b += step
		}
		lt.simpleSort(list[a:])

		for step < size {
			a, b = 0, step*2
			for b <= size {
				lt.symmerge(list[a:b], step)
				a = b
				b += step * 2
			}
			if a+step < size {
				lt.symmerge(list[a:], step)
			}
			step *= 2
		}
	}
}

func (lt lessFunc[E]) symmerge(list []E, border int) {
	size := len(list)

	if border == 1 {
		curr := list[0]
		a, b := 1, size
		for a < b {
			m := int(uint(a+b) / 2)
			if lt(list[m], curr) {
				a = m + 1
			} else {
				b = m
			}
		}
		for i := 1; i < a; i++ {
			list[i-1] = list[i]
		}
		list[a-1] = curr
		return
	}

	if border == size-1 {
		curr := list[border]
		a, b := 0, border
		for a < b {
			m := int(uint(a+b) / 2)
			if lt(curr, list[m]) {
				b = m
			} else {
				a = m + 1
			}
		}
		for i := border; i > a; i-- {
			list[i] = list[i-1]
		}
		list[a] = curr
		return
	}

	half := size / 2
	n := border + half
	a, b := 0, border
	if border > half {
		a, b = n-size, half
	}

	p := n - 1
	for a < b {
		m := int(uint(a+b) / 2)
		if lt(list[p-m], list[m]) {
			b = m
		} else {
			a = m + 1
		}
	}
	b = n - a

	if a < border && border < b {
		rotate(list[a:b], border-a)
	}
	if 0 < a && a < half {
		lt.symmerge(list[:half], a)
	}
	if half < b && b < size {
		lt.symmerge(list[half:], b-half)
	}
}
