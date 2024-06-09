package dejavu

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	const (
		nCases = 8
	)

	var (
		cache  *Cache
		e      error
		found  bool
		i      int
		j      int
		random []byte
		source *os.File
		stash  [][]byte
	)

	source, e = os.Open("/dev/urandom")
	if e != nil {
		t.Error(e)
	}

	defer source.Close()

	cache = NewCache128(nCases)

	assert.Equal(t, 0,
		cache.Length(),
	)

	assert.Equal(t, 144,
		cache.Size(),
	)

	for i = 0; i < nCases; i++ {
		random = make([]byte, 16)

		_, e = source.Read(random)
		if e != nil {
			t.Error(e)
		}

		stash = append(stash, random)
	}

	for i = 0; i < nCases; i++ {
		e = cache.Insert(stash[i])
		if e != nil {
			t.Error(e)
		}

		assert.Equal(t, i+1,
			cache.Length(),
		)

		for j = 0; j < nCases; j++ {
			found, e = cache.Recall(stash[j])
			if e != nil {
				t.Error(e)
			}

			if j > i {
				assert.False(t, found)
			} else {
				assert.True(t, found)
			}
		}
	}
}

func BenchmarkCacheInsert(b *testing.B) {
	const (
		nCases = 1 << 22 // 2^22
	)

	var (
		cache  *Cache
		e      error
		i      int
		random []byte
		source *os.File
		stash  [][]byte
	)

	source, e = os.Open("/dev/urandom")
	if e != nil {
		b.Error(e)
	}

	defer source.Close()

	cache = NewCache128(nCases)

	assert.Equal(b, 92274688,
		cache.Size(),
	)

	for i = 0; i < nCases; i++ {
		random = make([]byte, 16)

		_, e = source.Read(random)
		if e != nil {
			b.Error(e)
		}

		stash = append(stash, random)
	}

	b.ResetTimer()

	for i = 0; i < b.N; i++ {
		if i == nCases {
			break
		}

		e = cache.Insert(stash[i])
		if e != nil {
			b.Error(e)
		}
	}

	return
}

func BenchmarkCacheRecall(b *testing.B) {
	const (
		nCases = 1 << 22 // 2^22
	)

	var (
		cache  *Cache
		e      error
		i      int
		random []byte
		source *os.File
		stash  [][]byte
	)

	source, e = os.Open("/dev/urandom")
	if e != nil {
		b.Error(e)
	}

	defer source.Close()

	cache = NewCache128(nCases)

	assert.Equal(b, 92274688,
		cache.Size(),
	)

	for i = 0; i < nCases; i++ {
		random = make([]byte, 16)

		_, e = source.Read(random)
		if e != nil {
			b.Error(e)
		}

		stash = append(stash, random)
	}

	for i = 0; i < nCases; i++ {
		e = cache.Insert(stash[i])
		if e != nil {
			b.Error(e)
		}
	}

	b.ResetTimer()

	for i = 0; i < b.N; i++ {
		if i == nCases {
			break
		}

		_, e = cache.Recall(stash[i])
		if e != nil {
			b.Error(e)
		}
	}

	return
}

func TestLog(t *testing.T) {
	assert.Equal(t, 24,
		log(1<<22, 8),
	)

	return
}

func TestPutUint32(t *testing.T) {
	var (
		into0 = make([]byte, 1)
		into1 = make([]byte, 2)
	)

	putUint32(into0, 255)

	assert.Equal(t, []byte{0xff},
		into0,
	)

	putUint32(into1, 256)

	assert.Equal(t, []byte{0x01, 0x00},
		into1,
	)

	return
}

func TestGetUint32(t *testing.T) {
	var (
		from0 = []byte{0xff}
		from1 = []byte{0x01, 0x00}
	)

	assert.Equal(t, uint32(255),
		getUint32(from0),
	)

	assert.Equal(t, uint32(256),
		getUint32(from1),
	)

	return
}
