package reflect

import (
	"reflect"
	"testing"
	"unsafe"
)

// Test code from https://github.com/darkgopher/darkness/blob/master/deepcopy_test.go

func TestDeepCopy(t *testing.T) {
	// this variable is here to make sure that the unsafe.Pointer points to a
	// valid address, just typing in a random number makes the go runtime crash
	// when cleaning up the test as it tries to deallocate whatever is pointed
	// by the value. e.g. unsafe.Pointer(uintptr(4)) will end up trying to
	// deallocate the memory at 0x4.
	var aValidAddress struct{}

	tests := []struct {
		i interface{}
	}{
		{"5"},                              // 0, string
		{true},                             // 1, bool
		{1729},                             // 2, int
		{3.141592},                         // 3, float64
		{complex(1, 2)},                    // 4, complex
		{[4]float32{1, 2, 3, 4}},           // 5, array
		{[]bool{true, false, true, false}}, // 6, slice
		{map[int]float32{1: 1.2, 3: 3.4, 4: 4.5, 5: 5.5}}, // 7, map
		{struct {
			a int
			b float32
		}{5, 999.999}}, // 8
		{func() *int {
			i := 9
			return &i
		}()}, // 9, pointer to int
		{
			struct {
				a int
				b []float32
				c struct {
					d complex64
				}
				e *bool
			}{
				a: 5,
				b: []float32{1.1, 2.2, 3.3, 4.4},
				c: struct {
					d complex64
				}{d: complex(9.9, 8.8)},
				e: func() *bool {
					mybool := true
					return &mybool
				}(),
			},
		}, // 10, more complex test
		{uintptr(unsafe.Pointer(&aValidAddress))}, // 11, uintptr
		// {unsafe.Pointer(&aValidAddress)},          // 12, unsafe.Pointer
	}

	for i, test := range tests {
		t.Logf("starting test %d, on %+v", i, test.i)
		j := DeepCopy(test.i)
		if !reflect.DeepEqual(test.i, j) {
			t.Errorf("[%d] problem want %v, got %v", i, test.i, j)
		}
	}
}

func TestDeepCopyStringExceptions(t *testing.T) {
	s := "Hello world!"
	r := struct {
		s0 string
		s1 string
	}{
		s0: s,
		s1: s[:5],
	}
	u := DeepCopy(r)
	if !reflect.DeepEqual(u, r) {
		t.Fatalf("not equal got %v, want %v", u, r)
	}
}

// TestDeepCopyChannel tests the copying of channels, reflect.DeepEqual doesn't
// work on those.
func TestDeepCopyChannel(t *testing.T) {
	c := make(chan struct{})
	v, ok := any(DeepCopy(c)).(chan struct{})
	if !ok {
		t.Errorf("expected a chan struct{}, got %T", v)
	}
}

// TestDeepCopyPointers
func TestDeepCopyPointers(t *testing.T) {
	var i int
	ptr := &i
	cp, ok := any(DeepCopy(ptr)).(*int)
	if !ok {
		t.Fatal("expected a *int")
	}

	if cp == ptr {
		t.Fatalf("pointers should be different, %v", cp)
	}

	if *cp != *ptr {
		t.Fatalf("pointed values should be the same, got %d, want %d", *cp, *ptr)
	}
}

// TestDeepCopyRing tests that a circular reference doesn't go out of control
// and allocate everything.
func TestDeepCopyRing(t *testing.T) {
	type T struct {
		a    int
		next *T
	}

	// make a circular reference
	t0, t1, t2 := &T{a: 0}, &T{a: 1}, &T{a: 2}
	t0.next = t1
	t1.next = t2
	t2.next = t0

	i := DeepCopy(t0)
	r0, ok := any(i).(*T)
	if !ok {
		t.Fatalf("did not receive a *T, but a %T", i)
	}
	if r0.next == nil {
		t.Fatal("r0.next was nil ?")
	}
	r1 := r0.next
	if r1.next == nil {
		t.Fatal("r1.next was nil ?")
	}
	r2 := r1.next
	if r2.next != r0 {
		t.Fatal("r2.next wasn't linked back to r0")
	}

	if t0.a != r0.a || t1.a != r1.a || t2.a != r2.a {
		t.Fatalf("data not copied correctly %v, %v, %v != %v, %v, %v", t0, t1, t2, r0, r1, r2)
	}
}
