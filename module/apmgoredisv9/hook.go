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

package apmgoredisv9 // import "go.elastic.co/apm/module/apmgoredisv9/v2"

import (
	"bytes"
	"context"
	"net"
	"strings"

	"github.com/redis/go-redis/v9"

	"go.elastic.co/apm/v2"
)

// hook is an implementation of redis.Hook that reports cmds as spans to Elastic APM.
type hook struct{}

// NewHook returns a redis.Hook that reports cmds as spans to Elastic APM.
func NewHook() redis.Hook {
	return &hook{}
}

func (r *hook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		// It is possible to capture the address and port of the Redis server here.
		// However, due to the way go-redis implements sentinel support,
		// for redis behind sentinels,
		// we can only obtain the address of sentinel, instead of Redis, here.
		return next(ctx, network, addr)
	}
}

func (r *hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) (err error) {
		var span *apm.Span
		span, ctx = apm.StartSpanOptions(
			ctx, getCmdName(cmd), "db.redis", apm.SpanOptions{
				ExitSpan: true,
			},
		)
		if !span.Dropped() {
			defer r.finishSpan(span, &err)
		}

		err = next(ctx, cmd)
		return err
	}
}

func (r *hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) (err error) {
		var span *apm.Span
		span, ctx = apm.StartSpanOptions(
			ctx, getCmdsName(cmds), "db.redis", apm.SpanOptions{
				ExitSpan: true,
			},
		)
		if !span.Dropped() {
			defer r.finishSpan(span, &err)
		}

		err = next(ctx, cmds)
		return err
	}
}

func (r *hook) finishSpan(span *apm.Span, err *error) {
	if span.Outcome == "" {
		if *err == nil {
			span.Outcome = "success"
		} else {
			span.Outcome = "failure"
		}
	}
	span.End()
}

// getCmdsName Join all cmd names with “, ”.
func getCmdsName(cmds []redis.Cmder) string {
	var cmdNameBuf bytes.Buffer
	for i, cmd := range cmds {
		if i != 0 {
			cmdNameBuf.WriteString(", ")
		}
		cmdNameBuf.WriteString(getCmdName(cmd))
	}
	return cmdNameBuf.String()
}

func getCmdName(cmd redis.Cmder) string {
	cmdName := strings.ToUpper(cmd.Name())
	if cmdName == "" {
		cmdName = "(empty command)"
	}
	return cmdName
}
