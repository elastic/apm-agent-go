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

	"github.com/aws/aws-sdk-go/aws/request"
)

var (
	sqsErrMethodNotSupported = errors.New("method not supported")
	operationName            = map[string]string{
		"SendMessage":        "send",
		"SendMessageBatch":   "send_batch",
		"DeleteMessage":      "delete",
		"DeleteMessageBatch": "delete_batch",
		"ReceiveMessage":     "poll",
	}
)

type apmSQS struct {
	name, opName, resourceName string
}

func newSQS(req *request.Request) (*apmSQS, error) {
	opName, ok := operationName[req.Operation.Name]
	if !ok {
		return nil, sqsErrMethodNotSupported
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
	}

	return s, nil
}

func (s *apmSQS) spanName() string { return s.name }

func (s *apmSQS) resource() string { return s.resourceName }

func (s *apmSQS) setAdditional(span *apm.Span) {
	span.Action = s.opName
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
