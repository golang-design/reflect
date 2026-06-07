// Copyright 2022 The golang.design Initiative Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package reflect

import "testing"

// TestDeepCopyNilPointer is a regression test for golang-design/reflect#2:
// copying a nil pointer must yield a nil pointer, not a non-nil pointer to a
// zero value.
func TestDeepCopyNilPointer(t *testing.T) {
	type ListNode struct {
		Val  int
		Next *ListNode
	}
	var src *ListNode
	if dst := DeepCopy(src); dst != nil {
		t.Fatalf("DeepCopy(nil pointer) = %#v, want nil", dst)
	}
}

// TestDeepCopyBidirectionalChan guards the copyChan fix: a bidirectional
// channel must be replaced by a freshly created channel, not aliased to the
// source channel.
func TestDeepCopyBidirectionalChan(t *testing.T) {
	src := make(chan int, 1)
	dst := DeepCopy(src)
	if dst == src {
		t.Fatal("DeepCopy(bidirectional chan) aliased the source channel, want a new channel")
	}
	if cap(dst) != cap(src) {
		t.Fatalf("DeepCopy chan cap = %d, want %d", cap(dst), cap(src))
	}
}
