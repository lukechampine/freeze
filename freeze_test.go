package freeze

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// The panics caused by accessing protected memory are unrecoverable.
// Therefore, the only way to test freeze's behavior is to spawn a subprocess
// and check its return value. We accomplish this by rerunning the test binary
// with a special flag enabled.

var crash = flag.Bool("crash", false, "")

func execCrasher(t *testing.T, test string) {
	cmd := exec.Command(os.Args[0], "-test.run="+test, "-crash")
	output, _ := cmd.CombinedOutput()
	if !bytes.Contains(output, []byte("unexpected fault address")) {
		t.Fatalf("Test did not trigger 'unexpected fault address' panic")
	}
}

// TestWritePointerInt tests that modifying a frozen int pointer triggers a
// panic.
func TestWritePointerInt(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWritePointerInt")
		return
	}

	x := 3
	xp := Pointer(&x).(*int)
	*xp++
}

// TestWritePointerString tests that modifying a frozen string pointer
// triggers a panic.
func TestWritePointerString(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWritePointerString")
		return
	}

	s := "foo"
	sp := Pointer(&s).(*string)
	*sp = "bar"
}

// TestWritePointerStruct tests that modifying a frozen struct pointer
// triggers a panic.
func TestWritePointerStruct(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWritePointerStruct")
		return
	}

	type foo struct {
		X int
		y bool
	}
	f := foo{3, true}
	fp := Pointer(&f).(*foo)
	fp.X++
}

// TestWriteSlice tests that modifying a frozen slice triggers a panic.
func TestWriteSlice(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteSlice")
		return
	}

	xs := []int{1, 2, 3}
	xs = Slice(xs).([]int)
	xs[0]++
}

// TestWriteSliceAppend tests that appending to a frozen slice triggers a
// panic when cap > len.
func TestWriteSliceAppend(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteSliceAppend")
		return
	}

	xs := make([]int, 3, 4)
	xs = Slice(xs).([]int)
	_ = append(xs, 5)
}

// TestWriteMap tests that modifying a frozen map triggers a panic.
func TestWriteMap(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteMap")
		return
	}

	m := map[int]int{1: 1}
	m = Map(m).(map[int]int)
	m[1] = 1 // even a no-op overwrite should trigger
}

// TestDeleteMap tests that deleting an entry from a frozen map triggers a
// panic.
func TestDeleteMap(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestDeleteMap")
		return
	}

	m := map[int]int{1: 1}
	m = Map(m).(map[int]int)
	delete(m, 1)
}

// TestWriteObject1 tests that modifying a frozen object triggers a panic.
func TestWriteObject1(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObject1")
		return
	}

	type foo struct {
		S  string
		IP *int
		BS []*bool
	}
	f := &foo{"foo", new(int), []*bool{new(bool)}}
	f = Object(f).(*foo)
	*f.BS[0] = true
}

// TestWriteObject2 tests that modifying a frozen object triggers a panic.
func TestWriteObject2(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObject2")
		return
	}

	type foo struct {
		S  string
		IP *int
		BS []*bool
	}
	f := &foo{"foo", new(int), []*bool{new(bool)}}
	f = Object(f).(*foo)
	f.BS[0] = new(bool)
}

// TestWriteObject3 tests that modifying a frozen object triggers a panic.
func TestWriteObject3(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObject3")
		return
	}

	type foo struct {
		S  string
		IP *int
		BS []*bool
	}
	f := &foo{"foo", new(int), []*bool{new(bool)}}
	f = Object(f).(*foo)
	*f.IP = 8
}

// TestWriteObject4 tests that modifying a frozen object triggers a panic.
func TestWriteObject4(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObject4")
		return
	}

	type foo struct {
		S  string
		IP *int
		BS []*bool
	}
	f := &foo{"foo", new(int), []*bool{new(bool)}}
	f = Object(f).(*foo)
	f.S = "bar"
}

