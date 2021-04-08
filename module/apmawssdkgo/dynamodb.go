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
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"

	"go.elastic.co/apm"

	"github.com/aws/aws-sdk-go/aws/request"
)

type dynamoDB struct {
	TableName string
	// KeyConditionExpression is only available on Query operations.
	KeyConditionExpression string

	name, region string
}

func (d *dynamoDB) spanName() string {
	return d.name
}

func (d *dynamoDB) resource() string {
	return d.TableName
}

func (d *dynamoDB) setAdditional(span *apm.Span) {
	dbSpanCtx := apm.DatabaseSpanContext{
		Instance: d.region,
		Type:     serviceDynamoDB,
	}
	if span.Action == "Query" {
		dbSpanCtx.Statement = d.KeyConditionExpression
	}
	span.Context.SetDatabase(dbSpanCtx)
}

func newDynamoDB(req *request.Request) (*dynamoDB, error) {
	copyOfBody, values, err := readDynamoDBBody(req.HTTPRequest.Body)
	if err != nil {
		return nil, err
	}
	if r := req.Config.Region; r != nil {
		values.region = *r
	}

	req.HTTPRequest.Body = copyOfBody
	values.name = req.ClientInfo.ServiceID + " " + req.Operation.Name + " " + values.TableName
	return values, nil
}

// readDynamoDBBody reads the request body to parse out the TableName and
// KeyConditionExpression, then supply the http.Request with a copy of the
// original request body.
func readDynamoDBBody(r io.ReadCloser) (io.ReadCloser, *dynamoDB, error) {
	defer r.Close()

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	b := dynamoDB{}
	json.Unmarshal(body, &b)

	return ioutil.NopCloser(bytes.NewBuffer(body)), &b, nil
}
