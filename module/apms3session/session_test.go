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

package apms3session // import "go.elastic.co/apm/module/apms3session"

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
)

func TestSession(t *testing.T) {
	region := "us-west-2"
	cfg := aws.NewConfig().
		WithRegion(region).
		WithDisableSSL(true).
		WithCredentials(credentials.AnonymousCredentials)

	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)

	session := session.Must(session.NewSession(cfg))
	s3api := s3.New(WrapSession(session, WithTracer(tracer.Tracer)))

	bucketName := "BUCKET"
	s3api.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})

	tx.End()
	tracer.Flush(nil)

	payloads := tracer.Payloads()
	require.Len(t, payloads.Spans, 1)

	span := payloads.Spans[0]
	assert.Equal(t, "s3 CreateBucket BUCKET", span.Name)
	assert.Equal(t, spanType, span.Type)
	assert.Equal(t, spanSubtype, span.Subtype)
	assert.Equal(t, "CreateBucket", span.Action)

	service := span.Context.Destination.Service
	assert.Equal(t, "s3", service.Name)
	assert.Equal(t, bucketName, service.Resource)
	assert.Equal(t, spanType, service.Type)
	assert.Equal(t, "http://s3.us-west-2.amazonaws.com", span.Context.Destination.Address)

	require.NotNil(t, span.Context.Destination.Cloud)
	assert.Equal(t, region, span.Context.Destination.Cloud.Region)

	require.Len(t, payloads.Transactions, 1)
	transaction := payloads.Transactions[0]
	assert.Equal(t, transaction.ID, span.ParentID)
}
