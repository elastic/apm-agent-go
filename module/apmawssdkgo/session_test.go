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

//go:build go1.13
// +build go1.13

package apmawssdkgo // import "go.elastic.co/apm/module/apmawssdkgo"

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
)

func TestS3(t *testing.T) {
	region := "us-west-2"
	addr := "s3.testing.invalid"
	cfg := aws.NewConfig().
		WithEndpoint(addr).
		WithRegion(region).
		WithDisableSSL(true).
		WithCredentials(credentials.AnonymousCredentials)

	session := WrapSession(session.Must(session.NewSession(cfg)))

	// A bucket name in uppercase forces the S3 path style, but lowercase
	// will use the virtual bucket style. Check that both are properly
	// parsed in the handlers.
	// https://github.com/aws/aws-sdk-go/blob/d9428afe4490/service/s3/host_style_bucket.go#L107-L130
	for _, tc := range []struct {
		addr, bucketName string
	}{
		{addr, "BUCKET"},
		{"bucket." + addr, "bucket"},
	} {
		tx, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
			uploader := s3manager.NewUploader(session)
			uploader.UploadWithContext(ctx, &s3manager.UploadInput{
				Bucket: aws.String(tc.bucketName),
				Key:    aws.String("some key"),
				Body:   bytes.NewBuffer([]byte("some random body")),
			})
		})
		require.Len(t, errors, 1)
		require.Len(t, spans, 1)

		err := errors[0]
		span := spans[0]

		assert.Equal(t, tx.ID, err.TransactionID)
		assert.Equal(t, span.ID, err.ParentID)

		assert.Equal(t, "S3 PutObject "+tc.bucketName, span.Name)
		assert.Equal(t, "storage", span.Type)
		assert.Equal(t, "s3", span.Subtype)
		assert.Equal(t, "PutObject", span.Action)

		service := span.Context.Destination.Service
		assert.Equal(t, "s3", service.Name)
		assert.Equal(t, tc.bucketName, service.Resource)
		assert.Equal(t, "storage", service.Type)
		assert.Equal(t, tc.addr, span.Context.Destination.Address)

		require.NotNil(t, span.Context.Destination.Cloud)
		assert.Equal(t, region, span.Context.Destination.Cloud.Region)

		assert.Equal(t, tx.ID, span.ParentID)
	}
}

func TestDynamoDB(t *testing.T) {
	region := "us-west-2"
	cfg := aws.NewConfig().
		WithRegion(region).
		WithDisableSSL(true).
		WithCredentials(credentials.AnonymousCredentials)

	session := session.Must(session.NewSession(cfg))
	wrapped := WrapSession(session)
	svc := dynamodb.New(wrapped)

	for _, tc := range []struct {
		fn                      func(context.Context)
		name, action, statement string
	}{
		{
			name:      "DynamoDB Query Music",
			statement: "Artist = :v1",
			action:    "Query",
			fn: func(ctx context.Context) {
				input := &dynamodb.QueryInput{
					ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
						":v1": {S: aws.String("No One You Know")},
					},
					KeyConditionExpression: aws.String("Artist = :v1"),
					TableName:              aws.String("Music"),
				}
				svc.QueryWithContext(ctx, input)
			},
		},
		{
			name:      "DynamoDB GetItem Movies",
			statement: "",
			action:    "GetItem",
			fn: func(ctx context.Context) {
				input := &dynamodb.GetItemInput{
					TableName: aws.String("Movies"),
					Key: map[string]*dynamodb.AttributeValue{
						"Year":  {N: aws.String("2015")},
						"Title": {S: aws.String("The Big New Movie")},
					},
				}
				svc.GetItemWithContext(ctx, input)
			},
		},
	} {
		tx, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
			tc.fn(ctx)
		})

		require.Len(t, spans, 1)
		require.Len(t, errors, 1)

		span := spans[0]
		err := errors[0]

		assert.Equal(t, tx.ID, err.TransactionID)
		assert.Equal(t, span.ID, err.ParentID)

		assert.Equal(t, tc.name, span.Name)
		assert.Equal(t, "db", span.Type)
		assert.Equal(t, "dynamodb", span.Subtype)
		assert.Equal(t, tc.action, span.Action)

		service := span.Context.Destination.Service
		assert.Equal(t, "dynamodb", service.Name)
		assert.Equal(t, "db", service.Type)
		assert.Equal(t, "dynamodb.us-west-2.amazonaws.com", span.Context.Destination.Address)

		db := span.Context.Database
		assert.Equal(t, region, db.Instance)
		assert.Equal(t, tc.statement, db.Statement)
		assert.Equal(t, "dynamodb", db.Type)

		assert.Equal(t, region, span.Context.Destination.Cloud.Region)

		assert.Equal(t, tx.ID, span.ParentID)
	}

}

func TestUnsupportedServices(t *testing.T) {
	region := "us-west-2"
	cfg := aws.NewConfig().
		WithRegion(region).
		WithDisableSSL(true).
		WithCredentials(credentials.AnonymousCredentials)

	session := session.Must(session.NewSession(cfg))
	wrapped := WrapSession(session)
	svc := athena.New(wrapped)

	tx := apm.DefaultTracer.StartTransaction("send-email", "test-tx")
	span := tx.StartSpan("test-span", "send-email", nil)
	defer span.End()

	ctx := apm.ContextWithSpan(context.Background(), span)
	namedQuery := &athena.BatchGetNamedQueryInput{
		NamedQueryIds: []*string{aws.String("query")},
	}
	// Setting a timeout so the request fails fast. This doesn't affect
	// testing that we noop on unsupported services.
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond)
	defer cancel()

	svc.BatchGetNamedQueryWithContext(ctx, namedQuery)
	assert.NotPanics(t, func() { svc.BatchGetNamedQueryWithContext(ctx, namedQuery) })
}
