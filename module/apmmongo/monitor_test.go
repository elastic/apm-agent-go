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

package apmmongo_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmmongo"
)

func TestCommandMonitorSpanNames(t *testing.T) {
	test := func(
		commandName string, command interface{},
		expectedSpanName, expectedStatement string,
	) {
		cm := apmmongo.CommandMonitor()
		_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
			cm.Started(ctx, &event.CommandStartedEvent{
				DatabaseName: "test_db",
				CommandName:  commandName,
				RequestID:    42,
				ConnectionID: "rainbow",
				Command:      mustRawBSON(command),
			})
			cm.Succeeded(ctx, &event.CommandSucceededEvent{
				CommandFinishedEvent: event.CommandFinishedEvent{
					CommandName:  commandName,
					RequestID:    42,
					ConnectionID: "rainbow",
				},
			})
		})
		require.Len(t, spans, 1)
		assert.Equal(t, expectedSpanName, spans[0].Name)
		assert.Equal(t, expectedStatement, spans[0].Context.Database.Statement)
	}

	test("update", bson.D{
		{Key: "update", Value: "users"},
		{Key: "updates", Value: bson.A{bson.D{
			{Key: "q", Value: bson.D{}},
			{Key: "u", Value: bson.D{
				{Key: "$set", Value: bson.D{{Key: "status", Value: "A"}}},
				{Key: "$inc", Value: bson.D{{Key: "points", Value: 1}}}}},
			{Key: "multi", Value: true},
		}}},
		{Key: "ordered", Value: false},
		{Key: "writeConcern", Value: bson.D{
			{Key: "w", Value: "majority"},
			{Key: "wtimeout", Value: 500},
		}},
	},
		"users.update",
		`{"update":"users","updates":[{"q":{},"u":{"$set":{"status":"A"},"$inc":{"points":1}},"multi":true}],"ordered":false,"writeConcern":{"w":"majority","wtimeout":500}}`,
	)

	test("aggregate", bson.D{
		{Key: "aggregate", Value: 1},
		{Key: "pipeline", Value: bson.A{}},
	}, "aggregate", `{"aggregate":1,"pipeline":[]}`)

	test("aggregate", bson.D{
		{Key: "aggregate", Value: "foo"},
		{Key: "pipeline", Value: bson.A{}},
	}, "foo.aggregate", `{"aggregate":"foo","pipeline":[]}`)

	test("getMore", bson.D{
		{Key: "getMore", Value: 123},
		{Key: "collection", Value: "foo"},
	}, "foo.getMore", `{"getMore":123,"collection":"foo"}`)
}

func TestCommandMonitorSucceeded(t *testing.T) {
	testCommandMonitorFinished(t, "")
}

func TestCommandMonitorFailed(t *testing.T) {
	testCommandMonitorFinished(t, "bad things happened")
}

func testCommandMonitorFinished(t *testing.T, failure string) {
	cm := apmmongo.CommandMonitor()
	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		cm.Started(ctx, &event.CommandStartedEvent{
			DatabaseName: "test_db",
			CommandName:  "find",
			RequestID:    42,
			ConnectionID: "rainbow",
			Command:      mustRawBSON(bson.D{{Key: "find", Value: "test_coll"}}),
		})
		finished := event.CommandFinishedEvent{
			DurationNanos: int64(123 * time.Millisecond),
			CommandName:   "find",
			RequestID:     42,
			ConnectionID:  "rainbow",
		}
		if failure == "" {
			cm.Succeeded(ctx, &event.CommandSucceededEvent{
				CommandFinishedEvent: finished,
			})
		} else {
			cm.Failed(ctx, &event.CommandFailedEvent{
				CommandFinishedEvent: finished,
				Failure:              failure,
			})
		}
	})

	// We don't report errors, as they may be expected by the application.
	assert.Empty(t, errs)
	require.Len(t, spans, 1)

	assert.Equal(t, "test_coll.find", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "mongodb", spans[0].Subtype)
	assert.Equal(t, "query", spans[0].Action)
	assert.Equal(t, 123.0, spans[0].Duration)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Instance:  "test_db",
			Type:      "mongodb",
			Statement: `{"find":"test_coll"}`,
		},
	}, spans[0].Context)
}

func TestCommandMonitorStartedNotFinished(t *testing.T) {
	cm := apmmongo.CommandMonitor()
	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		cm.Started(ctx, &event.CommandStartedEvent{
			DatabaseName: "test_db",
			CommandName:  "find",
			RequestID:    42,
			ConnectionID: "rainbow",
		})
	})
	assert.Empty(t, spans)
	assert.Empty(t, errs)
}

func TestCommandMonitorFinishedNotStarted(t *testing.T) {
	cm := apmmongo.CommandMonitor()
	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		cm.Failed(ctx, &event.CommandFailedEvent{
			CommandFinishedEvent: event.CommandFinishedEvent{
				DurationNanos: int64(123 * time.Millisecond),
				CommandName:   "find",
				RequestID:     42,
				ConnectionID:  "rainbow",
			},
			Failure: "buhbow",
		})
	})
	assert.Empty(t, spans)
	assert.Empty(t, errs)
}

func TestCommandErrorDetails(t *testing.T) {
	_, _, errs := apmtest.WithTransaction(func(ctx context.Context) {
		apm.CaptureError(ctx, mongo.CommandError{
			Code:    11,
			Name:    "UserNotFound",
			Message: "Robert'); DROP TABLE Students;-- not found",
			Labels:  []string{"black", "blue", "red"},
		}).Send()
	})
	require.Len(t, errs, 1)

	errs[0].Exception.Stacktrace = nil
	assert.Equal(t, model.Exception{
		Message: `(UserNotFound) Robert'); DROP TABLE Students;-- not found`,
		Type:    "CommandError",
		Module:  "go.mongodb.org/mongo-driver/mongo",
		Code:    model.ExceptionCode{String: "UserNotFound"},
		Handled: true,
		Attributes: map[string]interface{}{
			"labels": []interface{}{"black", "blue", "red"},
		},
	}, errs[0].Exception)
}

func mustRawBSON(val interface{}) bson.Raw {
	out, err := bson.Marshal(val)
	if err != nil {
		panic(err)
	}
	return bson.Raw(out)
}
