<p align="center">
  <img src="logo.svg" width="40%" border="0" alt="avo" />
  <br />
  <a href="https://app.shippable.com/github/mmcloughlin/avo/dashboard"><img src="https://api.shippable.com/projects/5bf9e8f059e32e0700ec360f/badge?branch=master" alt="Build Status" /></a>
  <a href="http://godoc.org/github.com/mmcloughlin/avo"><img src="http://img.shields.io/badge/godoc-reference-5272B4.svg" alt="GoDoc" /></a>
</p>

<p align="center">Generate x86 Assembly with Go</p>

`avo` makes high-performance Go assembly easier to write, review and maintain. The `avo` package presents a familiar assembly-like interface that simplifies development without sacrificing performance:

* **Use Go control structures** for assembly generation; `avo` programs _are_ Go programs
* **Register allocation**: write functions with virtual registers and `avo` assigns physical registers for you
* **Automatically load arguments and store return values**: ensure memory offsets are always correct even for complex data structures
* **Generation of stub files** to interface with your Go package

_Note: APIs subject to change while `avo` is still in an experimental phase. You can use it to build [real things](examples) but we suggest you pin a version with your package manager of choice._

## Quick Start

Install `avo` with `go get`:

```
$ go get -u github.com/mmcloughlin/avo
```

`avo` assembly generators are pure Go programs. Here's a function that adds two `uint64` values:

[embedmd]:# (examples/add/asm.go)
```go
// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
)

func main() {
	TEXT("Add", NOSPLIT, "func(x, y uint64) uint64")
	Doc("Add adds x and y.")
	x := Load(Param("x"), GP64())
	y := Load(Param("y"), GP64())
	ADDQ(x, y)
	Store(y, ReturnIndex(0))
	RET()
	Generate()
}
```

