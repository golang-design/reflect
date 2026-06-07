// Copyright 2022 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

// Package reflect implements the proposal https://go.dev/issue/51520.
//
// Warning: Not largely tested. Use it with care.
package reflect

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

// DeepCopyOption represents an option to customize deep copied results.
type DeepCopyOption func(opt *copyConfig)

type copyConfig struct {
	disallowCopyUnexported        bool
	disallowCopyCircular          bool
	disallowCopyBidirectionalChan bool
	disallowCopyTypes             []reflect.Type
	copyFuncs                     []copyFuncEntry
}

// copyFuncEntry binds a type (concrete or interface) to a custom copy
// function that replaces the default deep copy behavior for that type.
type copyFuncEntry struct {
	typ reflect.Type
	fn  func(src any) (dst any)
}

// customCopy returns the custom copy function registered for srcType, if any.
// An exact (concrete) type match takes precedence over an interface match;
// among interfaces the first registered match wins.
func (c *copyConfig) customCopy(srcType reflect.Type) (func(any) any, bool) {
	var iface func(any) any
	found := false
	for i := range c.copyFuncs {
		e := c.copyFuncs[i]
		if e.typ == srcType {
			return e.fn, true
		}
		if !found && e.typ.Kind() == reflect.Interface && srcType.Implements(e.typ) {
			iface, found = e.fn, true
		}
	}
	return iface, found
}

// DisallowCopyUnexported returns a DeepCopyOption that disables the behavior
// of copying unexported fields.
func DisallowCopyUnexported() DeepCopyOption {
	return func(opt *copyConfig) {
		opt.disallowCopyUnexported = true
	}
}

// DisallowCopyCircular returns a DeepCopyOption that disables the behavior
// of copying circular structures.
func DisallowCopyCircular() DeepCopyOption {
	return func(opt *copyConfig) {
		opt.disallowCopyCircular = true
	}
}

// DisallowCopyBidirectionalChan returns a DeepCopyOption that disables
// the behavior of producing new channel when a bidirectional channel is copied.
func DisallowCopyBidirectionalChan() DeepCopyOption {
	return func(opt *copyConfig) {
		opt.disallowCopyBidirectionalChan = true
	}
}

// DisallowTypes returns a DeepCopyOption that disallows copying any types
// that are in given values.
func DisallowTypes(val ...any) DeepCopyOption {
	return func(opt *copyConfig) {
		for i := range val {
			opt.disallowCopyTypes = append(opt.disallowCopyTypes, reflect.TypeOf(val[i]))
		}
	}
}

// WithCopyFunc returns a DeepCopyOption that replaces the default deep copy
// behavior for values of type T with fn. This is the general escape hatch for
// types that must not be copied by their memory representation, such as
// singletons that must remain shared, or stateful objects like sync.Mutex,
// os.File, net.Conn, and js.Value.
//
// T may be a concrete type or an interface type. When T is an interface, fn
// applies to every value whose dynamic type implements T (unless a more
// specific concrete WithCopyFunc is also registered for that value's type).
//
// RetainTypes and ZeroTypes are convenience wrappers over WithCopyFunc.
func WithCopyFunc[T any](fn func(src T) (dst T)) DeepCopyOption {
	t := reflect.TypeOf((*T)(nil)).Elem()
	wrapped := func(src any) any { return fn(src.(T)) }
	return func(opt *copyConfig) {
		opt.copyFuncs = append(opt.copyFuncs, copyFuncEntry{t, wrapped})
	}
}

// RetainTypes returns a DeepCopyOption that retains (shares by reference) the
// values of the given types instead of deep copying them. Use it for pointers
// to singletons that must remain singletons in the copied result.
func RetainTypes(val ...any) DeepCopyOption {
	return func(opt *copyConfig) {
		for i := range val {
			t := reflect.TypeOf(val[i])
			opt.copyFuncs = append(opt.copyFuncs, copyFuncEntry{
				t, func(src any) any { return src },
			})
		}
	}
}

// ZeroTypes returns a DeepCopyOption that substitutes a fresh zero value for
// the values of the given types instead of deep copying them. Use it for
// stateful objects whose copied state would be invalid or dangerous to share,
// such as resetting a sync.Mutex to its unlocked state.
func ZeroTypes(val ...any) DeepCopyOption {
	return func(opt *copyConfig) {
		for i := range val {
			t := reflect.TypeOf(val[i])
			opt.copyFuncs = append(opt.copyFuncs, copyFuncEntry{
				t, func(src any) any { return reflect.Zero(t).Interface() },
			})
		}
	}
}

