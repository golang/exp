package vulncheck

// ImportChain is sequence of import paths starting with
// a client package and ending with a package with some
// known vulnerabilities.
type ImportChain []*PkgNode

// CallStack models a trace of function calls starting
// with a client function or method and ending with a
// call to a vulnerable symbol.
type CallStack []StackEntry

// StackEntry models an element of a call stack.
type StackEntry struct {
	// Function provides information on the function whose frame is on the stack.
	Function *FuncNode

	// Call provides information on the call site inducing this stack frame.
	// nil when the frame represents an entry point of the stack.
	Call *CallSite
}
