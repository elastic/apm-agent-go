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
	"context"
	"net"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.elastic.co/apm/v2"
)

type hook struct{}

// NewHook returns a redis.Hook that reports cmds as spans to Elastic APM.
func NewHook() redis.Hook {
	return &hook{}
}

func (h *hook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		span, ctx := apm.StartSpanOptions(ctx, "redis.connect", "db.redis", apm.SpanOptions{
			ExitSpan: true,
		})
		span.Context.SetDatabase(apm.DatabaseSpanContext{
			Type:     "redis",
			Instance: addr,
		})
		defer span.End()

		return next(ctx, network, addr)
	}
}

func (h *hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		span, ctx := apm.StartSpanOptions(ctx, getCmdName(cmd), "db.redis", apm.SpanOptions{
			ExitSpan: true,
		})
		defer span.End()

		return next(ctx, cmd)
	}
}

func (h *hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		cmdNames := make([]string, len(cmds))
		for i, cmd := range cmds {
			cmdNames[i] = getCmdName(cmd)
		}

		span, ctx := apm.StartSpanOptions(ctx, strings.Join(cmdNames, ", "), "db.redis", apm.SpanOptions{
			ExitSpan: true,
		})
		defer span.End()

		return next(ctx, cmds)
	}
}

func getCmdName(cmd redis.Cmder) string {
	cmdName := strings.ToUpper(cmd.Name())
	if cmdName == "" {
		cmdName = "(empty command)"
	}
	return cmdName
}
