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

// +build go1.13

package apmawssdkgo // import "go.elastic.co/apm/module/apmawssdkgo"

import (
	"bytes"
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	spanSubtype := "s3"
	spanType := serviceTypeMap[spanSubtype]

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
		assert.Equal(t, spanType, span.Type)
		assert.Equal(t, spanSubtype, span.Subtype)
		assert.Equal(t, "PutObject", span.Action)

		service := span.Context.Destination.Service
		assert.Equal(t, "s3", service.Name)
		assert.Equal(t, tc.bucketName, service.Resource)
		assert.Equal(t, spanType, service.Type)
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

	spanSubtype := "dynamodb"
	spanType := serviceTypeMap[spanSubtype]

	session := session.Must(session.NewSession(cfg))
	wrapped := WrapSession(session)
	svc := dynamodb.New(wrapped)

	tx, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		input := &dynamodb.QueryInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":v1": {
					S: aws.String("No One You Know"),
				},
			},
			KeyConditionExpression: aws.String("Artist = :v1"),
			TableName:              aws.String("Music"),
		}
		svc.QueryWithContext(ctx, input)
	})

	require.Len(t, spans, 1)
	require.Len(t, errors, 1)

	span := spans[0]
	assert.Equal(t, "DynamoDB Query Music", span.Name)
	assert.Equal(t, spanType, span.Type)
	assert.Equal(t, spanSubtype, span.Subtype)
	assert.Equal(t, "Query", span.Action)

	service := span.Context.Destination.Service
	assert.Equal(t, "dynamodb", service.Name)
	assert.Equal(t, spanType, service.Type)
	assert.Equal(t, "dynamodb.us-west-2.amazonaws.com", span.Context.Destination.Address)

	db := span.Context.Database
	assert.Equal(t, region, db.Instance)
	// For a Query operation, check the body to see if it's available there.
	assert.Equal(t, "Artist = :v1", db.Statement)
	// assert.Equal(t, "anon", db.User)
	assert.Equal(t, "dynamodb", db.Type)

	assert.Equal(t, region, span.Context.Destination.Cloud.Region)

	assert.Equal(t, tx.ID, span.ParentID)
}
