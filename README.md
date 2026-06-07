# reflect

[![PkgGoDev](https://pkg.go.dev/badge/golang.design/x/reflect)](https://pkg.go.dev/golang.design/x/reflect)

Package reflect provides a generic `DeepCopy` for Go, the external
implementation of proposal [go.dev/issue/51520](https://go.dev/issue/51520).

```go
import "golang.design/x/reflect"
```

## Usage

```go
src := &Node{Val: 1, Next: &Node{Val: 2}}
dst := reflect.DeepCopy(src) // a fully independent copy
```

`DeepCopy` copies `src` to a freshly allocated value of the same type and
returns it:

```go
func DeepCopy[T any](src T, opts ...DeepCopyOption) (dst T)
```

Two values of identical type are deeply copied as follows:

- Numbers, bools and strings are copied and have a different underlying memory
  address.
- Slices and arrays are deeply copied, including their elements.
- Maps are deeply copied for all keys and values.
- Pointers are deeply copied for the pointed value, and the copy points to the
  deeply copied value.
- Structs are deeply copied for all fields, including unexported ones.
- Interfaces are deeply copied if the underlying type can be deeply copied.

A few cases mean a deeply copied value is not necessarily `DeepEqual` to the
source:

1. Func values still refer to the same function.
2. Bidirectional channels are replaced by newly created channels.
3. One-way (send- or receive-only) channels still refer to the same channel.

## Customization

Stateful objects (`sync.Mutex`, `os.File`, `net.Conn`, `js.Value`, ...) and
singletons should usually not be copied by their memory representation. Use the
following options to override the default behavior per type:

```go
dst := reflect.DeepCopy(src,
	// Keep a singleton shared by reference instead of copying it.
	reflect.RetainType[*Config](),
	// Reset a stateful value to its zero value (e.g. an unlocked mutex).
	reflect.ZeroType[sync.Mutex](),
	// General escape hatch: provide arbitrary per-type copy logic. T may be a
	// concrete type or an interface (applies to any implementing dynamic type).
	reflect.WithCopyFunc(func(c net.Conn) net.Conn { return reconnect(c) }),
)
```

The type parameter may be a concrete type or an interface; an interface applies
to every value whose dynamic type implements it. `RetainType` and `ZeroType` are
thin wrappers over `WithCopyFunc`.

Additional options control structural behavior:

- `DisallowCopyUnexported()` skips unexported struct fields instead of copying them.
- `DisallowCopyCircular()` panics on circular structures instead of handling them.
- `DisallowCopyBidirectionalChan()` keeps the source channel instead of creating a new one.
- `DisallowType[T]()` panics if a value of type T (concrete or interface) is encountered.

## Design notes

These answer the questions raised in
[go.dev/issue/51520](https://go.dev/issue/51520):

- **Circular and shared structures** are handled by tracking already-copied
  pointers, so the copy preserves the original pointer aliasing. Opt out with
  `DisallowCopyCircular()`.
- **The destination can never alias the source.** `DeepCopy` always allocates a
  fresh result, so the "destination reachable from source" hazard does not arise.
- **Unexported fields** are copied (this requires the `unsafe` package, since
  `reflect` cannot set unexported fields directly). Opt out with
  `DisallowCopyUnexported()`.
- **Singletons and stateful objects** are the cases where blindly copying memory
  is wrong; `RetainType`, `ZeroType` and `WithCopyFunc` exist to handle them
  without baking a `Clone`-style interface into the type system.
- **Cost.** Deep copy is not free: it allocates roughly one allocation per
  copied element. See `bench_test.go`. Prefer a hand-written copy on hot paths.

## License

MIT | &copy; 2022 The golang.design Initiative Authors, written by [Changkun Ou](https://changkun.de).