// DeepCopy copies src to dst recursively.
//
// Two values of identical type are deeply copied if one of the following
// cases apply.
//
// Numbers, bools, strings are deeply copied and have different underlying
// memory address.
//
// Slice and Array values are deeply copied, including its elements.
//
// Map values are deeply copied for all of its key and corresponding
// values.
//
// Pointer values are deeply copied for their pointed value, and the
// pointer points to the deeply copied value.
//
// Struct values are deeply copied for all fields, including exported
// and unexported.
//
// Interface values are deeply copied if the underlying type can be
// deeply copied.
//
// There are a few exceptions that may result in a deeply copied value not
// deeply equal (asserted by DeepEqual(dst, src)) to the source value:
//
//  1. Func values are still refer to the same function
//  2. Chan values are replaced by newly created channels
//  3. One-way Chan values (receive or read-only) values are still refer
//     to the same channel
//
// Note that while correct uses of DeepCopy do exist, they are not rare.
// The use of DeepCopy often indicates the copying object does not contain
// a singleton or is never meant to be copied, such as sync.Mutex, os.File,
// net.Conn, js.Value, etc. In these cases, the copied value retains the
// memory representations of the source value but may result in unexpected
// consequences in follow-up usage, the caller should clear these values
// depending on their usage context.
//
// To change these predefined behaviors, use provided DeepCopyOption.
func DeepCopy[T any](src T, opts ...DeepCopyOption) (dst T) {
	ptrs := map[uintptr]any{}
	conf := &copyConfig{}
	for _, opt := range opts {
		opt(conf)
	}

	ret := copyAny(src, ptrs, conf)
	if v, ok := ret.(T); ok {
		dst = v
		return
	}
	panic(fmt.Sprintf("reflect: internal error: copied value is not typed in %T, got %T", src, ret))
}

func copyAny(src any, ptrs map[uintptr]any, copyConf *copyConfig) (dst any) {
	if len(copyConf.disallowCopyTypes) != 0 {
		for i := range copyConf.disallowCopyTypes {
			if reflect.TypeOf(src) == copyConf.disallowCopyTypes[i] {
				panic(fmt.Sprintf("reflect: deep copying type %T is disallowed", src))
			}
		}
	}

	v := reflect.ValueOf(src)
	if !v.IsValid() {
		return src
	}

	// A caller-provided copy function overrides the default behavior for
	// its type, e.g. retaining singletons or zeroing stateful objects.
	if fn, ok := copyConf.customCopy(v.Type()); ok {
		return fn(src)
	}

	// Look up the corresponding copy function.
	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.Func:
		dst = copyPremitive(src, ptrs, copyConf)
	case reflect.String:
		dst = strings.Clone(src.(string))
	case reflect.Slice:
		dst = copySlice(src, ptrs, copyConf)
	case reflect.Array:
		dst = copyArray(src, ptrs, copyConf)
	case reflect.Map:
		dst = copyMap(src, ptrs, copyConf)
	case reflect.Ptr:
		dst = copyPointer(src, ptrs, copyConf)
	case reflect.Struct:
		dst = copyStruct(src, ptrs, copyConf)
	case reflect.Interface:
		dst = copyAny(src, ptrs, copyConf)
	case reflect.Chan:
		dst = copyChan(src, ptrs, copyConf)
	default:
		panic(fmt.Sprintf("reflect: internal error: unknown type %v", v.Kind()))
	}
	return
}

func copyPremitive(src any, ptr map[uintptr]any, copyConf *copyConfig) (dst any) {
	kind := reflect.ValueOf(src).Kind()
	switch kind {
	case reflect.Array, reflect.Chan, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Struct, reflect.UnsafePointer:
		panic(fmt.Sprintf("reflect: internal error: type %v is not a primitive", kind))
	}
	dst = src
	return
}

