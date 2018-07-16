package elasticapm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestContextUser(t *testing.T) {
	t.Run("email", func(t *testing.T) {
		tx := testSendTransaction(t, func(tx *elasticapm.Transaction) {
			tx.Context.SetUserEmail("testing@host.invalid")
		})
		assert.Equal(t, &model.User{Email: "testing@host.invalid"}, tx.Context.User)
	})
	t.Run("username", func(t *testing.T) {
		tx := testSendTransaction(t, func(tx *elasticapm.Transaction) {
			tx.Context.SetUsername("schnibble")
		})
		assert.Equal(t, &model.User{Username: "schnibble"}, tx.Context.User)
	})
	t.Run("id", func(t *testing.T) {
		tx := testSendTransaction(t, func(tx *elasticapm.Transaction) {
			tx.Context.SetUserID("123")
		})
		assert.Equal(t, &model.User{ID: "123"}, tx.Context.User)
	})
}

func testSendTransaction(t *testing.T, f func(tx *elasticapm.Transaction)) model.Transaction {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	f(tx)
	tx.End()
	tracer.Flush(nil)

	payloads := r.Payloads()
	require.Len(t, payloads, 1)
	transactions := payloads[0].Transactions()
	require.Len(t, transactions, 1)
	return transactions[0]
}
