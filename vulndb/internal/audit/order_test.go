package audit

import (
	"reflect"
	"sort"
	"testing"
)

func TestFindingsOrdering(t *testing.T) {
	f1 := Finding{Trace: []TraceElem{
		{Description: "T1"},
	},
	}
	f2 := Finding{Trace: []TraceElem{
		{Description: "T1"},
		{Description: "T2"},
	},
	}
	f3 := Finding{Trace: []TraceElem{
		{Description: "T1"}},
		confidence: 1,
	}
	f4 := Finding{Trace: []TraceElem{
		{Description: "T1"}},
		confidence: 1,
		weight:     2,
	}

	finds := []Finding{f4, f3, f2, f1}
	sort.SliceStable(finds, func(i int, j int) bool { return findingCompare(&finds[i], &finds[j]) })
	if want := []Finding{f1, f2, f3, f4}; !reflect.DeepEqual(finds, want) {
		t.Errorf("want ordering %v; got %v", want, finds)
	}
}
