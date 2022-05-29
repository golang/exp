package ranges

import (
	"math"
)

type rangeTypes interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 |
		uint16 | uint32 | uint64 | float32 | float64
}

// RangeList return list of numbers between start and stop accourding step
func RangeList[R rangeTypes](start, stop, step R) (list []R) {
	if math.Signbit(float64(start)) || (start == 0) && math.Signbit(float64(stop)) && math.Signbit(float64(step)) {
		for i := start; i > stop; i += step {
			list = append(list, i)
		}
	} else if !math.Signbit(float64(start)) && !(start == 0) && math.Signbit(float64(stop)) && math.Signbit(float64(step)) {
		return
	} else if math.Signbit(float64(start)) || (start == 0) && !math.Signbit(float64(stop)) && math.Signbit(float64(step)) {
		return
	} else if math.Signbit(float64(start)) || (start == 0) && math.Signbit(float64(stop)) && !math.Signbit(float64(step)) {
		return
	} else {
		for i := start; i < stop; i += step {
			list = append(list, i)
		}
	}
	return
}
