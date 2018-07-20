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

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/module/apmgocql"
	"github.com/elastic/apm-agent-go/transport/transporttest"
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
	tx, errors := withTransaction(t, func(ctx context.Context) {
		start = time.Now()
		observer.ObserveQuery(ctx, gocql.ObservedQuery{
			Start:     start,
			End:       start.Add(3 * time.Second),
			Keyspace:  "quay ",
			Statement: "SELECT * FROM foo.bar",
			Err:       errors.New("baz"),
		})
	})

	require.Len(t, tx.Spans, 1)
	assert.Equal(t, "db.cassandra.query", tx.Spans[0].Type)
	assert.Equal(t, "SELECT FROM foo.bar", tx.Spans[0].Name)
	assert.WithinDuration(t,
		time.Time(tx.Timestamp).Add(time.Duration((tx.Spans[0].Start+tx.Spans[0].Duration)*1000000)),
		start.Add(3*time.Second),
		100*time.Millisecond, // allow some leeway for slow systems
	)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "cassandra",
			Instance:  "quay ",
			Statement: "SELECT * FROM foo.bar",
		},
	}, tx.Spans[0].Context)

	require.Len(t, errors, 1)
	assert.Equal(t, "TestQueryObserver.func1", errors[0].Culprit)
}

func TestBatchObserver(t *testing.T) {
	var start time.Time
	observer := apmgocql.NewObserver()
	tx, errors := withTransaction(t, func(ctx context.Context) {
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

	require.Len(t, tx.Spans, 3)
	assert.Equal(t, "db.cassandra.batch", tx.Spans[0].Type)
	assert.Equal(t, "db.cassandra.query", tx.Spans[1].Type)
	assert.Equal(t, "db.cassandra.query", tx.Spans[2].Type)
	assert.Nil(t, tx.Spans[0].Parent)
	assert.Equal(t, tx.Spans[0].ID, tx.Spans[1].Parent)
	assert.Equal(t, tx.Spans[0].ID, tx.Spans[2].Parent)

	assert.Equal(t, "BATCH", tx.Spans[0].Name)
	assert.Equal(t, "INSERT INTO foo.bar", tx.Spans[1].Name)
	assert.Equal(t, "UPDATE foo.bar", tx.Spans[2].Name)

	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:     "cassandra",
			Instance: "quay ",
		},
	}, tx.Spans[0].Context)

	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "cassandra",
			Instance:  "quay ",
			Statement: "INSERT INTO foo.bar(id) VALUES(1)",
		},
	}, tx.Spans[1].Context)

	require.Len(t, errors, 1)
	assert.Equal(t, "TestBatchObserver.func1", errors[0].Culprit)
}

func TestQueryObserverIntegration(t *testing.T) {
	session := newSession(t)
	defer session.Close()

	tx, _ := withTransaction(t, func(ctx context.Context) {
		err := execQuery(ctx, session, createKeyspaceStatement)
		assert.NoError(t, err)

		err = execQuery(ctx, session, `CREATE TABLE IF NOT EXISTS foo.bar (id int, PRIMARY KEY(id));`)
		assert.NoError(t, err)

		err = execQuery(ctx, session, "INSERT INTO foo.bar(id) VALUES(1)")
		assert.NoError(t, err)
	})

	require.Len(t, tx.Spans, 3)
	for _, span := range tx.Spans {
		assert.Equal(t, "db.cassandra.query", span.Type)
	}
	assert.Equal(t, "db.cassandra.query", tx.Spans[1].Type)
	assert.Equal(t, "CREATE", tx.Spans[0].Name)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "cassandra",
			Statement: createKeyspaceStatement,
		},
	}, tx.Spans[0].Context)
	assert.Equal(t, "CREATE", tx.Spans[1].Name)
	assert.Equal(t, "INSERT INTO foo.bar", tx.Spans[2].Name)
}

func TestBatchObserverIntegration(t *testing.T) {
	session := newSession(t)
	defer session.Close()

	err := execQuery(context.Background(), session, createKeyspaceStatement)
	assert.NoError(t, err)

	err = execQuery(context.Background(), session, `CREATE TABLE IF NOT EXISTS foo.bar (id int, PRIMARY KEY(id));`)
	assert.NoError(t, err)

	tx, _ := withTransaction(t, func(ctx context.Context) {
		batch := session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
		batch.Query("INSERT INTO foo.bar(id) VALUES(1)")
		batch.Query("INSERT INTO foo.bar(id) VALUES(2)")
		err := session.ExecuteBatch(batch)
		assert.NoError(t, err)
	})

	require.Len(t, tx.Spans, 3)
	assert.Equal(t, "db.cassandra.batch", tx.Spans[0].Type)
	assert.Equal(t, "db.cassandra.query", tx.Spans[1].Type)
	assert.Equal(t, "db.cassandra.query", tx.Spans[2].Type)
	assert.Nil(t, tx.Spans[0].Parent)
	assert.Equal(t, tx.Spans[0].ID, tx.Spans[1].Parent)
	assert.Equal(t, tx.Spans[0].ID, tx.Spans[2].Parent)

	assert.Equal(t, "BATCH", tx.Spans[0].Name)
	assert.Equal(t, "INSERT INTO foo.bar", tx.Spans[1].Name)
	assert.Equal(t, "INSERT INTO foo.bar", tx.Spans[2].Name)

	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type: "cassandra",
		},
	}, tx.Spans[0].Context)

	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "cassandra",
			Statement: "INSERT INTO foo.bar(id) VALUES(1)",
		},
	}, tx.Spans[1].Context)
}

func TestQueryObserverErrorIntegration(t *testing.T) {
	session := newSession(t)
	defer session.Close()

	var queryError error
	tx, errors := withTransaction(t, func(ctx context.Context) {
		queryError = execQuery(ctx, session, "ZINGA")
	})
	require.Len(t, errors, 1)
	require.Len(t, tx.Spans, 1)
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

func withTransaction(t *testing.T, f func(context.Context)) (model.Transaction, []*model.Error) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	f(elasticapm.ContextWithTransaction(context.Background(), tx))
	tx.End()

	tracer.Flush(nil)
	payloads := transport.Payloads()
	var errors []*model.Error
	if len(payloads) == 2 {
		errors = payloads[0].Errors()
	}
	transactions := payloads[len(payloads)-1].Transactions()
	require.Len(t, transactions, 1)
	return transactions[0], errors
}
