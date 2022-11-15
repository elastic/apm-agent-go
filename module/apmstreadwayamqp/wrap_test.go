package apmstreadwayamqp

import (
	"context"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/v2"
	"testing"
)

func TestWrappedChannel_Publish(t *testing.T) {
	ctx := context.Background()
	tx := apm.DefaultTracer().StartTransaction("name", "type")
	require.NotNil(t, tx)
	ctx = apm.ContextWithTransaction(ctx, tx)
	initialHeaders := map[string]interface{}{
		"ela":   "e",
		"stic":  "l",
		"stack": "k",
	}

	// Since in Go maps are always passed by reference, we must copy the initial map
	copyInitialMap := make(map[string]interface{})
	for key, value := range initialHeaders {
		copyInitialMap[key] = value
	}

	key := "key"
	exch := "exchange"
	msg := amqp.Publishing{
		Headers: initialHeaders,
	}

	ch := amqp.Channel{}
	wrCh := WrapChannel(&ch).WithContext(ctx)

	defer func() {
		if err := recover(); err != nil {
			assert.Len(t, msg.Headers, len(copyInitialMap)+2)
			extrH, extrErr := ExtractTraceContext(amqp.Delivery{Headers: msg.Headers})
			require.Nil(t, extrErr)
			assert.Equal(t, tx.TraceContext().Trace.String(), extrH.Trace.String())
			assert.Equal(t, tx.TraceContext().State.String(), extrH.State.String())

		}
	}()
	_ = wrCh.Publish(exch, key, true, true, msg)

}
