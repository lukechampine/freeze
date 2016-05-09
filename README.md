Freeze
======

[![GoDoc](https://godoc.org/github.com/lukechampine/freeze?status.svg)](https://godoc.org/github.com/lukechampine/freeze)
[![Go Report Card](http://goreportcard.com/badge/github.com/lukechampine/freeze)](https://goreportcard.com/report/github.com/lukechampine/freeze)

```
go get github.com/lukechampine/freeze
```

Package freeze enables the "freezing" of data, similar to JavaScript's
`Object.freeze()`. A frozen object cannot be modified; attempting to do so
will result in an unrecoverable panic.

To accomplish this, the `mprotect` syscall is used. Sadly, this necessitates
allocating new memory via `mmap` and copying the data into it. This
performance penalty should not be prohibitive, but it's something to be aware
of.

Freezing is useful for providing soft guarantees of immutability. That is: the
compiler can't prevent you from mutating an frozen object, but the runtime
can. One of the unfortunate aspects of Go is its limited support for
constants: structs, slices, and even arrays cannot be declared as consts. This
becomes a problem when you want to pass a slice around to many consumers
without worrying about them modifying it. With freeze, you can guard against
these unwanted or intended behaviors.

Functions are provided for freezing the three "pointer types:" `Pointer`,
`Slice`, and `Map`. Each function returns a copy of their input that is backed
by protected memory. In addition, `Object` is provided for freezing
recursively. Given a slice of pointers, `Object` will prevent modifications to
both the pointer data and the slice data, while `Slice` merely does the
latter.

To freeze an object:

```go
type foo struct {
	X int
	y bool // yes, freeze works on unexported fields!
}
f := &foo{3, true}
f = freeze.Object(f).(*foo)
println(f.X) // ok; prints 3
f.X++        // not ok; panics
```

Note that since `foo` does not contain any pointers, calling `Pointer(f)`
would have the same effect here.

It is recommended that, where convenient, you reassign the return value to its
original variable, as with append. Otherwise, you will retain both the mutable
original and the frozen copy.

Likewise, to freeze a slice:

```go
xs := []int{1, 2, 3}
xs = freeze.Slice(xs).([]int)
println(xs[0]) // ok; prints 1
xs[0]++        // not ok; panics
```

Interfaces can also be frozen, since internally they are just pointers to
objects. The effect of this is that the interface's pure methods can still be
called, but impure methods cannot. Unfortunately, the impurity of a given
method is defined by the implementation, not the interface. Even a `String()`
method could conceivably modify some internal state. Furthermore, the caveat
about unexported struct fields (see below) applies here, so many exported
objects cannot be completely frozen.

## Caveats ##

In general, you can't call `Object` on the same object twice. This is because
`Object` will attempt to rewrite the object's internal pointers -- which is a
memory modification. Calling `Pointer` or `Slice` twice should be fine.

`Object` cannot descend into unexported struct fields. It can still freeze the
field itself, but if the field contains a pointer, the data it points to will
not be frozen.

Appending to a frozen slice will trigger a panic iff `len(slice) < cap(slice)`.
This is because appending to a full slice will allocate new memory.

`Map` requires allocating two pages. For the specific reason why, see comments
in the implementation.

Unix is the only supported platform. Windows support is not planned, because
it doesn't support a syscall analogous to `mprotect`.
