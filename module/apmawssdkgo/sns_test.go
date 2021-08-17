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
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmhttp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSNS(t *testing.T) {
	for _, tc := range []struct {
		fn                                 func(context.Context, *sns.SNS)
		name, action, resource, topicName  string
		ignored, hasTraceContext, hasError bool
	}{
		{
			name:            "SNS PUBLISH myTopic",
			action:          "publish",
			resource:        "sns/myTopic",
			topicName:       "myTopic",
			hasError:        true,
			hasTraceContext: true,
			fn: func(ctx context.Context, svc *sns.SNS) {
				svc.PublishWithContext(ctx, &sns.PublishInput{
					Message:  aws.String("my message"),
					TopicArn: aws.String("arn:aws:sns:us-east-2:123456789012/myTopic"),
				})
			},
		},
		{
			name:            "SNS PUBLISH myTopic",
			action:          "publish",
			resource:        "sns/myTopic",
			topicName:       "myTopic",
			hasTraceContext: true,
			fn: func(ctx context.Context, svc *sns.SNS) {
				svc.PublishWithContext(ctx, &sns.PublishInput{
					Message:  aws.String("my message"),
					TopicArn: aws.String("arn:aws:sns:us-east-2:123456789012:myTopic"),
				})
			},
		},
		{
			ignored: true,
			fn: func(ctx context.Context, svc *sns.SNS) {
				svc.SubscribeWithContext(ctx, &sns.SubscribeInput{
					Endpoint:              aws.String("endpoint"),
					Protocol:              aws.String("email"),
					ReturnSubscriptionArn: aws.Bool(true),
					TopicArn:              aws.String("arn:aws:sns:us-east-2:123456789012:myTopic"),
				})
			},
		},
	} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tc.hasError {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}))
		defer ts.Close()

		region := "us-west-2"
		cfg := aws.NewConfig().
			WithEndpoint(ts.URL).
			WithRegion(region).
			WithDisableSSL(true).
			WithCredentials(credentials.AnonymousCredentials)

		session := session.Must(session.NewSession(cfg))
		wrapped := WrapSession(session)
		if tc.hasTraceContext {
			wrapped.Handlers.Build.PushBackNamed(request.NamedHandler{
				Name: "spy_message_attrs_added",
				Fn:   testTracingAttributesSNS(t),
			})
		}

		svc := sns.New(wrapped)

		tx, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
			tc.fn(ctx, svc)
		})

		if tc.ignored {
			require.Len(t, spans, 0)
			require.Len(t, errors, 0)
			return
		}

		require.Len(t, spans, 1)
		span := spans[0]

		if tc.hasError {
			require.Len(t, errors, 1)
			err := errors[0]
			assert.Equal(t, tx.ID, err.TransactionID)
			assert.Equal(t, span.ID, err.ParentID)
		}

		assert.Equal(t, tc.name, span.Name)
		assert.Equal(t, "messaging", span.Type)
		assert.Equal(t, "sns", span.Subtype)
		assert.Equal(t, tc.action, span.Action)

		service := span.Context.Destination.Service
		assert.Equal(t, "sns", service.Name)
		assert.Equal(t, "messaging", service.Type)
		assert.Equal(t, tc.resource, service.Resource)

		queue := span.Context.Message.Queue
		assert.Equal(t, tc.topicName, queue.Name)

		host, port, err := net.SplitHostPort(ts.URL[7:])
		require.NoError(t, err)
		assert.Equal(t, host, span.Context.Destination.Address)
		assert.Equal(t, port, strconv.Itoa(span.Context.Destination.Port))

		assert.Equal(t, region, span.Context.Destination.Cloud.Region)

		assert.Equal(t, tx.ID, span.ParentID)
	}
}

func testTracingAttributesSNS(t *testing.T) func(*request.Request) {
	return func(req *request.Request) {
		if req.Operation.Name != "Publish" {
			t.Fail()
		}

		input, ok := req.Params.(*sns.PublishInput)
		require.True(t, ok)
		attrs := input.MessageAttributes
		assert.Contains(t, attrs, apmhttp.W3CTraceparentHeader)
		assert.Contains(t, attrs, apmhttp.ElasticTraceparentHeader)
		assert.Contains(t, attrs, apmhttp.TracestateHeader)
	}
}
