package apmgoredisv9

import (
	"bytes"
	"context"
	"net"
	"strings"

	"github.com/redis/go-redis/v9"

	"go.elastic.co/apm/v2"
)

var _ redis.Hook = (*hook)(nil)

// hook is an implementation of redis.Hook that reports cmds as spans to Elastic APM.
type hook struct{}

func (r *hook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (r *hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		span, ctx := apm.StartSpanOptions(ctx, getCmdName(cmd), "db.redis", apm.SpanOptions{
			ExitSpan: true,
		})
		defer span.End()

		err := next(ctx, cmd)
		if err != nil {
			_ = apm.CaptureError(ctx, err)
		}
		return err
	}
}

func (r *hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		// Join all cmd names with ", ".
		var cmdNameBuf bytes.Buffer
		for i, cmd := range cmds {
			if i != 0 {
				cmdNameBuf.WriteString(", ")
			}
			cmdNameBuf.WriteString(getCmdName(cmd))
		}
		span, ctx := apm.StartSpanOptions(ctx, cmdNameBuf.String(), "db.redis", apm.SpanOptions{
			ExitSpan: true,
		})
		defer span.End()

		err := next(ctx, cmds)
		if err != nil {
			_ = apm.CaptureError(ctx, err)
		}
		return err
	}
}

// NewHook returns a redis.Hook that reports cmds as spans to Elastic APM.
func NewHook() redis.Hook {
	return &hook{}
}

func getCmdName(cmd redis.Cmder) string {
	cmdName := strings.ToUpper(cmd.Name())
	if cmdName == "" {
		cmdName = "(empty command)"
	}
	return cmdName
}
