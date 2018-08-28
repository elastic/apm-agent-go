package ringbuffer

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	b := New(150)
	assert.Equal(t, 0, b.Len())
	assert.Equal(t, 150, b.Cap())

	const block = `{"transaction":{"duration":0,"id":"00000000-0000-0000-0000-000000000000","name":"","timestamp":"0001-01-01T00:00:00Z","type":""}}`

	for i := 0; i < 10; i++ {
		b.Write([]byte(block))
		blen := b.Len()
		assert.NotEqual(t, 0, blen)
		assert.Equal(t, 150, b.Cap())

		var bb bytes.Buffer
		n, err := b.WriteTo(&bb)
		assert.Equal(t, int64(blen-BlockOverhead), n)
		assert.Equal(t, block, bb.String())
		assert.Equal(t, 0, b.Len())
		n, err = b.WriteTo(&bb)
		assert.Zero(t, n)
		assert.Equal(t, io.EOF, err)
	}
}

func TestBufferEviction(t *testing.T) {
	const block = `{"transaction":{"duration":0,"id":"00000000-0000-0000-0000-000000000000","name":"","timestamp":"0001-01-01T00:00:00Z","type":""}}`
	b := New(300)
	for i := 0; i < 100; i++ {
		b.Write([]byte(block))
	}
	assert.Equal(t, len(block)*2+2*BlockOverhead, b.Len())

	for i := 0; i < 2; i++ {
		var bb bytes.Buffer
		b.WriteTo(&bb)
		assert.Equal(t, block, bb.String())
	}
	assert.Equal(t, 0, b.Len())
}

func BenchmarkWrite(b *testing.B) {
	data := []byte(strings.Repeat("*", 1024))
	buf := New(10 * 1024 * 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n, err := buf.Write(data[:])
		if err != nil {
			panic(err)
		}
		b.SetBytes(int64(n))
	}
}

func BenchmarkWriteTo(b *testing.B) {
	data := []byte(strings.Repeat("*", 300))
	buf := New(b.N * (len(data) + BlockOverhead))
	for i := 0; i < b.N; i++ {
		buf.Write(data[:])
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n, err := buf.WriteTo(ioutil.Discard)
		if err != nil {
			panic(err)
		}
		b.SetBytes(n)
	}
}
