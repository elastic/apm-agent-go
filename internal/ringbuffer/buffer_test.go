package ringbuffer

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockHeaderSize(t *testing.T) {
	size := binary.Size(BlockHeader{})
	assert.Equal(t, BlockHeaderSize, size)
}

func TestBuffer(t *testing.T) {
	b := New(150)
	assert.Equal(t, 0, b.Len())
	assert.Equal(t, 150, b.Cap())

	const block = `{"transaction":{"duration":0,"id":"00000000-0000-0000-0000-000000000000","name":"","timestamp":"0001-01-01T00:00:00Z","type":""}}`

	for i := 0; i < 10; i++ {
		b.WriteBlock([]byte(block), 0)
		blen := b.Len()
		assert.NotEqual(t, 0, blen)
		assert.Equal(t, 150, b.Cap())

		var bb bytes.Buffer
		_, n, err := b.WriteBlockTo(&bb)
		assert.Equal(t, int64(blen-BlockHeaderSize), n)
		assert.Equal(t, block, bb.String())
		assert.Equal(t, 0, b.Len())
		_, n, err = b.WriteBlockTo(&bb)
		assert.Zero(t, n)
		assert.Equal(t, io.EOF, err)
	}
}

func TestBufferEviction(t *testing.T) {
	const block = `{"transaction":{"duration":0,"id":"00000000-0000-0000-0000-000000000000","name":"","timestamp":"0001-01-01T00:00:00Z","type":""}}`

	var evicted []BlockHeader
	b := New(300)
	b.Evicted = func(h BlockHeader) {
		evicted = append(evicted, h)
	}
	for i := 0; i < 100; i++ {
		b.WriteBlock([]byte(block), BlockTag(i))
	}
	assert.Equal(t, len(block)*2+2*BlockHeaderSize, b.Len())

	for i := 0; i < 2; i++ {
		var bb bytes.Buffer
		b.WriteBlockTo(&bb)
		assert.Equal(t, block, bb.String())
	}
	assert.Equal(t, 0, b.Len())
	assert.Len(t, evicted, 98)
	for i, h := range evicted {
		assert.Equal(t, BlockTag(i), h.Tag)
		assert.Equal(t, uint32(len(block)), h.Size)
	}
}

func BenchmarkWrite(b *testing.B) {
	data := []byte(strings.Repeat("*", 1024))
	buf := New(10 * 1024 * 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n, err := buf.WriteBlock(data[:], 0)
		if err != nil {
			panic(err)
		}
		b.SetBytes(int64(n))
	}
}

func BenchmarkWriteBlockTo(b *testing.B) {
	data := []byte(strings.Repeat("*", 300))
	buf := New(b.N * (len(data) + BlockHeaderSize))
	for i := 0; i < b.N; i++ {
		buf.WriteBlock(data[:], 0)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, n, err := buf.WriteBlockTo(ioutil.Discard)
		if err != nil {
			panic(err)
		}
		b.SetBytes(n)
	}
}
