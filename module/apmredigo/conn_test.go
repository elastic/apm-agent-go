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
	"errors"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmredigo"
)

func TestWrap(t *testing.T) {
	var conn mockConn
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		conn := apmredigo.Wrap(conn).WithContext(ctx)
		conn.Do("PING", "hello, world!")
	})
	require.Len(t, spans, 1)
	assert.Equal(t, "PING", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "redis", spans[0].Subtype)
}

func TestWithContext(t *testing.T) {
	ping := func(ctx context.Context, conn apmredigo.Conn) {
		span, ctx := apm.StartSpan(ctx, "ping", "custom")
		defer span.End()

		// bind conn to the ctx containing the span above
		conn = conn.WithContext(ctx)
		conn.Do("PING", "hello, world!")
	}

	var conn mockConn
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		conn := apmredigo.Wrap(conn)
		ping(ctx, conn)
	})
	require.Len(t, spans, 2)
	assert.Equal(t, "PING", spans[0].Name)
	assert.Equal(t, "ping", spans[1].Name)
	assert.Equal(t, spans[1].ID, spans[0].ParentID)
}

func TestConnWithTimeout(t *testing.T) {
	var conn mockConnWithTimeout
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		conn := apmredigo.Wrap(conn).WithContext(ctx)
		redis.DoWithTimeout(conn, time.Second, "PING", "hello, world!")
	})
	require.Len(t, spans, 1)
	assert.Equal(t, "PING", spans[0].Name)
}

func TestWrapPipeline(t *testing.T) {
	var conn mockConnWithTimeout
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		conn := apmredigo.Wrap(conn).WithContext(ctx)
		conn.Do("")
		redis.DoWithTimeout(conn, time.Second, "")
	})
	require.Len(t, spans, 2)
	assert.Equal(t, "(flush pipeline)", spans[0].Name)
	assert.Equal(t, "(flush pipeline)", spans[1].Name)
}

type mockConnWithTimeout struct{ mockConn }

func (mockConnWithTimeout) DoWithTimeout(timeout time.Duration, commandName string, args ...interface{}) (reply interface{}, err error) {
	return []byte("Done"), errors.New("DoWithTimeout failed")
}

func (mockConnWithTimeout) ReceiveWithTimeout(timeout time.Duration) (reply interface{}, err error) {
	return []byte("REceived"), errors.New("ReceiveWithTimeout failed")
}

type mockConn struct{}

func (mockConn) Close() error {
	panic("Close not implemented")
}

func (mockConn) Err() error {
	panic("Err not implemented")
}

func (mockConn) Flush() error {
	panic("Flush not implemented")
}

func (mockConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	return []byte("Done"), errors.New("Do failed")
}

func (mockConn) Send(commandName string, args ...interface{}) error {
	return errors.New("Send failed")
}

func (mockConn) Receive() (reply interface{}, err error) {
	return []byte("Received"), errors.New("Receive failed")
}
