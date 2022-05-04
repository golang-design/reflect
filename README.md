# reflect

Package reflect implements the proposal [go.dev/issue/51520](https://go.dev/issue/51520).

```go
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
// 1) Func values are still refer to the same function
// 2) Chan values are replaced by newly created channels
// 3) One-way Chan values (receive or read-only) values are still refer
//    to the same channel
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
func DeepCopy[T any](src T, opts ...DeepCopyOption) (dst T)
```

_Warning_: Not largely tested. Use it with care.

## License

MIT | &copy; 2022 The golang.design Initiative Authors, written by Changkun Ou.