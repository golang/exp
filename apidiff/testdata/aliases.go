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
