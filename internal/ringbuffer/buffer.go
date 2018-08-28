package ringbuffer

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
)

// BlockOverhead is the number of additional writes per block of data
// written to the buffer for block accounting.
const BlockOverhead = 4

// Buffer is a ring buffer of byte blocks.
type Buffer struct {
	buf     []byte
	sizebuf [4]byte
	len     int
	write   int
	read    int
}

// New returns a new Buffer with the given size in bytes.
func New(size int) *Buffer {
	return &Buffer{buf: make([]byte, size)}
}

// Len returns the number of bytes currently in the buffer, including
// block-accounting bytes.
func (b *Buffer) Len() int {
	return b.len
}

// Cap returns the capacity of the buffer.
func (b *Buffer) Cap() int {
	return len(b.buf)
}

// WriteTo writes the oldest block in b to w, and returns its size in bytes.
func (b *Buffer) WriteTo(w io.Writer) (written int64, err error) {
	if b.len == 0 {
		return 0, io.EOF
	}
	if n := copy(b.sizebuf[:], b.buf[b.read:]); n < len(b.sizebuf) {
		b.read = copy(b.sizebuf[n:], b.buf[:])
	} else {
		b.read = (b.read + n) % b.Cap()
	}
	b.len -= len(b.sizebuf)
	size := int(binary.LittleEndian.Uint32(b.sizebuf[:]))

	if b.read+size > b.Cap() {
		tail := b.buf[b.read:]
		n, err := w.Write(tail)
		if err != nil {
			b.read = (b.read + size) % b.Cap()
			b.len -= size + len(b.sizebuf)
			return int64(n), err
		}
		size -= n
		written = int64(n)
		b.read = 0
		b.len -= n
	}
	n, err := w.Write(b.buf[b.read : b.read+size])
	if err != nil {
		return written + int64(n), err
	}
	written += int64(n)
	b.read = (b.read + size) % b.Cap()
	b.len -= size
	return written, nil
}

// Write writes p as a block to b.
//
// If len(p)+BlockOverhead > b.Cap(), bytes.ErrTooLarge will be returned.
// If the buffer does not currently have room for the block, then the
// oldest blocks will be evicted until enough room is available.
func (b *Buffer) Write(p []byte) (int, error) {
	lenp := len(p)
	if lenp+BlockOverhead > b.Cap() {
		return 0, bytes.ErrTooLarge
	}
	for lenp+BlockOverhead > b.Cap()-b.Len() {
		b.WriteTo(ioutil.Discard)
	}
	binary.LittleEndian.PutUint32(b.sizebuf[:], uint32(lenp))
	if n := copy(b.buf[b.write:], b.sizebuf[:]); n < len(b.sizebuf) {
		b.write = copy(b.buf, b.sizebuf[n:])
	} else {
		b.write = (b.write + n) % b.Cap()
	}
	if n := copy(b.buf[b.write:], p); n < lenp {
		b.write = copy(b.buf, p[n:])
	} else {
		b.write = (b.write + n) % b.Cap()
	}
	b.len += lenp + BlockOverhead
	return lenp, nil
}
