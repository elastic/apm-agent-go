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

func TestContextTags(t *testing.T) {
	tx := testSendTransaction(t, func(tx *apm.Transaction) {
		tx.Context.SetTag("foo", "bar")
		tx.Context.SetTag("foo", "bar!") // Last instance wins
		tx.Context.SetTag("bar", "baz")
	})
	assert.Equal(t, model.StringMap{
		{Key: "bar", Value: "baz"},
		{Key: "foo", Value: "bar!"},
	}, tx.Context.Tags)
}

func TestContextUser(t *testing.T) {
	t.Run("email", func(t *testing.T) {
		tx := testSendTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetUserEmail("testing@host.invalid")
		})
		assert.Equal(t, &model.User{Email: "testing@host.invalid"}, tx.Context.User)
	})
	t.Run("username", func(t *testing.T) {
		tx := testSendTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetUsername("schnibble")
		})
		assert.Equal(t, &model.User{Username: "schnibble"}, tx.Context.User)
	})
	t.Run("id", func(t *testing.T) {
		tx := testSendTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetUserID("123")
		})
		assert.Equal(t, &model.User{ID: "123"}, tx.Context.User)
	})
}

func TestContextFramework(t *testing.T) {
	t.Run("name_unspecified", func(t *testing.T) {
		tx := testSendTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetFramework("", "1.0")
		})
		assert.Nil(t, tx.Context)
	})
	t.Run("version_specified", func(t *testing.T) {
		tx := testSendTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetFramework("framework", "1.0")
		})
		require.NotNil(t, tx.Context)
		require.NotNil(t, tx.Context.Service)
		assert.Equal(t, &model.Framework{
			Name:    "framework",
			Version: "1.0",
		}, tx.Context.Service.Framework)
	})
	t.Run("version_unspecified", func(t *testing.T) {
		tx := testSendTransaction(t, func(tx *apm.Transaction) {
			tx.Context.SetFramework("framework", "")
		})
		require.NotNil(t, tx.Context)
		require.NotNil(t, tx.Context.Service)
		assert.Equal(t, &model.Framework{
			Name:    "framework",
			Version: "unspecified",
		}, tx.Context.Service.Framework)
	})
}

func testSendTransaction(t *testing.T, f func(tx *apm.Transaction)) model.Transaction {
	transaction, _, _ := apmtest.WithTransaction(func(ctx context.Context) {
		f(apm.TransactionFromContext(ctx))
	})
	return transaction
}
