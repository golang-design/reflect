// Copyright 2022 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.

package reflect

import (
	"sync"
	"testing"
)

type singleton struct{ id int }

// TestRetainType verifies that RetainType keeps a value shared by reference
// in the copy instead of deep copying it (the singleton case from #51520).
func TestRetainType(t *testing.T) {
	shared := &singleton{id: 42}
	src := struct {
		S    *singleton
		Data []int
	}{S: shared, Data: []int{1, 2, 3}}

	// Without the option, the singleton is deep copied (different pointer).
	plain := DeepCopy(src)
	if plain.S == shared {
		t.Fatal("without RetainType the singleton should be deep copied")
	}

	// With the option, the singleton pointer is retained.
	dst := DeepCopy(src, RetainType[*singleton]())
	if dst.S != shared {
		t.Fatalf("RetainType should share the singleton: got %p, want %p", dst.S, shared)
	}
	// Surrounding data is still deep copied.
	if &dst.Data[0] == &src.Data[0] {
		t.Fatal("non-retained data should still be deep copied")
	}
}

// TestZeroType verifies that ZeroType substitutes a fresh zero value, so a
// copied sync.Mutex comes back unlocked (the stateful-object case from #51520).
func TestZeroType(t *testing.T) {
	type guarded struct {
		mu  sync.Mutex
		val int
	}
	src := &guarded{val: 7}
	src.mu.Lock() // src is held

	dst := DeepCopy(src, ZeroType[sync.Mutex]())
	if dst.val != 7 {
		t.Fatalf("non-zeroed field should be copied: got %d, want 7", dst.val)
	}
	if !dst.mu.TryLock() {
		t.Fatal("ZeroType should reset the copied mutex to unlocked")
	}
	dst.mu.Unlock()
	src.mu.Unlock()
}

// TestWithCopyFunc verifies the general escape hatch with a concrete type.
func TestWithCopyFunc(t *testing.T) {
	type box struct{ n int }
	src := &box{n: 1}
	dst := DeepCopy(src, WithCopyFunc(func(b *box) *box {
		return &box{n: b.n + 100}
	}))
	if dst.n != 101 {
		t.Fatalf("WithCopyFunc not applied: got %d, want 101", dst.n)
	}
}

type animal interface{ sound() string }

type dog struct{ name string }

func (d *dog) sound() string { return "woof" }

// TestWithCopyFuncInterface verifies that an interface-typed WithCopyFunc
// applies to values whose dynamic type implements the interface.
func TestWithCopyFuncInterface(t *testing.T) {
	src := struct{ A animal }{A: &dog{name: "rex"}}
	called := false
	dst := DeepCopy(src, WithCopyFunc(func(a animal) animal {
		called = true
		return a // share
	}))
	if !called {
		t.Fatal("interface WithCopyFunc was not invoked for implementing type")
	}
	if dst.A != src.A {
		t.Fatal("interface WithCopyFunc returning the source should share it")
	}
}

// TestDisallowTypeInterface verifies that DisallowType works for interface
// types, which the previous value-based DisallowTypes could not express
// (reflect.TypeOf of a nil interface value is nil).
func TestDisallowTypeInterface(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("DisallowType[animal] should panic on an implementing value")
		}
	}()
	src := struct{ A animal }{A: &dog{name: "rex"}}
	_ = DeepCopy(src, DisallowType[animal]())
}

// TestWithCopyFuncNilInterfaceField is a regression test: a WithCopyFunc that
// returns a nil interface for a struct field must not panic when setting the
// unexported field.
func TestWithCopyFuncNilInterfaceField(t *testing.T) {
	type holder struct{ a animal }
	src := &holder{a: &dog{name: "rex"}}
	dst := DeepCopy(src, WithCopyFunc(func(a animal) animal { return nil }))
	if dst.a != nil {
		t.Fatalf("expected nil interface field, got %v", dst.a)
	}
}
