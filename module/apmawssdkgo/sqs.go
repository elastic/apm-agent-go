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

package apmawssdkgo // import "go.elastic.co/apm/module/apmawssdkgo"

import (
	"errors"
	"strings"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	errMethodNotSupported = errors.New("method not supported")
	operationName         = map[string]string{
		"SendMessage":        "send",
		"SendMessageBatch":   "send_batch",
		"DeleteMessage":      "delete",
		"DeleteMessageBatch": "delete_batch",
		"ReceiveMessage":     "poll",
	}
)

type apmSQS struct {
	name, opName, resourceName, queueName string
}

func newSQS(req *request.Request) (*apmSQS, error) {
	opName, ok := operationName[req.Operation.Name]
	if !ok {
		return nil, errMethodNotSupported
	}
	name := req.ClientInfo.ServiceID + " " + strings.ToUpper(opName)
	resourceName := serviceSQS

	queueName := getQueueName(req)
	if queueName != "" {
		name += " " + operationDirection(req.Operation.Name) + " " + queueName
		resourceName += "/" + queueName
	}

	s := &apmSQS{
		name:         name,
		opName:       opName,
		resourceName: resourceName,
		queueName:    queueName,
	}

	return s, nil
}

func (s *apmSQS) spanName() string { return s.name }

func (s *apmSQS) resource() string { return s.resourceName }

func (s *apmSQS) setAdditional(span *apm.Span) {
	span.Action = s.opName
	if s.queueName != "" {
		span.Context.SetMessage(apm.MessageSpanContext{
			QueueName: s.queueName,
		})
	}
}

// addMessageAttributesSQS adds message attributes to `SendMessage` and
// `SendMessageBatch` RPC calls. Other SQS RPC calls are ignored.
func addMessageAttributesSQS(req *request.Request, span *apm.Span, propagateLegacyHeader bool) {
	switch req.Operation.Name {
	case "SendMessage", "SendMessageBatch":
		break
	default:
		return

	}

	traceContext := span.TraceContext()
	msgAttr := &sqs.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(apmhttp.FormatTraceparentHeader(traceContext)),
	}
	tracestate := traceContext.State.String()
	if req.Operation.Name == "SendMessage" {
		input, ok := req.Params.(*sqs.SendMessageInput)
		if !ok {
			return
		}
		setTracingAttributes(input.MessageAttributes, msgAttr, tracestate, propagateLegacyHeader)
	} else if req.Operation.Name == "SendMessageBatch" {
		input, ok := req.Params.(*sqs.SendMessageBatchInput)
		if !ok {
			return
		}
		for _, entry := range input.Entries {
			setTracingAttributes(entry.MessageAttributes, msgAttr, tracestate, propagateLegacyHeader)
		}
	}
}

func setTracingAttributes(
	attrs map[string]*sqs.MessageAttributeValue,
	value *sqs.MessageAttributeValue,
	tracestate string,
	propagateLegacyHeader bool,
) {
	attrs[apmhttp.W3CTraceparentHeader] = value
	if propagateLegacyHeader {
		attrs[apmhttp.ElasticTraceparentHeader] = value
	}
	if tracestate != "" {
		attrs[apmhttp.TracestateHeader] = &sqs.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(tracestate),
		}
	}
}

func supportedSQSMethod(req *request.Request) bool {
	_, ok := operationName[req.Operation.Name]
	return ok
}

func operationDirection(operationName string) string {
	switch operationName {
	case "SendMessage", "SendMessageBatch":
		return "to"
	default:
		return "from"
	}
}

func getQueueName(req *request.Request) string {
	parts := strings.Split(req.HTTPRequest.FormValue("QueueUrl"), "/")
	return parts[len(parts)-1]
}
