// Copyright 2022 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.

package reflect

import (
	"sync"
	"testing"
)

type singleton struct{ id int }

// TestRetainTypes verifies that RetainTypes keeps a value shared by reference
// in the copy instead of deep copying it (the singleton case from #51520).
func TestRetainTypes(t *testing.T) {
	shared := &singleton{id: 42}
	src := struct {
		S    *singleton
		Data []int
	}{S: shared, Data: []int{1, 2, 3}}

	// Without the option, the singleton is deep copied (different pointer).
	plain := DeepCopy(src)
	if plain.S == shared {
		t.Fatal("without RetainTypes the singleton should be deep copied")
	}

	// With the option, the singleton pointer is retained.
	dst := DeepCopy(src, RetainTypes(shared))
	if dst.S != shared {
		t.Fatalf("RetainTypes should share the singleton: got %p, want %p", dst.S, shared)
	}
	// Surrounding data is still deep copied.
	if &dst.Data[0] == &src.Data[0] {
		t.Fatal("non-retained data should still be deep copied")
	}
}

// TestZeroTypes verifies that ZeroTypes substitutes a fresh zero value, so a
// copied sync.Mutex comes back unlocked (the stateful-object case from #51520).
func TestZeroTypes(t *testing.T) {
	type guarded struct {
		mu  sync.Mutex
		val int
	}
	src := &guarded{val: 7}
	src.mu.Lock() // src is held

	dst := DeepCopy(src, ZeroTypes(sync.Mutex{}))
	if dst.val != 7 {
		t.Fatalf("non-zeroed field should be copied: got %d, want 7", dst.val)
	}
	if !dst.mu.TryLock() {
		t.Fatal("ZeroTypes should reset the copied mutex to unlocked")
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
