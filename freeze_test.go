package freeze

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
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

	var x int = 3
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
	xs = append(xs, 5)
}

// TestReadPointer tests that frozen pointers can be read without triggering a
// panic.
func TestReadPointer(t *testing.T) {
	var x int = 3
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
