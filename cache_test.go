package dejavu

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	nCasesMax = 1 << 22
)

var (
	values [][]byte = make([][]byte, nCasesMax)
)

func TestMain(m *testing.M) {
	var (
		e      error
		i      int
		random []byte
		source *os.File
	)

	source, e = os.Open("/dev/urandom")
	if e != nil {
		panic(e)
	}

	defer source.Close()

	for i = 0; i < nCasesMax; i++ {
		random = make([]byte, 16)

		_, e = source.Read(random)
		if e != nil {
			panic(e)
		}

		values[i] = random
	}

	os.Exit(m.Run())
}

func TestCache(t *testing.T) {
	const (
		nCases = 8
	)

	var (
		cache *Cache
		e     error
		found bool
		i     int
		j     int
	)

	cache = NewCache128(nCases)

	assert.Equal(t, 0,
		cache.Length(),
	)

	assert.Equal(t, 144,
		cache.Size(),
	)

	assert.Nil(t,
		cache.Last(),
	)

	for i = 0; i < nCases; i++ {
		e = cache.Insert(values[i])
		if e != nil {
			t.Error(e)
		}

		assert.Equal(t, i+1,
			cache.Length(),
		)

		assert.Equal(t, values[i],
			cache.Last(),
		)

		for j = 0; j < nCases; j++ {
			found, e = cache.Recall(values[j])
			if e != nil {
				t.Error(e)
			}

			if j > i {
				assert.False(t, found)
			} else {
				assert.True(t, found)
			}
		}

		if i == nCases-1 {
			assert.True(t,
				cache.Full(),
			)

		} else {
			assert.False(t,
				cache.Full(),
			)
		}
	}
}

func TestCacheSaveLoad(t *testing.T) {
	const (
		nCases = 1 << 8 // 256
	)

	var (
		found  bool
		buffer bytes.Buffer
		cache0 *Cache
		cache1 *Cache
		e      error
		i      int
	)

	cache0 = NewCache128(nCases)

	for i = 0; i < nCases; i++ {
		e = cache0.Insert(values[i])
		if e != nil {
			t.Error(e)
		}
	}

	e = cache0.Save(&buffer)
	if e != nil {
		t.Error(e)
	}

	cache1 = NewCache128(nCases)

	for i = 0; i < nCases; i++ {
		found, e = cache1.Recall(values[i])
		if e != nil {
			t.Error(e)
		}

		assert.False(t, found)
	}

	e = cache1.Load(&buffer)
	if e != nil {
		t.Error(e)
	}

	for i = 0; i < nCases; i++ {
		found, e = cache1.Recall(values[i])
		if e != nil {
			t.Error(e)
		}

		assert.True(t, found)
	}

	return
}

func BenchmarkCacheInsert(b *testing.B) {
	var (
		cache *Cache
		e     error
		i     int
	)

	cache = NewCache128(nCasesMax)

	b.ResetTimer()

	for i = 0; i < b.N; i++ {
		if i == nCasesMax {
			break
		}

		e = cache.Insert(values[i])
		if e != nil {
			b.Error(e)
		}
	}

	return
}

func BenchmarkCacheRecall(b *testing.B) {
	var (
		cache *Cache
		e     error
		i     int
	)

	cache = NewCache128(nCasesMax)

	for i = 0; i < nCasesMax; i++ {
		e = cache.Insert(values[i])
		if e != nil {
			b.Error(e)
		}
	}

	b.ResetTimer()

	for i = 0; i < b.N; i++ {
		if i == nCasesMax {
			break
		}

		_, e = cache.Recall(values[i])
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
