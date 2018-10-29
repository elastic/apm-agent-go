package apm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
)

func TestSpanContextSetTag(t *testing.T) {
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, _ := apm.StartSpan(ctx, "name", "type")
		span.Context.SetTag("foo", "bar")
		span.Context.SetTag("foo", "bar!") // Last instance wins
		span.Context.SetTag("bar", "baz")
		span.End()
	})
	require.Len(t, spans, 1)
	assert.Equal(t, model.StringMap{
		{Key: "bar", Value: "baz"},
		{Key: "foo", Value: "bar!"},
	}, spans[0].Context.Tags)
}
