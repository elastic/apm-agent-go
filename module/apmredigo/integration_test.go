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

package apmredigo_test

import (
	"context"
	"os"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmredigo"
)

func TestRequestContext(t *testing.T) {
	c := dialRedis(t)
	cleanRedis(t, c)
	c.Close()

	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return dialRedis(t), nil
		},
	}
	defer pool.Close()

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		for i := 0; i < 2; i++ {
			conn := apmredigo.Wrap(pool.Get()).WithContext(ctx)
			defer conn.Close()

			value, err := redis.Bytes(conn.Do("GET", "content"))
			if err == nil {
				return
			}

			value = []byte("Lorem ipsum dolor sit amet")
			if _, err := conn.Do("SET", "content", value); err != nil {
				require.NoError(t, err)
			}
		}
	})
	require.Len(t, spans, 3)

	assert.Equal(t, "GET", spans[0].Name)
	assert.Equal(t, "SET", spans[1].Name)
	assert.Equal(t, "GET", spans[2].Name)
}

func TestPipelineSendReceive(t *testing.T) {
	c := dialRedis(t)
	defer c.Close()
	cleanRedis(t, c)

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		c := apmredigo.Wrap(c).WithContext(ctx)

		err := c.Send("SET", "foo", "bar")
		require.NoError(t, err)

		err = c.Send("GET", "foo")
		require.NoError(t, err)

		err = c.Flush()
		require.NoError(t, err)

		setReply, err := c.Receive() // reply from SET
		require.NoError(t, err)
		_ = setReply

		getReply, err := c.Receive() // reply from GET
		require.NoError(t, err)
		_ = getReply
	})
	// Send and Receive calls are not currently captured.
	assert.Len(t, spans, 0)
}

func TestPipelinedTransaction(t *testing.T) {
	c := dialRedis(t)
	defer c.Close()
	cleanRedis(t, c)

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		c := apmredigo.Wrap(c).WithContext(ctx)
		c.Send("MULTI")
		c.Send("INCR", "foo")
		c.Send("INCR", "bar")
		c.Send("INCR", "bar")
		values, err := redis.Values(c.Do("EXEC"))
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{int64(1), int64(1), int64(2)}, values)
	})
	assert.Len(t, spans, 1)
	assert.Equal(t, "EXEC", spans[0].Name)
}

func dialRedis(t *testing.T) redis.Conn {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skipf("REDIS_URL not specified")
	}

	closeConn := true
	conn, err := redis.DialURL(redisURL)
	require.NoError(t, err)
	defer func() {
		if closeConn {
			conn.Close()
		}
	}()

	_, err = conn.Do("SELECT", "4")
	require.NoError(t, err)

	closeConn = false
	return conn
}

func cleanRedis(t *testing.T, conn redis.Conn) {
	_, err := conn.Do("FLUSHDB")
	require.NoError(t, err)
}
