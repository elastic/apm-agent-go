package apm_test

import (
	"context"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm"
	"go.elastic.co/apm/transport/transporttest"
)

func TestContextStartSpanTransactionEnded(t *testing.T) {
	tracer, err := apm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	defer tracer.Close()
	tracer.Transport = transporttest.Discard

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				tx := tracer.StartTransaction("name", "type")
				ctx := apm.ContextWithTransaction(context.Background(), tx)
				tx.End()
				apm.CaptureError(ctx, errors.New("boom")).Send()
				span, _ := apm.StartSpan(ctx, "name", "type")
				assert.True(t, span.Dropped())
				span.End()
			}
		}()
	}
	tracer.Flush(nil)
	wg.Wait()
}

func TestContextStartSpanSpanEnded(t *testing.T) {
	tracer, err := apm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	defer tracer.Close()
	tracer.Transport = transporttest.Discard

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				tx := tracer.StartTransaction("name", "type")
				ctx := apm.ContextWithTransaction(context.Background(), tx)
				span1, ctx := apm.StartSpan(ctx, "name", "type")
				span1.End()
				apm.CaptureError(ctx, errors.New("boom")).Send()
				span2, _ := apm.StartSpan(ctx, "name", "type")
				assert.True(t, span2.Dropped())
				span2.End()
				tx.End()
			}
		}()
	}
	tracer.Flush(nil)
	wg.Wait()
}
