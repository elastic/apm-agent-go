package apmstreadwayamqp

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	amqp2 "github.com/valinurovam/garagemq/amqp"
	"github.com/valinurovam/garagemq/config"
	"github.com/valinurovam/garagemq/metrics"
	"github.com/valinurovam/garagemq/server"

	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
)

func TestWrappedChannel_Publish(t *testing.T) {
	var spans []model.Span
	_, spans, _ = apmtest.WithTransaction(func(ctx context.Context) {
		port := startTestAmqpServer3(t)

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
		msg := amqp.Publishing{Headers: initialHeaders}

		//Sleeping is needed for the server to fully init
		time.Sleep(100 * time.Millisecond)
		conn, dialErr := amqp.Dial(fmt.Sprintf("amqp://guest:guest@%s:%d/", "0.0.0.0", port))
		require.Nil(t, dialErr)
		ch, chErr := conn.Channel()
		require.Nil(t, chErr)

		wrCh := WrapChannel(ch).WithContext(ctx)
		pubErr := wrCh.Publish("exch", "key", true, true, msg)
		require.Nil(t, pubErr)

		assert.Len(t, msg.Headers, len(copyInitialMap)+2)
		_, extrOk := ExtractTraceContext(amqp.Delivery{Headers: msg.Headers})
		require.True(t, extrOk)
	})
	require.Len(t, spans, 1)
	assert.Equal(t, "RabbitMQ SEND to exch", spans[0].Name)
	assert.Equal(t, "messaging", spans[0].Type)
	assert.Equal(t, "rabbitmq", spans[0].Subtype)
	assert.Equal(t, "success", spans[0].Outcome)
}

func startTestAmqpServer3(t testing.TB) int {
	rand.Seed(time.Now().UnixNano())
	port := rand.Intn((65535 - 1000) + 1000)
	cfg, _ := config.CreateDefault()
	metrics.NewTrackRegistry(15, time.Second, false)
	srv := server.NewServer("0.0.0.0", fmt.Sprint(port), amqp2.ProtoRabbit, cfg)

	go srv.Start()
	t.Cleanup(srv.Stop)
	return port
}
