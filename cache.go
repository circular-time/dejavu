package dejavu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

const (
	bitsPerByte  = 8
	maxUintLen32 = 4
)

// A Cache is a binary tree, held in memory as a byte array divided into
// segments of equal length, each representing a node in the tree.
//
// A node begins with a value of fixed length valLen, followed by two indices
// of length idxLen pointing to its left and right children.
//
// The Cache is insert-only and has a write lock to prevent concurrent writes.
type Cache struct {
	memory []byte // pointer to array in memory holding binary tree

	length int        // number of values cached
	mutex  sync.Mutex // write lock

	idxLen int // length of node indices, in number of bytes
	valLen int // length of values, in number of bytes

	maxCap int // maximum number of values that can be cached
}

// NewCache128 creates a new Cache that holds up to n 128-bit values in memory.
// Allocates an array of maximum size 96 GiB if n == math.MaxUint32.
func NewCache128(n uint32) *Cache {
	return newCache(128, n)
}

// Length returns the number of values currently cached.
func (c *Cache) Length() int {
	return c.length
}

// Size returns the size of the underlying array, in number of bytes.
func (c *Cache) Size() int {
	return len(c.memory)
}

// Insert caches a value.
func (c *Cache) Insert(value []byte) (e error) {
	if c.length == c.maxCap {
		e = fmt.Errorf("could not insert: no more free space left in cache")

		return
	}

	if len(value) != c.valLen {
		e = fmt.Errorf("could not insert: value length not equal to %d bytes",
			c.valLen,
		)

		return
	}

	c.mutex.Lock()

	defer c.mutex.Unlock()

	c.insert(0, value)

	return
}

// Recall returns true if a value has been cached, false otherwise.
func (c *Cache) Recall(value []byte) (cached bool, e error) {
	if len(value) != c.valLen {
		e = fmt.Errorf("could not recall: value length not equal to %d bytes",
			c.valLen,
		)

		return
	}

	return c.recall(0, value), nil
}

// Last returns the last-cached value, or an nil slice if the cache is empty.
func (c *Cache) Last() (value []byte) {
	switch c.length {
	case 0:
		return

	default:
		value = c.val(c.length - 1)
	}

	return
}

// Save writes all cached values to an [io.Writer] in the order of their
// insertion, after sending metadata about value length and quantity.
func (c *Cache) Save(writer io.Writer) (e error) {
	var (
		i int
	)

	c.mutex.Lock()

	defer c.mutex.Unlock()

	e = binary.Write(writer, binary.BigEndian,
		uint32(c.valLen),
	)
	if e != nil {
		return
	}

	e = binary.Write(writer, binary.BigEndian,
		uint32(c.length),
	)
	if e != nil {
		return
	}

	for i = 0; i < c.length; i++ {
		_, e = writer.Write(
			c.val(i),
		)
		if e != nil {
			return
		}
	}

	return
}

// Counterpart to Save, Load reads and inserts values from an [io.Reader],
// after verifying metadata about inbound value length and quantity.
func (c *Cache) Load(reader io.Reader) (e error) {
	var (
		i      uint32
		length uint32
		valLen uint32
		value  []byte
	)

	e = binary.Read(reader, binary.BigEndian, &valLen)
	if e != nil {
		return
	}

	e = binary.Read(reader, binary.BigEndian, &length)
	if e != nil {
		return
	}

	c.mutex.Lock()

	defer c.mutex.Unlock()

	if int(valLen) != c.valLen {
		e = fmt.Errorf("could not load: value length not equal to %d bytes",
			c.valLen,
		)

		return
	}

	value = make([]byte, valLen)

	if int(length) > (c.maxCap - c.length) {
		e = fmt.Errorf("could not load: not enough free space left in cache")

		return
	}

	for i = 0; i < length; i++ {
		_, e = reader.Read(value)
		if e != nil {
			return
		}

		c.insert(0, value)
	}

	return
}

func newCache(l uint8, n uint32) (c *Cache) {
	// Creates a new Cache that holds up to n l-bit values in memory.

	c = &Cache{
		idxLen: log(int(n), bitsPerByte) / bitsPerByte,
		valLen: int(l / bitsPerByte),

		maxCap: int(n),
	}

	c.memory = make([]byte,
		int(n)*c.nodeLen(),
	)

	return
}

func (c *Cache) insert(i int, val []byte) {
	// Appends a new node to the array by setting its value, and updates its
	// parent to point to it. Make sure this method is only called when the
	// mutex is locked!

	var (
		left bool
		next int
	)

	next, left = c.look(i, val)

	switch next {
	case -1: // do nothing; value already cached
		return

	case 0: // child does not exist; create child
		c.setVal(c.length, val)

		if left {
			c.setIdxL(i, c.length)
		} else {
			c.setIdxR(i, c.length)
		}

		c.length++

	default: // child exists; descend into child
		c.insert(next, val)
	}

	return
}

