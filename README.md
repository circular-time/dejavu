# Déjà vu
A zero-allocation, space-efficient, logarithmic-time solution in Go to the
exact membership query problem, for when you need to be 100% sure something has
been seen before. For when bloom filters just won't make the cut due to their
probabilistic nature.

## TL;DR
`dejavu` is a Go module that aims to provide an efficient and elegant way of
answering the following yes-no question: "Is element _x_ a member of the set
_S_?"

```go
package main

import (
	"fmt"
	"net"

	"github.com/circular-time/dejavu"
)

func main() {
	var (
		cache *dejavu.Cache = dejavu.NewCache128(1 << 22) // 4,194,304 elements
		seen  bool
		value = []byte(net.IPv6loopback)
	)

	fmt.Println(
		cache.Size(),
	)
	// 92274688
	// (88 MiB)

	seen, _ = cache.Recall(value)

	fmt.Println(seen)
	// false

	cache.Insert(value)

	seen, _ = cache.Recall(value)

	fmt.Println(seen)
	// true

	fmt.Println(
		cache.Length(),
	)
	// 1
}
```

## Caveats
`dejavu` requires cached values to be of fixed length. Users working with data
of varying sizes should consider caching not those values, but their hashes
instead.

Furthermore, it stores data in a binary tree that is not self-balancing. Values
that naturally occur in lexicographical order, such as time-based UUIDs, should
be hashed (or have their bytes rearranged so that the random bits come first)
before caching for optimal performance.

## Space complexity
`dejavu.Cache` occupies exactly _n_ (log _k_ + 2 log _n_) bits of memory to
store _n_ elements out of a set of _k_ possibilities. For example, it would
take 88 MiB to maintain a cache of more than four million IPv6 addresses.

The additional 2 log _n_ per element above the theoretical baseline of log _k_
is incurred to maintain a binary tree structure, as a trade-off against time
complexity.

The Go runtime has its own overheads that add to total memory consumption.

## Time complexity
`cache.Insert()` and `cache.Recall()` are operations in a binary tree averaging
an equivalent time complexity of _O_(log _n_).

In the worst-case scenario where values are cached in ascending/descending
order, the binary tree would end up a linked list with complexity _O_(_n_), in
which case it would be more space-efficient to use a regular list of complexity
Ω(_n_ log _k_).

## Benchmarks
```txt
goos: linux
goarch: arm64
pkg: github.com/circular-time/dejavu
BenchmarkCacheInsert
BenchmarkCacheInsert-2     2290011     739.7 ns/op     0 B/op     0 allocs/op
BenchmarkCacheRecall
BenchmarkCacheRecall-2     2246784     759.0 ns/op     0 B/op     0 allocs/op
```

Experiments suggest that `dejavu.Cache` is roughly two times more
memory-efficient than a Go `map[[16]byte]struct{}` with the same number of
keys, and somewhere in the region of ten times so compared to an LMDB instance
holding the same number of records consisting of 128-bit keys and 0-bit values.

```go
package main

import (
	"testing"

	"github.com/circular-time/dejavu"
)

const (
	n = 1 << 22 // 2^22 = 4,194,304
)

func BenchmarkDejavu(b *testing.B) {
	var (
		i int
	)

	for i = 0; i < b.N; i++ {
		dejavu.NewCache128(n)
	}
}

func BenchmarkMap(b *testing.B) {
	var (
		i int
	)

	for i = 0; i < b.N; i++ {
		_ = make(map[[16]byte]struct{}, n)
	}
}
```

```txt
BenchmarkDejavu
BenchmarkDejavu-2    1105    1475508 ns/op     92274688 B/op    1 allocs/op
BenchmarkMap
BenchmarkMap-2        592    2558122 ns/op    160432152 B/op    2 allocs/op
```
