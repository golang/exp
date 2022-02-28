<!-- To regenerate the readme, run: -->
<!-- go run golang.org/x/example/gotypes@latest generic-go-types.md -->

# Updating tools to support type parameters.

This guide is maintained by Rob Findley (`rfindley@google.com`).

**status**: this document is currently a rough-draft. See [golang/go#50447](https://go.dev/issues/50447) for more details.

%toc

# Who should read this guide

Read this guide if you are a tool author seeking to update your tools to
support generics Go code. Generics introduce significant new complexity to the
Go type system, because types can now be _parameterized_. While the
fundamentals of the `go/types` APIs remain the same, some previously valid
assumptions no longer hold. For example:

 - Type declarations need not correspond 1:1 with the types they define.
 - Interfaces are no longer determined entirely by their method set.
 - The set of concrete types implementing `types.Type` has grown to include
   `types.TypeParam` and `types.Union`.

# Introduction

With Go 1.18, Go now supports generic programming via type parameters. This
document is a guide for tool authors that want to update their tools to support
the new language constructs.

This guide assumes knowledge of the language changes to support generics. See
the following references for more information:

- The [original proposal](https://go.dev/issue/43651) for type parameters.
- The [addendum for type sets](https://go.dev/issue/45346).
- The [latest language specfication](https://tip.golang.org/ref/spec) (still in-progress as of 2021-01-11).
- The proposals for new APIs in
  [go/token and go/ast](https://go.dev/issue/47781), and in
  [go/types](https://go.dev/issue/47916).

It also assumes knowledge of `go/ast` and `go/types`. If you're just getting
started,
[x/example/gotypes](https://github.com/golang/example/tree/master/gotypes) is
a great introduction (and was the inspiration for this guide).

# Summary of new language features and their APIs

The introduction of of generic features appears as a large change to the
language, but a high level introduces only a few new concepts. We can break
down our discussion into the following three broad categories: generic types,
constraint interfaces, and instantiation. In each category below, the relevant
new APIs are listed (some constructors and getters/setters may be elided where
they are trivial):

**Generic types**. Types and functions may be _generic_, meaning their
declaration may have a non-empty _type parameter list_, as in
`type  List[T any] ...` or `func f[T1, T2 any]() { ... }`. Type parameter lists
define placeholder types (_type parameters_), scoped to the declaration, which
may be substituted by any type satisfying their corresponding _constraint
interface_ to _instantiate_ a new type or function.

Generic types may have methods, which declare `receiver type parameters` via
their receiver type expression: `func (r T[P1, ..., PN]) method(...) (...)
{...}`.

_New APIs_:
 - The field `ast.TypeSpec.TypeParams` holds the type parameter list syntax for
   type declarations.
 - The field `ast.FuncType.TypeParams` holds the type parameter list syntax for
   function declarations.
 - The type `types.TypeParam` is a `types.Type` representing a type parameter.
   On this type, the `Constraint` and `SetConstraint` methods allow
   getting/setting the constraint, the `Index` method returns the numeric index
   of the type parameter in the type parameter list that declares it, and the
   `Obj` method returns the object in the scope a for the type parameter (a
   `types.TypeName`). Generic type declarations have a new `*types.Scope` for
   type parameter declarations.
 - The type `types.TypeParamList` holds a list of type parameters.
 - The method `types.Named.TypeParams` returns the type parameters for a type
   declaration.
 - The method `types.Named.SetTypeParams` sets type parameters on a defined
   type.
 - The function `types.NewSignatureType` creates a new (possibly generic)
   signature type.
 - The method `types.Signature.RecvTypeParams` returns the receiver type
   parameters for a method.
 - The method `types.Signature.TypeParams` returns the type parameters for
   a function.

**Constraint Interfaces**: type parameter constraints are interfaces, expressed
by an interface type expression. Interfaces that are only used in constraint
position are permitted new embedded elements composed of tilde expressions
(`~T`) and unions (`A | B | ~C`). The new builtin interface type `comparable`
is implemented by types for which `==` and `!=` are valid (note that interfaces
must be statically comparable in this case, i.e., each type in the interface's
type set must be comparable). As a special case, the `interface` keyword may be
omitted from constraint expressions if it may be implied (in which case we say
the interface is _implicit_).

_New APIs_:
 - The constant `token.TILDE` is used to represent tilde expressions as an
   `ast.UnaryExpr`.
 - Union expressions are represented as an `ast.BinaryExpr` using `|`. This
   means that `ast.BinaryExpr` may now be both a type and value expression.
 - The method `types.Interface.IsImplicit` reports whether the `interface`
   keyword was elided from this interface.
 - The method `types.Interface.MarkImplicit` marks an interface as being
   implicit.
 - The method `types.Interface.IsComparable` reports whether every type in an
   interface's type set is comparable.
 - The method `types.Interface.IsMethodSet` reports whether an interface is
   defined entirely by its methods (has no _specific types_).
 - The type `types.Union` is a type that represents an embedded union
   expression in an interface. May only appear as an embedded element in
   interfaces.
 - The type `types.Term` represents a (possibly tilde) term of a union.

**Instantiation**: generic types and functions may be _instantiated_ to create
non-generic types and functions by providing _type arguments_ (`var x T[int]`).
Function type arguments may be _inferred_ via function arguments, or via
type parameter constraints.

_New APIs_:
 - The type `ast.IndexListExpr` holds index expressions with multiple indices,
   as in instantiation expressions with multiple type arguments or in receivers
   declaring multiple type parameters.
 - The function `types.Instantiate` instantiates a generic type with type arguments.
 - The type `types.Context` is an opaque instantiation context that may be
   shared to reduce duplicate instances.
 - The field `types.Config.Context` holds a shared `Context` to use for
   instantiation while type-checking.
 - The type `types.TypeList` holds a list of types.
 - The type `types.ArgumentError` holds an error associated with a specific
   type argument index. Used to represent instantiation errors.
 - The field `types.Info.Instances` maps instantiated identifiers to information
   about the resulting type instance.
 - The type `types.Instance` holds information about a type or function
   instance.
 - The method `types.Named.TypeArgs` reports the type arguments used to
   instantiate a named type.

# Examples

The following examples demonstrate the new APIs, and discuss their properties.
All examples are runnable, contained in subdirectories of the directory holding
this README.

## Generic types: type parameters

We say that a type is _generic_ if it has type parameters but no type
arguments. This section explains how we can inspect generic types with the new
`go/types` APIs.

### Type parameter lists

Suppose we want to understand the generic library below, which defines a generic
`Pair`, a constraint interface `Constraint`, and a generic function `MakePair`.

%include findtypeparams/main.go input -

We can use the new `TypeParams` fields in `ast.TypeSpec` and `ast.FuncType` to
access the type parameter list. From there, we can access type parameter types
in at least three ways:
 - by looking up type parameter definitions in `types.Info`
 - by calling `TypeParams()` on `types.Named` or `types.Signature`
 - by looking up type parameter objects in the declaration scope. Note that
   there now may be a scope associated with an `ast.TypeSpec` node.

%include findtypeparams/main.go print -

This program produces the following output. Note that not every type spec has
a scope.

%include findtypeparams/main.go output -

## Constraint Interfaces

In order to allow operations on type parameters, Go 1.18 introduces the notion
of [_type sets_](https://tip.golang.org/ref/spec#Interface_types), which is
abstractly the set of types that implement an interface. This section discusses
the new syntax for restrictions on interface type sets, and the APIs we can use
to understand them.

### New interface elements

Consider the generic library below:

%include interfaces/main.go input -

In this library, we can see a few new features added in Go 1.18. The first is
the new syntax in the `Numeric` type: unions of tilde-terms, specifying that
the numeric type may only be satisfied by types whose underlying type is `int`
or `float64`.

The `go/ast` package parses this new syntax as a combination of unary and
binary expressions, which we can see using the following program:

%include interfaces/main.go printsyntax -

Output:

%include interfaces/main.go outputsyntax -

Once type-checked, these embedded expressions are represented using the new
`types.Union` type, which flattens the expression into a list of `*types.Term`.
We can also investigate two new methods of interface:
`types.Interface.IsComparable`, which reports whether the type set of an
interface is comparable, and `types.Interface.IsMethodSet`, which reports
whether an interface is expressable using methods alone.

%include interfaces/main.go printtypes -

Output:

%include interfaces/main.go outputtypes -

The `Findable` type demonstrates another new feature of Go 1.18: the comparable
built-in. Comparable is a special interface type, not expressable using
ordinary Go syntax, whose type-set consists of all comparable types.

### Implicit interfaces

For interfaces that do not have methods, we can inline them in constraints and
elide the `interface` keyword. In the example above, we could have done this
for the `Square` function:

%include implicit/main.go input -

In such cases, the `types.Interface.IsImplicit` method reports whether the
interface type was implicit. This does not affect the behavior of the
interface, but is captured for more accurate type strings:

%include implicit/main.go show -

Output:

%include implicit/main.go output -

The `types.Interface.MarkImplicit` method is used to mark interfaces as
implicit by the importer.

### Type sets

The examples above demonstrate the new APIs for _accessing_ information about
the new interface elements, but how do we understand
[_type sets_](https://tip.golang.org/ref/spec#Interface_types), the new
abstraction that these elements help define? Type sets may be arbitrarily
complex, as in the following example:

%include typesets/main.go input -

Here, the type set of `D` simplifies to `~string|int`, but the current
`go/types` APIs do not expose this information. This will likely be added to
`go/types` in future versions of Go, but in the meantime we can use the
`typeparams.NormalTerms` helper:

%include typesets/main.go print -

which outputs:

%include typesets/main.go output -

See the documentation for `typeparams.NormalTerms` for more information on how
this calculation proceeds.

## Instantiation

We say that a type is _instantiated_ if it is created from a generic type by
substituting type arguments for type parameters. Instantiation can occur via
explicitly provided type arguments, as in the expression `T[A_1, ..., A_n]`, or
implicitly, through type inference.. This section describes how to find and
understand instantiated types.

### Finding instantiated types

Certain applications may find it useful to locate all instantiated types in
a package. For this purpose, `go/types` provides a new `types.Info.Instances`
field that maps instantiated identifiers to information about their instance.

For example, consider the following code:

%include instantiation/main.go input -

We can find instances by type-checking with the `types.Info.Instances` map
initialized:

%include instantiation/main.go check -

Output:

%include instantiation/main.go checkoutput -

The `types.Instance` type provides information about the (possibly inferred)
type arguments that were used to instantiate the generic type, and the
resulting type. Notably, it does not include the _generic_ type that was
instantiated, because this type can be found using `types.Info.Uses[id].Type()`
(where `id` is the identifier node being instantiated).

Note that the receiver type of method `Left` also appears in the `Instances`
map. This may be counterintuitive -- more on this below.

### Creating new instantiated types

`go/types` also provides an API for creating type instances:
`types.Instantiate`. This function accepts a generic type and type arguments,
and returns an instantiated type (or an error). The resulting instance may be
a newly constructed type, or a previously created instance with the same type
identity. To facilitate the reuse of frequently used instances,
`types.Instantiate` accepts a `types.Context` as its first argument, which
records instances.

If the final `validate` argument to `types.Instantiate` is set, the provided
type arguments will be verified against their corresponding type parameter
constraint; i.e., `types.Instantiate` will check that each type arguments
implements the corresponding type parameter constraint. If a type arguments
does not implement the respective constraint, the resulting error will wrap
a new `ArgumentError` type indicating which type argument index was bad.

%include instantiation/main.go instantiate -

Output:

%include instantiation/main.go instantiateoutput -

### Using a shared context while type checking

To share a common `types.Context` argument with a type-checking pass, set the
new `types.Config.Context` field.

## Generic types continued: method sets and predicates

Generic types are fundamentally different from ordinary types, in that they may
not be used without instantiation. In some senses they are not really types:
the go spec defines [types](https://tip.golang.org/ref/spec#Types) as "a set of
values, together with operations and methods", but uninstantiated generic types
do not define a set of values. Rather, they define a set of _types_. In that
sense, they are a "meta type", or a "type template" (disclaimer: I am using
these terms imprecisely).

However, for the purposes of `go/types` it is convenient to treat generic types
as a `types.Type`. This section explains how generic types behave in existing
`go/types` APIs.

### Method Sets

Methods on uninstantiated generic types are different from methods on an
ordinary type. Consider that for an ordinary type `T`, the receiver base type
of each method in its method set is `T`. However, this can't be the case for
a generic type: generic types cannot be used without instantation, and neither
can the type of the receiver variable. Instead, the receiver base type is an
_instantiated_ type, instantiated with the method's receiver type parameters.

This has some surprising consequences, which we observed in the section on
instantiation above: for a generic type `G`, each of its methods will define
a unique instantiation of `G`, as each method has distinct receiver type
parameters.

To see this, consider the following example:

%include genericmethods/main.go input -

Let's inspect the method sets of the types in this library:

%include genericmethods/main.go printmethods -

Output:

%include genericmethods/main.go printoutput -

In this example, we can see that all of `Pair`, `Pair[int, int]`, and
`Pair[L, _]` have distinct method sets, though the method set of `Pair` and
`Pair[L, _]` intersect in the `Left` method.

Only the objects in `Pair`'s method set are recorded in `types.Info.Defs`. To
get back to this "canonical" method object, the `typeparams` package provides
the `OriginMethod` helper:

%include genericmethods/main.go compareorigins -

Output:

%include genericmethods/main.go compareoutput -

### Predicates

Predicates on generic types are not defined by the spec. As a consequence,
using e.g. `types.AssignableTo` with operands of generic types leads to an
undefined result.

The behavior of predicates on generic `*types.Named` types may generally be
derived from the fact that type parameters bound to different names are
different types. This means that most predicates involving generic types will
return `false`.

`*types.Signature` types are treated differently. Two signatures are considered
identical if they are identical after substituting one's set of type parameters
for the other's, including having identical type parameter constraints. This is
analogous to the treatment of ordinary value parameters, whose names do not
affect type identity.

Consider the following code:

%include predicates/main.go ordinary -

Output:

%include predicates/main.go ordinaryoutput -

In this example, we see that despite their similarity the generic `Pair` type
is not assignable to the generic `LeftRighter` type. We also see the rules for
signature identity in practice.

This begs the question: how does one ask questions about the relationship
between generic types? In order to phrase such questions we need more
information: how does one relate the type parameters of `Pair` to the type
parameters of `LeftRighter`? Does it suffice for the predicate to hold for one
element of the type sets, or must it hold for all elements of the type sets?

We can use instantiation to answer some of these questions. In particular, by
instantiating both `Pair` and `LeftRighter` with the type parameters of `Pair`,
we can determine if, for all type arguments `[X, Y]` that are valid for `Pair`,
`[X, Y]` are also valid type arguments of `LeftRighter`, and `Pair[X, Y]` is
assignable to `LeftRighter[X, Y]`. The `typeparams.GenericAssignableTo`
function implements exactly this predicate:

%include predicates/main.go generic -

Output:

%include predicates/main.go genericoutput -

# Updating tools while building at older Go versions

In the examples above, we can see how a lot of the new APIs integrate with
existing usage of `go/ast` or `go/types`. However, most tools still need to
build at older Go versions, and handling the new language constructs in-line
will break builds at older Go versions.

For this purpose, the `x/exp/typeparams` package provides functions and types
that proxy the new APIs (with stub implementations at older Go versions).

# Further help

If you're working on updating a tool to support generics, and need help, please
feel free to reach out for help in any of the following ways:
 - By mailing the [golang-tools](https://groups.google.com/g/golang-tools) mailing list.
 - Directly to me via email (`rfindley@google.com`).
 - For bugs, you can [file an issue](https://github.com/golang/go/issues/new/choose).
