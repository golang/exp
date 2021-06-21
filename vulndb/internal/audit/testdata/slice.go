// go:build ignore

package testdata

func X() {}
func Y() {}

// not reachable
func id(i int) int {
	return i
}

// not reachable
func inc(i int) int {
	return i + 1
}

func Apply(b bool, h func()) {
	if b {
		func() {
			print("applied")
		}()
		return
	}
	h()
}

type I interface {
	Foo()
}

type A struct{}

func (a A) Foo() {}

// not reachable
func (a A) Bar() {}

type B struct{}

func (b B) Foo() {}

func debug(s string) {
	print(s)
}

func Do(i I, input string) {
	debug(input)

	i.Foo()

	func(x string) {
		func(l int) {
			print(l)
		}(len(x))
	}(input)
}
