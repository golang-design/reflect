// Copyright 2022 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.

package reflect

import "testing"

type benchNode struct {
	ID       int
	Name     string
	Weights  []float64
	Children []*benchNode
}

func newBenchTree(depth, fanout int) *benchNode {
	n := &benchNode{
		ID:      depth,
		Name:    "node",
		Weights: []float64{1, 2, 3, 4},
	}
	if depth <= 0 {
		return n
	}
	for i := 0; i < fanout; i++ {
		n.Children = append(n.Children, newBenchTree(depth-1, fanout))
	}
	return n
}

func BenchmarkDeepCopyPrimitive(b *testing.B) {
	src := 42
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = DeepCopy(src)
	}
}

func BenchmarkDeepCopySlice(b *testing.B) {
	src := make([]int, 1024)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = DeepCopy(src)
	}
}

func BenchmarkDeepCopyMap(b *testing.B) {
	src := make(map[int]string, 256)
	for i := 0; i < 256; i++ {
		src[i] = "value"
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = DeepCopy(src)
	}
}

func BenchmarkDeepCopyTree(b *testing.B) {
	// A tree of (3+1) levels with fanout 4: 1+4+16+64 = 85 nodes.
	src := newBenchTree(3, 4)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = DeepCopy(src)
	}
}
