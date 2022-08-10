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

package apmawssdkgo // import "go.elastic.co/apm/module/apmawssdkgo/v2"

import (
	"strings"

	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sns"
)

type apmSNS struct {
	name, opName, resourceName, topicName string
}

func newSNS(req *request.Request) (*apmSNS, error) {
	if req.Operation.Name != "Publish" {
		return nil, errMethodNotSupported
	}
	name := req.ClientInfo.ServiceID + " PUBLISH"
	resourceName := serviceSNS

	topicName := getTopicName(req)
	if topicName != "" {
		name += " to " + topicName
		resourceName += "/" + topicName
	}

	s := &apmSNS{
		name:         name,
		opName:       "publish",
		resourceName: resourceName,
		topicName:    topicName,
	}

	return s, nil
}

func (s *apmSNS) spanName() string { return s.name }

func (s *apmSNS) resource() string { return s.resourceName }

func (s *apmSNS) targetName() string { return s.topicName }

func (s *apmSNS) setAdditional(span *apm.Span) {
	span.Action = s.opName
	// According to the spec:
	// Wherever the broker terminology uses "topic", this field will
	// contain the topic name.
	if s.topicName != "" {
		span.Context.SetMessage(apm.MessageSpanContext{
			QueueName: s.topicName,
		})
	}
}

func getTopicName(req *request.Request) string {
	// format: arn:aws:sns:us-east-2:123456789012:My-Topic
	// should return My-Topic
	if topicArn := req.HTTPRequest.FormValue("TopicArn"); topicArn != "" {
		idx := strings.LastIndex(topicArn, ":")
		if idx == -1 {
			return ""
		}

		// special check for format: arn:aws:sns:us-east-2:123456789012/MyTopic
		if slashIdx := strings.LastIndex(topicArn, "/"); slashIdx != -1 {
			return topicArn[slashIdx+1:]
		}

		return topicArn[idx+1:]
	}

	// format: arn:aws:sns:us-west-2:123456789012:endpoint/GCM/gcmpushapp/5e3e9847-3183-3f18-a7e8-671c3a57d4b3
	// should return endpoint/GCM/gcmpushapp
	if targetArn := req.HTTPRequest.FormValue("TargetArn"); targetArn != "" {
		idx := strings.LastIndex(targetArn, ":")
		if idx == -1 {
			return ""
		}

		endIdx := strings.LastIndex(targetArn, "/")
		if endIdx == -1 {
			return ""
		}

		return targetArn[idx+1 : endIdx]
	}

	// The actual phone number MUST NOT be included because it is PII and cardinality is too high.
	if phoneNumber := req.HTTPRequest.FormValue("PhoneNumber"); phoneNumber != "" {
		return "[PHONENUMBER]"
	}

	return ""
}

// addMessageAttributesSNS adds message attributes to `Publish` RPC calls.
// Other SNS RPC calls are ignored.
func addMessageAttributesSNS(req *request.Request, span *apm.Span, propagateLegacyHeader bool) {
	if req.Operation.Name != "Publish" {
		return
	}

	traceContext := span.TraceContext()
	msgAttr := &sns.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(apmhttp.FormatTraceparentHeader(traceContext)),
	}
	tracestate := traceContext.State.String()

	input, ok := req.Params.(*sns.PublishInput)
	if !ok {
		return
	}

	if input.MessageAttributes == nil {
		input.MessageAttributes = make(map[string]*sns.MessageAttributeValue)
	}
	input.MessageAttributes[apmhttp.W3CTraceparentHeader] = msgAttr
	if propagateLegacyHeader {
		input.MessageAttributes[apmhttp.ElasticTraceparentHeader] = msgAttr
	}
	if tracestate != "" {
		input.MessageAttributes[apmhttp.TracestateHeader] = &sns.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(tracestate),
		}
	}
}

func supportedSNSMethod(req *request.Request) bool {
	return req.Operation.Name == "Publish"
}
