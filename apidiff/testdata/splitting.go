package p

// Splitting types

// OK: in both old and new, {J1, K1, L1} name the same type.
// old
type (
	J1 = K1
	K1 = L1
	L1 int
)

// new
type (
	J1 = K1
	K1 int
	L1 = J1
)

// Old has one type, K2; new has J2 and K2.
// both
type K2 int

// old
type J2 = K2

// new
// i K2: changed from K2 to K2
type J2 K2 // old K2 corresponds with new J2
// old K2 also corresponds with new K2: problem

// both
type k3 int

var Vj3 j3 // expose j3

// old
type j3 = k3

// new
// OK: k3 isn't exposed
type j3 k3

// both
type k4 int

var Vj4 j4 // expose j4
var VK4 k4 // expose k4

// old
type j4 = k4

// new
// i Vj4: changed from k4 to j4
// e.g. p.Vj4 = p.Vk4
type j4 k4
