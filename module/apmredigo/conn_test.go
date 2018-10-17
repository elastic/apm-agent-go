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
	assert.Equal(t, "cache.redis", spans[0].Type)
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
	assert.Equal(t, "cache.redis", spans[0].Type)
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