`go run` this code to see the assembly output. To integrate this into the rest of your Go package we recommend a [`go:generate`](https://blog.golang.org/generate) line to produce the assembly and the corresponding Go stub file.

[embedmd]:# (examples/add/add_test.go go /.*go:generate.*/)
```go
//go:generate go run asm.go -out add.s -stubs stub.go
```

After running `go generate` the [`add.s`](examples/add/add.s) file will contain the Go assembly.

[embedmd]:# (examples/add/add.s)
```s
// Code generated by command: go run asm.go -out add.s -stubs stub.go. DO NOT EDIT.

#include "textflag.h"

// func Add(x uint64, y uint64) uint64
TEXT ·Add(SB), NOSPLIT, $0-24
	MOVQ	x(FP), AX
	MOVQ	y+8(FP), CX
	ADDQ	AX, CX
	MOVQ	CX, ret+16(FP)
	RET
```

The same call will produce the stub file [`stub.go`](examples/add/stub.go) which will enable the function to be called from your Go code.

[embedmd]:# (examples/add/stub.go)
```go
// Code generated by command: go run asm.go -out add.s -stubs stub.go. DO NOT EDIT.

package add

// Add adds x and y.
func Add(x uint64, y uint64) uint64
```

See the [`examples/add`](examples/add) directory for the complete working example.

## Examples

See [`examples`](examples) for the full suite of examples.

### Slice Sum

Sum a slice of `uint64`s:

[embedmd]:# (examples/sum/asm.go /func main/ /^}/)
```go
func main() {
	TEXT("Sum", NOSPLIT, "func(xs []uint64) uint64")
	Doc("Sum returns the sum of the elements in xs.")
	ptr := Load(Param("xs").Base(), GP64())
	n := Load(Param("xs").Len(), GP64())

	// Initialize sum register to zero.
	s := GP64()
	XORQ(s, s)

	// Loop until zero bytes remain.
	Label("loop")
	CMPQ(n, Imm(0))
	JE(LabelRef("done"))

	// Load from pointer and add to running sum.
	ADDQ(Mem{Base: ptr}, s)

	// Advance pointer, decrement byte count.
	ADDQ(Imm(8), ptr)
	DECQ(n)
	JMP(LabelRef("loop"))

	// Store sum to return value.
	Label("done")
	Store(s, ReturnIndex(0))
	RET()
	Generate()
}
```

The result from this code generator is:

[embedmd]:# (examples/sum/sum.s)
```s
// Code generated by command: go run asm.go -out sum.s -stubs stub.go. DO NOT EDIT.

#include "textflag.h"

// func Sum(xs []uint64) uint64
TEXT ·Sum(SB), NOSPLIT, $0-32
	MOVQ	xs_base(FP), AX
	MOVQ	xs_len+8(FP), CX
	XORQ	DX, DX
loop:
	CMPQ	CX, $0x00
	JE	done
	ADDQ	(AX), DX
	ADDQ	$0x08, AX
	DECQ	CX
	JMP	loop
done:
	MOVQ	DX, ret+24(FP)
	RET
```

Full example at [`examples/sum`](examples/sum).

### Parameter Load/Store

`avo` provides deconstruction of complex data datatypes into components. For example, load the length of a string argument with:

[embedmd]:# (examples/args/asm.go go /.*TEXT.*StringLen/ /Load.*/)
```go
	TEXT("StringLen", NOSPLIT, "func(s string) int")
	strlen := Load(Param("s").Len(), GP64())
```

Index an array:

[embedmd]:# (examples/args/asm.go go /.*TEXT.*ArrayThree/ /Load.*/)
```go
	TEXT("ArrayThree", NOSPLIT, "func(a [7]uint64) uint64")
	a3 := Load(Param("a").Index(3), GP64())
```

Access a struct field (provided you have loaded your package with the `Package` function):

[embedmd]:# (examples/args/asm.go go /.*TEXT.*FieldFloat64/ /Load.*/)
```go
	TEXT("FieldFloat64", NOSPLIT, "func(s Struct) float64")
	f64 := Load(Param("s").Field("Float64"), XMM())
```

Component accesses can be arbitrarily nested:

[embedmd]:# (examples/args/asm.go go /.*TEXT.*FieldArrayTwoBTwo/ /Load.*/)
```go
	TEXT("FieldArrayTwoBTwo", NOSPLIT, "func(s Struct) byte")
	b2 := Load(Param("s").Field("Array").Index(2).Field("B").Index(2), GP8())
```

Very similar techniques apply to writing return values. See [`examples/args`](examples/args) and [`examples/returns`](examples/returns) for more.

### SHA-1

[SHA-1](https://en.wikipedia.org/wiki/SHA-1) is an excellent example of how powerful this kind of technique can be. The following is a (hopefully) clearly structured implementation of SHA-1 in `avo`, which ultimately generates a [1000+ line impenetrable assembly file](examples/sha1/sha1.s).

[embedmd]:# (examples/sha1/asm.go /func main/ /^}/)
```go
func main() {
	TEXT("block", 0, "func(h *[5]uint32, m []byte)")
	Doc("block SHA-1 hashes the 64-byte message m into the running state h.")
	h := Mem{Base: Load(Param("h"), GP64())}
	m := Mem{Base: Load(Param("m").Base(), GP64())}

	// Store message values on the stack.
	w := AllocLocal(64)
	W := func(r int) Mem { return w.Offset((r % 16) * 4) }

	// Load initial hash.
	hash := [5]Register{GP32(), GP32(), GP32(), GP32(), GP32()}
	for i, r := range hash {
		MOVL(h.Offset(4*i), r)
	}

	// Initialize registers.
	a, b, c, d, e := GP32(), GP32(), GP32(), GP32(), GP32()
	for i, r := range []Register{a, b, c, d, e} {
		MOVL(hash[i], r)
	}

	// Generate round updates.
	quarter := []struct {
		F func(Register, Register, Register) Register
		K uint32
	}{
		{choose, 0x5a827999},
		{xor, 0x6ed9eba1},
		{majority, 0x8f1bbcdc},
		{xor, 0xca62c1d6},
	}

	for r := 0; r < 80; r++ {
		q := quarter[r/20]

		// Load message value.
		u := GP32()
		if r < 16 {
			MOVL(m.Offset(4*r), u)
			BSWAPL(u)
		} else {
			MOVL(W(r-3), u)
			XORL(W(r-8), u)
			XORL(W(r-14), u)
			XORL(W(r-16), u)
			ROLL(U8(1), u)
		}
		MOVL(u, W(r))

		// Compute the next state register.
		t := GP32()
		MOVL(a, t)
		ROLL(U8(5), t)
		ADDL(q.F(b, c, d), t)
		ADDL(e, t)
		ADDL(U32(q.K), t)
		ADDL(u, t)

		// Update registers.
		ROLL(Imm(30), b)
		a, b, c, d, e = t, a, b, c, d
	}

	// Final add.
	for i, r := range []Register{a, b, c, d, e} {
		ADDL(r, hash[i])
	}

	// Store results back.
	for i, r := range hash {
		MOVL(r, h.Offset(4*i))
	}
	RET()

	Generate()
}
```

This relies on the bitwise functions that are defined as subroutines. For example here is bitwise `choose`; the others are similar.

[embedmd]:# (examples/sha1/asm.go /func choose/ /^}/)
```go
func choose(b, c, d Register) Register {
	r := GP32()
	MOVL(d, r)
	XORL(c, r)
	ANDL(b, r)
	XORL(d, r)
	return r
}
```

See the complete code at [`examples/sha1`](examples/sha1).

### Real Examples

* **[fnv1a](examples/fnv1a):** [FNV-1a](https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function#FNV-1a_hash) hash function.
* **[dot](examples/dot):** Vector dot product.
* **[geohash](examples/geohash):** Integer [geohash](https://en.wikipedia.org/wiki/Geohash) encoding.
* **[stadtx](examples/stadtx):** [`StadtX` hash](https://github.com/demerphq/BeagleHash) port from [dgryski/go-stadtx](https://github.com/dgryski/go-stadtx).

## Contributing

Contributions to `avo` are welcome:

* Feedback from using `avo` in a real project is incredibly valuable.
* [Submit bug reports](https://github.com/mmcloughlin/avo/issues/new) to the issues page.
* Pull requests accepted. Take a look at outstanding [issues](https://github.com/mmcloughlin/avo/issues) for ideas (especially the ["good first issue"](https://github.com/mmcloughlin/avo/labels/good%20first%20issue) label).

## Credits

Inspired by the [PeachPy](https://github.com/Maratyszcza/PeachPy) and [asmjit](https://github.com/asmjit/asmjit) projects.

## License

`avo` is available under the [BSD 3-Clause License](LICENSE).