// TestWriteObjectSlice tests that modifying a frozen slice triggers a panic.
func TestWriteObjectSlice(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObjectSlice")
		return
	}

	type foo struct {
		S  string
		IP *int
		BS []*bool
	}
	f := []foo{{"foo", new(int), []*bool{new(bool)}}}
	f = Object(f).([]foo)
	*f[0].BS[0] = false
}

// TestWriteObjectArray tests that modifying a frozen array triggers a panic.
func TestWriteObjectArray(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObjectArray")
		return
	}

	type foo struct {
		BS [3]*bool
	}
	f := &foo{[3]*bool{new(bool)}}
	f = Object(f).(*foo)
	*f.BS[0] = true
}

// TestWriteObjectMapKey tests that modifying a frozen map key triggers a
// panic.
func TestWriteObjectMapKey(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObjectMapKey")
		return
	}

	m := map[*int]int{new(int): 1}
	m = Object(m).(map[*int]int)
	for i := range m {
		*i = 1
	}
}

// TestWriteObjectMapVal tests that modifying a frozen map value triggers a
// panic.
func TestWriteObjectMapVal(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObjectMapVal")
		return
	}

	m := map[int]*int{1: new(int)}
	m = Object(m).(map[int]*int)
	*m[1] = 3
}

// TestWriteObjectInterface tests that calling impure methods on a frozen
// interface triggers a panic.
func TestWriteObjectInterface(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObjectInterface")
		return
	}

	type grower interface {
		// impure method; see TestReadObject for pure method
		Grow(int)
	}
	var g grower = new(bytes.Buffer)
	g = Object(g).(grower)
	g.Grow(1)
}

// TestWriteObjectTwice tests that freezing an object twice triggers a panic.
func TestWriteObjectTwice(t *testing.T) {
	if !*crash {
		execCrasher(t, "TestWriteObjectTwice")
		return
	}

	type foo struct {
		BS [3]*bool
	}
	f := Object(&foo{[3]*bool{new(bool)}}).(*foo)
	Object(f)
}

// TestReadPointer tests that frozen pointers can be read without triggering a
// panic.
func TestReadPointer(t *testing.T) {
	x := 3
	xp := Pointer(&x).(*int)
	y := *xp * 2
	if y != 6 {
		t.Fatal(y)
	}

	type foo struct {
		I int
		b bool
	}
	f := foo{3, true}
	fp := Pointer(&f).(*foo)
	y = fp.I * 2
	if y != 6 {
		t.Fatal(y)
	}

	// should be able to freeze nil
	Pointer(nil)
}

// TestReadMap tests that frozen maps can be read without triggering a panic.
func TestReadMap(t *testing.T) {
	m := map[int]int{1: 1, 2: 2}
	m = Map(m).(map[int]int)
	_, ok1 := m[1]
	_, ok2 := m[2]
	if len(m) != 2 || !ok1 || !ok2 {
		t.Fatal(m)
	}

	// should be able to freeze nil and empty map
	Map(nil)
	Map(map[int]int{})
}

// TestReadSlice tests that frozen slices can be read without triggering a
// panic.
func TestReadSlice(t *testing.T) {
	xs := []int{1, 2, 3}
	xs = Slice(xs).([]int)
	y := xs[2] * 2
	if y != 6 {
		t.Fatal(y)
	}

	type foo struct {
		I int
		b bool
	}
	fs := []foo{{3, true}}
	fs = Slice(fs).([]foo)
	y = fs[0].I * 2
	if y != 6 {
		t.Fatal(y)
	}

	// should be able to append as long as len == cap
	fs = append(fs, foo{0, false})
	fs[0].I++ // fs can now be modified

	// if we don't reassign the pointer, we should still be pointing to
	// writable memory
	xs = []int{1, 2, 3}
	Slice(xs)
	xs[0]++

	// should be able to freeze nil and empty slice
	Slice(nil)
	Slice([]int{})
}

