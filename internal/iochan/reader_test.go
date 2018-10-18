package iochan

import (
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	target := 9999
	r := NewReader()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var bytes int
		for req := range r.C {
			for i := range req.Buf {
				req.Buf[i] = '*'
			}
			n := len(req.Buf)
			var err error
			if bytes+n >= target {
				if bytes+n > target {
					n = target - bytes
				}
				err = io.EOF
			}
			bytes += n
			req.Respond(n, err)
		}
	}()

	data, err := ioutil.ReadAll(r)
	assert.NoError(t, err)
	r.CloseWrite() // unblocks the goroutine
	assert.Equal(t, strings.Repeat("*", target), string(data))
	wg.Wait()
}

func TestCloseRead(t *testing.T) {
	r := NewReader()
	r.CloseRead(io.EOF)
	r.CloseRead(errors.New("ignored"))
	n, err := r.Read(make([]byte, 1024))
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}

func TestCloseReadBlocked(t *testing.T) {
	r := NewReader()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-r.C // wait for ar request, but don't respond
		r.CloseRead(io.EOF)
	}()

	n, err := r.Read(make([]byte, 1024))
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
	wg.Wait()
}
