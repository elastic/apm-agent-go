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

package apmmongo // import "go.elastic.co/apm/module/apmmongo"

import (
	"context"
	"reflect"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"

	"go.elastic.co/apm"
)

var (
	extjPool = bsonrw.NewExtJSONValueWriterPool()
	swPool   = sync.Pool{
		New: func() interface{} {
			return &bsonrw.SliceWriter{}
		},
	}
)

func init() {
	apm.RegisterTypeErrorDetailer(
		reflect.TypeOf(mongo.CommandError{}),
		apm.ErrorDetailerFunc(func(err error, details *apm.ErrorDetails) {
			commandErr := err.(mongo.CommandError)
			details.Code.String = commandErr.Name
			if len(commandErr.Labels) > 0 {
				details.SetAttr("labels", commandErr.Labels)
			}
		}),
	)
}

// CommandMonitor returns a new event.CommandMonitor which will report a span
// for each command executed within a context containing a sampled transaction.
func CommandMonitor(opts ...Option) *event.CommandMonitor {
	cm := commandMonitor{
		bsonRegistry: bson.DefaultRegistry,
		spans:        make(map[commandKey]*apm.Span),
	}
	for _, o := range opts {
		o(&cm)
	}
	return &event.CommandMonitor{
		Started:   cm.started,
		Succeeded: cm.succeeded,
		Failed:    cm.failed,
	}
}

type commandMonitor struct {
	// TODO(axw) record number of active commands and report as a
	// metric so users can, for example, identify unclosed cursors.
	bsonRegistry *bsoncodec.Registry

	mu    sync.Mutex
	spans map[commandKey]*apm.Span
}

type commandKey struct {
	connectionID string
	requestID    int64
}

func (c *commandMonitor) started(ctx context.Context, event *event.CommandStartedEvent) {
	spanName := event.CommandName
	if collectionName, ok := collectionName(event.CommandName, event.Command); ok {
		spanName = collectionName + "." + spanName
	}
	span, _ := apm.StartSpan(ctx, spanName, "db.mongodb.query")
	if span.Dropped() {
		return
	}

	var statement string
	if len(event.Command) > 0 {
		// Encode the command as MongoDB Extended JSON
		// for the "statement" in database span context.
		sw := swPool.Get().(*bsonrw.SliceWriter)
		ejvw := extjPool.Get(sw, false /* non-canonical */, false /* don't escape HTML */)
		ec := bsoncodec.EncodeContext{Registry: c.bsonRegistry}
		if enc, err := bson.NewEncoderWithContext(ec, ejvw); err == nil {
			if err := enc.Encode(event.Command); err == nil {
				statement = string(*sw)
			}
		}
		*sw = (*sw)[:0]
		extjPool.Put(ejvw)
		swPool.Put(sw)
	}

	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Instance:  event.DatabaseName,
		Type:      "mongodb",
		Statement: statement,
	})

	// The command/event monitoring API does not provide a means of associating
	// arbitrary data with a request, so we must maintain our own map.
	//
	// https://jira.mongodb.org/browse/GODRIVER-837
	key := commandKey{connectionID: event.ConnectionID, requestID: event.RequestID}
	c.mu.Lock()
	c.spans[key] = span
	c.mu.Unlock()
}

func (c *commandMonitor) succeeded(ctx context.Context, event *event.CommandSucceededEvent) {
	c.finished(ctx, &event.CommandFinishedEvent)
}

func (c *commandMonitor) failed(ctx context.Context, event *event.CommandFailedEvent) {
	c.finished(ctx, &event.CommandFinishedEvent)
}

func (c *commandMonitor) finished(ctx context.Context, event *event.CommandFinishedEvent) {
	key := commandKey{connectionID: event.ConnectionID, requestID: event.RequestID}

	c.mu.Lock()
	span, ok := c.spans[key]
	if !ok {
		c.mu.Unlock()
		return
	}
	delete(c.spans, key)
	c.mu.Unlock()

	span.Duration = time.Duration(event.DurationNanos)
	span.End()
}

func collectionName(commandName string, command bson.Raw) (string, bool) {
	switch commandName {
	case
		// Aggregation Commands
		"aggregate",
		"count",
		"distinct",
		"mapReduce",

		// Geospatial Commands
		"geoNear",
		"geoSearch",

		// Query and Write Operation Commands
		"delete",
		"find",
		"findAndModify",
		"insert",
		"parallelCollectionScan",
		"update",

		// Administration Commands
		"compact",
		"convertToCapped",
		"create",
		"createIndexes",
		"drop",
		"dropIndexes",
		"killCursors",
		"listIndexes",
		"reIndex",

		// Diagnostic Commands
		"collStats":

		collectionValue := command.Lookup(commandName)
		return collectionValue.StringValueOK()
	case "getMore":
		collectionValue := command.Lookup("collection")
		return collectionValue.StringValueOK()
	}
	return "", false
}

// Option sets options for tracing MongoDB commands.
type Option func(*commandMonitor)