// TestReadObject tests that frozen objects can be read without triggering a
// panic.
func TestReadObject(t *testing.T) {
	// pointer
	type foo struct {
		S  string
		IP *int
		BS []*bool
	}
	x := 3
	tru, fals := true, false
	f := &foo{"foo", &x, []*bool{&tru, &fals, &tru}}
	f = Object(f).(*foo)
	if f.S != "foo" {
		t.Fatal(f.S)
	}
	if (*f.IP)*2 != 6 {
		t.Fatal(*f.IP)
	}
	if *f.BS[0] && *f.BS[2] == *f.BS[1] {
		t.Fatal(f.BS)
	}

	// slice
	fs := []foo{{"foo", &x, []*bool{&tru, &fals, &tru}}}
	fs = Object(fs).([]foo)
	if fs[0].S != "foo" {
		t.Fatal(fs[0].S)
	}
	if (*fs[0].IP)*2 != 6 {
		t.Fatal(*fs[0].IP)
	}
	if *fs[0].BS[0] && *fs[0].BS[2] == *fs[0].BS[1] {
		t.Fatal(fs[0].BS)
	}
	// empty non-nil slice
	Object([]int{})

	// array
	arr := [3][]int{{1, 2, 3}, nil, {4, 5, 6}}
	ap := Object(&arr).(*[3][]int)
	if len(ap[0]) != len(ap[2]) {
		t.Fatal(ap)
	}

	// map
	m := map[int]*foo{1: {"foo", &x, []*bool{&tru, &fals, &tru}}}
	m = Object(m).(map[int]*foo)
	if m[1].S != "foo" {
		t.Fatal(m)
	}

	// interface with pure method (see TestWriteObjectInterface for an impure
	// method)
	type stringer interface {
		String() string
	}
	var s stringer = bytes.NewBufferString("foo")
	s = Object(s).(stringer)
	if s.String() != "foo" {
		t.Fatal(s.String())
	}

	// empty object
	var empty struct{}
	Object(&empty)

	// nil
	Object(nil)
	Object([]*int{nil})
	Object(new(*int))

}

// TestFreezeUnexportedObject tests that Object will not descend into
// unexported fields.
func TestFreezeUnexportedObject(t *testing.T) {
	type foo struct {
		b []byte
	}
	f := &foo{[]byte{1, 2, 3}}
	f = Object(f).(*foo)
	// f.b should not be frozen
	f.b[0] = 9
}

// TestWriteSlicePointers tests that the elements of a frozen slice of
// pointers can be modified without triggering a panic.
func TestWriteSlicePointers(t *testing.T) {
	xs := []*int{new(int), new(int), new(int)}
	xs = Slice(xs).([]*int)
	*xs[0]++
	if *xs[0] != 1 {
		t.Fatal(*xs[0])
	}
}

// TestIllegalTypes tests that the Pointer, Slice, and Object functions will
// panic if supplied an invalid type.
func TestIllegalTypes(t *testing.T) {
	catchPanic := func(name string) {
		if r := recover(); r == nil {
			t.Fatal("test", name, "did not panic")
		}
	}

	func() {
		defer catchPanic("Pointer")
		Pointer(map[int]int{})
	}()

	func() {
		defer catchPanic("Slice")
		Slice(new(bool))
	}()

	func() {
		defer catchPanic("Slice")
		Map([]byte{})
	}()

	func() {
		defer catchPanic("Object")
		Object(true)
	}()
}

// TestGarbageCollection tests that frozen objects are properly garbage-collected.
func TestGarbageCollection(t *testing.T) {
	_ = Pointer(new(int)).(*int)
	runtime.GC() // manually verified via coverage inspection; finalizer should have run
}

// BenchmarkFreezeObject benchmarks freezing a complex object.
func BenchmarkFreezeObject(b *testing.B) {
	type foo struct {
		X []struct {
			Y *struct {
				Z [100]int
			}
		}
	}

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		f := new(foo)
		f.X = make([]struct {
			Y *struct {
				Z [100]int
			}
		}, 100)
		for i := range f.X {
			f.X[i].Y = new(struct {
				Z [100]int
			})
		}
		b.StartTimer()
		Object(f)
	}

	b.ReportAllocs()
}
