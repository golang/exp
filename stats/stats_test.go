// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"cmp"
	"math"
	"slices"
	"testing"
)

func TestMeanAndStdDev(t *testing.T) {
	tests := []struct {
		name         string
		data         []float64
		mean, stddev float64
	}{
		{
			"low count large positive reals",
			[]float64{1.0e10, 2.0e10, 3.0e10, 4.0e10, 5.0e10},
			3e10, 15811388300.8418960571289062,
		},
		{
			"low count large negative reals",
			[]float64{-1.0e10, -2.0e10, -3.0e10, -4.0e10, -5.0e10},
			-3e10, 15811388300.8418960571289062,
		},
		{
			"high count large positive reals",
			[]float64{1e10, 2e10, 3e10, 4e10, 5e10, 6e10, 7e10, 8e10, 9e10, 10e10, 11e10, 12e10, 13e10},
			7e10, 38944404818.4930725097656250,
		},
		{
			"high count large negative reals",
			[]float64{-1e10, -2e10, -3e10, -4e10, -5e10, -6e10, -7e10, -8e10, -9e10, -10e10, -11e10, -12e10, -13e10},
			-7e10, 38944404818.4930725097656250,
		},
		{
			"low count small positive reals",
			[]float64{0.1, 0.2, 0.3, 0.4, 0.5},
			0.3, 0.1581138830084190,
		},
		{
			"low count small negative reals",
			[]float64{-0.1, -0.2, -0.3, -0.4, -0.5},
			-0.3, 0.1581138830084190,
		},
		{
			"high count small positive reals",
			[]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.10, 0.11, 0.12, 0.13},
			0.38153846153846155, 0.2902540885111455,
		},
		{
			"high count small negative reals",
			[]float64{-0.1, -0.2, -0.3, -0.4, -0.5, -0.6, -0.7, -0.8, -0.9, -0.10, -0.11, -0.12, -0.13},
			-0.38153846153846155, 0.2902540885111455,
		},
		{
			"single value",
			[]float64{20.25},
			20.25, 0,
		},
		{
			"contains nan",
			[]float64{1.0, math.NaN()},
			math.NaN(), math.NaN(),
		},
		{
			"contains +inf and -inf",
			[]float64{math.Inf(1), 0.42, 314, math.Inf(-1)},
			math.NaN(), math.Inf(1),
		},
		{
			"contains -inf",
			[]float64{1.0, math.Inf(-1)},
			math.Inf(-1), math.Inf(1),
		},
		{
			"contains +inf",
			[]float64{1.0, math.Inf(1)},
			math.Inf(1), math.Inf(1),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preimage := slices.Clone(tc.data)
			mean, stddev := MeanAndStdDev(tc.data)
			if !slices.EqualFunc(preimage, tc.data, equateNaN) {
				t.Errorf("input slice cannot be modified\n\tgot:\t%v\n\twant:\t%v", tc.data, preimage)
			}
			if res := compareLastULP(mean, tc.mean); res != 0 {
				t.Errorf("miscalculated mean: got %.16f, want %.16f", mean, tc.mean)
			}
			if res := compareLastULP(stddev, tc.stddev); res != 0 {
				t.Errorf("miscalculated stddev: got %.16f, want %.16f", stddev, tc.stddev)
			}
		})
	}
}

func TestMedian(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want float64
	}{
		{"low count odd length positive reals", []float64{1, 5, 2, 8, 7}, 5},
		{"low count even length positive reals", []float64{1, 5, 2, 8, 7, 9}, 6},
		{"low count odd length negative reals", []float64{-1, -5, -2, -8, -7}, -5},
		{"low count even length negative reals", []float64{-1, -5, -2, -8, -7, -9}, -6},
		{"high count odd length positive reals", []float64{1, 5, 2, 8, 7, 9, 3, 4, 6, 11, 10, 12, 13}, 7},
		{"high count even length positive reals", []float64{1, 5, 2, 8, 7, 9, 3, 4, 6, 11, 10, 12}, 6.5},
		{"high count odd length negative reals", []float64{-1, -5, -2, -8, -7, -9, -3, -4, -6, -11, -10, -12, -13}, -7},
		{"high count even length negative reals", []float64{-1, -5, -2, -8, -7, -9, -3, -4, -6, -11, -10, -12}, -6.5},
		{"contains nan", []float64{1, math.NaN()}, math.NaN()},
		{"contains +inf", []float64{1, 2, math.Inf(1)}, 2},
		{"contains -inf", []float64{1, 2, math.Inf(-1)}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preimage := slices.Clone(tc.data)
			got := Median(tc.data)
			if !slices.EqualFunc(preimage, tc.data, equateNaN) {
				t.Errorf("input slice cannot be modified\n\tgot:\t%v\n\twant:\t%v", tc.data, preimage)
			}
			if res := compareLastULP(got, tc.want); res != 0 {
				t.Errorf("miscalculated median: got %.16f, want %.16f", got, tc.want)
			}
		})
	}
}