func copySlice(x any, ptrs map[uintptr]any, copyConf *copyConfig) any {
	v := reflect.ValueOf(x)
	kind := v.Kind()
	if kind != reflect.Slice {
		panic(fmt.Sprintf("reflect: internal error: type %v is not a slice", kind))
	}

	size := v.Len()
	t := reflect.TypeOf(x)
	dc := reflect.MakeSlice(t, size, size)
	for i := 0; i < size; i++ {
		iv := reflect.ValueOf(copyAny(v.Index(i).Interface(), ptrs, copyConf))
		if iv.IsValid() {
			dc.Index(i).Set(iv)
		}
	}
	return dc.Interface()
}

func copyArray(x any, ptrs map[uintptr]any, copyConf *copyConfig) any {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Array {
		panic(fmt.Errorf("reflect: internal error: must be an Array; got %v", v.Kind()))
	}
	t := reflect.TypeOf(x)
	size := t.Len()
	dc := reflect.New(reflect.ArrayOf(size, t.Elem())).Elem()
	for i := 0; i < size; i++ {
		item := copyAny(v.Index(i).Interface(), ptrs, copyConf)
		dc.Index(i).Set(reflect.ValueOf(item))
	}
	return dc.Interface()
}

func copyMap(x any, ptrs map[uintptr]any, copyConf *copyConfig) any {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Map {
		panic(fmt.Errorf("reflect: internal error: must be a Map; got %v", v.Kind()))
	}
	t := reflect.TypeOf(x)
	dc := reflect.MakeMapWithSize(t, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		item := copyAny(iter.Value().Interface(), ptrs, copyConf)
		k := copyAny(iter.Key().Interface(), ptrs, copyConf)
		dc.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(item))
	}
	return dc.Interface()
}

func copyPointer(x any, ptrs map[uintptr]any, copyConf *copyConfig) any {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Pointer {
		panic(fmt.Errorf("reflect: internal error: must be a Pointer or Ptr; got %v", v.Kind()))
	}
	if v.IsNil() {
		return x
	}
	addr := uintptr(v.UnsafePointer())
	if dc, ok := ptrs[addr]; ok {
		if copyConf.disallowCopyCircular {
			panic("reflect: deep copy dircular value is disallowed")
		}
		return dc
	}
	t := reflect.TypeOf(x)
	dc := reflect.New(t.Elem())
	ptrs[addr] = dc.Interface()
	item := copyAny(v.Elem().Interface(), ptrs, copyConf)
	iv := reflect.ValueOf(item)
	if iv.IsValid() {
		dc.Elem().Set(reflect.ValueOf(item))
	}
	return dc.Interface()
}

func copyStruct(x any, ptrs map[uintptr]any, copyConf *copyConfig) any {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Struct {
		panic(fmt.Errorf("reflect: internal error: must be a Struct; got %v", v.Kind()))
	}
	t := reflect.TypeOf(x)
	dc := reflect.New(t)
	for i := 0; i < t.NumField(); i++ {
		if copyConf.disallowCopyUnexported {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			item := copyAny(v.Field(i).Interface(), ptrs, copyConf)
			dc.Elem().Field(i).Set(reflect.ValueOf(item))
		} else {
			item := copyAny(valueInterfaceUnsafe(v.Field(i)), ptrs, copyConf)
			setField(dc.Elem().Field(i), reflect.ValueOf(item))
		}
	}
	return dc.Elem().Interface()
}

func copyChan(x any, ptrs map[uintptr]any, copyConf *copyConfig) any {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Chan {
		panic(fmt.Errorf("reflect: internal error: must be a Chan; got %v", v.Kind()))
	}
	t := reflect.TypeOf(x)
	dir := t.ChanDir()
	var dc any
	switch dir {
	case reflect.BothDir:
		if !copyConf.disallowCopyBidirectionalChan {
			dc = reflect.MakeChan(t, v.Cap()).Interface()
		}
	case reflect.SendDir, reflect.RecvDir:
		dc = x
	}
	return dc
}

// valueInterfaceUnsafe overpasses the reflect package check regarding
// unexported methods.
func valueInterfaceUnsafe(v reflect.Value) any {
	return reflect_valueInterface(v, false)
}

// setField sets the given value to the field value, regardless whether
// the filed is exported or not.
func setField(field reflect.Value, value reflect.Value) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(value)
}

//go:linkname reflect_valueInterface reflect.valueInterface
func reflect_valueInterface(v reflect.Value, safe bool) any
