/*

Package freeze enables the "freezing" of data, similar to JavaScript's
Object.freeze(). A frozen object cannot be modified; attempting to do so
will result in an unrecoverable panic.

To accomplish this, the mprotect syscall is used. Sadly, this necessitates
allocating new memory via mmap and copying the data into it. This performance
penalty should not be prohibitive, but it's something to be aware of.

Freezing is useful to providing soft guarantees of immutability. That is: the
compiler can't prevent you from mutating an frozen object, but the runtime
can. One of the unfortunate aspects of Go is its limited support for
constants: structs, slices, and even arrays cannot be declared as consts. This
becomes a problem when you want to pass a slice around to many consumers
without worrying about them modifying it. With freeze, you can guard against
these unwanted or intended behaviors.

Two functions are provided: Pointer and Slice. (I suppose these could have
been combined, but then the resulting function would have to be called Freeze,
which stutters.) To freeze a pointer, call Pointer like so:

	var x int = 3
	xp := freeze.Pointer(&x).(*int)
	println(*xp) // ok; prints 3
	*xp++        // not ok; panics

It is recommended that, where convenient, you reassign the returned pointer to
its original variable, as with append. Note that in the above example, x can
still be freely modified.

Likewise, to freeze a slice:

	xs := []int{1, 2, 3}
	xs = freeze.Slice(xs).([]int)
	println(xs[0]) // ok; prints 1
	xs[0]++        // not ok; panics

It may not be immediately obvious why these functions return a value that must
be reassigned. The reason is that we are allocating new memory, and therefore
the pointer must be updated. The same is true of the built-in append function.

Currently, only Unix is supported. Windows support is not planned, because it
doesn't support a syscall analogous to mprotect.
*/
package freeze

import (
	"reflect"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Pointer freezes v, which must be a pointer. Future writes to v's memory will
// result in a panic.
func Pointer(v interface{}) interface{} {
	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Ptr {
		panic("Pointer called on non-pointer type")
	}

	// allocate new memory to hold the frozen value
	size := typ.Elem().Size()
	newMem, err := unix.Mmap(-1, 0, int(size), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		panic(err)
	}

	// write v to the new memory (have fun figuring this out!)
	copy(newMem, *(*[]byte)(unsafe.Pointer(&[3]uintptr{(*[2]uintptr)(unsafe.Pointer(&v))[1], size, size})))

	// overwrite v's data pointer to point at the new memory
	(*[2]uintptr)(unsafe.Pointer(&v))[1] = uintptr(unsafe.Pointer(&newMem[0]))

	// freeze the new memory
	if err = unix.Mprotect(newMem, unix.PROT_READ); err != nil {
		panic(err)
	}

	// set a finalizer to unmap the memory when it would normally be GC'd
	runtime.SetFinalizer(&newMem, func(b *[]byte) { unix.Munmap(*b) })

	return v
}
