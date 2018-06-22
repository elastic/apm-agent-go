package transport_test

import (
	"context"
	"net"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/transport"
)

func TestInitDefault(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()

	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()

	tr, err := transport.InitDefault()
	assert.NoError(t, err)
	assert.NotNil(t, tr)
	assert.Exactly(t, tr, transport.Default)

	err = tr.SendTransactions(context.Background(), &model.TransactionsPayload{})
	assert.NoError(t, err)
	assert.Len(t, h.requests, 1)
}

func TestInitDefaultDiscard(t *testing.T) {
	var h recordingHandler
	server := httptest.NewUnstartedServer(&h)
	defer server.Close()

	lis, err := net.Listen("tcp", "localhost:8200")
	if err != nil {
		t.Skipf("cannot listen on default server address: %s", err)
	}
	server.Listener.Close()
	server.Listener = lis
	server.Start()

	defer patchEnv("ELASTIC_APM_SERVER_URL", "")()

	tr, err := transport.InitDefault()
	assert.NoError(t, err)
	assert.NotNil(t, tr)
	assert.Exactly(t, tr, transport.Default)

	err = tr.SendTransactions(context.Background(), &model.TransactionsPayload{})
	assert.NoError(t, err)
	assert.Len(t, h.requests, 1)
}

func TestInitDefaultError(t *testing.T) {
	defer patchEnv("ELASTIC_APM_SERVER_URL", ":")()

	tr, initErr := transport.InitDefault()
	assert.Error(t, initErr)
	assert.NotNil(t, tr)
	assert.Exactly(t, tr, transport.Default)

	sendErr := tr.SendTransactions(context.Background(), &model.TransactionsPayload{})
	assert.Exactly(t, initErr, sendErr)
}
