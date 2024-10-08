package p

// Here the same alias refers to different types in old and new.
// We correctly detect the problem, but the message is poor.

// both
type t1 int
type t2 bool

// old
type A = t1

// new
// i t1: changed from int to bool
type A = t2

// old
type B = int

// new
// i B: changed from int to B
type B int

// old
type C int

// new
// OK: merging types
type C = int

// both
// OK: identical (but this fails on 07ab4e7)
type TypedBucketRateLimiter[T comparable] struct{}
type BucketRateLimiter = TypedBucketRateLimiter[any]
