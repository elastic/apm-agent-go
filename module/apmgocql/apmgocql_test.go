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

//go:build go1.9
// +build go1.9

package apmgocql_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmgocql"
)

const (
	createKeyspaceStatement = `
CREATE KEYSPACE IF NOT EXISTS foo
WITH REPLICATION = {
	'class' : 'SimpleStrategy',
	'replication_factor' : 1
};`
)

var cassandraHost = os.Getenv("CASSANDRA_HOST")

func TestQueryObserver(t *testing.T) {
	var start time.Time
	observer := apmgocql.NewObserver()
	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		start = time.Now()
		observer.ObserveQuery(ctx, gocql.ObservedQuery{
			Start:     start,
			End:       start.Add(3 * time.Second),
			Keyspace:  "quay ",
			Statement: "SELECT * FROM foo.bar",
			Err:       errors.New("baz"),
		})
	})

	require.Len(t, spans, 1)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "cassandra", spans[0].Subtype)
	assert.Equal(t, "query", spans[0].Action)
	assert.Equal(t, "SELECT FROM foo.bar", spans[0].Name)
	assert.WithinDuration(t,
		time.Time(spans[0].Timestamp).Add(time.Duration(spans[0].Duration*1000000)),
		start.Add(3*time.Second),
		100*time.Millisecond, // allow some leeway for slow systems
	)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "cassandra",
			Instance:  "quay ",
			Statement: "SELECT * FROM foo.bar",
		},
	}, spans[0].Context)

	require.Len(t, errors, 1)
	assert.Equal(t, "TestQueryObserver.func1", errors[0].Culprit)
}

func TestBatchObserver(t *testing.T) {
	var start time.Time
	observer := apmgocql.NewObserver()
	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		start = time.Now()
		observer.ObserveBatch(ctx, gocql.ObservedBatch{
			Start:    start,
			End:      start.Add(3 * time.Second),
			Keyspace: "quay ",
			Statements: []string{
				"INSERT INTO foo.bar(id) VALUES(1)",
				"UPDATE foo.bar SET id=2",
			},
			Err: errors.New("baz"),
		})
	})

	require.Len(t, spans, 3)
	assert.Equal(t, "db", spans[2].Type)
	assert.Equal(t, "cassandra", spans[2].Subtype)
	assert.Equal(t, "batch", spans[2].Action) // sent last
	for _, span := range spans[:2] {
		assert.Equal(t, spans[2].ID, span.ParentID)
		assert.Equal(t, spans[2].TraceID, span.TraceID)
		assert.Equal(t, "db", span.Type)
		assert.Equal(t, "cassandra", span.Subtype)
		assert.Equal(t, "query", span.Action)
	}

	assert.Equal(t, "INSERT INTO foo.bar", spans[0].Name)
	assert.Equal(t, "UPDATE foo.bar", spans[1].Name)
	assert.Equal(t, "BATCH", spans[2].Name)

	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:     "cassandra",
			Instance: "quay ",
		},
	}, spans[2].Context)

	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "cassandra",
			Instance:  "quay ",
			Statement: "INSERT INTO foo.bar(id) VALUES(1)",
		},
	}, spans[0].Context)

	require.Len(t, errors, 1)
	assert.Equal(t, "TestBatchObserver.func1", errors[0].Culprit)
}

func TestQueryObserverIntegration(t *testing.T) {
	session := newSession(t)
	defer session.Close()

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		err := execQuery(ctx, session, createKeyspaceStatement)
		assert.NoError(t, err)

		err = execQuery(ctx, session, `CREATE TABLE IF NOT EXISTS foo.bar (id int, PRIMARY KEY(id));`)
		assert.NoError(t, err)

		err = execQuery(ctx, session, "INSERT INTO foo.bar(id) VALUES(1)")
		assert.NoError(t, err)
	})

	require.Len(t, spans, 3)
	for _, span := range spans {
		assert.Equal(t, "db", span.Type)
		assert.Equal(t, "cassandra", span.Subtype)
		assert.Equal(t, "query", span.Action)
	}
	assert.Equal(t, "CREATE", spans[0].Name)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "cassandra",
			Statement: createKeyspaceStatement,
		},
	}, spans[0].Context)
	assert.Equal(t, "CREATE", spans[1].Name)
	assert.Equal(t, "INSERT INTO foo.bar", spans[2].Name)
}

func TestBatchObserverIntegration(t *testing.T) {
	session := newSession(t)
	defer session.Close()

	err := execQuery(context.Background(), session, createKeyspaceStatement)
	assert.NoError(t, err)

	err = execQuery(context.Background(), session, `CREATE TABLE IF NOT EXISTS foo.bar (id int, PRIMARY KEY(id));`)
	assert.NoError(t, err)

	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		batch := session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
		batch.Query("INSERT INTO foo.bar(id) VALUES(1)")
		batch.Query("INSERT INTO foo.bar(id) VALUES(2)")
		err := session.ExecuteBatch(batch)
		assert.NoError(t, err)
	})

	require.Len(t, spans, 3)
	assert.Equal(t, tx.ID, spans[2].ParentID)
	assert.Equal(t, tx.TraceID, spans[2].TraceID)
	for _, span := range spans[:2] {
		assert.Equal(t, spans[2].ID, span.ParentID)
		assert.Equal(t, spans[2].TraceID, span.TraceID)
	}

	assert.Equal(t, "INSERT INTO foo.bar", spans[0].Name)
	assert.Equal(t, "INSERT INTO foo.bar", spans[1].Name)
	assert.Equal(t, "BATCH", spans[2].Name)

	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type: "cassandra",
		},
	}, spans[2].Context)

	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "cassandra",
			Statement: "INSERT INTO foo.bar(id) VALUES(1)",
		},
	}, spans[0].Context)
}

func TestQueryObserverErrorIntegration(t *testing.T) {
	session := newSession(t)
	defer session.Close()

	var queryError error
	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		queryError = execQuery(ctx, session, "ZINGA")
	})
	require.Len(t, errors, 1)
	require.Len(t, spans, 1)

	assert.Equal(t, errors[0].Culprit, "execQuery")
	assert.EqualError(t, queryError, errors[0].Exception.Message)
}

func execQuery(ctx context.Context, session *gocql.Session, query string) error {
	return session.Query(query).WithContext(ctx).Exec()
}

func newSession(t *testing.T) *gocql.Session {
	if cassandraHost == "" {
		t.Skipf("CASSANDRA_HOST not specified")
	}
	observer := apmgocql.NewObserver()
	config := gocql.NewCluster(cassandraHost)
	config.QueryObserver = observer
	config.BatchObserver = observer
	session, err := config.CreateSession()
	require.NoError(t, err)
	return session
}
