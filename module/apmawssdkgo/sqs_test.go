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
	"context"
	"testing"

	"go.elastic.co/apm/apmtest"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQS(t *testing.T) {
	region := "us-west-2"
	addr := "sqs.testing.invalid"
	cfg := aws.NewConfig().
		WithEndpoint(addr).
		WithRegion(region).
		WithDisableSSL(true).
		WithCredentials(credentials.AnonymousCredentials)

	session := session.Must(session.NewSession(cfg))
	wrapped := WrapSession(session)
	svc := sqs.New(wrapped)
	for _, tc := range []struct {
		fn                     func(context.Context, string)
		name, action, resource string
		queueURL               string
		ignored                bool
	}{
		{
			name:     "SQS POLL from MyQueue",
			action:   "poll",
			resource: "sqs/MyQueue",
			queueURL: "https://sqs.testing.invalid/123456789012/MyQueue",
			fn: func(ctx context.Context, queueURL string) {
				svc.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
					QueueUrl: &queueURL,
					AttributeNames: aws.StringSlice([]string{
						"SentTimestamp",
					}),
					MaxNumberOfMessages: aws.Int64(1),
					MessageAttributeNames: aws.StringSlice([]string{
						"All",
					}),
					WaitTimeSeconds: aws.Int64(1),
				})
			},
		},
		{
			name:     "SQS SEND to OtherQueue",
			action:   "send",
			resource: "sqs/OtherQueue",
			queueURL: "https://sqs.testing.invalid/123456789012/OtherQueue",
			fn: func(ctx context.Context, queueURL string) {
				svc.SendMessageWithContext(ctx, &sqs.SendMessageInput{
					QueueUrl: &queueURL,
					MessageAttributes: map[string]*sqs.MessageAttributeValue{
						"attr": &sqs.MessageAttributeValue{
							DataType:    aws.String("String"),
							StringValue: aws.String("string attr"),
						},
					},
					MessageBody: aws.String("msg body"),
				})
			},
		},
		{
			name:     "SQS SEND_BATCH to OtherQueue",
			action:   "send_batch",
			resource: "sqs/OtherQueue",
			queueURL: "https://sqs.testing.invalid/123456789012/OtherQueue",
			fn: func(ctx context.Context, queueURL string) {
				svc.SendMessageBatchWithContext(ctx, &sqs.SendMessageBatchInput{
					QueueUrl: &queueURL,
					Entries: []*sqs.SendMessageBatchRequestEntry{
						{
							Id: aws.String("1"),
							MessageAttributes: map[string]*sqs.MessageAttributeValue{
								"attr": &sqs.MessageAttributeValue{
									DataType:    aws.String("String"),
									StringValue: aws.String("string attr"),
								},
							},
							MessageBody: aws.String("msg body"),
						},
					},
				})
			},
		},
		{
			name:     "SQS DELETE from ThatQueue",
			action:   "delete",
			resource: "sqs/ThatQueue",
			queueURL: "https://sqs.testing.invalid/123456789012/ThatQueue",
			fn: func(ctx context.Context, queueURL string) {
				svc.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      &queueURL,
					ReceiptHandle: aws.String("receiptHandle"),
				})
			},
		},
		{
			name:     "SQS DELETE from ThatQueue",
			action:   "delete",
			resource: "sqs/ThatQueue",
			ignored:  true,
			fn: func(ctx context.Context, _ string) {
				svc.CreateQueueWithContext(ctx, &sqs.CreateQueueInput{
					QueueName: aws.String("SQS_QUEUE_NAME"),
					Attributes: map[string]*string{
						"DelaySeconds":           aws.String("60"),
						"MessageRetentionPeriod": aws.String("86400"),
					},
				})
			},
		},
	} {
		tx, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
			tc.fn(ctx, tc.queueURL)
		})

		if tc.ignored {
			require.Len(t, spans, 0)
			require.Len(t, errors, 0)
			return
		}

		require.Len(t, spans, 1)
		require.Len(t, errors, 1)

		span := spans[0]
		err := errors[0]

		assert.Equal(t, tx.ID, err.TransactionID)
		assert.Equal(t, span.ID, err.ParentID)

		assert.Equal(t, tc.name, span.Name)
		assert.Equal(t, "messaging", span.Type)
		assert.Equal(t, "sqs", span.Subtype)
		assert.Equal(t, tc.action, span.Action)

		service := span.Context.Destination.Service
		assert.Equal(t, "sqs", service.Name)
		assert.Equal(t, "messaging", service.Type)
		assert.Equal(t, tc.resource, service.Resource)
		assert.Equal(t, "sqs.testing.invalid", span.Context.Destination.Address)

		assert.Equal(t, region, span.Context.Destination.Cloud.Region)

		assert.Equal(t, tx.ID, span.ParentID)
	}
}
