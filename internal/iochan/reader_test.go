// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
