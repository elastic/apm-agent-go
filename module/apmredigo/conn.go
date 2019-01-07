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

package apmredigo

import (
	"context"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"

	"go.elastic.co/apm"
)

// Conn is the interface returned by ContextConn.
//
// Conn's Do method reports spans using the bound context.
type Conn interface {
	redis.Conn

	// WithContext returns a shallow copy of the connection with
	// its context changed to ctx.
	//
	// To report commands as spans, ctx must contain a transaction or span.
	WithContext(ctx context.Context) Conn
}

// Wrap wraps conn such that its Do method calls apmredigo.Do with
// context.Background(). The context can be changed using Conn.WithContext.
//
// If conn implements redis.ConnWithTimeout, then the DoWithTimeout method
// will similarly call apmredigo.DoWithTimeout.
//
// Send and Receive calls are not currently captured.
func Wrap(conn redis.Conn) Conn {
	ctx := context.Background()
	if cwt, ok := conn.(redis.ConnWithTimeout); ok {
		return contextConnWithTimeout{ConnWithTimeout: cwt, ctx: ctx}
	}
	return contextConn{Conn: conn, ctx: ctx}
}

type contextConnWithTimeout struct {
	redis.ConnWithTimeout
	ctx context.Context
}

func (c contextConnWithTimeout) WithContext(ctx context.Context) Conn {
	c.ctx = ctx
	return c
}

func (c contextConnWithTimeout) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	return Do(c.ctx, c.ConnWithTimeout, commandName, args...)
}

func (c contextConnWithTimeout) DoWithTimeout(timeout time.Duration, commandName string, args ...interface{}) (reply interface{}, err error) {
	return DoWithTimeout(c.ctx, c.ConnWithTimeout, timeout, commandName, args...)
}

type contextConn struct {
	redis.Conn
	ctx context.Context
}

func (c contextConn) WithContext(ctx context.Context) Conn {
	c.ctx = ctx
	return c
}

func (c contextConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	return Do(c.ctx, c.Conn, commandName, args...)
}

// Do calls conn.Do(commandName, args...), and also reports the operation as a span to Elastic APM.
func Do(ctx context.Context, conn redis.Conn, commandName string, args ...interface{}) (interface{}, error) {
	spanName := strings.ToUpper(commandName)
	if spanName == "" {
		spanName = "(flush pipeline)"
	}
	span, _ := apm.StartSpan(ctx, spanName, "db.redis")
	defer span.End()
	return conn.Do(commandName, args...)
}

// DoWithTimeout calls redis.DoWithTimeout(conn, timeout, commandName, args...), and also reports the operation as a span to Elastic APM.
func DoWithTimeout(ctx context.Context, conn redis.Conn, timeout time.Duration, commandName string, args ...interface{}) (interface{}, error) {
	spanName := strings.ToUpper(commandName)
	if spanName == "" {
		spanName = "(flush pipeline)"
	}
	span, _ := apm.StartSpan(ctx, spanName, "db.redis")
	defer span.End()
	return redis.DoWithTimeout(conn, timeout, commandName, args...)
}
