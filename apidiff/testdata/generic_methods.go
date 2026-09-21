package p

// Generic methods:

// both
type S struct{}

// old
func (S) M1[T any](x T) T { return x }
func (S) M2[T any](x T)   {}
func (S) M4[T any](x T)   {}

// new
// OK: type param rename
func (S) M1[U any](x U) U { return x }

// i S.M2: changed from func[T any](T) to func[T comparable](T)
func (S) M2[T comparable](x T) {}

// c S.M3: added
func (S) M3[T any](x T) {}

// i S.M4: removed

// Generic methods on generic types:

// both
type C[X any] interface {
	~[]X
}

type GR[R any] struct{}

// old
func (GR[R]) M1[T any](x R, y T) (R, T)  { return x, y }
func (GR[R]) M2[T any](x R, y T)         {}
func (GR[R]) M3[T any, U C[R]](x T, y U) {}
func (GR[R]) M4[T any](x R, y T)         {}

// new
// OK: receiver and method type param rename
func (GR[R2]) M1[T2 any](x R2, y T2) (R2, T2) { return x, y }

// i GR[R].M2: changed from func[T any](R, T) to func[T any](T, R)
func (GR[R]) M2[T any](x T, y R) {}

// i GR[R].M3: changed from func[T any, U C[R]](T, U) to func[T any, U C[T]](T, U)
func (GR[R]) M3[T any, U C[T]](x T, y U) {}

// i GR[R].M4: changed from func[T any](R, T) to func[T comparable](R, T)
func (GR[R]) M4[T comparable](x R, y T) {}

// old
type embedG struct{}

func (embedG) M[T any](x U2, y T) {}

type U1 struct{ embedG }
type U2 struct{ embedG }

// new
// OK: replacing embedded generic method with direct methods
type U1 struct{}

func (U1) M[T any](x U2, y T) {}

type U2 struct{}

func (U2) M[T any](x U2, y T) {}
