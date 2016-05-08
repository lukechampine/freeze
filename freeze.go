/*

Package freeze enables the "freezing" of data, similar to JavaScript's
Object.freeze(). A frozen object cannot be modified; attempting to do so
will result in an unrecoverable panic.

To accomplish this, the mprotect syscall is used. Sadly, this necessitates
allocating new memory via mmap and copying the data into it. This performance
penalty should not be prohibitive, but it's something to be aware of.

Freezing is useful for providing soft guarantees of immutability. That is: the
compiler can't prevent you from mutating an frozen object, but the runtime
can. One of the unfortunate aspects of Go is its limited support for
constants: structs, slices, and even arrays cannot be declared as consts. This
becomes a problem when you want to pass a slice around to many consumers
without worrying about them modifying it. With freeze, you can guard against
these unwanted or intended behaviors.

Three functions are provided: Pointer, Slice, and Object. Object is a generic
function that freezes either a pointer or a slice, but does so recursively.
That is, calling Object on a slice of pointers will freeze both the slice and
the pointers. To freeze an object:

	type foo struct {
		X int
		y bool // yes, freeze works on unexported fields!
	}
	f := foo{3, true}
	fp := freeze.Object(&f).(*foo)
	println(fp.X) // ok; prints 3
	fp.X++        // not ok; panics

Since foo does not contain any pointers, calling Pointer(&f) would have
the same effect.

It is recommended that, where convenient, you reassign the returned pointer to
its original variable, as with append. Note that in the above example, f can
still be freely modified.

Likewise, to freeze a slice:

	xs := []int{1, 2, 3}
	xs = freeze.Slice(xs).([]int)
	println(xs[0]) // ok; prints 1
	xs[0]++        // not ok; panics

It may not be immediately obvious why these functions return a value that must
be reassigned. The reason is that we are allocating new memory, and therefore
the pointer must be updated. The same is true of the built-in append function.
Well, not quite; if a slice has greater capacity than length, then append will
use that memory. For the same reason, appending to a frozen slice with spare
capacity will trigger a panic.

Caveats

In general, you can't call Object on the same object twice. This is because
Object will attempt to rewrite the object's internal pointers -- which is a
memory modification. Calling Pointer or Slice twice should be fine.

Object cannot descend into unexported struct fields. It can still freeze the
field itself, but if the field contains a pointer, the data it points to will
not be frozen.

Maps and channels are not supported, due to the complexity of their internal
memory layout. They may be supported in the future, but don't count on it. An
immutable channel wouldn't be very useful anyway.

Unix is the only supported platform. Windows support is not planned, because
it doesn't support a syscall analogous to mprotect.
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
	if v == nil {
		return v
	}
	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Ptr {
		panic("Pointer called on non-pointer type")
	}

	// freeze the memory pointed to by the interface's data pointer
	size := typ.Elem().Size()
	ptrs := (*[2]uintptr)(unsafe.Pointer(&v))
	ptrs[1] = copyAndFreeze(ptrs[1], size)

	return v
}

// Slice freezes v, which must be a slice. Future writes to v's memory will
// result in a panic.
func Slice(v interface{}) interface{} {
	if v == nil {
		return v
	}
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Slice {
		panic("Slice called on non-slice type")
	}

	// freeze the memory pointed to by the slice's data pointer
	size := val.Type().Elem().Size() * uintptr(val.Len())
	slice := (*[3]uintptr)((*[2]unsafe.Pointer)(unsafe.Pointer(&v))[1]) // should be [2]uintptr, but go vet complains
	slice[0] = copyAndFreeze(slice[0], size)

	return v
}

// Object recursively freezes v, which must be a pointer or a slice. It will
// follow pointers until "bottoming out," freezing the entire chain. Passing a
// cyclic structure to Object will result in infinite recursion. Note that
// Object can only follow pointer fields if they are exported (the pointers
// themselves will still be frozen).
func Object(v interface{}) interface{} {
	if v == nil {
		return v
	}
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr && val.Kind() != reflect.Slice {
		panic("Object called on non-slice type")
	}

	return object(val).Interface()
}

// object updates all pointers in val to point to frozen memory containing the
// same data.
func object(val reflect.Value) reflect.Value {
	// helper function to identify types that may contain pointers
	hasPtrs := func(t reflect.Type) bool {
		k := t.Kind()
		return k == reflect.Ptr || k == reflect.Array || k == reflect.Slice || k == reflect.Struct
	}

	switch val.Type().Kind() {
	default:
		return val

	case reflect.Ptr:
		if val.IsNil() {
			return val
		}
		val.Elem().Set(object(val.Elem()))
		return reflect.ValueOf(Pointer(val.Interface()))

	case reflect.Array:
		// only recurse if elements might have pointers
		if hasPtrs(val.Type().Elem()) {
			for i := 0; i < val.Len(); i++ {
				val.Index(i).Set(object(val.Index(i)))
			}
		}
		return val

	case reflect.Slice:
		// only recurse if elements might have pointers
		if hasPtrs(val.Type().Elem()) {
			for i := 0; i < val.Len(); i++ {
				val.Index(i).Set(object(val.Index(i)))
			}
		}
		return reflect.ValueOf(Slice(val.Interface()))

	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			// only recurse if field is exported and might have pointers
			t := val.Type().Field(i)
			if !(t.PkgPath != "" && !t.Anonymous) && hasPtrs(t.Type) {
				val.Field(i).Set(object(val.Field(i)))
			}
		}
		return val
	}
}

// copyAndFreeze copies n bytes from dataptr into new memory, freezes it, and
// returns a uintptr to the new memory.
func copyAndFreeze(dataptr, n uintptr) uintptr {
	if n == 0 {
		return dataptr
	}
	// allocate new memory to be frozen
	newMem, err := unix.Mmap(-1, 0, int(n), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		panic(err)
	}
	// set a finalizer to unmap the memory when it would normally be GC'd
	runtime.SetFinalizer(&newMem, func(b *[]byte) { _ = unix.Munmap(*b) })

	// copy n bytes into newMem
	copy(newMem, *(*[]byte)(unsafe.Pointer(&[3]uintptr{dataptr, n, n})))

	// freeze the new memory
	if err = unix.Mprotect(newMem, unix.PROT_READ); err != nil {
		panic(err)
	}

	// return pointer to new memory
	return uintptr(unsafe.Pointer(&newMem[0]))
}
