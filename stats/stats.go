// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package stats provides basic descriptive statistics.
//
// This is intended not as a comprehensive statistics package, but
// to provide common, everyday statistical functions.
//
// As a rule of thumb, a statistical function belongs in this package
// if it would be explained in a typical high school.
//
// These functions aim to balance performance and accuracy, but some
// amount of error is inevitable in floating-point computations.
// The underlying implementations may change, resulting in small
// changes in their results from version to version. If the caller
// needs particular guarantees on accuracy and overflow behavior or
// version stability, they should use a more specialized
// implementation.
package stats

// References:
//
// Hyndman, Rob J.; Fan, Yanan (November 1996).
// "Sample Quantiles in Statistical Packages".
// American Statistician. 50 (4).
// American Statistical Association: 361–365.
// doi:10.2307/2684934. JSTOR 2684934.

import (
	"math"
	"slices"
)

// Mean returns the arithmetic mean of the values in values.
//
// Mean does not modify the array.
//
// Mean panics if values is an empty slice.
//
// If values contains NaN or both Inf and -Inf, it returns NaN.
// If values contains Inf, it returns Inf. If values contains -Inf, it returns -Inf.
func Mean(values []float64) float64 {
	mean, infs := meanInf(values)
	switch infs {
	case negInf:
		return math.Inf(-1)
	case posInf:
		return math.Inf(1)
	case negInf | posInf:
		return math.NaN()
	default: // passthrough mean or NaN
	}
	return mean
}

// MeanAndStdDev returns the arithmetic mean and
// sample standard deviation of values; the standard
// deviation is only defined for len(values) > 1.
//
// MeanAndStdDev does not modify the array.
//
// MeanAndStdDev panics if values is an empty slice.
//
// If values contains NaN, it returns NaN, NaN.
// If values contains both Inf and -Inf, it returns NaN, Inf.
// If values contains Inf, it returns Inf, Inf.
// If values contains -Inf, it returns -Inf, Inf.
func MeanAndStdDev(values []float64) (float64, float64) {
	mean, infs := meanInf(values)
	switch infs {
	case 0:
		if math.IsNaN(mean) {
			return mean, math.NaN()
		}
	case negInf, posInf, negInf | posInf:
		return mean, math.Inf(1)
	}
	if len(values) == 1 {
		return mean, 0
	}
	squaredDiffs := 0.0
	for _, v := range values {
		diff := v - mean
		squaredDiffs += diff * diff
	}
	return mean, math.Sqrt(squaredDiffs / float64(len(values)-1))
}

// meanInf calculates a naive mean value
// and reports the infinities status.
func meanInf(values []float64) (float64, infinities) {
	if len(values) == 0 {
		panic("mean: empty slice")
	}
	sum, infs := 0.0, infinities(0)
	for _, v := range values {
		switch {
		case math.IsInf(v, 1):
			infs |= posInf
		case math.IsInf(v, -1):
			infs |= negInf
		}
		sum += v
	}
	return sum / float64(len(values)), infs
}

// infinities is a bitset that records the presence of ±Inf in the input
type infinities uint8

const (
	negInf infinities = 1 << iota
	posInf
)

// Median returns the median of the values in values.
//
// Median does not modify the array.
//
// Median may perform asymptotically faster and allocate
// asymptotically less if the slice is already sorted.
//
// If values is an empty slice, it panics.
// If values contains NaN, it returns NaN.
// -Inf is treated as smaller than all other values,
// Inf is treated as larger than all other values, and
// -0.0 is treated as smaller than 0.0.
func Median(values []float64) float64 { return Quantiles(values, 0.5)[0] }

// Quantiles returns a sequence of quantiles of values.
//
// The returned slice has the same length as the quantiles slice,
// and the elements are one-to-one with the input quantiles.
// A quantile of 0 corresponds to the minimum value in values and
// a quantile of 1 corresponds to the maximum value in values.
// A quantile of 0.5 is the same as the value returned by [Median].
//
// Quantiles does not modify the array.
//
// Quantiles may perform asymptotically faster and allocate
// asymptotically less if the slice is already sorted.
//
// Quantiles panics if values is an empty slice or any
// quantile is not contained in the interval [0, 1].
//
// If values contains NaN, it returns [NaN, ..., NaN].
// -Inf is treated as smaller than all other values,
// Inf is treated as larger than all other values, and
// -0.0 is treated as smaller than 0.0.
func Quantiles(values []float64, quantiles ...float64) []float64 {
	if len(values) == 0 {
		panic("quantiles: empty slice")
	}
	if !slices.IsSorted(values) {
		values = slices.Clone(values)
		slices.Sort(values)
	}
	res := make([]float64, len(quantiles))
	if math.IsNaN(values[0]) {
		for i := range res {
			res[i] = math.NaN()
		}
		return res
	}
	for i, q := range quantiles {
		if !(0 <= q && q <= 1) {
			panic("quantile must be contained in the interval [0, 1]")
		}
		// There are many methods for computing quantiles. Quantiles uses the
		// "inclusive" method, also known as Q7 in Hyndman and Fan, or the
		// "linear" or "R-7" method. This assumes that the data is either a
		// population or a sample that includes the most extreme values of the
		// underlying population.
		res[i] = hyndmanFanR7(values, q)
	}
	return res
}

// hyndmanFanR7 implements the Hyndman and Fan "R-7"
// method of computing interpolated quantile values
// over a sorted slice of vals.
//
// hyndmanFanR7 does not modify the array.
func hyndmanFanR7(values []float64, q float64) float64 {
	h := float64(len(values)-1)*q + 1
	// the h-th smallest of len(vals) values is at fn(h)-1.
	return values[floor(h-1)] + (h-math.Floor(h))*(values[ceil(h-1)]-values[floor(h-1)])
}

// ceil returns the integer value of [math.Ceil].
func ceil(n float64) int { return int(math.Ceil(n)) }

// floor returns the integer value of [math.Floor].
func floor(n float64) int { return int(math.Floor(n)) }
