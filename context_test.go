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

func TestContextLabels(t *testing.T) {
	type customInt int
	tx := testSendTransaction(t, func(tx *apm.Transaction) {
		tx.Context.SetTag("foo", "bar")    // deprecated
		tx.Context.SetLabel("foo", "bar!") // Last instance wins
		tx.Context.SetLabel("bar", "baz")
		tx.Context.SetLabel("baz", 123.456)
		tx.Context.SetLabel("qux", true)
		tx.Context.SetLabel("quux", customInt(123))
	})
	assert.Equal(t, model.IfaceMap{
		{Key: "bar", Value: "baz"},
		{Key: "baz", Value: 123.456},
		{Key: "foo", Value: "bar!"},
		{Key: "quux", Value: 123.0},
		{Key: "qux", Value: true},
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

func TestContextCustom(t *testing.T) {
	type arbitraryStruct struct {
		Field string
	}
	tx := testSendTransaction(t, func(tx *apm.Transaction) {
		tx.Context.SetCustom("string", "string")
		tx.Context.SetCustom("bool", true)
		tx.Context.SetCustom("number", 1.23)
		tx.Context.SetCustom("struct", arbitraryStruct{Field: "foo"})
	})
	require.NotNil(t, tx.Context)
	assert.Equal(t, model.IfaceMap{
		{Key: "bool", Value: true},
		{Key: "number", Value: 1.23},
		{Key: "string", Value: "string"},
		{Key: "struct", Value: map[string]interface{}{"Field": "foo"}},
	}, tx.Context.Custom)
}

func testSendTransaction(t *testing.T, f func(tx *apm.Transaction)) model.Transaction {
	transaction, _, _ := apmtest.WithTransaction(func(ctx context.Context) {
		f(apm.TransactionFromContext(ctx))
	})
	return transaction
}
