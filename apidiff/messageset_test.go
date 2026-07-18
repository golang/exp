package apidiff

import "testing"

func TestDotjoin(t *testing.T) {
	tests := []struct {
		s1, s2 string
		want   string
	}{
		{"", "part", "part"},
		{"obj", "", "obj"},
		{"AnonField", "Inner", "AnonField.Inner"},
	}
	for _, test := range tests {
		if got := dotjoin(test.s1, test.s2); got != test.want {
			t.Errorf("dotjoin(%q, %q) = %q, want %q", test.s1, test.s2, got, test.want)
		}
	}
}
