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

//go:build go1.10
// +build go1.10

package apmsql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmsql"
)

func TestConnect(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Note: only in Go 1.10 do we have context during connection.

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		err := db.PingContext(ctx)
		assert.NoError(t, err)
	})
	require.Len(t, spans, 2)
	assert.Equal(t, "connect", spans[0].Name)
	assert.Equal(t, "connect", spans[0].Action)
	assert.Equal(t, "ping", spans[1].Name)
	assert.Equal(t, "ping", spans[1].Action)
}