func (c *Cache) recall(i int, val []byte) bool {
	// Returns true if a node with value val is found; otherwise false.

	var (
		next int
	)

	next, _ = c.look(i, val)

	switch next {
	case -1: // value found
		return true

	case 0: // value not found
		return false

	default: // go deeper
		return c.recall(next, val)
	}
}

func (c *Cache) look(i int, val []byte) (int, bool) {
	// Returns either
	// (1) the index of the left child of the i-th node in the array, if
	//     val is less than the value of that node, or
	// (2) the index of the right child, if val is greater, or
	// (3) 0, if the relevant child does not exist, or
	// (4) -1, if val is equal to the value of that node, and
	// true if the index returned is of the left child of that node.

	switch bytes.Compare(c.val(i), val) {
	case 0:
		return -1, false

	case 1: // c.val(i) > val
		return c.idxL(i), true

	case -1: // c.val(i) < val
		return c.idxR(i), false
	}

	return 0, false
}

func (c *Cache) val(i int) []byte {
	// Returns the value of the i-th node in the array.

	var (
		valPos int = i * c.nodeLen()
	)

	return c.memory[valPos : valPos+c.valLen]
}

func (c *Cache) idxL(i int) int {
	// Returns the index of the left child of the i-th node in the array.

	var (
		idxPos int = i*c.nodeLen() + c.valLen
		idxVal uint32
	)

	idxVal = getUint32(c.memory[idxPos : idxPos+c.idxLen])

	return int(idxVal)
}

func (c *Cache) idxR(i int) int {
	// Returns the index of the right child of the i-th node in the array.

	var (
		idxPos int = i*c.nodeLen() + c.valLen + c.idxLen
		idxVal uint32
	)

	idxVal = getUint32(c.memory[idxPos : idxPos+c.idxLen])

	return int(idxVal)
}

func (c *Cache) setVal(i int, val []byte) {
	// Overwrites the value of the i-th node in the array.
	// Ensure it is only called while the mutex is locked!

	var (
		j      int
		valPos int = i * c.nodeLen()
	)

	// copy(c.memory[valPos:valPos+c.valLen], val)
	for j = 0; j < len(val); j++ {
		c.memory[valPos+j] = val[j]
	}

	return
}

func (c *Cache) setIdxL(i int, idxVal int) {
	// Overwrites the index of the left child of the i-th node in the array.
	// Ensure it is only called while the mutex is locked!

	var (
		idxPos int = i*c.nodeLen() + c.valLen
	)

	putUint32(c.memory[idxPos:idxPos+c.idxLen],
		uint32(idxVal),
	)

	return
}

func (c *Cache) setIdxR(i int, idxVal int) {
	// Overwrites the index of the right child of the i-th node in the array.
	// Ensure it is only called while the mutex is locked!

	var (
		idxPos int = i*c.nodeLen() + c.valLen + c.idxLen
	)

	putUint32(c.memory[idxPos:idxPos+c.idxLen],
		uint32(idxVal),
	)

	return
}

func (c *Cache) nodeLen() int {
	// Returns the length of a node, in number of bytes.

	return c.valLen + 2*c.idxLen
}

func log(n int, m int) (x int) {
	// Returns x >= log2(n) such that x is a multiple of m, or 0 if n == 0.

	if n == 0 {
		return
	}

	for {
		if 1<<x >= n {
			break
		}

		x += m
	}

	return
}

func putUint32(into []byte, value uint32) {
	// Copies the last bytes in the big-endian representation of value into a
	// byte slice, sans leading zeroes.

	var (
		b = make([]byte, maxUintLen32)
		i int
	)

	binary.BigEndian.PutUint32(b, value)

	// copy(into, b[maxUintLen32-len(into):])
	for i = 0; i < len(into); i++ {
		into[i] = b[maxUintLen32-len(into)+i]
	}

	return
}

func getUint32(from []byte) uint32 {
	// Returns a 32-bit unsigned integer given its big-endian representation.
	// If shorter than four bytes, the representation is assumed to be
	// right-aligned with leading zeroes omitted.

	var (
		b = make([]byte, maxUintLen32)
		i int
	)

	// copy(b[maxUintLen32-len(from):], from)
	for i = 0; i < len(from); i++ {
		b[maxUintLen32-len(from)+i] = from[i]
	}

	return binary.BigEndian.Uint32(b)
}