func TestQuantiles(t *testing.T) {
	tests := []struct {
		name      string
		data      []float64
		quantiles []float64
		want      []float64
	}{
		{
			"quartiles positive reals",
			[]float64{1, 2, 3, 4, 5, 6, 7, 8},
			[]float64{0.25, 0.5, 0.75}, []float64{2.75, 4.5, 6.25},
		},
		{
			"deciles positive reals",
			[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			[]float64{0.1, 0.5, 0.9}, []float64{1.9000000000000001, 5.5, 9.1},
		},
		{
			"quartiles negative reals",
			[]float64{-1, -2, -3, -4, -5, -6, -7, -8},
			[]float64{0.25, 0.5, 0.75}, []float64{-6.25, -4.5, -2.75},
		},
		{
			"deciles negative reals",
			[]float64{-1, -2, -3, -4, -5, -6, -7, -8, -9, -10},
			[]float64{0.1, 0.5, 0.9}, []float64{-9.1, -5.5, -1.9000000000000004},
		},
		{
			"small positive floats",
			[]float64{0.000001, 0.000002, 0.000003, 0.000004, 0.000005, 0.000006, 0.000007, 0.000008, 0.000009},
			[]float64{0.25, 0.5, 0.75}, []float64{0.000003, 0.000005, 0.000007},
		},
		{
			"small negative floats",
			[]float64{-0.000001, -0.000002, -0.000003, -0.000004, -0.000005, -0.000006, -0.000007, -0.000008, -0.000009},
			[]float64{0.25, 0.5, 0.75}, []float64{-0.000007, -0.000005, -0.000003},
		},
		{
			"large positive floats",
			[]float64{1e10, 2e10, 3e10, 4e10, 5e10, 6e10, 7e10, 8e10, 9e10},
			[]float64{0.25, 0.5, 0.75}, []float64{3e10, 5e10, 7e10},
		},
		{
			"large negative floats",
			[]float64{-1e10, -2e10, -3e10, -4e10, -5e10, -6e10, -7e10, -8e10, -9e10},
			[]float64{0.25, 0.5, 0.75}, []float64{-7e10, -5e10, -3e10},
		},
		{
			"contains +inf and -inf",
			[]float64{1.0, math.Inf(1), 0.0, -0.0000001, 2, math.Inf(-1)},
			[]float64{0.25, 0.5, 0.75}, []float64{-0.000000075, 0.5, 1.75},
		},
		{
			"contains nan",
			[]float64{-1, 0.000001, math.Inf(-1), 0.0, 1, math.NaN()},
			[]float64{0.5}, []float64{math.NaN()},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preimage := slices.Clone(tc.data)
			res := Quantiles(tc.data, tc.quantiles...)
			if !slices.EqualFunc(preimage, tc.data, equateNaN) {
				t.Errorf("input slice cannot be modified\n\tgot:\t%v\n\twant:\t%v", tc.data, preimage)
			}
			for i := range res {
				if compareLastULP(res[i], tc.want[i]) != 0 {
					t.Errorf("miscalculated quantile: got %.16f, want %.16f", res[i], tc.want[i])
				}
			}
		})
	}
}

// compareLastULP performs up to two comparisons
// using ±1 ulp; similarly to [cmp.Compare], it
// will return -1, 0, or 1.
func compareLastULP(x, y float64) int {
	switch cmp.Compare(x, y) {
	case -1, 1:
		return cmp.Compare(math.Nextafter(x, y), y)
	}
	return 0
}

// equateNaN is used to ensure that NaN
// values are equated in order verify that
// input slices are not modified.
func equateNaN(x, y float64) bool {
	return cmp.Compare(x, y) == 0
}
