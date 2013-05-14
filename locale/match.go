// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package locale

// regionDistance computes the distance between two regions based
// on the distance in the graph of region containments as defined in CLDR.
// It iterates over increasingly inclusive sets of groups, represented as
// bit vectors, until the source bit vector has bits in common with the
// destination vector.
func regionDistance(a, b regionID) int {
	if a == b {
		return 0
	}
	p, q := regionInclusion[a], regionInclusion[b]
	if p < nRegionGroups {
		p, q = q, p
	}
	set := regionInclusionBits
	if q < nRegionGroups && set[p]&(1<<q) != 0 {
		return 1
	}
	d := 2
	for goal := set[q]; set[p]&goal == 0; p = regionInclusionNext[p] {
		d++
	}
	return d
}
